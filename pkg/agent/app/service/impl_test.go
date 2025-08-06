package service

import (
	"github.com/abrander/go-supervisord"
	"github.com/stretchr/testify/mock"
)

// MockConfig is a mock for the configuration used in the Config method.
type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) GetSupervisorClient() (*supervisord.Client, error) {
	args := m.Called()
	return args.Get(0).(*supervisord.Client), args.Error(1)
}

// TestServiceConfig tests the Config method of the service struct.
//func TestServiceConfig(t *testing.T) {
//	// Define test cases
//	tests := []struct {
//		name            string
//		mockClient      *supervisord.Client
//		mockClientError error
//		expectError     bool
//	}{
//		{
//			name:            "Successfully configure service",
//			mockClient:      &supervisord.Client{}, // Mock client
//			mockClientError: nil,
//			expectError:     false,
//		},
//		{
//			name:            "Failed to get supervisor client",
//			mockClient:      nil,
//			mockClientError: errors.New("failed to get supervisor client"),
//			expectError:     true,
//		},
//	}
//
//	// Store the original app.GetGrpcApp function
//	//originalGetGrpcApp := app.GetGrpcApp
//
//	// Defer restoration of the original function
//	//defer func() { app.GetGrpcApp = originalGetGrpcApp }()
//
//	// Mock the app.GetGrpcApp function
//	//app.GetGrpcApp = MockApp
//
//	// Iterate through test cases
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a new mock configuration
//			mockConf := new(MockConfig)
//			mockConf.On("GetSupervisorClient").Return(tt.mockClient, tt.mockClientError)
//
//			// Initialize the service
//			s := &service{
//				logger: zap.NewNop().Sugar(),
//			}
//
//			// Inject the mock configuration into conf.GetConf().Supervisor
//			//conf.GetConf().Supervisor = mockConf
//
//			// Call the Config method and check the result
//			err := s.Config()
//
//			if tt.expectError {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, s.client)
//				assert.NotNil(t, s.service)
//			}
//
//			// Verify that the mock was called
//			mockConf.AssertExpectations(t)
//		})
//	}
//}

// TestServiceName tests the Name method of the service struct.
//func TestServiceName(t *testing.T) {
//	// Define test cases
//	tests := []struct {
//		name     string
//		appName  string
//		expected string
//	}{
//		{
//			name:     "Standard App Name",
//			appName:  "TestApp",
//			expected: "TestApp",
//		},
//		{
//			name:     "Empty App Name",
//			appName:  "",
//			expected: "",
//		},
//		{
//			name:     "Long App Name",
//			appName:  "ThisIsAVeryLongApplicationNameForTestingPurposes",
//			expected: "ThisIsAVeryLongApplicationNameForTestingPurposes",
//		},
//		{
//			name:     "App Name with Special Characters",
//			appName:  "AppName_123!@#",
//			expected: "AppName_123!@#",
//		},
//	}
//
//	// Iterate through test cases
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Set the appName variable
//			//appName = tt.appName
//
//			// Initialize the service
//			s := &service{}
//
//			// Call the Name method and check the result
//			result := s.Name()
//			assert.Equal(t, tt.expected, result)
//		})
//	}
//}

type mockSupervisorClient struct {
	processInfo       supervisord.ProcessInfo
	err               error
	startErr          error
	stopErr           error
	startProcessErr   error
	stopProcessErr    error
	getProcessInfoErr error
}

func (m *mockSupervisorClient) GetProcessInfo(_ string) (supervisord.ProcessInfo, error) {
	return m.processInfo, m.err
}

func (m *mockSupervisorClient) StartProcess(_ string, _ bool) error {
	return m.startErr
}

//func TestStartService(t *testing.T) {
//	tests := []struct {
//		name           string
//		mockClient     *mockSupervisorClient
//		expectedMsg    string
//		expectedErr    error
//		expectedCalled bool
//	}{
//		{
//			name: "Get ProcessInfo failed",
//			mockClient: &mockSupervisorClient{
//				err: errors.New("unable to get process info"),
//			},
//			expectedMsg: "Get ProcessInfo failed, error: unable to get process info",
//			expectedErr: errors.New("Get ProcessInfo failed, error: unable to get process info"),
//		},
//		{
//			name: "Process not running, Start service success",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStopped},
//			},
//			expectedMsg: "Start service success.",
//			expectedErr: nil,
//		},
//		{
//			name: "Process not running, Start service failed",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStopped},
//				startErr:    errors.New("unable to start process"),
//			},
//			expectedMsg: "Start service failed, error: unable to start process",
//			expectedErr: errors.New("Start service failed, error: unable to start process"),
//		},
//		{
//			name: "Service already running",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateRunning},
//			},
//			expectedMsg: "Service already running, No need to start.",
//			expectedErr: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a service with a mocked supervisor client and logger
//			service := &service{
//				//client: tt.mockClient,
//				logger: zap.NewNop().Sugar(),
//			}
//
//			// Execute the StartService method
//			resp, err := service.StartService(context.Background(), &ServiceRequest{})
//
//			// Validate the response
//			assert.Equal(t, tt.expectedMsg, resp.Message)
//			assert.Equal(t, tt.expectedErr, err)
//		})
//	}
//}

