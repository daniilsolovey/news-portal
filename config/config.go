package config

import (
	"github.com/go-pg/pg/v10"
)

type Config struct {
	Database pg.Options
	App      struct {
		Host string
		Port int
	}
}
