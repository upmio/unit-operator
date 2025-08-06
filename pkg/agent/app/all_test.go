package app

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// Mock implementations of the Config interface
type MockConfigurable struct {
	mock.Mock
}

func (m *MockConfigurable) Config() error {
	args := m.Called()
	return args.Error(0)
}

type Configurable interface {
	Config() error
}

func TestInitAllApp(t *testing.T) {
	tests := []struct {
		name          string
		grpcApps      []Configurable
		httpApps      []Configurable
		expectedError error
	}{
		{
			name: "All apps config success",
			grpcApps: []Configurable{
				&MockConfigurable{},
				&MockConfigurable{},
			},
			httpApps: []Configurable{
				&MockConfigurable{},
				&MockConfigurable{},
			},
			expectedError: nil,
		},
		//{
		//	name: "GRPC app config failure",
		//	grpcApps: []Configurable{
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(assert.AnError)
		//			return m
		//		}(),
		//	},
		//	httpApps: []Configurable{
		//		&MockConfigurable{},
		//		&MockConfigurable{},
		//	},
		//	expectedError: assert.AnError,
		//},
		//{
		//	name: "HTTP app config failure",
		//	grpcApps: []Configurable{
		//		&MockConfigurable{},
		//		&MockConfigurable{},
		//	},
		//	httpApps: []Configurable{
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(assert.AnError)
		//			return m
		//		}(),
		//	},
		//	expectedError: assert.AnError,
		//},
		//{
		//	name: "GRPC and HTTP apps config failure",
		//	grpcApps: []Configurable{
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(assert.AnError)
		//			return m
		//		}(),
		//	},
		//	httpApps: []Configurable{
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(nil)
		//			return m
		//		}(),
		//		func() *MockConfigurable {
		//			m := &MockConfigurable{}
		//			m.On("Config").Return(assert.AnError)
		//			return m
		//		}(),
		//	},
		//	expectedError: assert.AnError,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//grpcApps = tt.grpcApps
			//httpApps = tt.httpApps

			err := InitAllApp()

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
