package mongodb

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/slm"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// service instance
	svr = &service{}
)

type service struct {
	mongoOps MongoDBOperationServer
	UnimplementedMongoDBOperationServer
	logger *zap.SugaredLogger

	slm              slm.ServiceLifecycleServer
	clientSet        kubernetes.Interface
	namespace        string
	serviceName      string
	serviceGroupName string
}

func (s *service) Config() error {
	s.mongoOps = app.GetGrpcApp(appName).(MongoDBOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	s.slm = app.GetGrpcApp("slm").(slm.ServiceLifecycleServer)
	clientSet, err := conf.GetConf().Kube.GetClientSet()
	if err != nil {
		return err
	}

	s.clientSet = clientSet

	namespace, err := util.IsEnvVarSet(vars.NamespaceEnvKey)
	if err != nil {
		return err
	}

	serviceName, err := util.IsEnvVarSet(vars.ServiceNameEnvKey)
	if err != nil {
		return err
	}

	serviceGroupName, err := util.IsEnvVarSet(vars.ServiceGroupNameEnvKey)
	if err != nil {
		return err
	}

	s.namespace = namespace
	s.serviceName = serviceName
	s.serviceGroupName = serviceGroupName

	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterMongoDBOperationServer(server, svr)
}

func (s *service) Backup(ctx context.Context, req *BackupRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mongodb backup", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_file": req.GetBackupFile(),
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

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	uri, err := s.getMongoDBURL(ctx)
	if err != nil {
		s.logger.Errorw("failed to get MongoDB URL", zap.Error(err))
		return nil, err
	}

	// Example:
	// mongodump --uri="mongodb://admin:xxxxxxx@ \
	//                  demo-mongodb-6rn-0.demo-mongodb-6rn-headless-svc.demo:27017, \
	//                  demo-mongodb-6rn-1.demo-mongodb-6rn-headless-svc.demo:27017, \
	//                  demo-mongodb-6rn-2.demo-mongodb-6rn-headless-svc.demo:27017/ \
	//                  ?replicaSet=demo&readPreference=secondaryPreferred" \
	//                  --oplog --gzip --archive=/backup/mongodb-backup-xxxxxxxx --numParallelCollections=4

	cmd := exec.CommandContext(ctx,
		"mongodump",
		"--uri", fmt.Sprintf("mongodb://%s:%s@%s/?replicaSet=%s&readPreference=secondaryPreferred", req.GetUsername(), password, uri, s.serviceGroupName),
		"--oplog",
		"--gzip",
		"--archive",
		"--numParallelCollections=4",
	)

	executor := common.NewCommandExecutor(s.logger)
	factory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}

	if err := executor.ExecuteCommandStreamToS3(ctx, cmd, factory, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), "backup"); err != nil {
		s.logger.Errorw("failed to execute backup", zap.Error(err))
		return nil, err
	}

	s.logger.Info("backup mongodb successfully")
	return nil, nil
}

func (s *service) Restore(ctx context.Context, req *RestoreRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mongodb restore", map[string]interface{}{
		"username":    req.GetUsername(),
		"backup_file": req.GetBackupFile(),
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

	password, err := util.DecryptPlainTextPassword(req.GetUsername())
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), zap.String("username", req.GetUsername()))
		return nil, err
	}

	// Example:
	// mongorestore --uri="mongodb://admin:xxxxxxx@demo-mongodb-6rn-0.demo-mongodb-6rn-headless-svc.demo:27017/?replicaSet=demo" \
	// 					--drop \
	// 					--gzip \
	// 					--archive
	cmd := exec.CommandContext(ctx,
		"mongorestore",
		"--uri", fmt.Sprintf("mongodb://%s:%s@127.0.0.1:27017/?replicaSet=%s", req.GetUsername(), password, s.serviceGroupName),
		"--drop",
		"--gzip",
		"--archive",
	)

	executor := common.NewCommandExecutor(s.logger)
	factory, err := req.GetObjectStorage().GenerateFactory()
	if err != nil {
		s.logger.Errorw("failed to generate storage factory", zap.Error(err))
		return nil, err
	}
	if err := executor.ExecuteCommandStreamFromS3(ctx, cmd, factory, req.GetObjectStorage().GetBucket(), req.GetBackupFile(), "restore"); err != nil {
		s.logger.Errorw("failed to execute restore", zap.Error(err))
		return nil, err
	}

	s.logger.Info("restore mongodb successfully")
	return nil, nil
}

