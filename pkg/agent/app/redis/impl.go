package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	slm "github.com/upmio/unit-operator/pkg/agent/app/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	// service service instance
	svr = &service{}
)

type service struct {
	redisOps RedisOperationServer
	UnimplementedRedisOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer
}

const (
	redisAddr            = "localhost:6379"
	redisDB              = 0
	defaultBGSaveTimeout = 2 * time.Minute

	persistencePollInterval = 2 * time.Second
)

// Common helper methods

// newRedisResponse creates a new Sentinel Response with the given message
func newRedisResponse(message string) *Response {
	return &Response{Message: message}
}

// newRedisClient creates a Redis connection
func (s *service) newRedisClient(ctx context.Context, encrypt string) (*redis.Client, error) {
	password, err := common.GetPlainTextPassword(encrypt)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password, %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       redisDB,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to ping redis server, %v", err)
	}

	return rdb, nil
}

func (s *service) closeRedisClient(client *redis.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		s.logger.Errorf("failed to close redis client: %v", err)
	}
}

// createMinioClient creates an S3 client
func (s *service) createMinioClient(s3Config *S3Storage) (*minio.Client, error) {
	if s3Config == nil {
		return nil, fmt.Errorf("s3 storage configuration is required")
	}

	endpoint := s3Config.GetEndpoint()
	secure := false
	if strings.HasPrefix(endpoint, "https://") {
		secure = true
		endpoint = strings.TrimPrefix(endpoint, "https://")
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	}

	if endpoint == "" {
		return nil, fmt.Errorf("s3 endpoint must be provided")
	}

	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Config.GetAccessKey(), s3Config.GetSecretKey(), ""),
		Secure: secure,
	})
}

// getEnvVarOrError gets environment variable or returns error if not found
func getEnvVarOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}

func (s *service) Config() error {
	s.redisOps = app.GetGrpcApp(appName).(RedisOperationServer)
	s.logger = zap.L().Named("[REDIS]").Sugar()
	s.slm = app.GetGrpcApp("service").(slm.ServiceLifecycleServer)

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterRedisOperationServer(server, svr)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"password": req.GetPassword(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to check service status", err)
	}

	// 2. Create connection
	client, err := s.newRedisClient(ctx, req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to new redis client", err)
	}
	defer s.closeRedisClient(client)

	// 3. Execute config set
	err = client.ConfigSet(ctx, req.GetKey(), req.GetValue()).Err()
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, fmt.Sprintf("failed to SET %s=%s", req.GetKey(), req.GetValue()), err)
	}

	return common.LogAndReturnSuccess(s.logger, newRedisResponse, fmt.Sprintf("set variable %s=%s successfully", req.GetKey(), req.GetValue()))
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis backup", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetS3Storage().GetBucket(),
		"endpoint":    req.GetS3Storage().GetEndpoint(),
		"access_key":  req.GetS3Storage().GetAccessKey(),
		"secret_key":  req.GetS3Storage().GetSecretKey(),
		"password":    req.GetPassword(),
	})

	// 1. Check service status
	if _, err := s.slm.CheckServiceStatus(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "service status check failed", err)
	}

	// 2. Create connection
	client, err := s.newRedisClient(ctx, req.GetPassword())
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to new redis client", err)
	}
	defer s.closeRedisClient(client)

	rdbPath, err := discoverRDBPath(ctx, client)
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to discover rdb path", err)
	}
	s.logger.Infof("discovered redis rdb path: %s", rdbPath)

	if err := ensureFreshRDBSnapshot(ctx, client, defaultBGSaveTimeout); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to create new redis snapshot", err)
	}

	if _, err := os.Stat(rdbPath); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "redis rdb file not found after snapshot", err)
	}

	minioClient, err := s.createMinioClient(req.GetS3Storage())
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to create s3 client", err)
	}

	if err := uploadFileToS3(ctx, minioClient, req.GetS3Storage().GetBucket(), req.GetBackupFile(), rdbPath); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to upload redis backup to s3", err)
	}

	return common.LogAndReturnSuccess(s.logger, newRedisResponse, "backup redis and upload to s3 success")
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "redis restore", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetS3Storage().GetBucket(),
		"endpoint":    req.GetS3Storage().GetEndpoint(),
		"access_key":  req.GetS3Storage().GetAccessKey(),
		"secret_key":  req.GetS3Storage().GetSecretKey(),
		"password":    req.GetPassword(),
	})

	// 1. Check if service is stopped
	if _, err := s.slm.CheckServiceStopped(ctx, &slm.ServiceRequest{}); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "service status check failed", err)
	}

	// 2. Rename dump.rdb to .bak
	dataDir, err := getEnvVarOrError("DATA_DIR")
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to get DATA_DIR environment variable", err)
	}

	rdbPath := filepath.Join(dataDir, "dump.rdb")
	if err := renameWithBak(rdbPath); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to backup original rdb file", err)
	}

	minioClient, err := s.createMinioClient(req.GetS3Storage())
	if err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to create s3 client", err)
	}

	// 3. download rdb from s3
	if err := downloadFileFromS3(ctx, minioClient, req.GetS3Storage().GetBucket(), req.GetBackupFile(), rdbPath); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to download rdb file from s3 storage", err)
	}

	if err := os.Chmod(rdbPath, 0644); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to chmod rdb file", err)
	}

	if err := os.Chown(rdbPath, 1001, 1001); err != nil {
		return common.LogAndReturnError(s.logger, newRedisResponse, "failed to chown rdb file", err)
	}

	return common.LogAndReturnSuccess(s.logger, newRedisResponse, "restore from s3 succeeded")
}

