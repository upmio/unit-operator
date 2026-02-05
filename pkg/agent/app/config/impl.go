package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends"
	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/template"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"github.com/upmio/unit-operator/pkg/agent/pkg/util"
	"github.com/upmio/unit-operator/pkg/agent/vars"
	"k8s.io/apimachinery/pkg/util/json"

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
	s.logger = zap.L().Named(appName).Sugar()
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

func (s *service) SyncConfig(ctx context.Context, req *SyncConfigRequest) (*common.Empty, error) {
	util.LogRequestSafely(s.logger, "sync config", map[string]interface{}{
		"key":              req.GetKey(),
		"namespace":        req.GetNamespace(),
		"configmap":        req.GetValueConfigmapName(),
		"extend_configmap": req.GetExtendValueConfigmaps(),
		"template":         req.GetTemplateConfigmapName(),
	})

	if err := s.fetchDataFromConfigMap(ctx, req.GetValueConfigmapName(), req.GetTemplateConfigmapName(), req.GetKey(), req.GetNamespace()); err != nil {
		s.logger.Errorw("failed to fetch data from config map", zap.Error(err))
		return nil, err
	}

	if len(req.GetExtendValueConfigmaps()) != 0 {
		for _, name := range req.GetExtendValueConfigmaps() {
			obj, err := s.clientSet.CoreV1().ConfigMaps(req.GetNamespace()).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("can't find config map %s/%s: %v", req.GetNamespace(), name, err)
			}

			value, err := json.Marshal(obj.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal config map %s/%s: %v", req.GetNamespace(), name, err)
			}
			s.confdConfig.BackendsConfig.Contents = append(s.confdConfig.BackendsConfig.Contents, string(value))
		}
	}

	// Initialize the storage client
	storeClient, err := backends.New(s.confdConfig.BackendsConfig)
	if err != nil {
		s.logger.Errorw("failed to init backend store", zap.Error(err))
		return nil, err
	}

	s.confdConfig.TemplateConfig.StoreClient = storeClient
	if err := template.Process(s.confdConfig.TemplateConfig); err != nil {
		s.logger.Errorw("failed to process template config", zap.Error(err))
		return nil, err
	}

	s.logger.Info("sync config successfully")
	return nil, nil
}

func (s *service) fetchDataFromConfigMap(ctx context.Context, valueObjName, templateObjName, key, namespace string) error {
	// Get CONFIG_PATH environment variable
	path, err := util.IsEnvVarSet(vars.ConfigPathEnvKey)
	if err != nil {
		return err
	}
	s.confdConfig.DestFile = path

	// Check template configmap
	templateObj, err := s.clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, templateObjName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find config map %s/%s: %v", namespace, templateObjName, err)
	}

	templateContent, ok := templateObj.Data[key]
	if !ok {
		return fmt.Errorf("config map %s/%s doesn't has key %s", namespace, templateObjName, key)
	}

	tmplFile := filepath.Join("/tmp", "template.tmpl")
	if err := os.WriteFile(tmplFile, []byte(templateContent), 0644); err != nil {
		return fmt.Errorf("failed to write template file %s: %v", tmplFile, err)
	}

	//defer func() { _ = os.Remove(tmplFile) }()
	s.confdConfig.TemplateConfig.TemplateFile = tmplFile

	// Check value configmap
	s.confdConfig.BackendsConfig.Contents = make([]string, 0)

	valueObj, err := s.clientSet.CoreV1().ConfigMaps(namespace).Get(ctx, valueObjName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find config map %s/%s: %v", namespace, valueObjName, err)
	}

	valueContent, ok := valueObj.Data[key]
	if !ok {
		return fmt.Errorf("config map %s/%s doesn't has key %s", namespace, valueObjName, key)
	}

	s.confdConfig.BackendsConfig.Contents = append(s.confdConfig.BackendsConfig.Contents, valueContent)

	return nil
}

func init() {
	app.RegistryGrpcApp(svr)
}
