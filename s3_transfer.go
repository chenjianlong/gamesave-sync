package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Transfer struct {
	Client *minio.Client
	bucketName string
}

func NewS3Transfer(endpoint, bucketName, accessKeyID, secretAccessKey string) (*S3Transfer, error) {
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	transfer := new(S3Transfer)
	transfer.Client = s3Client
	transfer.bucketName = bucketName
	return transfer, nil
}

func (t *S3Transfer) upload(localFile, remoteFile string) error {
	_, err := t.Client.FPutObject(context.Background(), t.bucketName, remoteFile, localFile, minio.PutObjectOptions{
		ContentType: "application/zip",
	})

	return err
}

func (t *S3Transfer) download(remoteFile, localFile string) error {
	return t.Client.FGetObject(context.Background(), t.bucketName, remoteFile, localFile, minio.GetObjectOptions{})
}