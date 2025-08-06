package app

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// MockHTTPApp is a mock implementation of HTTPApp for testing purposes
type MockHTTPApp struct {
	mock.Mock
	name string
}

func (m *MockHTTPApp) Registry(r *gin.Engine, appName string) {
	m.Called(r, appName)
}

func (m *MockHTTPApp) Config() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockHTTPApp) Name() string {
	return m.name
}

// TestLoadHttpApp tests the LoadHttpApp function
//func TestLoadHttpApp(t *testing.T) {
//	tests := []struct {
//		name           string
//		httpApps       map[string]HTTPApp
//		expectedCalled map[string]bool
//	}{
//		{
//			name: "All apps register successfully",
//			httpApps: map[string]HTTPApp{
//				"App1": &MockHTTPApp{name: "App1"},
//				"App2": &MockHTTPApp{name: "App2"},
//			},
//			expectedCalled: map[string]bool{
//				"App1": true,
//				"App2": true,
//			},
//		},
//		{
//			name:           "No apps registered",
//			httpApps:       map[string]HTTPApp{},
//			expectedCalled: map[string]bool{},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a new Gin router
//			router := gin.Default()
//
//			// Setup mocks
//			for name, app := range tt.httpApps {
//				mockApp, ok := app.(*MockHTTPApp)
//				if !ok {
//					t.Fatalf("expected MockHTTPApp but got %T", app)
//				}
//				mockApp.On("Registry", router, name).Return()
//			}
//
//			// Load HTTP apps
//			LoadHttpApp(router)
//
//			// Verify that Registry is called for each app
//			for name, app := range tt.httpApps {
//				mockApp, ok := app.(*MockHTTPApp)
//				if !ok {
//					t.Fatalf("expected MockHTTPApp but got %T", app)
//				}
//				if tt.expectedCalled[name] {
//					mockApp.AssertCalled(t, "Registry", router, name)
//				} else {
//					mockApp.AssertNotCalled(t, "Registry", router, name)
//				}
//			}
//		})
//	}
//}

// TestRegistryHttpApp tests the RegistryHttpApp function
func TestRegistryHttpApp(t *testing.T) {
	tests := []struct {
		name          string
		httpApps      map[string]HTTPApp
		appToRegister HTTPApp
		expectPanic   bool
	}{
		{
			name: "Register new app successfully",
			httpApps: map[string]HTTPApp{
				"App1": &MockHTTPApp{name: "App1"},
			},
			appToRegister: &MockHTTPApp{name: "App2"},
			expectPanic:   false,
		},
		{
			name: "Attempt to re-register an existing app",
			httpApps: map[string]HTTPApp{
				"App1": &MockHTTPApp{name: "App1"},
			},
			appToRegister: &MockHTTPApp{name: "App1"},
			expectPanic:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize httpApps with the current state
			httpApps = tt.httpApps

			if tt.expectPanic {
				assert.Panics(t, func() { RegistryHttpApp(tt.appToRegister) }, "Expected panic but got none")
			} else {
				assert.NotPanics(t, func() { RegistryHttpApp(tt.appToRegister) }, "Did not expect panic but got one")
				assert.Contains(t, httpApps, tt.appToRegister.Name(), "App should be registered in httpApps map")
			}
		})
	}
}

// TestLoadedHttpApp tests the LoadedHttpApp function.
func TestLoadedHttpApp(t *testing.T) {
	// Setup mock data
	httpApps = map[string]HTTPApp{
		"App1": &MockHTTPApp{name: "App1"},
		"App2": &MockHTTPApp{name: "App2"},
		"App3": &MockHTTPApp{name: "App3"},
	}

	expectedApps := []string{"App1", "App2", "App3"}

	// Call the function
	result := LoadedHttpApp()

	// Check if the result matches the expected value
	assert.ElementsMatch(t, expectedApps, result, "LoadedHttpApp should return the names of all registered HTTP apps")
}
