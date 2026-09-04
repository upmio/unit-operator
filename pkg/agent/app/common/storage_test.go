package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateFactoryS3Compatible(t *testing.T) {
	for _, storageType := range []ObjectStorageType{ObjectStorageType_Minio, ObjectStorageType_Aws} {
		t.Run(storageType.String(), func(t *testing.T) {
			storage := &ObjectStorage{
				Endpoint:  "localhost:9000",
				Bucket:    "backup",
				AccessKey: "access",
				SecretKey: "secret",
				Ssl:       false,
				Type:      storageType,
			}

			factory, err := storage.GenerateFactory()
			require.NoError(t, err)
			require.NotNil(t, factory)
		})
	}
}

func TestGenerateFactoryUnsupported(t *testing.T) {
	storage := &ObjectStorage{
		Type: ObjectStorageType(99),
	}

	factory, err := storage.GenerateFactory()
	require.Error(t, err)
	require.Nil(t, factory)
}
