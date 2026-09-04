package hugegraphhubble

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigureGraphConnectionCreatesConnection(t *testing.T) {
	var postBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, graphConnectionsPath, r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{})
		case http.MethodPost:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&postBody))
			writeHubbleResponse(t, w, http.StatusOK, graphConnection{ID: 1, Name: "upm_hugegraph_example"})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.NoError(t, err)
	require.Equal(t, "upm_hugegraph_example", postBody["name"])
	require.Equal(t, "hugegraph", postBody["graph"])
	require.Equal(t, "hugegraph-svc.example.svc", postBody["host"])
	require.EqualValues(t, 8080, postBody["port"])
}

func TestConfigureGraphConnectionIsIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
			ID:    1,
			Name:  "upm_hugegraph_example",
			Graph: "hugegraph",
			Host:  "hugegraph-svc.example.svc",
			Port:  8080,
		}}})
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.NoError(t, err)
}

func TestConfigureGraphConnectionRejectsConflictingName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		writeHubbleResponse(t, w, http.StatusOK, graphConnectionsPage{Records: []graphConnection{{
			Name:  "upm_hugegraph_example",
			Graph: "hugegraph",
			Host:  "different-hugegraph-svc.example.svc",
			Port:  8080,
		}}})
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.EqualError(t, err, "graph connection \"upm_hugegraph_example\" already exists with different graph, host, or port")
}

func TestConfigureGraphConnectionReportsHubbleApplicationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":400,"data":null,"message":"Hubble is unavailable"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	service := newTestService(t, server)
	_, err := service.ConfigureGraphConnection(context.Background(), &ConfigureGraphConnectionRequest{
		Name:  "upm_hugegraph_example",
		Graph: "hugegraph",
		Host:  "hugegraph-svc.example.svc",
		Port:  8080,
	})
	require.EqualError(t, err, "list graph connections: hubble API returned status 400: Hubble is unavailable")
}

func TestValidateConfigureRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *ConfigureGraphConnectionRequest
	}{
		{name: "nil request", req: nil},
		{name: "missing name", req: &ConfigureGraphConnectionRequest{Graph: "hugegraph", Host: "hugegraph-svc", Port: 8080}},
		{name: "invalid port", req: &ConfigureGraphConnectionRequest{Name: "connection", Graph: "hugegraph", Host: "hugegraph-svc"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, validateConfigureRequest(test.req))
		})
	}
}

func newTestService(t *testing.T, server *httptest.Server) *service {
	t.Helper()
	address := strings.TrimPrefix(server.URL, "http://")
	host, port, err := splitAddress(address)
	require.NoError(t, err)
	return &service{
		logger: zap.NewNop().Sugar(),
		opsCfg: &opsConfig{
			hubbleHost: host,
			hubblePort: port,
		},
		client: server.Client(),
	}
}

func splitAddress(address string) (string, string, error) {
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] != ':' {
			continue
		}
		port, err := strconv.Atoi(address[i+1:])
		if err != nil {
			return "", "", err
		}
		return address[:i], strconv.Itoa(port), nil
	}
	return "", "", strconv.ErrSyntax
}

func writeHubbleResponse(t *testing.T, w http.ResponseWriter, status int, data interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(hubbleResponse{Status: status, Data: mustMarshal(t, data)}))
}

func mustMarshal(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}
