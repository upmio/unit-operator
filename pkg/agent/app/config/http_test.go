package config

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GetGrpcApp  get gRPC
var GetGrpcApp = func(name string) interface{} {
	return nil //
}

// MockSyncConfigServiceServer mock to SyncConfigServiceServer
type HttpTestMockSyncConfigServiceServer struct {
	mock.Mock
}

func (m *HttpTestMockSyncConfigServiceServer) SyncConfig(ctx context.Context, req *SyncConfigRequest) (*SyncConfigResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*SyncConfigResponse), args.Error(1)
}

//func TestConfig_Success(t *testing.T) {
//	mockService := new(HttpTestMockSyncConfigServiceServer)
//	GetGrpcApp = func(name string) interface{} {
//		return mockService
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.NoError(t, err)
//	assert.Equal(t, mockService, h.service)
//}

//func TestConfig_Failure(t *testing.T) {
//	GetGrpcApp = func(name string) interface{} {
//		return nil
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.Error(t, err)
//	assert.Nil(t, h.service)
//}

//func TestConfig_TypeAssertionFailure(t *testing.T) {
//	GetGrpcApp = func(name string) interface{} {
//		return "not a SyncConfigServiceServer"
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.Error(t, err)
//	assert.Nil(t, h.service)
//}

//func TestConfig_EmptyService(t *testing.T) {
//	GetGrpcApp = func(name string) interface{} {
//		return nil
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.Error(t, err)
//	assert.Nil(t, h.service)
//}

//func TestConfig_InvalidAppName(t *testing.T) {
//	GetGrpcApp = func(name string) interface{} {
//		if name == appName {
//			return nil
//		}
//		return new(MockSyncConfigServiceServer)
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.Error(t, err)
//	assert.Nil(t, h.service)
//}

//func TestConfig_WithValidService(t *testing.T) {
//	mockService := new(MockSyncConfigServiceServer)
//	GetGrpcApp = func(name string) interface{} {
//		return mockService
//	}
//
//	h := &handler{}
//	err := h.Config()
//
//	assert.NoError(t, err)
//	assert.Equal(t, mockService, h.service)
//}

func TestRegistry_InvalidPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()

	h := &handler{}
	h.Registry(r, "/config")

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/invalid/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSyncConfigRouter_BindJSONError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	h := &handler{}

	r.POST("/sync", h.SyncConfigRouter)
	body := bytes.NewBufferString(`invalid JSON`)

	req, _ := http.NewRequest(http.MethodPost, "/sync", body)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Request body binding failed")
}

func TestSyncConfigRouter_EmptyRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()

	h := &handler{}
	r.POST("/sync", h.SyncConfigRouter)

	req, _ := http.NewRequest(http.MethodPost, "/sync", bytes.NewBuffer(nil))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Request body binding failed")
}
