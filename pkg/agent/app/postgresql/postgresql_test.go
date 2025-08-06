package postgresql

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TestMarshalToLogicalBackupRequest(t *testing.T) {
	params := map[string]interface{}{
		"backupFile":        "postgresql-backup-20250520",
		"username":          "root",
		"password":          "password",
		"database":          "test",
		"table":             "table1",
		"logicalBackupMode": 0,
		"s3Storage": map[string]interface{}{
			"endpoint":  "s3://192.168.1.1",
			"bucket":    "mysql-backup",
			"accessKey": "accesskey",
			"secretKey": "secretkey",
		},
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	var req LogicalBackupRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "postgresql-backup-20250520", req.GetBackupFile(), "BackupFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
	require.Equal(t, "test", req.GetDatabase(), "Database mismatch")
	require.Equal(t, "table1", req.GetTable(), "Table mismatch")
	require.Equal(t, LogicalBackupMode_Full, req.GetLogicalBackupMode(), "LogicalBackupMode mismatch")

	// Validate nested S3Storage message
	s3 := req.GetS3Storage()
	require.NotNil(t, s3, "S3Storage should not be nil")
	require.Equal(t, "s3://192.168.1.1", s3.GetEndpoint(), "S3Storage.Endpoint mismatch")
	require.Equal(t, "mysql-backup", s3.GetBucket(), "S3Storage.Bucket mismatch")
	require.Equal(t, "accesskey", s3.GetAccessKey(), "S3Storage.AccessKey mismatch")
	require.Equal(t, "secretkey", s3.GetSecretKey(), "S3Storage.SecretKey mismatch")
}

func TestMarshalToPhysicalBackupRequest(t *testing.T) {
	data := map[string]interface{}{
		"backupFile": "postgresql-backup-20250520",
		"username":   "root",
		"password":   "password",
		"s3Storage": map[string]interface{}{
			"endpoint":  "s3://192.168.1.1",
			"bucket":    "mysql-backup",
			"accessKey": "accesskey",
			"secretKey": "secretkey",
		},
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req PhysicalBackupRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "postgresql-backup-20250520", req.GetBackupFile(), "BackupFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")

	// Validate nested S3Storage message
	s3 := req.GetS3Storage()
	require.NotNil(t, s3, "S3Storage should not be nil")
	require.Equal(t, "s3://192.168.1.1", s3.GetEndpoint(), "S3Storage.Endpoint mismatch")
	require.Equal(t, "mysql-backup", s3.GetBucket(), "S3Storage.Bucket mismatch")
	require.Equal(t, "accesskey", s3.GetAccessKey(), "S3Storage.AccessKey mismatch")
	require.Equal(t, "secretkey", s3.GetSecretKey(), "S3Storage.SecretKey mismatch")
}

func TestMarshalToRestoreRequest(t *testing.T) {
	data := map[string]interface{}{
		"backupFile": "postgresql-backup-20250520",
		"s3Storage": map[string]interface{}{
			"endpoint":  "s3://192.168.1.1",
			"bucket":    "mysql-backup",
			"accessKey": "accesskey",
			"secretKey": "secretkey",
		},
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req RestoreRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "postgresql-backup-20250520", req.GetBackupFile(), "BackupFile mismatch")

	// Validate nested S3Storage message
	s3 := req.GetS3Storage()
	require.NotNil(t, s3, "S3Storage should not be nil")
	require.Equal(t, "s3://192.168.1.1", s3.GetEndpoint(), "S3Storage.Endpoint mismatch")
	require.Equal(t, "mysql-backup", s3.GetBucket(), "S3Storage.Bucket mismatch")
	require.Equal(t, "accesskey", s3.GetAccessKey(), "S3Storage.AccessKey mismatch")
	require.Equal(t, "secretkey", s3.GetSecretKey(), "S3Storage.SecretKey mismatch")

}
