package configs

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

func Init() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = "envs/.env.dev"
	}

	viper.SetConfigFile(envFile)
	viper.SetConfigType("env")

	// load .env file if exist
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config: no config file loaded (%v), using ENV or defaults", err)
	}

	// read environment variables (if file not exist)
	viper.AutomaticEnv()

	// default values if file not exist and environment variables not set
	viper.SetDefault("SERVICE_NAME", "news-portal")
	viper.SetDefault("HTTP_PORT", "3000")
	viper.SetDefault(
		"DATABASE_URL", "postgres://user:password@localhost:5432/news_portal?sslmode=disable")
	viper.SetDefault("DB_MAX_CONNS", 5)
	viper.SetDefault("DB_MAX_CONN_LIFETIME", "300s")
}
