package protocol

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/upmio/unit-operator/pkg/agent/conf"
	"net/http"
	"testing"
)

//type MockConfig struct {
//	mock.Mock
//}

func (m *MockConfig) GetConf() *conf.Config {
	args := m.Called()
	return args.Get(0).(*conf.Config)
}

//func TestNewHTTPService(t *testing.T) {
//	testCases := []struct {
//		name             string
//		mockConfig       *MockConfig
//		expectedAddr     string
//		expectedTimeout  time.Duration
//		expectedLogLevel zapcore.Level
//	}{
//		{
//			name: "Debug Level Logging",
//			mockConfig: func() *MockConfig {
//				mockConfig := new(MockConfig)
//				mockConfig.On("GetConf").Return(&conf.Config{
//					App: &conf.App{
//						Host: "localhost",
//						Port: 8080,
//					},
//					Log: &conf.Log{
//						Level: "debug",
//					},
//				})
//				return mockConfig
//			}(),
//			expectedAddr:     "localhost:8080",
//			expectedTimeout:  60 * time.Second,
//			expectedLogLevel: zap.DebugLevel,
//		},
//		{
//			name: "Info Level Logging",
//			mockConfig: func() *MockConfig {
//				mockConfig := new(MockConfig)
//				mockConfig.On("GetConf").Return(&conf.Config{
//					App: &conf.App{
//						Host: "localhost",
//						Port: 8081,
//					},
//					Log: &conf.Log{
//						Level: "info",
//					},
//				})
//				return mockConfig
//			}(),
//			expectedAddr:     "localhost:8081",
//			expectedTimeout:  60 * time.Second,
//			expectedLogLevel: zap.InfoLevel,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			//conf.GetConf = tc.mockConfig.GetConf
//
//			httpService := NewHTTPService()
//
//			// Verify HTTP Server configuration
//			assert.Equal(t, tc.expectedAddr, httpService.server.Addr)
//			assert.Equal(t, tc.expectedTimeout, httpService.server.ReadHeaderTimeout)
//			assert.Equal(t, tc.expectedTimeout, httpService.server.ReadTimeout)
//			assert.Equal(t, tc.expectedTimeout, httpService.server.WriteTimeout)
//			assert.Equal(t, tc.expectedTimeout, httpService.server.IdleTimeout)
//			assert.Equal(t, 1<<20, httpService.server.MaxHeaderBytes)
//
//			// Verify Gin Mode
//			assert.Equal(t, gin.ReleaseMode, gin.Mode())
//
//			// Verify Logger Level
//			logger, _ := zap.NewDevelopment()
//			assert.Equal(t, tc.expectedLogLevel, logger.Core().Enabled(0))
//		})
//	}
//}

//func TestEnableAPIRoot(t *testing.T) {
//	// Set up the config
//	//conf.GetConf = func() *conf.Config {
//	//	return &conf.Config{
//	//		App: &conf.App{
//	//			Host: "localhost",
//	//			Port: 8080,
//	//		},
//	//		Log: &conf.Log{
//	//			Level: "debug",
//	//		},
//	//	}
//	//}
//
//	// Initialize the HTTPService
//	httpService := NewHTTPService()
//
//	// Enable the API root
//	httpService.EnableAPIRoot()
//
//	// Create a test request
//	req, err := http.NewRequest("GET", "/", nil)
//	if err != nil {
//		t.Fatalf("Failed to create request: %v", err)
//	}
//
//	// Create a response recorder to capture the response
//	w := httptest.NewRecorder()
//
//	// Perform the request
//	httpService.r.ServeHTTP(w, req)
//
//	// Verify the response
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Equal(t, "API Root", w.Body.String())
//}