//func TestStopService(t *testing.T) {
//	tests := []struct {
//		name        string
//		mockClient  *mockSupervisorClient
//		expectedMsg string
//		expectedErr error
//	}{
//		{
//			name: "Get ProcessInfo failed",
//			mockClient: &mockSupervisorClient{
//				err: errors.New("unable to get process info"),
//			},
//			expectedMsg: "Get ProcessInfo failed, error: unable to get process info",
//			expectedErr: errors.New("Get ProcessInfo failed, error: unable to get process info"),
//		},
//		{
//			name: "Process not stopped, Stop service success",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateRunning},
//			},
//			expectedMsg: "Stop service success.",
//			expectedErr: nil,
//		},
//		{
//			name: "Process not stopped, Stop service failed",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateRunning},
//				stopErr:     errors.New("unable to stop process"),
//			},
//			expectedMsg: "Stop service failed, error: unable to stop process",
//			expectedErr: errors.New("Stop service failed, error: unable to stop process"),
//		},
//		{
//			name: "Service already stopped",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStopped},
//			},
//			expectedMsg: "Service already stopped, No need to stop.",
//			expectedErr: nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a service with a mocked supervisor client and logger
//			service := &service{
//				//client: tt.mockClient,
//				logger: zap.NewNop().Sugar(),
//			}
//
//			// Execute the StopService method
//			resp, err := service.StopService(context.Background(), &ServiceRequest{})
//
//			// Validate the response
//			assert.Equal(t, tt.expectedMsg, resp.Message)
//			assert.Equal(t, tt.expectedErr, err)
//		})
//	}
//}

//func TestGetServiceStatus(t *testing.T) {
//	tests := []struct {
//		name           string
//		mockClient     *mockSupervisorClient
//		expectedStatus ProcessState
//		expectedErr    error
//	}{
//		{
//			name: "Get ProcessInfo failed",
//			mockClient: &mockSupervisorClient{
//				err: errors.New("unable to get process info"),
//			},
//			expectedStatus: ProcessState_StateUnknown,
//			expectedErr:    errors.New("Get ProcessInfo failed, error: unable to get process info"),
//		},
//		{
//			name: "Process is stopped",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStopped},
//			},
//			expectedStatus: ProcessState_StateStopped,
//			expectedErr:    nil,
//		},
//		{
//			name: "Process is starting",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStarting},
//			},
//			expectedStatus: ProcessState_StateStarting,
//			expectedErr:    nil,
//		},
//		{
//			name: "Process is running",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateRunning},
//			},
//			expectedStatus: ProcessState_StateRunning,
//			expectedErr:    nil,
//		},
//		{
//			name: "Process is in fatal state",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateFatal},
//			},
//			expectedStatus: ProcessState_StateFatal,
//			expectedErr:    nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a service with a mocked supervisor client and logger
//			service := &service{
//				//client: tt.mockClient,
//				logger: zap.NewNop().Sugar(),
//			}
//
//			// Execute the GetServiceStatus method
//			resp, err := service.GetServiceStatus(context.Background(), &ServiceRequest{})
//
//			// Validate the response
//			if tt.expectedErr != nil {
//				assert.Error(t, err)
//				assert.Equal(t, tt.expectedErr.Error(), err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, resp)
//				assert.Equal(t, tt.expectedStatus, resp.ServiceStatus)
//			}
//		})
//	}
//}

//func TestRestartService(t *testing.T) {
//	tests := []struct {
//		name            string
//		mockClient      *mockSupervisorClient
//		expectedMessage string
//		expectedErr     error
//	}{
//		{
//			name: "Get ProcessInfo failed",
//			mockClient: &mockSupervisorClient{
//				getProcessInfoErr: errors.New("unable to get process info"),
//			},
//			expectedMessage: "Get ProcessInfo failed, error: unable to get process info",
//			expectedErr:     errors.New("Get ProcessInfo failed, error: unable to get process info"),
//		},
//		{
//			name: "Stop service failed",
//			mockClient: &mockSupervisorClient{
//				processInfo:    supervisord.ProcessInfo{State: supervisord.StateRunning},
//				stopProcessErr: errors.New("failed to stop process"),
//			},
//			expectedMessage: "Stop service failed, error: failed to stop process",
//			expectedErr:     errors.New("Stop service failed, error: failed to stop process"),
//		},
//		{
//			name: "Start service failed",
//			mockClient: &mockSupervisorClient{
//				processInfo:     supervisord.ProcessInfo{State: supervisord.StateStopped},
//				startProcessErr: errors.New("failed to start process"),
//			},
//			expectedMessage: "Start service failed, error: failed to start process",
//			expectedErr:     errors.New("Start service failed, error: failed to start process"),
//		},
//		{
//			name: "Service already stopped, start service success",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateStopped},
//			},
//			expectedMessage: "Start service success.",
//			expectedErr:     nil,
//		},
//		{
//			name: "Service running, stop and start success",
//			mockClient: &mockSupervisorClient{
//				processInfo: supervisord.ProcessInfo{State: supervisord.StateRunning},
//			},
//			expectedMessage: "Start service success.",
//			expectedErr:     nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a service with a mocked supervisor client and logger
//			service := &service{
//				//client: tt.mockClient,
//				logger: zap.NewNop().Sugar(),
//			}
//
//			// Execute the RestartService method
//			resp, err := service.RestartService(context.Background(), &ServiceRequest{})
//
//			// Validate the response
//			if tt.expectedErr != nil {
//				assert.Error(t, err)
//				assert.Equal(t, tt.expectedErr.Error(), err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, resp)
//				assert.Equal(t, tt.expectedMessage, resp.Message)
//			}
//		})
//	}
//}
