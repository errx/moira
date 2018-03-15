package main

import (
	"runtime"

	"github.com/gosexy/to"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/cmd"
)

type config struct {
	Redis    cmd.RedisConfig    `yaml:"redis"`
	Graphite cmd.GraphiteConfig `yaml:"graphite"`
	Logger   cmd.LoggerConfig   `yaml:"log"`
	Checker  checkerConfig      `yaml:"checker"`
	Pprof    cmd.ProfilerConfig `yaml:"pprof"`
}

type checkerConfig struct {
	NoDataCheckInterval  string `yaml:"nodata_check_interval"`  // Period for check all triggers for NODATA
	CheckInterval        string `yaml:"check_interval"`         // Min period for re-check triggers. Reduce this duration may be increase CPU and memory usage
	MetricsTTL           string `yaml:"metrics_ttl"`            // Time interval to store metrics. Increase of this value also increases redis memory consumption
	StopCheckingInterval string `yaml:"stop_checking_interval"` // Если у вас перестали идти метрики, то вероятней всего все ваши триггеры упадут в NODATA. Это время указывает, сколько должно пройти времени после последней метрики, чтобы перестать чекать триггеры
	MaxParallelChecks    int    `yaml:"max_parallel_checks"`    // Max concurrent checks of triggers, 0 - processor cores number.
}

func (config *checkerConfig) getSettings() *checker.Config {
	if config.MaxParallelChecks == 0 {
		config.MaxParallelChecks = runtime.NumCPU()
	}
	return &checker.Config{
		MetricsTTLSeconds:           int64(to.Duration(config.MetricsTTL).Seconds()),
		CheckInterval:               to.Duration(config.CheckInterval),
		NoDataCheckInterval:         to.Duration(config.NoDataCheckInterval),
		StopCheckingIntervalSeconds: int64(to.Duration(config.StopCheckingInterval).Seconds()),
		MaxParallelChecks:           config.MaxParallelChecks,
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
		Checker: checkerConfig{
			NoDataCheckInterval:  "60s",
			CheckInterval:        "5s",
			MetricsTTL:           "1h",
			StopCheckingInterval: "30s",
			MaxParallelChecks:    0,
		},
		Graphite: cmd.GraphiteConfig{
			URI:      "localhost:2003",
			Prefix:   "DevOps.Moira",
			Interval: "60s",
		},
		Pprof: cmd.ProfilerConfig{
			Listen: "",
		},
	}
}
