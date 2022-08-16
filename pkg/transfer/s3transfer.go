package transfer

import (
	"context"
	. "github.com/chenjianlong/gamesave-sync/pkg/gsutils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

func (t *S3Transfer) Upload(localFile, remoteFile string) error {
	_, err := t.client.FPutObject(context.Background(), t.bucketName, remoteFile, localFile, minio.PutObjectOptions{
		ContentType: "application/zip",
	})

	return err
}

func (t *S3Transfer) Download(remoteFile, localFile string) error {
	return t.client.FGetObject(context.Background(), t.bucketName, remoteFile, localFile, minio.GetObjectOptions{})
}

func (t *S3Transfer) ListFile(dir string) chan string {
	objectCh := t.client.ListObjects(context.Background(), t.bucketName, minio.ListObjectsOptions{Prefix: dir, Recursive: true})
	resultCh := make(chan string)
	go func() {
		for obj := range objectCh {
			CheckError(obj.Err)
			resultCh <- obj.Key
		}
		close(resultCh)
	}()
	return resultCh
}

func (t *S3Transfer) Rename(src, dst string) error {
	srcOpt := minio.CopySrcOptions{
		Bucket: t.bucketName,
		Object: src,
	}

	dstOpt := minio.CopyDestOptions{
		Bucket:  t.bucketName,
		Object: dst,
	}

	if _, err := t.client.CopyObject(context.Background(), dstOpt, srcOpt); err != nil {
		return err
	}

	return t.client.RemoveObject(context.Background(), t.bucketName, src, minio.RemoveObjectOptions{})
}