//func TestEnableSwagger(t *testing.T) {
//	// Set up the config
//	//conf.GetConf = func() *conf.Config {
//	//	return &conf.Config{
//	//		App: &conf.App{
//	//			Host: "localhost",
//	//			Port: 8080,
//	//		},
//	//		Log: &conf.Log{
//	//			Level: "debug",
//	//		},
//	//	}
//	//}
//
//	// Initialize the HTTPService
//	httpService := NewHTTPService()
//
//	// Enable Swagger
//	httpService.EnableSwagger()
//
//	// Create a test request for Swagger
//	req, err := http.NewRequest("GET", "/swagger/doc.json", nil)
//	if err != nil {
//		t.Fatalf("Failed to create request: %v", err)
//	}
//
//	// Create a response recorder to capture the response
//	w := httptest.NewRecorder()
//
//	// Perform the request
//	httpService.r.ServeHTTP(w, req)
//
//	// Verify the response
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
//
//	// Verify the Swagger base path
//	assert.Equal(t, ApiV1, docs.SwaggerInfo.BasePath)
//}

// TestTransferRouteInfo tests the transferRouteInfo function using table-driven tests.
func TestTransferRouteInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    gin.RouteInfo
		expected RouteInfo
	}{
		{
			name: "GET method with home route",
			input: gin.RouteInfo{
				Method:  "GET",
				Path:    "/",
				Handler: "HomeHandler",
			},
			expected: RouteInfo{
				Method:       "GET",
				FunctionName: "HomeHandler",
				Path:         "/",
			},
		},
		{
			name: "POST method with submit route",
			input: gin.RouteInfo{
				Method:  "POST",
				Path:    "/submit",
				Handler: "SubmitHandler",
			},
			expected: RouteInfo{
				Method:       "POST",
				FunctionName: "SubmitHandler",
				Path:         "/submit",
			},
		},
		{
			name: "PUT method with update route",
			input: gin.RouteInfo{
				Method:  "PUT",
				Path:    "/update",
				Handler: "UpdateHandler",
			},
			expected: RouteInfo{
				Method:       "PUT",
				FunctionName: "UpdateHandler",
				Path:         "/update",
			},
		},
		{
			name: "DELETE method with delete route",
			input: gin.RouteInfo{
				Method:  "DELETE",
				Path:    "/delete",
				Handler: "DeleteHandler",
			},
			expected: RouteInfo{
				Method:       "DELETE",
				FunctionName: "DeleteHandler",
				Path:         "/delete",
			},
		},
		{
			name: "OPTIONS method with options route",
			input: gin.RouteInfo{
				Method:  "OPTIONS",
				Path:    "/options",
				Handler: "OptionsHandler",
			},
			expected: RouteInfo{
				Method:       "OPTIONS",
				FunctionName: "OptionsHandler",
				Path:         "/options",
			},
		},
		{
			name: "PATCH method with patch route",
			input: gin.RouteInfo{
				Method:  "PATCH",
				Path:    "/patch",
				Handler: "PatchHandler",
			},
			expected: RouteInfo{
				Method:       "PATCH",
				FunctionName: "PatchHandler",
				Path:         "/patch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transferRouteInfo(tt.input)
			assert.Equal(t, tt.expected, *result)
		})
	}
}

// Mock HTTPService with predefined routes.
func setupTestHTTPService() *HTTPService {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Define some mock routes
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "Home") })
	r.POST("/submit", func(c *gin.Context) { c.String(http.StatusOK, "Submit") })
	r.PUT("/update", func(c *gin.Context) { c.String(http.StatusOK, "Update") })
	r.DELETE("/delete", func(c *gin.Context) { c.String(http.StatusOK, "Delete") })

	server := &http.Server{
		Handler: r,
	}

	return &HTTPService{
		server: server,
		r:      r,
	}
}