func uploadFileToS3(ctx context.Context, client *minio.Client, bucket, objectName, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file %s: %w", localPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat backup file %s: %w", localPath, err)
	}

	if _, err := client.PutObject(ctx, bucket, objectName, file, info.Size(), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return fmt.Errorf("failed to upload backup to s3: %w", err)
	}

	return nil
}

func downloadFileFromS3(ctx context.Context, client *minio.Client, bucket, objectName, destination string) error {
	if err := client.FGetObject(ctx, bucket, objectName, destination, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("failed to download backup from s3: %w", err)
	}
	return nil
}

func redisLastSaveTime(ctx context.Context, client *redis.Client) (time.Time, error) {
	lastSaveUnix, err := client.LastSave(ctx).Result()
	if err != nil {
		return time.Time{}, err
	}
	if lastSaveUnix <= 0 {
		return time.Time{}, nil
	}
	return time.Unix(lastSaveUnix, 0), nil
}

func ensureFreshRDBSnapshot(ctx context.Context, client *redis.Client, timeout time.Duration) error {
	prevLastSave, err := redisLastSaveTime(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to query redis LASTSAVE: %w", err)
	}

	if err := waitForExistingBGSave(ctx, client, prevLastSave, timeout); err != nil {
		return err
	}

	if err := client.BgSave(ctx).Err(); err != nil {
		if !strings.Contains(err.Error(), "Background save already in progress") {
			return fmt.Errorf("failed to trigger redis BGSAVE: %w", err)
		}

		if err := waitForBGSave(ctx, client, prevLastSave, timeout); err != nil {
			return err
		}

		prevLastSave, err = redisLastSaveTime(ctx, client)
		if err != nil {
			return fmt.Errorf("failed to query redis LASTSAVE after existing save: %w", err)
		}

		if err := client.BgSave(ctx).Err(); err != nil {
			return fmt.Errorf("failed to trigger redis BGSAVE after waiting: %w", err)
		}
	}

	return waitForBGSave(ctx, client, prevLastSave, timeout)
}

func waitForExistingBGSave(ctx context.Context, client *redis.Client, baseline time.Time, timeout time.Duration) error {
	info, err := client.Info(ctx, "persistence").Result()
	if err != nil {
		return fmt.Errorf("failed to query redis persistence info: %w", err)
	}

	if parseRedisInfo(info)["rdb_bgsave_in_progress"] != "1" {
		return nil
	}

	return waitForBGSave(ctx, client, baseline, timeout)
}

func waitForBGSave(ctx context.Context, client *redis.Client, baseline time.Time, timeout time.Duration) error {
	ticker := time.NewTicker(persistencePollInterval)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for redis background save to finish")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := client.Info(ctx, "persistence").Result()
			if err != nil {
				continue
			}
			if parseRedisInfo(info)["rdb_bgsave_in_progress"] == "0" {
				lastSave, err := redisLastSaveTime(ctx, client)
				if err != nil {
					return fmt.Errorf("failed to query redis LASTSAVE: %w", err)
				}
				if baseline.IsZero() {
					if !lastSave.IsZero() {
						return nil
					}
				} else if lastSave.After(baseline) {
					return nil
				}
			}
		}
	}
}

func getConfigValue(ctx context.Context, client *redis.Client, key string) (string, error) {
	resp, err := client.Do(ctx, "CONFIG", "GET", key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to CONFIG GET %s: %w", key, err)
	}

	var value string
	switch arr := resp.(type) {
	case []interface{}:
		if len(arr) == 2 {
			if s, ok := arr[1].(string); ok {
				value = s
			}
		}
	}

	if value == "" {
		return "", fmt.Errorf("unexpected redis CONFIG GET %s response", key)
	}

	return value, nil
}

func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func discoverRDBPath(ctx context.Context, rdb *redis.Client) (string, error) {
	dir, err := getConfigValue(ctx, rdb, "dir")
	if err != nil {
		return "", err
	}

	dbfn, err := getConfigValue(ctx, rdb, "dbfilename")
	if err != nil {
		return "", err
	}

	if dir == "" || dbfn == "" {
		return "", fmt.Errorf("unexpected CONFIG GET format")
	}

	return filepath.Join(dir, dbfn), nil
}

// renameWithBak renames dump.rdb to dump.rdb.bak.
func renameWithBak(src string) error {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("source is a directory, not a file: %s", src)
	}

	baseBak := src + ".bak"
	if err := os.Rename(src, baseBak); err != nil {
		return err
	}

	return nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
