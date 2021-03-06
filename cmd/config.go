package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gosexy/to"
	"gopkg.in/yaml.v2"

	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/metrics/graphite"
)

// RedisConfig is redis config structure, which are taken on the start of moira
type RedisConfig struct {
	MasterName    string `yaml:"master_name"`
	SentinelAddrs string `yaml:"sentinel_addrs"`
	Host          string `yaml:"host"`
	Port          string `yaml:"port"`
	DBID          int    `yaml:"dbid"`
}

// GetSettings return redis config parsed from moira config files
func (config *RedisConfig) GetSettings() redis.Config {
	return redis.Config{
		MasterName:        config.MasterName,
		SentinelAddresses: strings.Split(config.SentinelAddrs, ","),
		Host:              config.Host,
		Port:              config.Port,
		DBID:              config.DBID,
	}
}

// GraphiteConfig is graphite metrics config, which are taken on the start of moira
type GraphiteConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URI      string `yaml:"uri"`
	Prefix   string `yaml:"prefix"`
	Interval string `yaml:"interval"`
}

// GetSettings return graphite metrics config parsed from moira config files
func (graphiteConfig *GraphiteConfig) GetSettings() graphite.Config {
	return graphite.Config{
		Enabled:  graphiteConfig.Enabled,
		URI:      graphiteConfig.URI,
		Prefix:   graphiteConfig.Prefix,
		Interval: to.Duration(graphiteConfig.Interval),
	}
}

// LoggerConfig is logger settings, which are taken on the start of moira
type LoggerConfig struct {
	LogFile  string `yaml:"log_file"`
	LogLevel string `yaml:"log_level"`
}

// ProfilerConfig is pprof settings, which are taken on the start of moira
type ProfilerConfig struct {
	Listen string `yaml:"listen"`
}

// ReadConfig gets config file by given file and marshal it to moira-used type
func ReadConfig(configFileName string, config interface{}) error {
	configYaml, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return fmt.Errorf("Can't read file [%s] [%s]", configFileName, err.Error())
	}
	err = yaml.Unmarshal(configYaml, config)
	if err != nil {
		return fmt.Errorf("Can't parse config file [%s] [%s]", configFileName, err.Error())
	}
	return nil
}

// PrintConfig prints config to std
func PrintConfig(config interface{}) {
	d, _ := yaml.Marshal(&config)
	fmt.Println(string(d))
}
