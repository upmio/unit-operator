package app

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"testing"
)

// Mock implementation of GRPCApp interface
type MockGRPCApp struct {
	mock.Mock

	name string
}

func (m *MockGRPCApp) Registry(server *grpc.Server) {
	m.Called(server)
}

func (m *MockGRPCApp) Config() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGRPCApp) Name() string {
	args := m.Called()
	return args.String(0)
}

// TestRegistryGrpcApp tests the RegistryGrpcApp function
func TestRegistryGrpcApp(t *testing.T) {
	tests := []struct {
		name        string
		app         *MockGRPCApp
		shouldPanic bool
	}{
		{
			name:        "Successful registration",
			app:         func() *MockGRPCApp { m := &MockGRPCApp{}; m.On("Name").Return("TestService1"); return m }(),
			shouldPanic: false,
		},
		{
			name:        "Re-registration",
			app:         func() *MockGRPCApp { m := &MockGRPCApp{}; m.On("Name").Return("TestService1"); return m }(),
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				assert.Panics(t, func() { RegistryGrpcApp(tt.app) })
			} else {
				assert.NotPanics(t, func() { RegistryGrpcApp(tt.app) })

				// Verify that the app is registered
				registeredApp, ok := grpcApps[tt.app.Name()]
				assert.True(t, ok, "App should be registered")
				assert.Same(t, tt.app, registeredApp, "Registered app should be the same instance")
			}
		})
	}
}

// TestLoadedGrpcApp tests the LoadedGrpcApp function
//func TestLoadedGrpcApp(t *testing.T) {
//	tests := []struct {
//		name     string
//		grpcApps map[string]GRPCApp
//		expected []string
//	}{
//		{
//			name:     "No services loaded",
//			grpcApps: map[string]GRPCApp{}, // Empty map
//			expected: []string{},
//		},
//		{
//			name: "One service loaded",
//			grpcApps: map[string]GRPCApp{
//				"TestService1": &MockGRPCApp{},
//			},
//			expected: []string{"TestService1"},
//		},
//		{
//			name: "Multiple services loaded",
//			grpcApps: map[string]GRPCApp{
//				"TestService1": &MockGRPCApp{},
//				"TestService2": &MockGRPCApp{},
//				"TestService3": &MockGRPCApp{},
//			},
//			expected: []string{"TestService1", "TestService2", "TestService3"},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Set the grpcApps variable to the test case value
//			grpcApps = tt.grpcApps
//
//			// Call LoadedGrpcApp and check the result
//			actual := LoadedGrpcApp()
//			assert.ElementsMatch(t, tt.expected, actual)
//		})
//	}
//}

// TestGetGrpcApp tests the GetGrpcApp function
//func TestGetGrpcApp(t *testing.T) {
//	tests := []struct {
//		name     string
//		grpcApps map[string]GRPCApp
//		request  string
//		expected string
//		panics   bool
//	}{
//		{
//			name: "Application exists",
//			grpcApps: map[string]GRPCApp{
//				"TestService": &MockGRPCApp{name: "TestService"},
//			},
//			request:  "TestService",
//			expected: "TestService",
//			panics:   false,
//		},
//		{
//			name: "Application does not exist",
//			grpcApps: map[string]GRPCApp{
//				"TestService": &MockGRPCApp{name: "TestService"},
//			},
//			request:  "NonExistentService",
//			expected: "",
//			panics:   true,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Set the grpcApps variable to the test case value
//			grpcApps = tt.grpcApps
//
//			if tt.panics {
//				require.Panics(t, func() {
//					GetGrpcApp(tt.request)
//				})
//			} else {
//				app := GetGrpcApp(tt.request)
//				assert.Equal(t, tt.expected, app.Name())
//			}
//		})
//	}
//}

// TestLoadGrpcApp tests the LoadGrpcApp function
//func TestLoadGrpcApp(t *testing.T) {
//	tests := []struct {
//		name           string
//		grpcApps       map[string]GRPCApp
//		configErr      error
//		registryCalled bool
//		expectedError  string
//	}{
//		//{
//		//	name: "All apps configure and register successfully",
//		//	grpcApps: map[string]GRPCApp{
//		//		"App1": &MockGRPCApp{name: "App1"},
//		//		"App2": &MockGRPCApp{name: "App2"},
//		//	},
//		//	configErr:      nil,
//		//	registryCalled: true,
//		//	expectedError:  "",
//		//},
//		{
//			name: "One app configuration fails",
//			grpcApps: map[string]GRPCApp{
//				"App1": &MockGRPCApp{name: "App1"},
//				"App2": &MockGRPCApp{name: "App2"},
//			},
//			configErr:      errors.New("config error"),
//			registryCalled: false,
//			expectedError:  "config grpc app App1 error config error",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			server := grpc.NewServer()
//
//			// Setup mocks
//			for _, app := range tt.grpcApps {
//				mockApp, ok := app.(*MockGRPCApp)
//				if !ok {
//					t.Fatalf("expected MockGRPCApp but got %T", app)
//				}
//
//				mockApp.On("Config").Return(tt.configErr)
//				if tt.registryCalled {
//					mockApp.On("Registry", server).Return()
//				}
//			}
//
//			// Call LoadGrpcApp
//			err := LoadGrpcApp(server)
//
//			if tt.expectedError != "" {
//				assert.EqualError(t, err, tt.expectedError)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			// Verify that Registry is called if no error occurred
//			for _, app := range tt.grpcApps {
//				mockApp, ok := app.(*MockGRPCApp)
//				if !ok {
//					t.Fatalf("expected MockGRPCApp but got %T", app)
//				}
//				if tt.registryCalled {
//					mockApp.AssertCalled(t, "Registry", server)
//				} else {
//					mockApp.AssertNotCalled(t, "Registry", server)
//				}
//			}
//		})
//	}
//}
