package main

import (
	"errors"
	"github.com/alexflint/go-arg"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeandeaual/go-locale"
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
	checkError(err)
	initBundle(loc)

	iniFile, err := ini.Load(args.Path)
	checkError(err)
	iniSection := iniFile.Section("main")
	endpoint := iniSection.Key("endpoint").String()
	bucketName := iniSection.Key("bucketName").String()
	accessKeyID := iniSection.Key("accessKeyID").String()
	secretAccessKey := iniSection.Key("secretAccessKey").String()

	transfer, err := NewS3Transfer(endpoint, bucketName, accessKeyID, secretAccessKey)
	checkError(err)

	appData := getAppdata()
	for _, info := range LoadGameList("conf.d/") {
		log.Println(getSyncGameMessage(info.Name))
		p := info.Dir
		valid, _ := isDir(p)
		if !valid {
			log.Printf("%s not exist\n", p)
			continue
		}

		localGameSaveTime := getLocalGameSaveTime(info.Dir)
		downloadObjName, needUpload := getDownloadName(transfer, localGameSaveTime, info.Name+"/")
		log.Printf("Game: %s, needUpload: %v, downloadObject: %s\n", info.Name, needUpload, downloadObjName)
		zipPath := filepath.Join(appData, info.Name+".zip")
		if needUpload && localGameSaveTime != nil {
			objName := path.Join(info.Name, localGameSaveTime.Format(time.RFC3339)+".zip")
			uploadGameSave(transfer, p, zipPath, objName)
		}

		if downloadObjName != "" {
			downloadGameSave(transfer, p, zipPath, downloadObjName)
		}
	}
}

func getDownloadName(transfer Transfer, localTime *time.Time, dir string) (string, bool) {
	needUpload := false
	var downloadTime time.Time
	if localTime != nil {
		needUpload = true
		downloadTime = *localTime
	}
	downloadObjName := ""
	for file := range transfer.listFile(dir) {
		if !strings.HasSuffix(file, ".zip") {
			continue
		}

		objTime, err := time.Parse(time.RFC3339, strings.TrimPrefix(strings.TrimSuffix(file, ".zip"), dir))
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

	checkError(err)
	return mtime
}

func getAppdata() string {
	appData, err := windows.KnownFolderPath(windows.FOLDERID_RoamingAppData, 0)
	checkError(err)
	appData = filepath.Join(appData, AppName)
	checkError(os.MkdirAll(appData, 0755))
	return appData
}

func uploadGameSave(uploader Uploader, p, zipPath, objName string) {
	err := zipSource(p, zipPath)
	checkError(err)
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			log.Println(err)
		}
	}()

	checkError(uploader.upload(zipPath, objName))
	log.Printf("Successfully uploaded %s\n", objName)
}

func downloadGameSave(downloader Downloader, p, zipPath, objName string) {
	err := os.Remove(zipPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	checkError(downloader.download(objName, zipPath))
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			log.Println(err)
		}
	}()
	checkError(os.RemoveAll(p))
	checkError(unzipSource(zipPath, p))
}

// exists returns whether the given file or directory exists
func isDir(path string) (bool, error) {
	st, err := os.Stat(path)
	if err == nil {
		return st.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
