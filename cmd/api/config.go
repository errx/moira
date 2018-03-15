package main

import (
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis  cmd.RedisConfig    `yaml:"redis"`
	Logger cmd.LoggerConfig   `yaml:"log"`
	API    apiConfig          `yaml:"api"`
	Pprof  cmd.ProfilerConfig `yaml:"pprof"`
}

type apiConfig struct {
	Listen        string `yaml:"listen"`          // Listen announces on api local network address.
	EnableCORS    bool   `yaml:"enable_cors"`     // Then true enable CORS for cross-domain requests. Use it only for local debug
	WebConfigPath string `yaml:"web_config_path"` // Web_UI config file path. If file not found, api request "api/config" will return 404
}

func (config *apiConfig) getSettings() *api.Config {
	return &api.Config{
		Listen:     config.Listen,
		EnableCORS: config.EnableCORS,
	}
}

func getDefault() config {
	return config{
		Redis: cmd.RedisConfig{
			Host: "localhost",
			Port: "6379",
		},
		Logger: cmd.LoggerConfig{
			LogFile:  "stdout",
			LogLevel: "info",
		},
		API: apiConfig{
			Listen:        ":8081",
			WebConfigPath: "/etc/moira/web.json",
			EnableCORS:    false,
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}
