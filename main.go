package main

import (
	"context"
	"errors"
	"github.com/alexflint/go-arg"
	"github.com/minio/minio-go/v7"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sys/windows"
)

const AppName = "GameSaveSyncing"

type SearchType uint16

const (
	ST_KnownFolder SearchType = 1
	ST_Registry    SearchType = 2
)

type RegistryInfo struct {
	RootKey registry.Key
	Key     string
	Name    string
}
type GameSearchInfo struct {
	Name      string
	StateGlob string
	Type      SearchType
	FolderID  *windows.KNOWNFOLDERID
	Reg       *RegistryInfo
	SubDir    string
}

type GameInfo struct {
	Name     string
	Dir      string
	StatGlob string
}

func getGameList() []GameInfo {
	gameSearchInfo := []GameSearchInfo{
		{`The Witcher 3`, `*.sav`, ST_KnownFolder, windows.FOLDERID_Documents,
			nil, `The Witcher 3\gamesaves`},
		{`Skyrim`, `*.ess`, ST_KnownFolder, windows.FOLDERID_Documents,
			nil, `My Games\Skyrim\Saves`},
		{`NewPAL`, `*.sav`, ST_KnownFolder, windows.FOLDERID_Documents,
			nil, `My Games\NewPAL`},
		{`Wind3`, `*\Event.data`, ST_Registry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind3`, `Path`}, `Save`},
		{`Wind4`, `*\EventData_Total.DATA`, ST_Registry, nil,
			&RegistryInfo{registry.CURRENT_USER, `Wind4`, `Path`}, `Save`},
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
		case ST_KnownFolder:
			dir, err = windows.KnownFolderPath(info.FolderID, 0)
			checkError(err)
		case ST_Registry:
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

		gameList = append(gameList, GameInfo{info.Name, dir, info.StateGlob})
	}

	return gameList
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var args struct {
		Path string `arg:"-p" default:"config.ini" help:"config path"`
	}

	arg.MustParse(&args)

	iniFile, err := ini.Load(args.Path)
	checkError(err)
	iniSection := iniFile.Section("main")
	endpoint := iniSection.Key("endpoint").String()
	bucketName := iniSection.Key("bucketName").String()
	accessKeyID := iniSection.Key("accessKeyID").String()
	secretAccessKey := iniSection.Key("secretAccessKey").String()

	// Initialize minio client object.
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	checkError(err)

	appData := getAppdata()
	for _, info := range getGameList() {
		log.Printf("Syncing game: %s\n", info.Name)
		p := info.Dir
		valid, _ := isDir(p)
		if !valid {
			log.Printf("%s not exist\n", p)
			continue
		}

		matches, err := filepath.Glob(filepath.Join(p, info.StatGlob))
		checkError(err)

		var lastTime time.Time
		for _, name := range matches {
			fileInfo, err := os.Lstat(name)
			if err != nil {
				log.Printf("Failed lstat, path: %s, err: %v\n", name, err)
				continue
			}

			if fileInfo.ModTime().After(lastTime) {
				lastTime = fileInfo.ModTime()
			}
		}

		needUpload := len(matches) > 0
		downloadTime := lastTime
		downloadObjName := ""
		objectCh := s3Client.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{Prefix: info.Name + "/"})
		for obj := range objectCh {
			checkError(obj.Err)
			if !strings.HasSuffix(obj.Key, ".zip") {
				continue
			}

			objTime, err := time.Parse(time.RFC3339, strings.TrimPrefix(strings.TrimSuffix(obj.Key, ".zip"), info.Name+"/"))
			if err != nil {
				log.Printf("Failed to parse time %s\n", obj.Key)
				continue
			}

			if lastTime.Unix() == objTime.Unix() {
				needUpload = false
			} else if objTime.After(downloadTime) {
				downloadObjName = obj.Key
				downloadTime = objTime
			}
		}

		log.Printf("Game: %s, needUpload: %v, downloadObject: %s\n", info.Name, needUpload, downloadObjName)
		zipPath := filepath.Join(appData, info.Name+".zip")
		if needUpload {
			objName := path.Join(info.Name, lastTime.Format(time.RFC3339)+".zip")
			uploadGameSave(s3Client, p, zipPath, bucketName, objName)
		}

		if downloadObjName != "" {
			downloadGameSave(s3Client, p, zipPath, bucketName, downloadObjName)
		}
	}
}

func getAppdata() string {
	appData, err := windows.KnownFolderPath(windows.FOLDERID_RoamingAppData, 0)
	checkError(err)
	appData = filepath.Join(appData, AppName)
	checkError(os.MkdirAll(appData, 0755))
	return appData
}

func uploadGameSave(s3 *minio.Client, p, zipPath, bucketName, objName string) {
	err := zipSource(p, zipPath)
	checkError(err)
	defer func() {
		err = os.Remove(zipPath)
		if err != nil {
			log.Println(err)
		}
	}()

	_, err = s3.FPutObject(context.Background(), bucketName, objName, zipPath, minio.PutObjectOptions{
		ContentType: "application/zip",
	})
	checkError(err)
	log.Printf("Successfully uploaded %s\n", objName)
}

func downloadGameSave(s3 *minio.Client, p, zipPath, bucketName, objName string) {
	err := os.Remove(zipPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}

	checkError(s3.FGetObject(context.Background(), bucketName, objName, zipPath, minio.GetObjectOptions{}))
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
