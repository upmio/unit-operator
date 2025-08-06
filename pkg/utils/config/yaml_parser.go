package config

import (
	"bytes"
	"fmt"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

func NewViper(originConfig, configType string) (*viper.Viper, error) {
	viperConfig := viper.New()
	// TODO: support KeyDelimiter eg: '#'
	keyDelimiter := ""
	if keyDelimiter != "" {
		viperConfig = viper.NewWithOptions(viper.KeyDelimiter(keyDelimiter))
	}

	// 目前配置文件格式都改为yaml
	if configType == "" {
		configType = "yaml"
	}
	viperConfig.SetConfigType(configType)

	err := viperConfig.ReadConfig(bytes.NewBuffer([]byte(originConfig)))
	if err != nil {
		return nil, fmt.Errorf("Viper read origin config ERROR:[%s] ", err.Error())
	}

	return viperConfig, nil
}

func Viper2String(viperConfig *viper.Viper) (string, error) {
	return yamlStringSettings(viperConfig)
}

func yamlStringSettings(viperConfig *viper.Viper) (string, error) {
	c := viperConfig.AllSettings()
	bs, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("unable to marshal config to YAML: %v", err)
	}
	return string(bs), nil
}
