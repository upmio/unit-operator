package config

import (
	"context"
	"fmt"
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
	// service service instance
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

func (s *service) Config() error {
	clientSet, err := conf.GetConf().Kube.GetClientSet()
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

func (s *service) SyncConfig(ctx context.Context, req *SyncConfigRequest) (*SyncConfigResponse, error) {
	s.logger.With(
		"key", req.GetKey(),
		"namespace", req.GetNamespace(),
		"configmap", req.GetValueConfigmapName(),
		"extend_configmap", req.GetExtendValueConfigmaps(),
		"template", req.GetTemplateConfigmapName(),
	).Info("receive sync config request")

	// Check CONFIG_PATH environment is exists
	if path := os.Getenv(confPathEnvKey); path == "" {
		errMsg := fmt.Sprintf("failed to get environment variables[%s]", confPathEnvKey)
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	} else {
		s.confdConfig.TemplateConfig.DestFile = path
	}

	// Check template configmap
	templateConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetTemplateConfigmapName(), metav1.GetOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch template configmap[%s] in namespace[%s]: %v", req.GetTemplateConfigmapName(), req.GetNamespace(), err)
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	}

	if value, ok := templateConfigMapObj.Data[req.GetKey()]; !ok {
		errMsg := fmt.Sprintf("failed to found template configmap[%s] key[%s]", req.GetTemplateConfigmapName(), req.GetKey())
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	} else {
		tmplFile := filepath.Join("/tmp", "template.tmpl")
		err := os.WriteFile(tmplFile, []byte(value), 0644)
		if err != nil {
			errMsg := fmt.Sprintf("failed to write template file [%s]: %v", tmplFile, err)
			s.logger.Errorf(errMsg)
			return &SyncConfigResponse{
				Message: errMsg,
			}, fmt.Errorf(errMsg)
		} else {
			//defer os.Remove(tmplFile)
			s.logger.Debugf("write template file [%s] successfully", tmplFile)
			s.confdConfig.TemplateConfig.TemplateFile = tmplFile
		}

	}

	// Check value configmap
	valueConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetValueConfigmapName(), metav1.GetOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch value configmap[%s] in namespace[%s]: %v", req.GetValueConfigmapName(), req.GetNamespace(), err)
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	}

	s.confdConfig.BackendsConfig.Contents = make([]string, 0)

	if value, ok := valueConfigMapObj.Data[req.GetKey()]; !ok {
		errMsg := fmt.Sprintf("failed to found value configmap[%s] key[%s]", req.GetValueConfigmapName(), req.GetKey())
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	} else {
		s.confdConfig.BackendsConfig.Contents = append(s.confdConfig.BackendsConfig.Contents, value)
	}

	if len(req.GetExtendValueConfigmaps()) != 0 {
		for _, extendConfigmapName := range req.GetExtendValueConfigmaps() {
			extendConfigMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, extendConfigmapName, metav1.GetOptions{})
			if err != nil {
				errMsg := fmt.Sprintf("failed to fetch extend value configmap[%s] in namespace[%s]: %v", extendConfigmapName, req.GetNamespace(), err)
				s.logger.Errorf(errMsg)
				return &SyncConfigResponse{
					Message: errMsg,
				}, fmt.Errorf(errMsg)
			}

			if temp, err := json.Marshal(extendConfigMapObj.Data); err != nil {
				errMsg := fmt.Sprintf("failed to marshal extend value configmap[%s] in namespace[%s]: %v", extendConfigMapObj, req.GetNamespace(), err)
				s.logger.Errorf(errMsg)
				return &SyncConfigResponse{
					Message: errMsg,
				}, fmt.Errorf(errMsg)
			} else {
				s.confdConfig.BackendsConfig.Contents = append(s.confdConfig.BackendsConfig.Contents, string(temp))
			}
		}
	}

	// Initialize the storage client
	storeClient, err := backends.New(s.confdConfig.BackendsConfig)

	s.confdConfig.TemplateConfig.StoreClient = storeClient
	if err := template.Process(s.confdConfig.TemplateConfig); err != nil {
		errMsg := fmt.Sprintf("failed to generate config file: %v", err)
		s.logger.Errorf(errMsg)
		return &SyncConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	}

	successMsg := "generate config file successfully"
	s.logger.Info(successMsg)

	return &SyncConfigResponse{Message: successMsg}, nil
}

func (s *service) RewriteConfig(ctx context.Context, req *RewriteConfigRequest) (*RewriteConfigResponse, error) {
	s.logger.With(
		"key", req.GetKey(),
		"namespace", req.GetNamespace(),
		"value", req.GetValue(),
		"configmap", req.GetConfigmapName(),
	).Info("receive rewrite config request")

	configMapObj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, req.GetConfigmapName(), metav1.GetOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch configmap[%s] in namespace[%s]: %v", req.GetConfigmapName(), req.GetNamespace(), err)
		s.logger.Errorf(errMsg)
		return &RewriteConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	}

	if value, ok := configMapObj.Data[req.GetKey()]; ok && value == req.GetValue() {
		successMsg := fmt.Sprintf("configmap[%s] key[%s] value unchanged, skipping update", req.GetConfigmapName(), req.GetKey())
		s.logger.Info(successMsg)

		return &RewriteConfigResponse{Message: successMsg}, nil
	}

	configMapObj.Data[req.GetKey()] = req.GetValue()
	if _, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Update(ctx, configMapObj, metav1.UpdateOptions{}); err != nil {
		errMsg := fmt.Sprintf("failed to update configmap[%s]: %v", req.GetConfigmapName(), err)
		s.logger.Errorf(errMsg)
		return &RewriteConfigResponse{
			Message: errMsg,
		}, fmt.Errorf(errMsg)
	}

	successMsg := fmt.Sprintf("rewrite configmap[%s] successfully", req.GetConfigmapName())
	s.logger.Info(successMsg)

	return &RewriteConfigResponse{Message: successMsg}, nil
}

func init() {
	app.RegistryGrpcApp(svr)
}
