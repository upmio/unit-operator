package event

import (
	"context"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	args := m.Called(ctx, key, obj)
	return args.Error(0)
}

// MockEventRecorder
type MockEventRecorder struct {
	mock.Mock
}

func (m *MockEventRecorder) Event(object client.Object, eventType, reason, message string) {
	m.Called(object, eventType, reason, message)
}

//func TestSendNormalEventToUnit(t *testing.T) {
//	logger, _ := zap.NewDevelopment()
//	mockClient := new(MockClient)
//	mockRecorder := new(MockEventRecorder)
//
//	s := &service{
//		//unitClient: mockClient,
//		//EventRecorder:   mockRecorder,
//		logger: logger.Sugar(),
//	}
//
//	tests := []struct {
//		name          string
//		unitName      string
//		namespace     string
//		reason        string
//		message       string
//		getError      error
//		expectedError bool
//		expectedMsg   string
//	}{
//		//{
//		//	name:          "UnitNotFound",
//		//	unitName:      "non-existent-unit",
//		//	namespace:     "default",
//		//	reason:        "UnitNotFound",
//		//	message:       "Unit not found in the given namespace",
//		//	getError:      errors.New("unit not found"),
//		//	expectedError: true,
//		//	expectedMsg:   "unit not found",
//		//},
//		{
//			name:          "SuccessfulEventCreation",
//			unitName:      "test-unit",
//			namespace:     "default",
//			reason:        "UnitStarted",
//			message:       "Unit has started successfully",
//			getError:      nil,
//			expectedError: false,
//			expectedMsg:   "",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			unit := &apiV1alpha1.Unit{}
//
//			mockClient.On("Get", mock.Anything, types.NamespacedName{Name: tt.unitName, Namespace: tt.namespace}, unit).Return(tt.getError)
//
//			if tt.getError == nil {
//				mockRecorder.On("Event", unit, v1.EventTypeNormal, tt.reason, tt.message).Return()
//			}
//
//			err := s.SendNormalEventToUnit(tt.unitName, tt.namespace, tt.reason, tt.message)
//
//			if tt.expectedError {
//				assert.Error(t, err)
//				assert.Contains(t, err.Error(), tt.expectedMsg)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			mockClient.AssertExpectations(t)
//			mockRecorder.AssertExpectations(t)
//		})
//	}
//}

//func TestSendWarningEventToUnit(t *testing.T) {
//	logger, _ := zap.NewDevelopment()
//	mockClient := new(MockClient)
//	mockRecorder := new(MockEventRecorder)
//
//	s := &service{
//		//unitClient: mockClient,
//		//EventRecorder:   mockRecorder,
//		logger: logger.Sugar(),
//	}
//
//	tests := []struct {
//		name          string
//		unitName      string
//		namespace     string
//		reason        string
//		message       string
//		getError      error
//		expectedError bool
//		expectedMsg   string
//	}{
//		{
//			name:          "UnitNotFound",
//			unitName:      "non-existent-unit",
//			namespace:     "default",
//			reason:        "UnitNotFound",
//			message:       "Unit not found in the given namespace",
//			getError:      errors.New("unit not found"),
//			expectedError: true,
//			expectedMsg:   "unit not found",
//		},
//		{
//			name:          "SuccessfulEventCreation",
//			unitName:      "test-unit",
//			namespace:     "default",
//			reason:        "UnitWarning",
//			message:       "There is a warning for the unit",
//			getError:      nil,
//			expectedError: false,
//			expectedMsg:   "",
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			unit := &apiV1alpha1.Unit{}
//
//			mockClient.On("Get", mock.Anything, types.NamespacedName{Name: tt.unitName, Namespace: tt.namespace}, unit).Return(tt.getError)
//
//			if tt.getError == nil {
//				mockRecorder.On("Event", unit, v1.EventTypeWarning, tt.reason, tt.message).Return()
//			}
//
//			err := s.SendWarningEventToUnit(tt.unitName, tt.namespace, tt.reason, tt.message)
//
//			if tt.expectedError {
//				assert.Error(t, err)
//				assert.Contains(t, err.Error(), tt.expectedMsg)
//			} else {
//				assert.NoError(t, err)
//			}
//
//			mockClient.AssertExpectations(t)
//			mockRecorder.AssertExpectations(t)
//		})
//	}
//}

type MockConf struct {
	mock.Mock
}

func (m *MockConf) GetClientSet() (*fake.Clientset, error) {
	args := m.Called()
	return args.Get(0).(*fake.Clientset), args.Error(1)
}

func (m *MockConf) GetTesseractClient() (*fake.Clientset, error) {
	args := m.Called()
	return args.Get(0).(*fake.Clientset), args.Error(1)
}

//func TestNewIEventRecorder(t *testing.T) {
//	//logger, _ := zap.NewDevelopment()
//	mockConf := new(MockConf)
//	fakeClientSet := fake.NewSimpleClientset()
//
//	tests := []struct {
//		name          string
//		setupMocks    func()
//		expectedError error
//	}{
//		{
//			name: "SuccessfulInitialization",
//			setupMocks: func() {
//				mockConf.On("GetClientSet").Return(fakeClientSet, nil).Once()
//				mockConf.On("GetUnitClient").Return(fakeClientSet, nil).Once()
//			},
//			expectedError: nil,
//		},
//		{
//			name: "ClientSetError",
//			setupMocks: func() {
//				mockConf.On("GetClientSet").Return(nil, errors.New("client set error")).Once()
//			},
//			expectedError: errors.New("client set error"),
//		},
//		{
//			name: "TesseractClientError",
//			setupMocks: func() {
//				mockConf.On("GetClientSet").Return(fakeClientSet, nil).Once()
//				mockConf.On("GetUnitClient").Return(nil, errors.New("tesseract client error")).Once()
//			},
//			expectedError: errors.New("tesseract client error"),
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.setupMocks()
//
//			recorder, err := NewIEventRecorder()
//			if tt.expectedError != nil {
//				assert.Error(t, err)
//				assert.EqualError(t, err, tt.expectedError.Error())
//				assert.Nil(t, recorder)
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, recorder)
//			}
//
//			mockConf.AssertExpectations(t)
//		})
//	}
//}
