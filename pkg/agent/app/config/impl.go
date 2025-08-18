package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/template"
	"github.com/upmio/unit-operator/pkg/agent/conf"
)

var (
	// service instance
	svr = &service{}
)

const (
	confPathEnvKey = "CONFIG_PATH"
)

type service struct {
	syncConfig SyncConfigServiceServer
	clientSet  kubernetes.Interface
	UnimplementedSyncConfigServiceServer
	logger      *zap.SugaredLogger
	confdConfig *confd.Config
}

// Common helper methods

// newConfigResponse creates a new Config Response with the given message
func newConfigResponse(message string) *Response {
	return &Response{Message: message}
}

// getEnvVarOrError
// gets environment variable or returns error if not found
func getEnvVarOrError(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}

func (s *service) Config() error {
	clientSet, err := conf.GetConf().GetClientSet()
	if err != nil {
		return err
	}

	s.clientSet = clientSet
	s.syncConfig = app.GetGrpcApp(appName).(SyncConfigServiceServer)
	s.logger = zap.L().Named("[CONFIG]").Sugar()
	s.confdConfig = &confd.Config{
		BackendsConfig: confd.BackendsConfig{
			Backend: "content",
		},
	}
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterSyncConfigServiceServer(server, svr)
}

func (s *service) SyncConfig(ctx context.Context, req *SyncConfigRequest) (*Response, error) {
	common.LogRequestSafely(s.logger, "sync config clone", map[string]interface{}{
		"key":              req.GetKey(),
		"namespace":        req.GetNamespace(),
		"configmap":        req.GetValueConfigmapName(),
		"extend_configmap": req.GetExtendValueConfigmaps(),
		"template":         req.GetTemplateConfigmapName(),
	})

	// Get CONFIG_PATH environment variable
	path, err := getEnvVarOrError(confPathEnvKey)
	if err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, "failed to get CONFIG_PATH environment variable", err)
	}
	s.confdConfig.DestFile = path

	// Check template configmap
	templateConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetTemplateConfigmapName(), metav1.GetOptions{})
	if err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to fetch template configmap[%s] in namespace[%s]", req.GetTemplateConfigmapName(), req.GetNamespace()), err)
	}

	if value, ok := templateConfigMapObj.Data[req.GetKey()]; !ok {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to found template configmap[%s] key[%s]", req.GetTemplateConfigmapName(), req.GetKey()), errors.New("not found"))
	} else {
		tmplFile := filepath.Join("/tmp", "template.tmpl")
		err := os.WriteFile(tmplFile, []byte(value), 0644)
		if err != nil {
			return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to write template file [%s]", tmplFile), err)
		} else {
			//defer os.Remove(tmplFile)
			s.logger.Debugf("write template file [%s] successfully", tmplFile)
			s.confdConfig.TemplateFile = tmplFile
		}

	}

	// Check value configmap
	valueConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetValueConfigmapName(), metav1.GetOptions{})
	if err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to fetch value configmap[%s] in namespace[%s]", req.GetValueConfigmapName(), req.GetNamespace()), err)
	}

	s.confdConfig.Contents = make([]string, 0)

	if value, ok := valueConfigMapObj.Data[req.GetKey()]; !ok {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to found value configmap[%s] key[%s]", req.GetValueConfigmapName(), req.GetKey()), errors.New("not found"))
	} else {
		s.confdConfig.Contents = append(s.confdConfig.Contents, value)
	}

	if len(req.GetExtendValueConfigmaps()) != 0 {
		for _, extendConfigmapName := range req.GetExtendValueConfigmaps() {
			extendConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, extendConfigmapName, metav1.GetOptions{})
			if err != nil {
				return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to fetch extend configmap[%s] in namespace[%s]", extendConfigmapName, req.GetNamespace()), err)
			}

			if temp, err := json.Marshal(extendConfigMapObj.Data); err != nil {
				return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to marshal extend value configmap[%s] in namespace[%s]", extendConfigMapObj, req.GetNamespace()), err)
			} else {
				s.confdConfig.Contents = append(s.confdConfig.Contents, string(temp))
			}
		}
	}

	// Initialize the storage client
	storeClient, err := backends.New(s.confdConfig.BackendsConfig)
	if err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, "failed to generate store client", err)
	}

	s.confdConfig.StoreClient = storeClient
	if err := template.Process(s.confdConfig.TemplateConfig); err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, "failed to generate config file", err)
	}

	return common.LogAndReturnSuccess(s.logger, newConfigResponse, "generate config file successfully")
}

func (s *service) RewriteConfig(ctx context.Context, req *RewriteConfigRequest) (*Response, error) {
	s.logger.With(
		"key", req.GetKey(),
		"namespace", req.GetNamespace(),
		"value", req.GetValue(),
		"configmap", req.GetConfigmapName(),
	).Info("receive rewrite config request")

	configMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetConfigmapName(), metav1.GetOptions{})
	if err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to fetch configmap[%s] in namespace[%s]", req.GetConfigmapName(), req.GetNamespace()), err)
	}

	if value, ok := configMapObj.Data[req.GetKey()]; ok && value == req.GetValue() {
		return common.LogAndReturnSuccess(s.logger, newConfigResponse, fmt.Sprintf("configmap[%s] key[%s] value unchanged, skipping update", req.GetConfigmapName(), req.GetKey()))
	}

	configMapObj.Data[req.GetKey()] = req.GetValue()
	if _, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Update(ctx, configMapObj, metav1.UpdateOptions{}); err != nil {
		return common.LogAndReturnError(s.logger, newConfigResponse, fmt.Sprintf("failed to update configmap[%s]", req.GetConfigmapName()), err)
	}

	return common.LogAndReturnSuccess(s.logger, newConfigResponse, fmt.Sprintf("rewrite configmap[%s] successfully", req.GetConfigmapName()))
}

func init() {
	app.RegistryGrpcApp(svr)
}
