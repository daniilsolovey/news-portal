package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-pg/pg/v10"
)

// config default values
const (
	dbURL             = "postgres://user:password@localhost:5432/news_portal?sslmode=disable"
	dbMaxConns        = 5
	dbMaxConnLifetime = "300s"
	dbMaxRetries      = 3
	appHost           = "0.0.0.0"
	appPort           = 3000
	tomlFile          = "config.toml"
)

type Config struct {
	Database pg.Options
	Host     string
	Port     int
	Debug    bool
}

type Toml struct {
	Database struct {
		URL             string `toml:"url"`
		MaxConns        int    `toml:"max_conns"`
		MaxConnLifetime string `toml:"max_conn_lifetime"`
	} `toml:"database"`
	Server struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
	} `toml:"server"`
}

func Init() (*Config, error) {
	var (
		configFile string
		debug      bool
	)

	flag.StringVar(&configFile, "config", tomlFile, "path to TOML configuration file")
	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.Parse()

	config := &Config{
		Host:  appHost,
		Port:  appPort,
		Debug: debug,
	}

	tomlConfig, err := loadTOML(configFile)
	if err != nil {
		return nil, fmt.Errorf("load TOML config: %w", err)
	}

	databaseURL := dbURL
	maxConns := dbMaxConns
	maxConnLifetime := dbMaxConnLifetime

	if tomlConfig != nil {
		if tomlConfig.Database.URL != "" {
			databaseURL = tomlConfig.Database.URL
		}
		if tomlConfig.Database.MaxConns > 0 {
			maxConns = tomlConfig.Database.MaxConns
		}
		if tomlConfig.Database.MaxConnLifetime != "" {
			maxConnLifetime = tomlConfig.Database.MaxConnLifetime
		}

		if tomlConfig.Server.Host != "" {
			config.Host = tomlConfig.Server.Host
		}
		if tomlConfig.Server.Port > 0 {
			config.Port = tomlConfig.Server.Port
		}

	}

	opt, err := pg.ParseURL(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	opt.MaxRetries = dbMaxRetries
	opt.PoolSize = maxConns

	if maxConnLifetime != "" {
		lifetime, err := time.ParseDuration(maxConnLifetime)
		if err != nil {
			return nil, fmt.Errorf("parse max connection lifetime: %w", err)
		}

		opt.MaxConnAge = lifetime
	}

	config.Database = *opt
	return config, nil
}

func loadTOML(configFile string) (*Toml, error) {
	if configFile == "" {
		configFile = tomlFile
	}

	_, err := os.Stat(configFile)
	if err != nil {
		return nil, nil
	}

	var config Toml
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, fmt.Errorf("decode TOML: %w", err)
	}

	return &config, nil
}
