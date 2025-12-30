package wire

import (
	"context"
	"log/slog"
	"os"
	"time"

	postgres "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rest"
	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func ProvideDB(logger *slog.Logger) (*postgres.Repository, func(), error) {
	url := viper.GetString("DATABASE_URL")

	opt, err := pg.ParseURL(url)
	if err != nil {
		logger.Error("failed to parse database URL", "error", err)
		return nil, nil, err
	}

	opt.MaxRetries = 3
	opt.PoolSize = viper.GetInt("DB_MAX_CONNS")

	lifetimeStr := viper.GetString("DB_MAX_CONN_LIFETIME")
	if lifetimeStr != "" {
		lifetime, err := time.ParseDuration(lifetimeStr)
		if err != nil {
			logger.Error("failed to parse DB_MAX_CONN_LIFETIME", "error", err, "value", lifetimeStr)
			return nil, nil, err
		}
		opt.MaxConnAge = lifetime
	}

	db := pg.Connect(opt)

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		db.Close()
		return nil, nil, err
	}

	repo := postgres.New(db, logger)
	cleanup := func() {
		if err := repo.Close(); err != nil {
			logger.Error("error closing database connection", "error", err)
		}
	}

	return repo, cleanup, nil
}

func ProvideLogger() *slog.Logger {
	return slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo},
		),
	)
}

func ProvideNewsPortal(repo *postgres.Repository, logger *slog.Logger) *newsportal.Manager {
	return newsportal.NewNewsUseCase(repo, logger)
}

func ProvideHandler(uc *newsportal.Manager, logger *slog.Logger) *rest.NewsHandler {
	return rest.NewNewsHandler(uc, logger)
}

func ProvideEngine(handler *rest.NewsHandler) *echo.Echo {
	return handler.RegisterRoutes()
}
