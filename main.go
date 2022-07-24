package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sys/windows"
	"gopkg.in/ini.v1"
)

type GameInfo struct {
	Name string
	Dir string
	StatGlob string
}

var GameList = []GameInfo{
	{`The Witcher 3`, `The Witcher 3\gamesaves`,`*.sav`},
	{`Skyrim`, `My Games\Skyrim\Saves`, `*.ess`},
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var args struct {
		Path  string `arg:"-p" default:"config.ini" help:"config path"`
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

			objTime, err := time.Parse(time.RFC3339, strings.TrimPrefix(strings.TrimSuffix(obj.Key, ".zip"), info.Name + "/"))
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
		zipPath := filepath.Join(document, info.Name + ".zip")
		if needUpload {
			err = zipSource(p, zipPath)
			checkError(err)

			objName := path.Join(info.Name, lastTime.Format(time.RFC3339) + ".zip")
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

func zipSource(source, destination string) error {
	writer, err := os.Create(destination)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(writer)
	defer zw.Close()
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			return addFileToZip(zw, path, source)
		}

		return nil
	})

	return err
}

func addFileToZip(zipWriter *zip.Writer, filename string, dirname string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	if len(dirname) < len(filename) && strings.HasPrefix(filename, dirname) {
		header.Name = filename[len(dirname):]
		if header.Name[0] == filepath.Separator {
			header.Name = header.Name[1:]
		}
	} else {
		header.Name = filepath.Base(filename)
	}

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}

	if err := destinationFile.Close(); err != nil {
		return err
	}

	if err := os.Chtimes(filePath, time.Now(), f.FileInfo().ModTime()); err != nil {
		return err
	}

	return nil
}

// exists returns whether the given file or directory exists
func isDir(path string) (bool, error) {
	st, err := os.Stat(path)
	if err == nil { return st.IsDir(), nil }
	if os.IsNotExist(err) { return false, nil }
	return false, err
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
