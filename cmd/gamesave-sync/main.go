package main

import (
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/ini.v1"

	"github.com/chenjianlong/gamesave-sync/pkg/gsutils"
	"github.com/chenjianlong/gamesave-sync/pkg/i18n"
	"github.com/chenjianlong/gamesave-sync/pkg/transfer"
	"github.com/chenjianlong/gamesave-sync/pkg/ziputils"
	"github.com/jeandeaual/go-locale"
	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
)

const AppName = "GameSaveSyncing"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var args struct {
		Path string `arg:"-p" default:"config.ini" help:"config path"`
	}

	arg.MustParse(&args)

	loc, err := locale.GetLocale()
	gsutils.CheckError(err)
	i18n.InitBundle(loc)

	transfer := newTransfer(args.Path)
	appData := getAppdata()
	hasMonitor := false
	for _, info := range LoadGameList("conf.d/") {
		log.Println(i18n.GetSyncGameMessage(info.Name))
		p := info.Dir
		valid, _ := gsutils.IsDir(p)
		if !valid {
			log.Printf("%s not exist\n", p)
			continue
		}

		localGameSaveTime := getLocalGameSaveTime(info.Dir)
		downloadObjName, needUpload := getDownloadName(transfer, localGameSaveTime, info.Name+"/")
		log.Printf("Game: %s, needUpload: %v, downloadObject: %s\n", info.Name, needUpload, downloadObjName)
		zipPath := filepath.Join(appData, info.Name+".zip")
		if needUpload && localGameSaveTime != nil {
			objName := path.Join(info.Name, localGameSaveTime.UTC().Format(gsutils.TimeFormat)+".zip")
			uploadGameSave(transfer, p, zipPath, objName)
		}

		if downloadObjName != "" {
			downloadGameSave(transfer, p, zipPath, downloadObjName)
		}

		if info.ProcName != "" {
			go monitorDir(args.Path, info)
			hasMonitor = true
		}
	}

	if hasMonitor {
		// Block main goroutine forever.
		<-make(chan struct{})
	}
}

func monitorDir(iniPath string, info GameInfo) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		fatalError := false
		gameSaveModify := false
		for !fatalError {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("modified file:", event.Name)
					gameSaveModify = true
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					fatalError = true
				}
				log.Println("error:", err)
			case <-time.After(time.Second * 5):
				break
			}

			if gameSaveModify {
				uploadGameSaveIfGameExited(iniPath, info)
			}
		}
	}()

	// Add a path.
	gsutils.CheckError(watcher.Add(info.Dir))
	// TODO exit if watcher is error on monitor
	<-make(chan struct{})
}

func processRunning(name string) bool {
	processes, err := ps.Processes()
	gsutils.CheckError(err)
	for _, proc := range processes {
		if proc.Executable() == name {
			return true
		}
	}

	return false
}

func uploadGameSaveIfGameExited(iniPath string, info GameInfo) {
	if processRunning(info.ProcName) {
		return
	}

	zipPath := filepath.Join(getAppdata(), info.Name+".zip")
	localGameSaveTime := getLocalGameSaveTime(info.Dir)
	objName := path.Join(info.Name, localGameSaveTime.UTC().Format(gsutils.TimeFormat)+".zip")
	uploadGameSave(newTransfer(iniPath), info.Dir, zipPath, objName)
}

func newTransfer(path string) transfer.Transfer {
	iniFile, err := ini.Load(path)
	gsutils.CheckError(err)
	s3Section, err := iniFile.GetSection("s3")
	if err == nil {
		endpoint := s3Section.Key("endpoint").String()
		bucketName := s3Section.Key("bucketName").String()
		accessKeyID := s3Section.Key("accessKeyID").String()
		secretAccessKey := s3Section.Key("secretAccessKey").String()
		transfer, err := transfer.NewS3Transfer(endpoint, bucketName, accessKeyID, secretAccessKey)
		gsutils.CheckError(err)
		return transfer
	}

	ftpSection, err := iniFile.GetSection("ftp")
	if err == nil {
		addr := ftpSection.Key("addr").String()
		user := ftpSection.Key("user").String()
		password := ftpSection.Key("password").String()
		subDir := ftpSection.Key("subDir").String()
		transfer, err := transfer.NewFTPTransfer(addr, user, password, subDir)
		gsutils.CheckError(err)
		return transfer
	}

	panic("Invalid config no s3 and ftp section")
}

func getDownloadName(transfer transfer.Transfer, localTime *time.Time, dir string) (string, bool) {
	needUpload := false
	var downloadTime time.Time
	if localTime != nil {
		needUpload = true
		downloadTime = *localTime
	}
	downloadObjName := ""
	for file := range transfer.ListFile(dir) {
		if !strings.HasSuffix(file, ".zip") {
			continue
		}

		objTime, err := time.Parse(gsutils.TimeFormat, strings.TrimPrefix(strings.TrimSuffix(file, ".zip"), dir))
		if err != nil {
			log.Printf("Failed to parse time %s\n", file)
			continue
		}

		if localTime != nil && localTime.Unix() == objTime.Unix() {
			needUpload = false
		} else if objTime.After(downloadTime) {
			downloadObjName = file
			downloadTime = objTime
		}
	}

	return downloadObjName, needUpload
}

func getLocalGameSaveTime(dir string) *time.Time {
	var mtime *time.Time = nil
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			if mtime == nil {
				mtime = new(time.Time)
			}

			if info.ModTime().After(*mtime) {
				*mtime = info.ModTime()
			}
		}

		return nil
	})

	gsutils.CheckError(err)
	return mtime
}

func getAppdata() string {
	appData, err := windows.KnownFolderPath(windows.FOLDERID_RoamingAppData, 0)
	gsutils.CheckError(err)
	appData = filepath.Join(appData, AppName)
	gsutils.CheckError(os.MkdirAll(appData, 0755))
	return appData
}

func uploadGameSave(uploader transfer.Uploader, p, zipPath, objName string) {
	err := ziputils.ZipSource(p, zipPath)
	gsutils.CheckError(err)
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			log.Println(err)
		}
	}()

	gsutils.CheckError(uploader.Upload(zipPath, objName))
	log.Printf("Successfully uploaded %s\n", objName)
}

func downloadGameSave(downloader transfer.Downloader, p, zipPath, objName string) {
	err := os.Remove(zipPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	gsutils.CheckError(downloader.Download(objName, zipPath))
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			log.Println(err)
		}
	}()
	gsutils.CheckError(os.RemoveAll(p))
	gsutils.CheckError(ziputils.UnzipSource(zipPath, p))
}
