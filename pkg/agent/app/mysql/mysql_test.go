package mysql

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"testing"
)

func TestMarshalToLogicalBackupRequest(t *testing.T) {
	params := map[string]interface{}{
		"backupFile":        "mysql-backup-20250516",
		"confFile":          "/DATA_MOUNT/conf/mysqld.cnf",
		"socketFile":        "/DATA_MOUNT/mysqld.sock",
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
	require.Equal(t, "mysql-backup-20250516", req.GetBackupFile(), "BackupFile mismatch")
	require.Equal(t, "/DATA_MOUNT/conf/mysqld.cnf", req.GetConfFile(), "ConfFile mismatch")
	require.Equal(t, "/DATA_MOUNT/mysqld.sock", req.GetSocketFile(), "SocketFile mismatch")
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
		"backupFile":         "mysql-backup-20250516",
		"confFile":           "/DATA_MOUNT/conf/mysqld.cnf",
		"socketFile":         "/DATA_MOUNT/mysqld.sock",
		"username":           "root",
		"password":           "password",
		"parallel":           "6",
		"physicalBackupTool": 0,
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
	require.Equal(t, "mysql-backup-20250516", req.GetBackupFile(), "BackupFile mismatch")
	require.Equal(t, "/DATA_MOUNT/conf/mysqld.cnf", req.GetConfFile(), "ConfFile mismatch")
	require.Equal(t, "/DATA_MOUNT/mysqld.sock", req.GetSocketFile(), "SocketFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
	require.Equal(t, int64(6), req.GetParallel(), "Parallel mismatch")
	require.Equal(t, PhysicalBackupTool_Xtrabackup, req.GetPhysicalBackupTool(), "PhysicalBackupTool mismatch")

	// Validate nested S3Storage message
	s3 := req.GetS3Storage()
	require.NotNil(t, s3, "S3Storage should not be nil")
	require.Equal(t, "s3://192.168.1.1", s3.GetEndpoint(), "S3Storage.Endpoint mismatch")
	require.Equal(t, "mysql-backup", s3.GetBucket(), "S3Storage.Bucket mismatch")
	require.Equal(t, "accesskey", s3.GetAccessKey(), "S3Storage.AccessKey mismatch")
	require.Equal(t, "secretkey", s3.GetSecretKey(), "S3Storage.SecretKey mismatch")
}

func TestMarshalToCloneRequest(t *testing.T) {
	data := map[string]interface{}{
		"sourceCloneUser":     "replication",
		"sourceClonePassword": "replication",
		"sourceHost":          "mysql-source-0",
		"sourcePort":          3306,
		"socketFile":          "/DATA_MOUNT/mysqld.sock",
		"username":            "root",
		"password":            "password",
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req CloneRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "replication", req.GetSourceCloneUser(), "SourceCloneUser mismatch")
	require.Equal(t, "replication", req.GetSourceClonePassword(), "SourceClonePassword mismatch")
	require.Equal(t, "mysql-source-0", req.GetSourceHost(), "SourceHost mismatch")
	require.Equal(t, int64(3306), req.GetSourcePort(), "SourcePort mismatch")
	require.Equal(t, "/DATA_MOUNT/mysqld.sock", req.GetSocketFile(), "SocketFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
}

func TestMarshalToRestoreRequest(t *testing.T) {
	data := map[string]interface{}{
		"backupFile": "mysql-backup-20250516",
		"parallel":   2,
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
	require.Equal(t, "mysql-backup-20250516", req.GetBackupFile(), "BackupFile mismatch")
	require.Equal(t, int64(2), req.GetParallel(), "Parallel mismatch")

	// Validate nested S3Storage message
	s3 := req.GetS3Storage()
	require.NotNil(t, s3, "S3Storage should not be nil")
	require.Equal(t, "s3://192.168.1.1", s3.GetEndpoint(), "S3Storage.Endpoint mismatch")
	require.Equal(t, "mysql-backup", s3.GetBucket(), "S3Storage.Bucket mismatch")
	require.Equal(t, "accesskey", s3.GetAccessKey(), "S3Storage.AccessKey mismatch")
	require.Equal(t, "secretkey", s3.GetSecretKey(), "S3Storage.SecretKey mismatch")

}

func TestMarshalToGtidPurgeRequest(t *testing.T) {
	data := map[string]interface{}{
		"socketFile": "/DATA_MOUNT/mysqld.sock",
		"username":   "root",
		"password":   "password",
		"archMode":   0,
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req GtidPurgeRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "/DATA_MOUNT/mysqld.sock", req.GetSocketFile(), "SocketFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
	require.Equal(t, ArchMode_Replication, req.GetArchMode(), "ArchMode mismatch")
}

func TestMarshalToGSetVariableRequest(t *testing.T) {
	data := map[string]interface{}{
		"key":        "auto_increment",
		"value":      "2",
		"socketFile": "/DATA_MOUNT/mysqld.sock",
		"username":   "root",
		"password":   "password",
	}

	// Perform unmarshal into proto struct
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var req SetVariableRequest
	if err := protojson.Unmarshal(jsonBytes, &req); err != nil {
		t.Fatal(err)
	}

	// Validate all top-level fields
	require.Equal(t, "auto_increment", req.GetKey(), "Key mismatch")
	require.Equal(t, "2", req.GetValue(), "Value mismatch")
	require.Equal(t, "/DATA_MOUNT/mysqld.sock", req.GetSocketFile(), "SocketFile mismatch")
	require.Equal(t, "root", req.GetUsername(), "Username mismatch")
	require.Equal(t, "password", req.GetPassword(), "Password mismatch")
}
