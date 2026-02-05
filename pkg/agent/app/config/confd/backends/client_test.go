package backends

import (
	"reflect"
	"testing"

	"github.com/upmio/unit-operator/pkg/agent/app/config/confd/backends/content"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantClient StoreClient
		wantErr    bool
	}{
		{
			name:       "Valid content backend",
			config:     Config{Backend: "content", Contents: []string{"key: value"}},
			wantClient: content.NewContentClient([]string{"key: value"}),
			wantErr:    false,
		},
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