//func TestAPIRoot(t *testing.T) {
//	// Setup mock HTTP service
//	s := setupTestHTTPService()
//
//	// Create a test request and recorder
//	req, _ := http.NewRequest(http.MethodGet, "/", nil)
//	recorder := httptest.NewRecorder()
//	c, _ := gin.CreateTestContext(recorder)
//	c.Request = req
//
//	// Call the apiRoot handler
//	s.apiRoot(c)
//
//	// Check the status code
//	assert.Equal(t, http.StatusOK, recorder.Code)
//
//	// Check the response body
//	var result []*RouteInfo
//	err := json.NewDecoder(recorder.Body).Decode(&result)
//	if err != nil {
//		t.Fatalf("Failed to decode response body: %v", err)
//	}
//
//	// Define expected routes
//	expected := []*RouteInfo{
//		{Method: "GET", FunctionName: "GET", Path: "/"},
//		{Method: "POST", FunctionName: "POST", Path: "/submit"},
//		{Method: "PUT", FunctionName: "PUT", Path: "/update"},
//		{Method: "DELETE", FunctionName: "DELETE", Path: "/delete"},
//	}
//
//	// Assert the response matches the expected result
//	assert.ElementsMatch(t, expected, result)
//}

func setupMockHTTPService() *HTTPService {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Create a mock HTTP server
	server := &http.Server{
		Handler: r,
	}

	// Create the HTTPService with mock config
	return &HTTPService{
		server: server,
		c: &conf.Config{
			App: &conf.App{
				Host: "localhost",
				Port: 8080,
			},
		},
		r: r,
	}
}

// Mocked dependencies
type MockApp struct {
	mock.Mock
}

func (m *MockApp) LoadHttpApp(r *gin.Engine) {
	m.Called(r)
}

//func TestStart(t *testing.T) {
//	tests := []struct {
//		name        string
//		mockSetup   func(*MockApp)
//		expectedErr error
//	}{
//		{
//			name: "successful start",
//			mockSetup: func(m *MockApp) {
//				m.On("LoadHttpApp", mock.Anything).Return()
//			},
//			expectedErr: nil,
//		},
//		{
//			name: "start error",
//			mockSetup: func(m *MockApp) {
//				m.On("LoadHttpApp", mock.Anything).Return()
//			},
//			expectedErr: errors.New("start service error, test error"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Set up mocks and HTTPService
//			mockApp := new(MockApp)
//			//originalLoadHttpApp := app.LoadHttpApp
//			//app.LoadHttpApp = mockApp.LoadHttpApp
//			//defer func() { app.LoadHttpApp = originalLoadHttpApp }() // Restore original function
//
//			// Create HTTPService
//			s := setupMockHTTPService()
//
//			// Set up mock expectations
//			tt.mockSetup(mockApp)
//
//			// Mock ListenAndServe
//			//s.server.ListenAndServe = func() error {
//			//	if tt.expectedErr != nil {
//			//		return tt.expectedErr
//			//	}
//			//	return nil
//			//}
//
//			// Run the Start method
//			err := s.Start()
//
//			// Assert the results
//			if tt.expectedErr == nil {
//				assert.NoError(t, err)
//			} else {
//				assert.EqualError(t, err, tt.expectedErr.Error())
//			}
//		})
//	}
//}

// Mocked HTTP server
type MockServer struct {
	mock.Mock
}

func (m *MockServer) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Test Stop method
//func TestStop(t *testing.T) {
//	tests := []struct {
//		name        string
//		mockSetup   func(*MockServer)
//		expectedErr error
//	}{
//		{
//			name: "successful shutdown",
//			mockSetup: func(m *MockServer) {
//				m.On("Shutdown", mock.Anything).Return(nil)
//			},
//			expectedErr: nil,
//		},
//		{
//			name: "shutdown timeout",
//			mockSetup: func(m *MockServer) {
//				m.On("Shutdown", mock.Anything).Return(errors.New("shutdown timeout"))
//			},
//			expectedErr: fmt.Errorf("graceful shutdown timeout, force exit"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Set up mocks and HTTPService
//			mockServer := new(MockServer)
//			s := &HTTPService{
//				//server: mockServer,
//				c: conf.GetConf(), // Assume config is properly set up
//				r: gin.New(),      // Create a new Gin engine
//			}
//
//			// Set up mock expectations
//			tt.mockSetup(mockServer)
//
//			// Run the Stop method
//			err := s.Stop()
//
//			// Assert the results
//			if tt.expectedErr == nil {
//				assert.NoError(t, err)
//			} else {
//				assert.EqualError(t, err, tt.expectedErr.Error())
//			}
//		})
//	}
//}
