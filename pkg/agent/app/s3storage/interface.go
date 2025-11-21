package s3storage

import (
	"context"
	"crypto/tls"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"net/http"
)

type S3Storage interface {
	UploadFileToS3(ctx context.Context, bucket, objectName, file string) error
	UploadContentToS3(ctx context.Context, bucket, objectName string, content []byte) error
	DownloadFileFromS3(ctx context.Context, bucket, objectName, file string) error
	DownloadContentFromS3(ctx context.Context, bucket, objectName string) (*minio.Object, error)
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
