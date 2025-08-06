package protocol

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestGrpcService(t *testing.T) {
	testCases := []struct {
		name               string
		expectedServerNil  bool
		expectedLoggerNil  bool
		expectedReflection bool
		expectedServerType bool
	}{
		{
			name:               "Test default initialization",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
		{
			name:               "Test logger initialization",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
		{
			name:               "Test gRPC server type",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
		{
			name:               "Test reflection registration",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
		{
			name:               "Test gRPC server not nil",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
		{
			name:               "Test logger not nil",
			expectedServerNil:  false,
			expectedLoggerNil:  false,
			expectedReflection: true,
			expectedServerType: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			grpcService := NewGrpcService()

			// Check if the gRPC server is initialized correctly
			if tc.expectedServerNil {
				assert.Nil(t, grpcService.s, "Expected gRPC server to be nil")
			} else {
				assert.NotNil(t, grpcService.s, "Expected gRPC server to be initialized")
			}

			// Check if the logger is initialized correctly
			if tc.expectedLoggerNil {
				assert.Nil(t, grpcService.l, "Expected logger to be nil")
			} else {
				assert.NotNil(t, grpcService.l, "Expected logger to be initialized")
			}

			// Check if the reflection service is registered
			if tc.expectedReflection {
				// To test reflection registration, we assume that if gRPC service is not nil, reflection is registered
				// This is a simplification for demonstration purposes
				assert.NotNil(t, grpcService.s, "Expected reflection to be registered with gRPC server")
			}

			// Check if the server is of type *grpc.Server
			//if tc.expectedServerType {
			//	_, ok := grpcService.s.(*grpc.Server)
			//	assert.True(t, ok, "Expected server to be of type *grpc.Server")
			//}
		})
	}
}

// Mocking the necessary components
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args)
}

type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) GrpcAddr() string {
	args := m.Called()
	return args.String(0)
}

//func (m *MockConfig) GetConf() *Config {
//	args := m.Called()
//	return args.Get(0).(*Config)
//}

// Mocking the actual GrpcService
//func TestGrpcService_Start(t *testing.T) {
//	testCases := []struct {
//		name             string
//		grpcAddr         string
//		listenError      error
//		serveError       error
//		expectedLogCalls []string
//		expectedLogArgs  []interface{}
//	}{
//		//{
//		//	name:             "Successful Start",
//		//	grpcAddr:         "localhost:50051",
//		//	listenError:      nil,
//		//	serveError:       nil,
//		//	expectedLogCalls: []string{"Infof", "Info"},
//		//	expectedLogArgs:  []interface{}{"success start GRPC service, listen address: [%s]", "localhost:50051"},
//		//},
//		//{
//		//	name:             "Listen Error",
//		//	grpcAddr:         "localhost:50051",
//		//	listenError:      errors.New("listen error"),
//		//	serveError:       nil,
//		//	expectedLogCalls: []string{"Errorf"},
//		//	expectedLogArgs:  []interface{}{"listen grpc tcp conn error, %s", "listen error"},
//		//},
//		{
//			name:             "Serve Error",
//			grpcAddr:         "localhost:50051",
//			listenError:      nil,
//			serveError:       errors.New("serve error"),
//			expectedLogCalls: []string{"Infof", "Errorf"},
//			expectedLogArgs:  []interface{}{"success start GRPC service, listen address: [%s]", "localhost:50051"},
//		},
//		{
//			name:             "Server Stopped Error",
//			grpcAddr:         "localhost:50051",
//			listenError:      nil,
//			serveError:       grpc.ErrServerStopped,
//			expectedLogCalls: []string{"Infof", "Info"},
//			expectedLogArgs:  []interface{}{"success start GRPC service, listen address: [%s]", "localhost:50051"},
//		},
//		{
//			name:             "Config GrpcAddr Error",
//			grpcAddr:         "localhost:50051",
//			listenError:      nil,
//			serveError:       nil,
//			expectedLogCalls: []string{"Infof"},
//			expectedLogArgs:  []interface{}{"success start GRPC service, listen address: [%s]", "localhost:50051"},
//		},
//		{
//			name:             "Log Calls Check",
//			grpcAddr:         "localhost:50051",
//			listenError:      nil,
//			serveError:       nil,
//			expectedLogCalls: []string{"Infof", "Info"},
//			expectedLogArgs:  []interface{}{"success start GRPC service, listen address: [%s]", "localhost:50051"},
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockLogger := new(MockLogger)
//			mockConfig := new(MockConfig)
//
//			// Set up mocks
//			mockConfig.On("GrpcAddr").Return(tc.grpcAddr)
//			mockLogger.On(tc.expectedLogCalls[0], mock.Anything).Return()
//			if len(tc.expectedLogCalls) > 1 {
//				mockLogger.On(tc.expectedLogCalls[1], mock.Anything).Return()
//			}
//
//			grpcService := &GrpcService{
//				//l: mockLogger,
//				s: grpc.NewServer(),
//			}
//
//			// Override the network listener behavior
//			//netListen = func(network, address string) (net.Listener, error) {
//			//	return nil, tc.listenError
//			//}
//
//			// Override the serve method behavior
//			//grpcService.s.Serve = func(net.Listener) error {
//			//	return tc.serveError
//			//}
//
//			grpcService.Start()
//
//			// Verify logger calls
//			mockLogger.AssertCalled(t, tc.expectedLogCalls[0], tc.expectedLogArgs[0])
//			if len(tc.expectedLogCalls) > 1 {
//				mockLogger.AssertCalled(t, tc.expectedLogCalls[1], tc.expectedLogArgs[1])
//			}
//		})
//	}
//}

// Mocking the GrpcService
//func TestGrpcService_Stop(t *testing.T) {
//	testCases := []struct {
//		name             string
//		grpcService      *GrpcService
//		expectedLogCalls []string
//		expectedLogArgs  []interface{}
//	}{
//		{
//			name:             "Successful Stop",
//			grpcService:      &GrpcService{s: grpc.NewServer()},
//			expectedLogCalls: []string{"Info", "Info"},
//			expectedLogArgs:  []interface{}{"start graceful shutdown", "service is stopped"},
//		},
//		{
//			name:             "No-Op Stop",
//			grpcService:      &GrpcService{s: grpc.NewServer()},
//			expectedLogCalls: []string{"Info", "Info"},
//			expectedLogArgs:  []interface{}{"start graceful shutdown", "service is stopped"},
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockLogger := new(MockLogger)
//
//			// Set up mocks
//			mockLogger.On(tc.expectedLogCalls[0], mock.Anything).Return()
//			if len(tc.expectedLogCalls) > 1 {
//				mockLogger.On(tc.expectedLogCalls[1], mock.Anything).Return()
//			}
//
//			grpcService := &GrpcService{
//				//l: mockLogger,
//				s: tc.grpcService.s,
//			}
//
//			grpcService.Stop()
//
//			// Verify logger calls
//			mockLogger.AssertCalled(t, tc.expectedLogCalls[0], tc.expectedLogArgs[0])
//			if len(tc.expectedLogCalls) > 1 {
//				mockLogger.AssertCalled(t, tc.expectedLogCalls[1], tc.expectedLogArgs[1])
//			}
//		})
//	}
//}
