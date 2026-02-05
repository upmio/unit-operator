package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFactoryMinio(t *testing.T) {
	storage := &ObjectStorage{
		Endpoint:  "localhost:9000",
		Bucket:    "backup",
		AccessKey: "access",
		SecretKey: "secret",
		Ssl:       false,
		Type:      ObjectStorageType_Minio,
	}

	factory, err := storage.GenerateFactory()
	require.NoError(t, err)
	require.NotNil(t, factory)
}

func TestGenerateFactoryUnsupported(t *testing.T) {
	storage := &ObjectStorage{
		Type: ObjectStorageType_Aws,
	}

	factory, err := storage.GenerateFactory()
	require.Error(t, err)
	require.Nil(t, factory)
}

