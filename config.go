package main

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var configFilePath string

type KeyAction struct {
	Key      string  `yaml:"key"`
	Interval float64 `yaml:"interval"`
}

type RemapAction struct {
	Key   string   `yaml:"key"`
	To    []string `yaml:"to"`
	Delay int      `yaml:"delay"`
}

type AppConfig struct {
	AppClass string        `yaml:"app_class"`
	PauseKey string        `yaml:"pause_key"`
	Keys     []KeyAction   `yaml:"keys"`
	Remaps   []RemapAction `yaml:"remaps"`
}

type Config struct {
	PauseKey string      `yaml:"pause_key"`
	Apps     []AppConfig `yaml:"apps"`
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var c Config
	err = yaml.Unmarshal(data, &c)
	return &c, err
}

func findAppConfig(config *Config, appClass string) *AppConfig {
	for _, app := range config.Apps {
		if strings.EqualFold(app.AppClass, appClass) {
			return &app
		}
	}
	return nil
}
