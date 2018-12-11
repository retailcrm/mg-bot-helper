package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
)

// BotConfig struct
type BotConfig struct {
	Version    string           `yaml:"version"`
	LogLevel   logging.Level    `yaml:"log_level"`
	Database   DatabaseConfig   `yaml:"database"`
	SentryDSN  string           `yaml:"sentry_dsn"`
	HTTPServer HTTPServerConfig `yaml:"http_server"`
	Debug      bool             `yaml:"debug"`
	BotInfo    BotInfo          `yaml:"bot_info"`
}

type BotInfo struct {
	Name     string `yaml:"name"`
	Code     string `yaml:"code"`
	LogoPath string `yaml:"logo_path"`
}

// DatabaseConfig struct
type DatabaseConfig struct {
	Connection         string `yaml:"connection"`
	Logging            bool   `yaml:"logging"`
	TablePrefix        string `yaml:"table_prefix"`
	MaxOpenConnections int    `yaml:"max_open_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
	ConnectionLifetime int    `yaml:"connection_lifetime"`
}

// HTTPServerConfig struct
type HTTPServerConfig struct {
	Host   string `yaml:"host"`
	Listen string `yaml:"listen"`
}

// LoadConfig read configuration file
func LoadConfig(path string) *BotConfig {
	var err error

	path, err = filepath.Abs(path)
	if err != nil {
		panic(err)
	}

	source, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var c BotConfig
	if err = yaml.Unmarshal(source, &c); err != nil {
		panic(err)
	}

	return &c
}
