package cmd

import (
	"go.uber.org/zap/zapcore"
)

//func TestNewService(t *testing.T) {
//	tests := []struct {
//		name    string
//		wantErr bool
//	}{
//		{
//			name:    "Valid service creation",
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			svc, err := newService()
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("newService() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//
//			if svc == nil {
//				t.Errorf("newService() returned nil service")
//			}
//
//			if svc.http == nil {
//				t.Errorf("newService() failed to initialize HTTPService")
//			}
//
//			if svc.grpc == nil {
//				t.Errorf("newService() failed to initialize GrpcService")
//			}
//
//			if svc.logger == nil {
//				t.Errorf("newService() failed to initialize logger")
//			}
//		})
//	}
//}

type mockHTTPService struct {
	startFunc func() error
	stopFunc  func() error
}

func (m *mockHTTPService) Start() error {
	if m.startFunc != nil {
		return m.startFunc()
	}
	return nil
}

func (m *mockHTTPService) Stop() error {
	if m.stopFunc != nil {
		return m.stopFunc()
	}
	return nil
}

type mockGrpcService struct {
	stopFunc func()
}

func (m *mockGrpcService) Start() {
}

func (m *mockGrpcService) Stop() {
	if m.stopFunc != nil {
		m.stopFunc()
	}
}

//func TestService_Start(t *testing.T) {
//	tests := []struct {
//		name       string
//		httpEnable bool
//		httpStart  func() error
//		wantErr    bool
//	}{
//		//{
//		//	name:       "HTTP enabled and start successfully",
//		//	httpEnable: true,
//		//	httpStart:  func() error { return nil },
//		//	wantErr:    false,
//		//},
//		{
//			name:       "HTTP enabled and start fails",
//			httpEnable: true,
//			httpStart:  func() error { return errors.New("HTTP start failed") },
//			wantErr:    true,
//		},
//		{
//			name:       "HTTP disabled",
//			httpEnable: false,
//			httpStart:  nil,
//			wantErr:    false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			httpEnable = tt.httpEnable
//			//mockHTTP := &mockHTTPService{startFunc: tt.httpStart}
//			//mockGrpc := &mockGrpcService{}
//			logger := zap.NewNop().Sugar()
//
//			s := &service{
//				//http:   mockHTTP,
//				//grpc:   mockGrpc,
//				logger: logger,
//			}
//
//			err := s.start()
//
//			if (err != nil) != tt.wantErr {
//				t.Errorf("service.start() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

//func TestService_waitSign(t *testing.T) {
//	tests := []struct {
//		name          string
//		httpEnable    bool
//		httpStopError error
//	}{
//		{
//			name:       "Graceful shutdown with HTTP enabled",
//			httpEnable: true,
//		},
//		{
//			name:       "Graceful shutdown with HTTP disabled",
//			httpEnable: false,
//		},
//		{
//			name:          "Graceful shutdown with HTTP stop error",
//			httpEnable:    true,
//			httpStopError: errors.New("HTTP stop error"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			httpEnable = tt.httpEnable
//			sign := make(chan os.Signal, 1)
//			var wg sync.WaitGroup
//
//			//mockHTTP := &mockHTTPService{
//			//	stopFunc: func() error {
//			//		return tt.httpStopError
//			//	},
//			//}
//			//
//			//mockGrpc := &mockGrpcService{
//			//	stopFunc: func() {
//			//		// Simulate gRPC stop logic
//			//	},
//			//}
//
//			s := &service{
//				//http:   mockHTTP,
//				//grpc:   mockGrpc,
//				logger: zap.NewNop().Sugar(),
//			}
//
//			wg.Add(1)
//			go s.waitSign(sign, &wg)
//
//			// Send a shutdown signal
//			sign <- os.Interrupt
//
//			// Wait for the service to shut down
//			done := make(chan struct{})
//			go func() {
//				wg.Wait()
//				close(done)
//			}()
//
//			select {
//			case <-done:
//				// Test passed
//			case <-time.After(3 * time.Second):
//				t.Errorf("Test %s timed out", tt.name)
//			}
//		})
//	}
//}

// Mock configuration and dependencies for testing
type mockConf struct {
	LogPathDir string
	LogLevel   zapcore.Level
}

func (m *mockConf) GetLogLevel() zapcore.Level {
	return m.LogLevel
}

// Mock version for testing
type mockVersion struct {
	ServiceName string
}

var daemonTestConf = &mockConf{}
var testLoadGlobalLoggerVersion = &mockVersion{}

//func TestLoadGlobalLogger(t *testing.T) {
//	tests := []struct {
//		name        string
//		setup       func()
//		cleanup     func()
//		expectError bool
//	}{
//		{
//			name: "Successful logger initialization with existing directory",
//			setup: func() {
//				daemonTestConf.LogPathDir = "./logs"
//				testLoadGlobalLoggerVersion.ServiceName = "testservice"
//				_ = os.Mkdir(daemonTestConf.LogPathDir, 0755)
//			},
//			cleanup: func() {
//				os.RemoveAll("./logs")
//			},
//			expectError: false,
//		},
//		{
//			name: "Successful logger initialization with non-existing directory",
//			setup: func() {
//				daemonTestConf.LogPathDir = "./new_logs"
//				testLoadGlobalLoggerVersion.ServiceName = "testservice"
//			},
//			cleanup: func() {
//				os.RemoveAll("./new_logs")
//			},
//			expectError: false,
//		},
//		{
//			name: "Failed logger initialization due to file creation error",
//			setup: func() {
//				daemonTestConf.LogPathDir = "/invalid_path"
//				testLoadGlobalLoggerVersion.ServiceName = "testservice"
//			},
//			cleanup: func() {
//				// No cleanup necessary
//			},
//			expectError: true,
//		},
//		{
//			name: "Logger initialization with JSON and Console encoding",
//			setup: func() {
//				daemonTestConf.LogPathDir = "./logs"
//				testLoadGlobalLoggerVersion.ServiceName = "testservice"
//				_ = os.Mkdir(daemonTestConf.LogPathDir, 0755)
//			},
//			cleanup: func() {
//				os.RemoveAll("./logs")
//			},
//			expectError: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setup()
//
//			err := loadGlobalLogger()
//
//			if (err != nil) != tt.expectError {
//				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
//			}
//
//			if err == nil {
//				if _, statErr := os.Stat(filepath.Join(daemonTestConf.LogPathDir, testLoadGlobalLoggerVersion.ServiceName+"-json.log")); statErr != nil {
//					t.Errorf("expected log file to be created, but got error: %v", statErr)
//				}
//
//				if _, statErr := os.Stat(filepath.Join(daemonTestConf.LogPathDir, testLoadGlobalLoggerVersion.ServiceName+".log")); statErr != nil {
//					t.Errorf("expected log file to be created, but got error: %v", statErr)
//				}
//			}
//
//			tt.cleanup()
//		})
//	}
//}
