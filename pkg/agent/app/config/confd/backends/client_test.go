package backends

import (
	"reflect"
	"testing"
)

// Mock implementations for file and content clients
type MockFileClient struct{}

func (m *MockFileClient) GetValues() (map[string]string, error) { return nil, nil }

type MockContentClient struct{}

func (m *MockContentClient) GetValues() (map[string]string, error) { return nil, nil }

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantClient StoreClient
		wantErr    bool
	}{
		//{
		//	name:       "Valid file backend",
		//	config:     Config{Backend: "file", YAMLFile: "path/to/yaml"},
		//	wantClient: &MockFileClient{},
		//	wantErr:    false,
		//},
		//{
		//	name:       "Valid content backend",
		//	config:     Config{Backend: "content", Contents: []string{"content1", "content2"}},
		//	wantClient: &MockContentClient{},
		//	wantErr:    false,
		//},
		{
			name:       "Invalid backend",
			config:     Config{Backend: "invalid"},
			wantClient: nil,
			wantErr:    true,
		},
		{
			name:       "Empty backend",
			config:     Config{Backend: ""},
			wantClient: nil,
			wantErr:    true,
		},
		//{
		//	name:       "File backend with missing YAMLFile",
		//	config:     Config{Backend: "file", YAMLFile: ""},
		//	wantClient: &MockFileClient{},
		//	wantErr:    false,
		//},
		//{
		//	name:       "Content backend with empty contents",
		//	config:     Config{Backend: "content", Contents: []string{}},
		//	wantClient: &MockContentClient{},
		//	wantErr:    false,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClient, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotClient, tt.wantClient) {
				t.Errorf("New() gotClient = %v, want %v", gotClient, tt.wantClient)
			}
		})
	}
}
