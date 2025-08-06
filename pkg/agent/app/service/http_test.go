package service

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockApp simulates the app package behavior for testing purposes.
type MockApp struct {
	GetGrpcAppFunc func(name string) interface{}
}

// GetGrpcApp calls the mock function defined in MockApp.
func (m *MockApp) GetGrpcApp(name string) interface{} {
	return m.GetGrpcAppFunc(name)
}

func TestHandler_Config(t *testing.T) {
	tests := []struct {
		name           string
		getGrpcAppFunc func(name string) interface{}
		expectError    bool
	}{
		{
			name: "Success - Correct type",
			getGrpcAppFunc: func(name string) interface{} {
				return &MockServiceLifecycleServer{}
			},
			expectError: false,
		},
		//{
		//	name: "Failure - Incorrect type",
		//	getGrpcAppFunc: func(name string) interface{} {
		//		return "invalid type"
		//	},
		//	expectError: true,
		//},
		//{
		//	name: "Failure - Nil return",
		//	getGrpcAppFunc: func(name string) interface{} {
		//		return nil
		//	},
		//	expectError: true,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the mock app.
			//mockApp := &MockApp{
			//	GetGrpcAppFunc: tt.getGrpcAppFunc,
			//}

			// Replace the app.GetGrpcApp function with the mock.
			//originalGetGrpcApp := app.GetGrpcApp
			//defer func() { app.GetGrpcApp = originalGetGrpcApp }()
			//app.GetGrpcApp = mockApp.GetGrpcApp

			// Create the handler and call Config.
			h := &handler{}
			err := h.Config()

			// Check if the error matches the expectation.
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, h.service)
			}
		})
	}
}

// MockServiceLifecycleServer is a mock implementation of ServiceLifecycleServer.
//type MockServiceLifecycleServer struct {
//	UnimplementedServiceLifecycleServer
//}

// MockHandler provides mock implementations for handler methods.
type MockHandler struct{}

func (h *MockHandler) StartServiceRouter(c *gin.Context) {
	c.String(http.StatusOK, "StartServiceRouter called")
}

func (h *MockHandler) StopServiceRouter(c *gin.Context) {
	c.String(http.StatusOK, "StopServiceRouter called")
}

//func TestHandler_Registry(t *testing.T) {
//	// Create a new Gin engine
//	gin.SetMode(gin.TestMode)
//	router := gin.New()
//
//	// Initialize the handler
//	//h := &MockHandler{}
//	//h.Registry(router, "/services")
//
//	// Define test cases
//	tests := []struct {
//		name       string
//		method     string
//		path       string
//		statusCode int
//		body       string
//	}{
//		{
//			name:       "POST /api/v1/services/start",
//			method:     http.MethodPost,
//			path:       "/api/v1/services/start",
//			statusCode: http.StatusOK,
//			body:       "StartServiceRouter called",
//		},
//		{
//			name:       "POST /api/v1/services/stop",
//			method:     http.MethodPost,
//			path:       "/api/v1/services/stop",
//			statusCode: http.StatusOK,
//			body:       "StopServiceRouter called",
//		},
//		{
//			name:       "POST /api/v1/services/invalid",
//			method:     http.MethodPost,
//			path:       "/api/v1/services/invalid",
//			statusCode: http.StatusNotFound,
//			body:       "",
//		},
//	}
//
//	// Run the tests
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			req := httptest.NewRequest(tt.method, tt.path, nil)
//			resp := httptest.NewRecorder()
//
//			router.ServeHTTP(resp, req)
//
//			assert.Equal(t, tt.statusCode, resp.Code)
//			assert.Equal(t, tt.body, resp.Body.String())
//		})
//	}
//}

// MockService implements the ServiceLifecycleServer interface for testing.
type MockService struct{}

//func (m *MockService) StartService(ctx *gin.Context, req *ServiceRequest) (*Response, error) {
//	// You can customize this mock behavior based on test case needs.
//	if req.Param == "error" {
//		return &Response{Message: "Service failed to start"}, errors.New("service start error")
//	}
//	return &Response{Message: "Service started successfully"}, nil
//}

// TestHandler_StartServiceRouter tests the StartServiceRouter method.
//func TestHandler_StartServiceRouter(t *testing.T) {
//	// Set up Gin in test mode
//	gin.SetMode(gin.TestMode)
//
//	// Create a new Gin engine
//	router := gin.New()
//
//	// Initialize the handler with a mock service
//	//h := &handler{service: &MockService{}}
//	router.POST("/start", h.StartServiceRouter)
//
//	// Define test cases
//	tests := []struct {
//		name           string
//		requestBody    string
//		expectedStatus int
//		expectedBody   string
//	}{
//		//{
//		//	name:           "Successful service start",
//		//	requestBody:    `{"param":"success"}`,
//		//	expectedStatus: http.StatusCreated,
//		//	expectedBody:   `{"Message":"Service started successfully"}`,
//		//},
//		{
//			name:           "Service start failure",
//			requestBody:    `{"param":"error"}`,
//			expectedStatus: http.StatusInternalServerError,
//			expectedBody:   `{"Message":"Service failed to start"}`,
//		},
//	}
//
//	// Run test cases
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a new HTTP request
//			req := httptest.NewRequest(http.MethodPost, "/start", nil)
//			req.Header.Set("Content-Type", "application/json")
//			resp := httptest.NewRecorder()
//
//			// Perform the request
//			router.ServeHTTP(resp, req)
//
//			// Assert the status code and response body
//			assert.Equal(t, tt.expectedStatus, resp.Code)
//			assert.JSONEq(t, tt.expectedBody, resp.Body.String())
//		})
//	}
//}

// TestHandler_StopServiceRouter tests the StopServiceRouter method.
//func TestHandler_StopServiceRouter(t *testing.T) {
//	// Set up Gin in test mode
//	gin.SetMode(gin.TestMode)
//
//	// Create a new Gin engine
//	router := gin.New()
//
//	// Initialize the handler with a mock service
//	//h := &handler{service: &MockService{}}
//	router.POST("/stop", h.StopServiceRouter)
//
//	// Define test cases
//	tests := []struct {
//		name           string
//		requestBody    string
//		expectedStatus int
//		expectedBody   string
//	}{
//		{
//			name:           "Successful service stop",
//			requestBody:    `{"param":"success"}`,
//			expectedStatus: http.StatusCreated,
//			expectedBody:   `{"Message":"Service stopped successfully"}`,
//		},
//		{
//			name:           "Service stop failure",
//			requestBody:    `{"param":"error"}`,
//			expectedStatus: http.StatusInternalServerError,
//			expectedBody:   `{"Message":"Service failed to stop"}`,
//		},
//	}
//
//	// Run test cases
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a new HTTP request
//			req := httptest.NewRequest(http.MethodPost, "/stop", nil)
//			req.Header.Set("Content-Type", "application/json")
//			resp := httptest.NewRecorder()
//
//			// Perform the request
//			router.ServeHTTP(resp, req)
//
//			// Assert the status code and response body
//			assert.Equal(t, tt.expectedStatus, resp.Code)
//			assert.JSONEq(t, tt.expectedBody, resp.Body.String())
//		})
//	}
//}
