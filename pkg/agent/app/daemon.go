package app

import (
	"context"
	"fmt"
	"sync"
)

// DaemonApp DaemonService instance
type DaemonApp interface {
	Registry(context.Context, *sync.WaitGroup)
	Config() error
	Name() string
}

var (
	daemonApps = map[string]DaemonApp{}
)

func RegistryDaemonApp(app DaemonApp) {
	_, ok := daemonApps[app.Name()]
	if ok {
		panic(fmt.Sprintf("daemon app %s has registered", app.Name()))
	}

	daemonApps[app.Name()] = app
}

// LoadedDaemonApp query successfully loaded services
func LoadedDaemonApp() (apps []string) {
	for k := range daemonApps {
		apps = append(apps, k)
	}
	return
}
