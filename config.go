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
	Hold     bool    `yaml:"hold"`
	Delay    int     `yaml:"delay"`
	Reassert int     `yaml:"reassert"`
}

type RemapAction struct {
	Key   string   `yaml:"key"`
	To    []string `yaml:"to"`
	Delay int      `yaml:"delay"`
}

type AutoclickAction struct {
	Trigger  string  `yaml:"trigger"`
	Mode     string  `yaml:"mode"`
	Button   string  `yaml:"button"`
	Interval float64 `yaml:"interval"`
}

type AppConfig struct {
	AppClass     string            `yaml:"app_class"`
	PauseKey     string            `yaml:"pause_key"`
	Keys         []KeyAction       `yaml:"keys"`
	Remaps       []RemapAction     `yaml:"remaps"`
	Autoclickers []AutoclickAction `yaml:"autoclickers"`
}

type NotifyConfig struct {
	System bool   `yaml:"system"`
	Sound  string `yaml:"sound"`
}

type Config struct {
	PauseKey      string            `yaml:"pause_key"`
	Notifications NotifyConfig      `yaml:"notifications"`
	Autoclickers  []AutoclickAction `yaml:"autoclickers"`
	Apps          []AppConfig       `yaml:"apps"`
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
