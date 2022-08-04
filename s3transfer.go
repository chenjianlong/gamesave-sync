package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"strings"
)

type S3Transfer struct {
	client     *minio.Client
	bucketName string
}

func NewS3Transfer(endpoint, bucketName, accessKeyID, secretAccessKey string) (Transfer, error) {
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	transfer := new(S3Transfer)
	transfer.client = s3Client
	transfer.bucketName = bucketName
	return transfer, nil
}

func (t *S3Transfer) upload(localFile, remoteFile string) error {
	_, err := t.client.FPutObject(context.Background(), t.bucketName, remoteFile, localFile, minio.PutObjectOptions{
		ContentType: "application/zip",
	})

	return err
}

func (t *S3Transfer) download(remoteFile, localFile string) error {
	return t.client.FGetObject(context.Background(), t.bucketName, remoteFile, localFile, minio.GetObjectOptions{})
}

func (t *S3Transfer) listFile(dir string) chan string {
	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}

	objectCh := t.client.ListObjects(context.Background(), t.bucketName, minio.ListObjectsOptions{Prefix: dir})
	resultCh := make(chan string)
	go func() {
		for obj := range objectCh {
			checkError(obj.Err)
			resultCh <- obj.Key
		}
		close(resultCh)
	}()
	return resultCh
}
