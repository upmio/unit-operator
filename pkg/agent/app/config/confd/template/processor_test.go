package template

import (
	"testing"
)

func TestProcess(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		//{
		//	name:    "Successful processing",
		//	config:  Config{TemplateFile: "valid_template"},
		//	wantErr: false,
		//},
		{
			name:    "Error in NewTemplateResource",
			config:  Config{TemplateFile: "error"},
			wantErr: true,
		},
		{
			name:    "Error in process method",
			config:  Config{TemplateFile: "process_error"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Process(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
