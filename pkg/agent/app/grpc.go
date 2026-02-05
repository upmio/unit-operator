package app

import (
	"fmt"

	"google.golang.org/grpc"
)

// GRPCApp GRPCService instance
type GRPCApp interface {
	Registry(*grpc.Server)
	Config() error
	Name() string
}

var (
	grpcApps = map[string]GRPCApp{}
)

// RegistryGrpcApp RegistryService service Instance Registration
func RegistryGrpcApp(app GRPCApp) {
	// re-registration of already registered services is prohibited
	_, ok := grpcApps[app.Name()]
	if ok {
		panic(fmt.Sprintf("grpc app %s has registered", app.Name()))
	}

	grpcApps[app.Name()] = app
}

// LoadedGrpcApp query successfully loaded services
func LoadedGrpcApp() (apps []string) {
	for k := range grpcApps {
		apps = append(apps, k)
	}
	return
}

func GetGrpcApp(name string) GRPCApp {
	app, ok := grpcApps[name]
	if !ok {
		panic(fmt.Sprintf("grpc app %s not registered", name))
	}

	return app
}

// LoadGrpcApp load all Grpc app
func LoadGrpcApp(server *grpc.Server) error {
	for name, app := range grpcApps {
		err := app.Config()
		if err != nil {
			return fmt.Errorf("config grpc app %s error %s", name, err)
		}

		app.Registry(server)
	}
	return nil
}
