package main

import (
	"context"
	"errors"
	"github.com/alexflint/go-arg"
	"github.com/minio/minio-go/v7"
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

type GameInfo struct {
	Name     string
	Dir      string
	StatGlob string
}

var GameList = []GameInfo{
	{`The Witcher 3`, `The Witcher 3\gamesaves`, `*.sav`},
	{`Skyrim`, `My Games\Skyrim\Saves`, `*.ess`},
	{`NewPAL`, `My Games\NewPAL`, `*.sav`},
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

	document, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
	checkError(err)
	log.Println(document)
	for _, info := range GameList {
		p := filepath.Join(document, info.Dir)
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
		zipPath := filepath.Join(document, info.Name+".zip")
		if needUpload {
			err = zipSource(p, zipPath)
			checkError(err)

			objName := path.Join(info.Name, lastTime.Format(time.RFC3339)+".zip")
			_, err = s3Client.FPutObject(context.Background(), bucketName, objName, zipPath, minio.PutObjectOptions{
				ContentType: "application/zip",
			})
			checkError(err)
			log.Printf("Successfully uploaded %s\n", objName)
		}

		if downloadObjName != "" {
			err = os.Remove(zipPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				panic(err)
			}

			checkError(s3Client.FGetObject(context.Background(), bucketName, downloadObjName, zipPath, minio.GetObjectOptions{}))
			checkError(os.RemoveAll(p))
			checkError(unzipSource(zipPath, p))
		}
	}
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
