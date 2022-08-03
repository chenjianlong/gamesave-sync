package main

import (
	"errors"
	"github.com/alexflint/go-arg"
	"golang.org/x/sys/windows/registry"
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

type SearchType uint16

const (
	STKnownFolder SearchType = 1
	STRegistry    SearchType = 2
)

type RegistryInfo struct {
	RootKey registry.Key
	Key     string
	Name    string
}
type GameSearchInfo struct {
	Name      string
	Type      SearchType
	FolderID  *windows.KNOWNFOLDERID
	Reg       *RegistryInfo
	SubDir    string
}

type GameInfo struct {
	Name     string
	Dir      string
}

func getGameList() []GameInfo {
	gameSearchInfo := []GameSearchInfo{
		{`The Witcher 3`, STKnownFolder, windows.FOLDERID_Documents,
			nil, `The Witcher 3\gamesaves`},
		{`Shin Sangokumusou 7 TC`, STKnownFolder, windows.FOLDERID_Documents,
			nil, `TecmoKoei\Shin Sangokumusou 7 TC\Savedata`},
		{`Skyrim`, STKnownFolder, windows.FOLDERID_Documents,
			nil, `My Games\Skyrim\Saves`},
		{`NewPAL`, STKnownFolder, windows.FOLDERID_Documents,
			nil, `My Games\NewPAL`},
		{`Wind3`, STRegistry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind3`, `Path`}, `Save`},
		{`Wind4`, STRegistry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind4`, `Path`}, `Save`},
		{`Wind5`, STRegistry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind5`, `Path`}, `Save`},
		{`Wind6`, STRegistry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind6`, `Path`}, `Save`},
		{`WindXX`, STRegistry, nil,
			&RegistryInfo{registry.CURRENT_USER, `WindXX`, `Path`}, `Save`},
	}

	var gameList []GameInfo
	var err error
	for _, info := range gameSearchInfo {
		if info.Name == `` || info.SubDir == `` {
			log.Printf("Invalid search info: %#v\n", info)
			continue
		}

		var dir string
		switch info.Type {
		case STKnownFolder:
			dir, err = windows.KnownFolderPath(info.FolderID, 0)
			checkError(err)
		case STRegistry:
			key, err := registry.OpenKey(info.Reg.RootKey, info.Reg.Key, registry.QUERY_VALUE|registry.WOW64_64KEY)
			if err != nil {
				continue
			}

			dir, _, err = key.GetStringValue(info.Reg.Name)
			if err != nil {
				continue
			}
		default:
			log.Fatalf("Invalid search type: %d\n", info.Type)
		}

		if dir == `` {
			continue
		}

		dir = filepath.Join(dir, info.SubDir)
		valid, _ := isDir(dir)
		if !valid {
			continue
		}

		gameList = append(gameList, GameInfo{info.Name, dir})
	}

	return gameList
}

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
	for _, info := range getGameList() {
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
