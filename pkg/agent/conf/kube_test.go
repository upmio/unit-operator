package conf

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sync"
	"testing"
)

func TestKubeGetClientSet(t *testing.T) {
	tests := []struct {
		name            string
		setupClientset  func() kubernetes.Interface
		getClientSetErr error
		expectedErr     error
	}{
		{
			name: "First call, clientset is nil",
			setupClientset: func() kubernetes.Interface {
				clientset = nil // Ensure clientset is nil
				return fake.NewSimpleClientset()
			},
			expectedErr: nil,
		},
		{
			name: "Second call, clientset already initialized",
			setupClientset: func() kubernetes.Interface {
				clientset = fake.NewSimpleClientset()
				return clientset
			},
			expectedErr: nil,
		},
		//{
		//	name: "getClientSet returns an error",
		//	setupClientset: func() kubernetes.Interface {
		//		clientset = nil
		//		return nil
		//	},
		//	getClientSetErr: errors.New("failed to create clientset"),
		//	expectedErr:     errors.New("failed to create clientset"),
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kube{
				lock: sync.Mutex{},
			}
			clientset = tt.setupClientset()

			// Mock getClientSet function
			//originalGetClientSet := k.getClientSet
			//k.getClientSet = func() (kubernetes.Interface, error) {
			//	if tt.getClientSetErr != nil {
			//		return nil, tt.getClientSetErr
			//	}
			//	return originalGetClientSet()
			//}

			cs, err := k.GetClientSet()

			if tt.expectedErr != nil {
				if err == nil || err.Error() != tt.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if cs == nil {
					t.Errorf("expected non-nil clientset, got nil")
				}
			}
		})
	}
}

// Mock variables for testing
var (
	inClusterConfigError    error
	buildConfigFromFlagsErr error
	newForConfigError       error
	mockClientset           kubernetes.Interface
)

func mockInClusterConfig() (*rest.Config, error) {
	return &rest.Config{}, inClusterConfigError
}

func mockBuildConfigFromFlags(_, _ string) (*rest.Config, error) {
	return &rest.Config{}, buildConfigFromFlagsErr
}

func mockNewForConfig(config *rest.Config) (kubernetes.Interface, error) {
	return mockClientset, newForConfigError
}

//func TestGetClientSet(t *testing.T) {
//	tests := []struct {
//		name            string
//		kubeConfig      string
//		inClusterErr    error
//		outClusterErr   error
//		newForConfigErr error
//		expectedErr     error
//	}{
//		{
//			name:        "InClusterConfig Success",
//			kubeConfig:  "",
//			expectedErr: nil,
//		},
//		{
//			name:        "OutClusterConfig Success",
//			kubeConfig:  "/fake/path/to/kubeconfig",
//			expectedErr: nil,
//		},
//		{
//			name:         "InClusterConfig Failure",
//			kubeConfig:   "",
//			inClusterErr: errors.New("in-cluster config error"),
//			expectedErr:  errors.New("create in-cluster config fail, error: in-cluster config error"),
//		},
//		{
//			name:          "OutClusterConfig Failure",
//			kubeConfig:    "/fake/path/to/kubeconfig",
//			outClusterErr: errors.New("out-cluster config error"),
//			expectedErr:   errors.New("create out-of-cluster config fail, error: out-cluster config error"),
//		},
//		{
//			name:            "NewForConfig Failure",
//			kubeConfig:      "/fake/path/to/kubeconfig",
//			newForConfigErr: errors.New("new for config error"),
//			expectedErr:     errors.New("create clientset fail, error: new for config error"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			k := &Kube{
//				KubeConfig: tt.kubeConfig,
//				lock:       sync.Mutex{},
//			}
//
//			// Set up mocks
//			inClusterConfigError = tt.inClusterErr
//			buildConfigFromFlagsErr = tt.outClusterErr
//			newForConfigError = tt.newForConfigErr
//			mockClientset = fake.NewSimpleClientset()
//
//			// Replace actual functions with mocks
//			//rest.InClusterConfig = mockInClusterConfig
//			//clientcmd.BuildConfigFromFlags = mockBuildConfigFromFlags
//			//kubernetes.NewForConfig = mockNewForConfig
//
//			_, err := k.getClientSet()
//
//			if tt.expectedErr != nil {
//				if err == nil || err.Error() != tt.expectedErr.Error() {
//					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
//				}
//			} else if err != nil {
//				t.Errorf("expected no error, got %v", err)
//			}
//		})
//	}
//}

// Variables to simulate different scenarios
var (
	getGauntletClientErr error
	//gauntletClient       client.Client
)

// Mocking the `client.Client` interface
type MockClient struct {
	mock.Mock
}

