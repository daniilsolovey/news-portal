package configs

import (
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/namsral/flag"
)

type Config struct {
	Database pg.Options
	Host     string
	Port     int
}

var cfg Config

func Init() *Config {
	var databaseURL string
	var dbMaxConns int
	var dbMaxConnLifetime string

	flag.StringVar(&databaseURL, "database-url", "postgres://user:password@localhost:5432/news_portal?sslmode=disable", "database connection URL (DATABASE_URL)")
	flag.IntVar(&dbMaxConns, "db-max-conns", 5, "maximum number of database connections (DB_MAX_CONNS)")
	flag.StringVar(&dbMaxConnLifetime, "db-max-conn-lifetime", "300s", "maximum lifetime of database connection (DB_MAX_CONN_LIFETIME)")
	flag.StringVar(&cfg.Host, "host", "0.0.0.0", "host to bind server (HOST)")
	flag.IntVar(&cfg.Port, "port", 3000, "HTTP server port (PORT)")

	flag.Parse()

	opt, err := pg.ParseURL(databaseURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse database URL: %w", err))
	}

	opt.MaxRetries = 3
	opt.PoolSize = dbMaxConns

	if dbMaxConnLifetime != "" {
		lifetime, err := time.ParseDuration(dbMaxConnLifetime)
		if err != nil {
			panic(fmt.Errorf("failed to parse DB_MAX_CONN_LIFETIME: %w", err))
		}
		opt.MaxConnAge = lifetime
	}

	cfg.Database = *opt

	return &cfg
}

func Get() *Config {
	return &cfg
}
