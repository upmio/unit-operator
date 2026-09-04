package hugegraphhubble

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/upmio/unit-operator/pkg/agent/app"
	"github.com/upmio/unit-operator/pkg/agent/app/common"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const graphConnectionsPath = "/api/v1.2/graph-connections"

var svr = &service{}

type opsConfig struct {
	hubbleHost string
	hubblePort string
}

type service struct {
	hubbleOps HugeGraphHubbleOperationServer
	UnimplementedHugeGraphHubbleOperationServer
	logger *zap.SugaredLogger

	opsCfg *opsConfig
	client *http.Client

	operationMu    sync.Mutex
	runner         commandRunner
	storageFactory func(*common.ObjectStorage) (common.ObjectStorageFactory, error)
}

type graphConnection struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Graph    string `json:"graph"`
	Host     string `json:"host"`
	Port     int32  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type graphConnectionsPage struct {
	Records []graphConnection `json:"records"`
}

type hubbleResponse struct {
	Status  int             `json:"status"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

func (s *service) Config() error {
	s.hubbleOps = app.GetGrpcApp(appName).(HugeGraphHubbleOperationServer)
	s.logger = zap.L().Named(appName).Sugar()

	port := strings.TrimSpace(os.Getenv("HUBBLE_PORT"))
	if port == "" {
		return fmt.Errorf("HUBBLE_PORT must be set")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("HUBBLE_PORT must be an integer in [1, 65535]")
	}

	host := strings.TrimSpace(os.Getenv("HUBBLE_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	s.opsCfg = &opsConfig{
		hubbleHost: host,
		hubblePort: port,
	}
	s.client = &http.Client{Timeout: 10 * time.Second}
	s.runner = commandRunnerFunc(runCommand)
	s.storageFactory = func(storage *common.ObjectStorage) (common.ObjectStorageFactory, error) {
		return storage.GenerateFactory()
	}
	return nil
}

func (s *service) Name() string {
	return appName
}

func (s *service) Registry(server *grpc.Server) {
	RegisterHugeGraphHubbleOperationServer(server, svr)
}

func (s *service) ConfigureGraphConnection(ctx context.Context, req *ConfigureGraphConnectionRequest) (*common.Empty, error) {
	if err := validateConfigureRequest(req); err != nil {
		return nil, err
	}

	s.logger.Infow("configure hugegraph hubble graph connection",
		"name", req.GetName(), "graph", req.GetGraph(), "host", req.GetHost(), "port", req.GetPort())

	connections, err := s.listGraphConnections(ctx)
	if err != nil {
		return nil, err
	}

	for _, existing := range connections {
		if existing.Name != req.GetName() {
			continue
		}
		if sameGraphConnection(existing, req) {
			s.logger.Infof("graph connection %q already has the requested configuration", req.GetName())
			return &common.Empty{}, nil
		}
		return nil, fmt.Errorf("graph connection %q already exists with different graph, host, or port", req.GetName())
	}

	payload, err := json.Marshal(map[string]interface{}{
		"name":  req.GetName(),
		"graph": req.GetGraph(),
		"host":  req.GetHost(),
		"port":  req.GetPort(),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal graph connection request: %w", err)
	}

	if _, err := s.callHubble(ctx, http.MethodPost, graphConnectionsPath, payload); err != nil {
		return nil, fmt.Errorf("create graph connection %q: %w", req.GetName(), err)
	}

	s.logger.Infof("created graph connection %q", req.GetName())
	return &common.Empty{}, nil
}

func validateConfigureRequest(req *ConfigureGraphConnectionRequest) error {
	if req == nil {
		return fmt.Errorf("configure graph connection request must not be nil")
	}
	for field, value := range map[string]string{
		"name":  req.GetName(),
		"graph": req.GetGraph(),
		"host":  req.GetHost(),
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be set", field)
		}
	}
	if req.GetPort() < 1 || req.GetPort() > 65535 {
		return fmt.Errorf("port must be in [1, 65535]")
	}
	return nil
}

func sameGraphConnection(existing graphConnection, requested *ConfigureGraphConnectionRequest) bool {
	return existing.Graph == requested.GetGraph() &&
		existing.Host == requested.GetHost() &&
		existing.Port == requested.GetPort()
}

func (s *service) listGraphConnections(ctx context.Context) ([]graphConnection, error) {
	data, err := s.callHubble(ctx, http.MethodGet, graphConnectionsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("list graph connections: %w", err)
	}

	var page graphConnectionsPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, fmt.Errorf("decode graph connections response: %w", err)
	}
	return page.Records, nil
}

func (s *service) graphConnection(ctx context.Context, name string) (graphConnection, error) {
	connections, err := s.listGraphConnections(ctx)
	if err != nil {
		return graphConnection{}, err
	}
	for _, connection := range connections {
		if connection.Name == name {
			return connection, nil
		}
	}
	return graphConnection{}, fmt.Errorf("graph connection %q was not found", name)
}

func (s *service) callHubble(ctx context.Context, method, requestPath string, payload []byte) (json.RawMessage, error) {
	if s.opsCfg == nil {
		return nil, fmt.Errorf("hugegraph-hubble agent is not configured")
	}

	endpoint := "http://" + net.JoinHostPort(s.opsCfg.hubbleHost, s.opsCfg.hubblePort) + requestPath
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create hubble request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := s.client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call hubble API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read hubble API response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("hubble API returned HTTP %d", response.StatusCode)
	}

	var result hubbleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode hubble API response: %w", err)
	}
	if result.Status != http.StatusOK {
		if result.Message == "" {
			return nil, fmt.Errorf("hubble API returned status %d", result.Status)
		}
		return nil, fmt.Errorf("hubble API returned status %d: %s", result.Status, result.Message)
	}
	return result.Data, nil
}

func RegistryGrpcApp() {
	app.RegistryGrpcApp(svr)
}