// TestGetGauntletClient tests the GetGauntletClient method
func TestKubeGetGauntletClient(t *testing.T) {
	tests := []struct {
		name                 string
		getGauntletClientErr error
		expectedErr          error
	}{
		//{
		//	name:                 "Success",
		//	getGauntletClientErr: nil,
		//	expectedErr:          nil,
		//},
		//{
		//	name:                 "Error from getGauntletClient",
		//	getGauntletClientErr: errors.New("failed to get gauntlet client"),
		//	expectedErr:          errors.New("failed to get gauntlet client"),
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kube{
				lock: sync.Mutex{},
			}

			// Set up the mock error scenario
			getGauntletClientErr = tt.getGauntletClientErr
			//gauntletClient = &MockClient{}

			// Call the method under test
			client, err := k.GetGauntletClient()

			// Check if the error matches the expected result
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, gauntletClient, client)
			}
		})
	}
}

// Test for Kube.GetGauntletClient method
//func TestGetGauntletClient(t *testing.T) {
//	tests := []struct {
//		name                 string
//		kubeConfig           string
//		getClientSetError    error
//		getGauntletClientErr error
//		expectedErr          error
//	}{
//		{
//			name:                 "Success - InClusterConfig",
//			kubeConfig:           "",
//			getClientSetError:    nil,
//			getGauntletClientErr: nil,
//			expectedErr:          nil,
//		},
//		{
//			name:                 "Success - OutOfClusterConfig",
//			kubeConfig:           "/path/to/kubeconfig",
//			getClientSetError:    nil,
//			getGauntletClientErr: nil,
//			expectedErr:          nil,
//		},
//		{
//			name:                 "Failure - InClusterConfig",
//			kubeConfig:           "",
//			getClientSetError:    errors.New("create in-cluster config fail"),
//			getGauntletClientErr: errors.New("create in-cluster config fail"),
//			expectedErr:          errors.New("create in-cluster config fail"),
//		},
//		{
//			name:                 "Failure - OutOfClusterConfig",
//			kubeConfig:           "/path/to/kubeconfig",
//			getClientSetError:    errors.New("create out-of-cluster config fail"),
//			getGauntletClientErr: errors.New("create out-of-cluster config fail"),
//			expectedErr:          errors.New("create out-of-cluster config fail"),
//		},
//		{
//			name:                 "Failure - Scheme Build",
//			kubeConfig:           "/path/to/kubeconfig",
//			getClientSetError:    nil,
//			getGauntletClientErr: errors.New("create scheme fail"),
//			expectedErr:          errors.New("create scheme fail"),
//		},
//		{
//			name:                 "Failure - Client New",
//			kubeConfig:           "/path/to/kubeconfig",
//			getClientSetError:    nil,
//			getGauntletClientErr: errors.New("create client fail"),
//			expectedErr:          errors.New("create client fail"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Reset global variables
//			clientset = nil
//			gauntletClient = nil
//
//			k := &Kube{
//				KubeConfig: tt.kubeConfig,
//				lock:       sync.Mutex{},
//			}
//
//			// Mock the errors
//			//if tt.getClientSetError != nil {
//			//clientcmd.BuildConfigFromFlags = func(kubeconfig string, masterURL string) (*rest.Config, error) {
//			//	return nil, tt.getClientSetError
//			//}
//			//}
//
//			//// Mock the client creation
//			//client.New = func(cfg *rest.Config, opts client.Options) (client.Client, error) {
//			//	if tt.getGauntletClientErr != nil {
//			//		return nil, tt.getGauntletClientErr
//			//	}
//			//	return fake.NewFakeClientWithScheme(opts.Scheme), nil
//			//}
//
//			// Call the method under test
//			client, err := k.GetGauntletClient()
//
//			// Validate the results
//			if tt.expectedErr != nil {
//				assert.Error(t, err)
//				assert.Equal(t, tt.expectedErr.Error(), err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, client)
//			}
//		})
//	}
//}

