package common

import (
	"context"
	"fmt"
	"io"
)

type ObjectStorageFactory interface {
	PutFile(ctx context.Context, bucket, object, path string) error
	GetFile(ctx context.Context, bucket, object, path string) error

	PutObject(ctx context.Context, bucket, objectName string, reader io.Reader) error
	GetObject(ctx context.Context, bucket, object string) (io.ReadCloser, error)
}

func (s *ObjectStorage) GenerateFactory() (ObjectStorageFactory, error) {
	switch s.GetType() {
	case ObjectStorageType_Minio, ObjectStorageType_Aws:
		// minio-go implements the S3 API and works with both an AWS S3
		// endpoint and a self-hosted MinIO endpoint. The caller supplies the
		// endpoint and whether TLS is enabled.
		return newMinioClient(s.GetEndpoint(), s.GetAccessKey(), s.GetSecretKey(), s.GetSsl())
	}

	return nil, fmt.Errorf("unsupported s3 storage type: %s", s.GetType().String())
}
