package s3storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"os"
)

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

func (mc *MinioClient) UploadContentToS3(ctx context.Context, bucket, objectName string, content []byte) error {

	if _, err := mc.client.PutObject(ctx, bucket, objectName, bytes.NewReader(content), -1, minio.PutObjectOptions{
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

func (mc *MinioClient) DownloadContentFromS3(ctx context.Context, bucket, objectName string) (*minio.Object, error) {
	if object, err := mc.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{}); err != nil {
		return nil, fmt.Errorf("failed to download object from s3: %w", err)
	} else {
		return object, nil
	}
}