func (s *service) getMongoDBURL(ctx context.Context) (string, error) {

	obj, err := s.clientSet.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("unitset_internal=%s", s.serviceName),
	})

	if err != nil {
		return "", err
	}

	uri := make([]string, 0)
	for _, pod := range obj.Items {
		endpoint := fmt.Sprintf("%s.%s-headless-svc.%s:27017", pod.GetName(), s.serviceName, pod.GetNamespace())
		uri = append(uri, endpoint)
	}

	return strings.Join(uri, ","), nil
}

func (s *service) SetVariable(ctx context.Context, req *SetVariableRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "mongodb set variable", map[string]interface{}{
		"key":      req.GetKey(),
		"value":    req.GetValue(),
		"username": req.GetUsername(),
		"type":     req.GetType(),
	})

	// Check process is started
	if _, err := s.slm.CheckProcessStarted(ctx, nil); err != nil {
		s.logger.Errorw("failed to check process started", zap.Error(err))
		return nil, err
	}

	// Create mongo connection
	client, err := s.newMongoClient(ctx, req.GetUsername())
	if err != nil {
		return nil, err
	}
	defer s.closeMongoClient(ctx, client)

	val, err := s.parseValueByType(req.GetType(), req.GetValue())
	if err != nil {
		s.logger.Errorw("invalid parameter value",
			zap.Error(err),
			zap.String("key", req.GetKey()),
			zap.String("type", req.GetType()),
			zap.String("value", req.GetValue()),
		)
		return nil, err
	}

	// Execute set variable
	cmd := bson.D{
		{Key: "setParameter", Value: 1},
		{Key: req.GetKey(), Value: val},
	}

	var resp bson.M
	if err := client.Database("admin").RunCommand(ctx, cmd).Decode(&resp); err != nil {
		s.logger.Errorw("failed to set parameter", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))
		return nil, fmt.Errorf("failed to set parameter failed, %v", err)
	}
	if ok, _ := resp["ok"].(float64); ok != 1 {
		err = fmt.Errorf("set parameter not ok: resp=%v", resp)
		s.logger.Errorw("failed to set parameter", zap.Error(err), zap.String("key", req.GetKey()), zap.String("value", req.GetValue()))

		return nil, err
	}

	s.logger.Info("set variable successfully")
	return nil, nil
}

func (s *service) parseValueByType(typeStr, raw string) (any, error) {
	t := strings.ToLower(strings.TrimSpace(typeStr))
	v := strings.TrimSpace(raw)

	switch t {
	case "bool", "boolean":
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid bool value: %q", raw)
		}
		return b, nil

	case "int", "integer":
		i64, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid int value: %q", raw)
		}
		// MongoDB 多数参数是 int32 语义，优先收敛
		if i64 >= math.MinInt32 && i64 <= math.MaxInt32 {
			return int32(i64), nil
		}
		return i64, nil

	case "float", "double":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float value: %q", raw)
		}
		return f, nil

	case "string", "":
		return raw, nil

	default:
		return nil, fmt.Errorf("unsupported value type: %q", typeStr)
	}
}

// newMongoClient creates a client with sane defaults.
func (s *service) newMongoClient(ctx context.Context, username string) (*mongo.Client, error) {
	password, err := util.DecryptPlainTextPassword(username)
	if err != nil {
		s.logger.Errorw("failed to decrypt password", zap.Error(err), "username", username)
		return nil, err
	}

	opts := options.Client().
		SetHosts([]string{
			"127.0.0.1:27017",
		}).
		SetAuth(options.Credential{
			Username:   username,
			Password:   password,
			AuthSource: "admin",
		}).
		SetServerSelectionTimeout(5 * time.Second).
		SetConnectTimeout(5 * time.Second)

	c, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := c.Ping(ctx, nil); err != nil {
		s.logger.Errorw("failed to ping mongo", zap.Error(err))

		_ = c.Disconnect(ctx)
		return nil, err
	}
	return c, nil
}

func (s *service) closeMongoClient(ctx context.Context, client *mongo.Client) {
	if client == nil {
		return
	}

	if err := client.Disconnect(ctx); err != nil {
		s.logger.Errorw("failed to close mongo connection", zap.Error(err))
	}
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
