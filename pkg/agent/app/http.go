package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

var (
	httpApps = map[string]HTTPApp{}
)

type HTTPApp interface {
	Registry(r *gin.Engine, appName string)
	Config() error
	Name() string
}

func LoadHttpApp(router *gin.Engine) {
	for appName, api := range httpApps {
		api.Registry(router, appName)
	}
}

func RegistryHttpApp(app HTTPApp) {
	// re-registration of already registered services is prohibited
	_, ok := httpApps[app.Name()]
	if ok {
		panic(fmt.Sprintf("http app %s has registed", app.Name()))
	}

	httpApps[app.Name()] = app
}

func LoadedHttpApp() (apps []string) {
	for k := range httpApps {
		apps = append(apps, k)
	}
	return
}
