package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage interface {
	UploadFileToS3(ctx context.Context, bucket, objectName, file string) error
	DownloadFileFromS3(ctx context.Context, bucket, objectName, file string) error

	StreamToS3(ctx context.Context, bucket, objectName string, reader io.Reader) error
	StreamFromS3(ctx context.Context, bucket, objectName string) (*minio.Object, error)
}

func NewMinioClient(endpoint, accessKey, secretKey string, ssl bool) (S3Storage, error) {
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: ssl,
	}

	if ssl {
		opts.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}

	return &MinioClient{
		client: client,
	}, nil

}

type MinioClient struct {
	client *minio.Client
}

func (mc *MinioClient) UploadFileToS3(ctx context.Context, bucket, objectName, file string) error {
	_, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", file, err)
	}

	if _, err := mc.client.FPutObject(ctx, bucket, objectName, file, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return fmt.Errorf("failed to upload file to s3: %w", err)
	}

	return nil
}

func (mc *MinioClient) StreamToS3(ctx context.Context, bucket, objectName string, reader io.Reader) error {

	if _, err := mc.client.PutObject(ctx, bucket, objectName, reader, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return fmt.Errorf("failed to upload content to s3: %w", err)
	}

	return nil
}

func (mc *MinioClient) DownloadFileFromS3(ctx context.Context, bucket, objectName, file string) error {
	if err := mc.client.FGetObject(ctx, bucket, objectName, file, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("failed to download file from s3: %w", err)
	}
	return nil
}

func (mc *MinioClient) StreamFromS3(ctx context.Context, bucket, objectName string) (*minio.Object, error) {
	if object, err := mc.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{}); err != nil {
		return nil, fmt.Errorf("failed to download object from s3: %w", err)
	} else {
		return object, nil
	}
}
