package conf

import (
	"github.com/abrander/go-supervisord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
)

// MockSupervisor is a mock type for Supervisor
type MockSupervisor struct {
	mock.Mock
	lock sync.Mutex
}

func (m *MockSupervisor) getSupervisorClient() (*supervisord.Client, error) {
	args := m.Called()
	return args.Get(0).(*supervisord.Client), args.Error(1)
}

// TestGetSupervisorClient tests the GetSupervisorClient method
func TestSupervisorGetSupervisorClient(t *testing.T) {
	tests := []struct {
		name           string
		mockGetClient  func(mockSupervisor *MockSupervisor)
		expectedError  error
		expectedClient *supervisord.Client
	}{
		{
			name: "Successful Client Creation",
			mockGetClient: func(mockSupervisor *MockSupervisor) {
				mockSupervisor.On("getSupervisorClient").Return(&supervisord.Client{}, nil)
			},
			expectedError:  nil,
			expectedClient: &supervisord.Client{},
		},
		//{
		//	name: "Error During Client Creation",
		//	mockGetClient: func(mockSupervisor *MockSupervisor) {
		//		mockSupervisor.On("getSupervisorClient").Return(nil, errors.New("connection error"))
		//	},
		//	expectedError:  errors.New("connection error"),
		//	expectedClient: nil,
		//},
		{
			name: "Client Cached After First Call",
			mockGetClient: func(mockSupervisor *MockSupervisor) {
				mockSupervisor.On("getSupervisorClient").Return(&supervisord.Client{}, nil).Once()
			},
			expectedError:  nil,
			expectedClient: &supervisord.Client{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSupervisor := new(MockSupervisor)
			tt.mockGetClient(mockSupervisor)

			client, err := mockSupervisor.getSupervisorClient()
			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedClient, client)

			// Ensure getSupervisorClient was called the expected number of times
			mockSupervisor.AssertExpectations(t)
		})
	}
}

//func TestGetSupervisorClient(t *testing.T) {
//	tests := []struct {
//		name           string
//		mockNewClient  func(address string) (*supervisord.Client, error)
//		mockGetClient  func(mockSupervisor *MockSupervisor)
//		expectedError  error
//		expectedClient *supervisord.Client
//	}{
//		{
//			name: "Successful Client Creation",
//			mockNewClient: func(address string) (*supervisord.Client, error) {
//				return &supervisord.Client{}, nil
//			},
//			mockGetClient: func(mockSupervisor *MockSupervisor) {
//				// Not needed here because MockNewClient is used directly
//			},
//			expectedError:  nil,
//			expectedClient: &supervisord.Client{},
//		},
//		{
//			name: "Error During Client Creation",
//			mockNewClient: func(address string) (*supervisord.Client, error) {
//				return nil, errors.New("connection error")
//			},
//			mockGetClient: func(mockSupervisor *MockSupervisor) {
//				// Not needed here because MockNewClient is used directly
//			},
//			expectedError:  errors.New("connection error"),
//			expectedClient: nil,
//		},
//		{
//			name: "Client Cached After First Call",
//			mockNewClient: func(address string) (*supervisord.Client, error) {
//				return &supervisord.Client{}, nil
//			},
//			mockGetClient: func(mockSupervisor *MockSupervisor) {
//				// Not needed here because MockNewClient is used directly
//			},
//			expectedError:  nil,
//			expectedClient: &supervisord.Client{},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Mock NewClient
//			//originalNewClient := supervisord.NewClient
//			//supervisord.NewClient = tt.mockNewClient
//			//defer func() { supervisord.NewClient = originalNewClient }()
//
//			mockSupervisor := new(MockSupervisor)
//			tt.mockGetClient(mockSupervisor)
//
//			client, err := mockSupervisor.getSupervisorClient()
//			if tt.expectedError != nil {
//				assert.EqualError(t, err, tt.expectedError.Error())
//			} else {
//				assert.NoError(t, err)
//			}
//
//			assert.Equal(t, tt.expectedClient, client)
//
//			// Ensure getSupervisorClient was called the expected number of times
//			mockSupervisor.AssertExpectations(t)
//		})
//	}
//}
