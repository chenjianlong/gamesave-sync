package main

import (
	"fmt"
	"github.com/alexflint/go-arg"
	. "github.com/chenjianlong/gamesave-syncing/pkg/gsutils"
	. "github.com/chenjianlong/gamesave-syncing/pkg/transfer"
	"gopkg.in/ini.v1"
	"log"
	"strings"
	"time"
)

func main() {
	oldFormat := time.RFC3339
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var args struct {
		Path string `arg:"-p" default:"config.ini" help:"config path"`
	}

	arg.MustParse(&args)
	iniFile, err := ini.Load(args.Path)
	CheckError(err)
	iniSection := iniFile.Section("main")
	endpoint := iniSection.Key("endpoint").String()
	bucketName := iniSection.Key("bucketName").String()
	accessKeyID := iniSection.Key("accessKeyID").String()
	secretAccessKey := iniSection.Key("secretAccessKey").String()
	transfer, err := NewS3Transfer(endpoint, bucketName, accessKeyID, secretAccessKey)
	CheckError(err)
	s3Transfer := transfer.(*S3Transfer)
	ch := transfer.ListFile("")
	sourceToDest := map[string]string{}
	_, offset := time.Now().Zone()
	for name := range ch {
		if !strings.HasSuffix(name, ".zip") {
			continue
		}

		sepIdx := strings.LastIndex(name, "/")
		if sepIdx == -1 {
			continue
		}

		t := name[sepIdx + 1: len(name) - 4]
		tm, err := time.Parse(oldFormat, t)
		CheckError(err)

		sourceToDest[name] = fmt.Sprintf("%s/%s.zip", name[:sepIdx], time.Unix(tm.Unix()-int64(offset), 0).UTC().Format(TimeFormat))
	}

	for src, dst := range sourceToDest {
		CheckError(s3Transfer.Rename(src, dst))
	}
}
