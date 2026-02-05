package conf

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
)

var config *Config

type Config struct {
	*Log        `toml:"log"`
	*App        `toml:"app"`
	*Kube       `toml:"kube"`
	*Supervisor `toml:"supervisor"`
}

type Log struct {
	Level   string `toml:"level"`
	PathDir string `toml:"dir"`
}

type App struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	GrpcHost string `toml:"grpc_host"`
	GrpcPort int    `toml:"grpc_port"`
}

type Kube struct {
	KubeConfig string `toml:"kubeConfigPath"`
	lock       sync.Mutex
}

type Supervisor struct {
	Addr string `toml:"address"`
	Port int    `toml:"port"`
	lock sync.Mutex
}

func GetConf() *Config {
	if config == nil {
		panic("cannot read config")
	}
	return config
}

// LoadConfigFromToml load from configuration file
func LoadConfigFromToml(path string) error {
	_, err := toml.DecodeFile(path, &config)

	if err != nil {
		return err
	}

	return nil
}

func (a *App) GrpcAddr() string {
	return fmt.Sprintf("%s:%d", a.GrpcHost, a.GrpcPort)
}

func (a *App) Addr() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}
