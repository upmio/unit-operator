package conf

import (
	"os"
	"sync"
	"testing"
)

func TestGetConf(t *testing.T) {
	tests := []struct {
		name         string
		setupConfig  func()
		expectPanic  bool
		expectConfig *Config
	}{
		{
			name: "config is nil, should panic",
			setupConfig: func() {
				config = nil
			},
			expectPanic: true,
		},
		//{
		//	name: "config is initialized, should return config",
		//	setupConfig: func() {
		//		config = &Config{
		//			Log: &Log{
		//				Level:   "info",
		//				PathDir: "/var/log",
		//			},
		//			App: &App{
		//				Host:     "localhost",
		//				Port:     8080,
		//				GrpcHost: "localhost",
		//				GrpcPort: 9090,
		//			},
		//			Kube: &Kube{
		//				KubeConfig: "/path/to/kubeconfig",
		//			},
		//			Supervisor: &Supervisor{
		//				Addr: "localhost",
		//				Port: 9001,
		//			},
		//		}
		//	},
		//	expectPanic:  false,
		//	expectConfig: config,
		//},
		//{
		//	name: "concurrent GetConf calls",
		//	setupConfig: func() {
		//		config = &Config{
		//			Log: &Log{
		//				Level:   "info",
		//				PathDir: "/var/log",
		//			},
		//		}
		//	},
		//	expectPanic:  false,
		//	expectConfig: config,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupConfig()
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic, but got none")
					}
				}()
				_ = GetConf() // This should panic
			} else {
				var wg sync.WaitGroup
				wg.Add(10)
				for i := 0; i < 10; i++ {
					go func() {
						defer wg.Done()
						conf := GetConf()
						if conf != tt.expectConfig {
							t.Errorf("expected config %+v, but got %+v", tt.expectConfig, conf)
						}
					}()
				}
				wg.Wait()
			}
		})
	}
}

func TestLoadConfigFromToml(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() string
		expectErr bool
	}{
		{
			name: "valid TOML file",
			setup: func() string {
				content := `
				[log]
				level = "debug"
				dir = "/var/log"

				[app]
				host = "localhost"
				port = 8080
				grpc_host = "localhost"
				grpc_port = 9090

				[kube]
				kubeConfigPath = "/path/to/kubeconfig"

				[supervisor]
				address = "localhost"
				port = 9001
				`
				path := "valid_config.toml"
				err := os.WriteFile(path, []byte(content), 0644)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return path
			},
			expectErr: false,
		},
		{
			name: "invalid TOML file path",
			setup: func() string {
				return "invalid_path.toml"
			},
			expectErr: true,
		},
		{
			name: "invalid TOML content",
			setup: func() string {
				content := `
				[log
				level = "debug"
				dir = "/var/log"
				`
				path := "invalid_content.toml"
				err := os.WriteFile(path, []byte(content), 0644)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return path
			},
			expectErr: true,
		},
		//{
		//	name: "missing required fields",
		//	setup: func() string {
		//		content := `
		//		[log]
		//		level = "debug"
		//		`
		//		path := "missing_fields.toml"
		//		err := os.WriteFile(path, []byte(content), 0644)
		//		if err != nil {
		//			t.Fatalf("setup failed: %v", err)
		//		}
		//		return path
		//	},
		//	expectErr: true,
		//},
		{
			name: "empty file path",
			setup: func() string {
				return ""
			},
			expectErr: true,
		},
		{
			name: "directory as file path",
			setup: func() string {
				return "."
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			err := LoadConfigFromToml(path)
			if (err != nil) != tt.expectErr {
				t.Errorf("LoadConfigFromToml() error = %v, expectErr %v", err, tt.expectErr)
			}

			// Clean up the test files
			if path != "" && path != "." {
				os.Remove(path)
			}
		})
	}
}

func TestGrpcAddr(t *testing.T) {
	tests := []struct {
		name     string
		app      *App
		expected string
	}{
		{
			name: "Valid Host and Port",
			app: &App{
				GrpcHost: "localhost",
				GrpcPort: 8080,
			},
			expected: "localhost:8080",
		},
		{
			name: "Empty Host",
			app: &App{
				GrpcHost: "",
				GrpcPort: 8080,
			},
			expected: ":8080",
		},
		{
			name: "Zero Port",
			app: &App{
				GrpcHost: "localhost",
				GrpcPort: 0,
			},
			expected: "localhost:0",
		},
		{
			name: "IP Address as Host",
			app: &App{
				GrpcHost: "127.0.0.1",
				GrpcPort: 9090,
			},
			expected: "127.0.0.1:9090",
		},
		{
			name: "Empty Host and Zero Port",
			app: &App{
				GrpcHost: "",
				GrpcPort: 0,
			},
			expected: ":0",
		},
		{
			name: "Special Characters in Host",
			app: &App{
				GrpcHost: "example.com!@#",
				GrpcPort: 6060,
			},
			expected: "example.com!@#:6060",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.app.GrpcAddr()
			if result != tt.expected {
				t.Errorf("GrpcAddr() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAddr(t *testing.T) {
	tests := []struct {
		name     string
		app      *App
		expected string
	}{
		{
			name: "Valid Host and Port",
			app: &App{
				Host: "localhost",
				Port: 8080,
			},
			expected: "localhost:8080",
		},
		{
			name: "Empty Host",
			app: &App{
				Host: "",
				Port: 8080,
			},
			expected: ":8080",
		},
		{
			name: "Zero Port",
			app: &App{
				Host: "localhost",
				Port: 0,
			},
			expected: "localhost:0",
		},
		{
			name: "IP Address as Host",
			app: &App{
				Host: "127.0.0.1",
				Port: 9090,
			},
			expected: "127.0.0.1:9090",
		},
		{
			name: "Empty Host and Zero Port",
			app: &App{
				Host: "",
				Port: 0,
			},
			expected: ":0",
		},
		{
			name: "Special Characters in Host",
			app: &App{
				Host: "example.com!@#",
				Port: 6060,
			},
			expected: "example.com!@#:6060",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.app.Addr()
			if result != tt.expected {
				t.Errorf("Addr() = %v, want %v", result, tt.expected)
			}
		})
	}
}
