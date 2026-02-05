package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	redisOps RedisOperationServer
	UnimplementedRedisOperationServer
	logger *zap.SugaredLogger

	slm slm.ServiceLifecycleServer

	dataDir string
}

func (s *service) Config() error {
	s.redisOps = app.GetGrpcApp(appName).(RedisOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)

	dataDir, err := util.IsEnvVarSet(vars.DataDirEnvKey)
	if err != nil {
		return err
	}

	s.dataDir = dataDir

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterRedisOperationServer(server, svr)
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "redis set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"username": req.GetUsername(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create connection
	rdb, err := s.newRedisClient(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeRedisClient(rdb)

	// Execute config set
	err = rdb.ConfigSet(ctx, req.GetKey(), req.GetValue()).Err()
	if err != nil {
		s.logger.Errorw("failed to set variable", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, err
	}

	s.logger.Info("set variable successfully")
	return nil, nil
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "redis backup", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"username":    req.GetUsername(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
		"ssl":         req.GetObjectStorage().GetSsl(),
		"type":        req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create connection
	rdb, err := s.newRedisClient(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeRedisClient(rdb)

	if err := ensureFreshRDBSnapshot(ctx, rdb, 2*time.Minute); err != nil {
		s.logger.Errorw("failed to ensure fresh rdb snapshot", zap.Error(err))
		return nil, err
	}

	// Discover RDB file path
	rdbPath, err := discoverRDBPath(s.dataDir)
	if err != nil {
		s.logger.Errorw("failed to discover rdb path", zap.Error(err))
		return nil, err
	}

	storageFactory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	if err := storageFactory.PutFile(ctx, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), rdbPath); err != nil {
		s.logger.Errorw("failed to put backup file", zap.Error(err))
		return nil, err
	}

	s.logger.Info("backup redis rdb file successfully")

	return nil, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "redis restore", map[string]interface{}{
		"backup_file": req.GetBackupFile(),
		"bucket":      req.GetObjectStorage().GetBucket(),
		"endpoint":    req.GetObjectStorage().GetEndpoint(),
		"access_key":  req.GetObjectStorage().GetAccessKey(),
		"secret_key":  req.GetObjectStorage().GetSecretKey(),
		"ssl":         req.GetObjectStorage().GetSsl(),
		"type":        req.GetObjectStorage().GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStopped(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process stopped", zap.Error(err))
		return nil, err
	}

	// Discover RDB file path
	rdbPath, err := discoverRDBPath(s.dataDir)
	if err != nil {
		s.logger.Errorw("failed to discover rdb path", zap.Error(err))
		return nil, err
	}

	storageFactory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	// Rename dump.rdb to .bak
	if err := renameWithBak(rdbPath); err != nil {
		s.logger.Errorw("failed to rename rdb file", zap.Error(err))
		return nil, err
	}

	if err := storageFactory.GetFile(ctx, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), rdbPath); err != nil {
		s.logger.Errorw("failed to get backup file", zap.Error(err))
		return nil, err
	}

	if err := os.Chmod(rdbPath, 0644); err != nil {
		s.logger.Errorw("failed to chmod rdb file", zap.Error(err))
		return nil, err
	}

	if err := os.Chown(rdbPath, 1001, 1001); err != nil {
		s.logger.Errorw("failed to chown rdb file", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore redis rdb file successfully")

	return nil, nil
}

// newRedisClient creates a Redis connection
func (s *service) newRedisClient(ctx context.Context, username string) (*redis.Client, error) {
	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: password,
		DB:       0,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		s.logger.Errorw("failed to ping redis", zap.Error(err))

		s.closeRedisClient(rdb)
		return nil, err
	}

	return rdb, nil
}

func (s *service) closeRedisClient(client *redis.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		s.logger.Errorw("failed to close redis connection", zap.Error(err))
	}
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
	ticker := time.NewTicker(2 * time.Second)
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

//func getConfigValue(ctx context.Context, client *redis.Client, key string) (string, error) {
//	resp, err := client.Do(ctx, "CONFIG", "GET", key).Result()
//	if err != nil {
//		return "", fmt.Errorf("failed to CONFIG GET %s: %w", key, err)
//	}
//
//	var (
//		result string
//	)
//
//	switch arr := resp.(type) {
//	case map[interface{}]interface{}:
//		if value, ok := arr[key]; ok {
//			result = value.(string)
//		}
//	}
//
//	if result == "" {
//		return "", fmt.Errorf("unexpected redis CONFIG GET %s response", key)
//	}
//
//	return result, nil
//}

//func discoverRDBPath(ctx context.Context, rdb *redis.Client) (string, error) {
//	dir, err := getConfigValue(ctx, rdb, "dir")
//	if err != nil {
//		return "", err
//	}
//
//	dbfn, err := getConfigValue(ctx, rdb, "dbfilename")
//	if err != nil {
//		return "", err
//	}
//
//	if dir == "" || dbfn == "" {
//		return "", fmt.Errorf("unexpected CONFIG GET format")
//	}
//
//	return filepath.Join(dir, dbfn), nil
//}

func discoverRDBPath(dataDir string) (string, error) {
	rdbPath := filepath.Join(dataDir, "dump.rdb")

	if exists := util.IsFileExist(rdbPath); !exists {
		return rdbPath, fmt.Errorf("rdb path %s does not exist", rdbPath)
	}

	return rdbPath, nil
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