// Test for Kube.GetUnitClient method
func TestKubeGetUnitClient(t *testing.T) {
	tests := []struct {
		name             string
		kubeConfig       string
		getUnitClientErr error
		expectedErr      error
	}{
		//{
		//	name:                  "Success - InClusterConfig",
		//	kubeConfig:            "",
		//	getUnitClientErr: nil,
		//	expectedErr:           nil,
		//},
		//{
		//	name:                  "Success - OutOfClusterConfig",
		//	kubeConfig:            "/path/to/kubeconfig",
		//	getUnitClientErr: nil,
		//	expectedErr:           nil,
		//},
		//{
		//	name:                  "Failure - InClusterConfig",
		//	kubeConfig:            "",
		//	getUnitClientErr: errors.New("create in-cluster config fail"),
		//	expectedErr:           errors.New("create in-cluster config fail"),
		//},
		//{
		//	name:                  "Failure - OutOfClusterConfig",
		//	kubeConfig:            "/path/to/kubeconfig",
		//	getUnitClientErr: errors.New("create out-of-cluster config fail"),
		//	expectedErr:           errors.New("create out-of-cluster config fail"),
		//},
		//{
		//	name:                  "Failure - Scheme Build",
		//	kubeConfig:            "/path/to/kubeconfig",
		//	getUnitClientErr: errors.New("create scheme fail"),
		//	expectedErr:           errors.New("create scheme fail"),
		//},
		//{
		//	name:                  "Failure - Client New",
		//	kubeConfig:            "/path/to/kubeconfig",
		//	getUnitClientErr: errors.New("create client fail"),
		//	expectedErr:           errors.New("create client fail"),
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			clientset = nil
			unitClient = nil

			k := &Kube{
				KubeConfig: tt.kubeConfig,
				lock:       sync.Mutex{},
			}

			// Mock the errors
			//if tt.getTesseractClientErr != nil {
			//	// Mocking the getUnitClient method
			//	k.getUnitClient = func() (client.Client, error) {
			//		return nil, tt.getTesseractClientErr
			//	}
			//} else {
			//	// Mocking the successful client creation
			//	k.getUnitClient = func() (client.Client, error) {
			//		return fake.NewFakeClient(), nil
			//	}
			//}

			// Call the method under test
			client, err := k.GetUnitClient()

			// Validate the results
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// Mocks
type MockConfig struct {
	mock.Mock
}

// Test for Kube.getUnitClient method
//func TestGetTesseractClient(t *testing.T) {
//	tests := []struct {
//		name               string
//		kubeConfig         string
//		inClusterConfigErr error
//		buildConfigErr     error
//		schemeBuildErr     error
//		clientNewErr       error
//		expectedErr        error
//	}{
//		{
//			name:               "Success - InClusterConfig",
//			kubeConfig:         "",
//			inClusterConfigErr: nil,
//			buildConfigErr:     nil,
//			schemeBuildErr:     nil,
//			clientNewErr:       nil,
//			expectedErr:        nil,
//		},
//		{
//			name:               "Success - OutOfClusterConfig",
//			kubeConfig:         "/path/to/kubeconfig",
//			inClusterConfigErr: nil,
//			buildConfigErr:     nil,
//			schemeBuildErr:     nil,
//			clientNewErr:       nil,
//			expectedErr:        nil,
//		},
//		{
//			name:               "Failure - InClusterConfig",
//			kubeConfig:         "",
//			inClusterConfigErr: errors.New("create in-cluster config fail"),
//			buildConfigErr:     nil,
//			schemeBuildErr:     nil,
//			clientNewErr:       nil,
//			expectedErr:        errors.New("create in-cluster config fail"),
//		},
//		{
//			name:               "Failure - BuildConfigFromFlags",
//			kubeConfig:         "/path/to/kubeconfig",
//			inClusterConfigErr: nil,
//			buildConfigErr:     errors.New("create out-of-cluster config fail"),
//			schemeBuildErr:     nil,
//			clientNewErr:       nil,
//			expectedErr:        errors.New("create out-of-cluster config fail"),
//		},
//		{
//			name:               "Failure - Scheme Build",
//			kubeConfig:         "/path/to/kubeconfig",
//			inClusterConfigErr: nil,
//			buildConfigErr:     nil,
//			schemeBuildErr:     errors.New("create scheme fail"),
//			clientNewErr:       nil,
//			expectedErr:        errors.New("create scheme fail"),
//		},
//		{
//			name:               "Failure - Client New",
//			kubeConfig:         "/path/to/kubeconfig",
//			inClusterConfigErr: nil,
//			buildConfigErr:     nil,
//			schemeBuildErr:     nil,
//			clientNewErr:       errors.New("create client fail"),
//			expectedErr:        errors.New("create client fail"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Mocking dependencies
//			mockConfig := new(MockConfig)
//			mockConfig.On("InClusterConfig").Return(&rest.Config{}, tt.inClusterConfigErr)
//			mockConfig.On("BuildConfigFromFlags", "", tt.kubeConfig).Return(&rest.Config{}, tt.buildConfigErr)
//
//			k := &Kube{
//				KubeConfig: tt.kubeConfig,
//				lock:       sync.Mutex{},
//			}
//
//			// Replace actual methods with mocks
//			//getClientSet := func() (*rest.Config, error) {
//			//	return mockConfig.InClusterConfig()
//			//}
//			//
//			//getGauntletClient := func() (client.Client, error) {
//			//	return fake.NewFakeClient(), nil
//			//}
//
//			//getUnitClient := func() (client.Client, error) {
//			//	cfg, err := mockConfig.BuildConfigFromFlags("", k.KubeConfig)
//			//	if err != nil {
//			//		return nil, err
//			//	}
//			//
//			//	scheme, err := tesseractv1alpha1.SchemeBuilder.Build()
//			//	if err != nil {
//			//		return nil, err
//			//	}
//			//
//			//	c, err := client.New(cfg, client.Options{Scheme: scheme})
//			//	if err != nil {
//			//		return nil, err
//			//	}
//			//
//			//	return c, nil
//			//}
//
//			// Call the method under test
//			client, err := k.GetUnitClient()
//
//			// Validate the results
//			if tt.expectedErr != nil {
//				assert.Error(t, err)
//				assert.Equal(t, tt.expectedErr.Error(), err.Error())
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, client)
//			}
//		})
//	}
//}
