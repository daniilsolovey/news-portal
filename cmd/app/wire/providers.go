package wire

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/daniilsolovey/news-portal/internal/delivery"
	"github.com/daniilsolovey/news-portal/internal/repository"
	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
	"github.com/daniilsolovey/news-portal/internal/usecase"
	"github.com/go-pg/pg/v10"
	"github.com/spf13/viper"
)

func ProvidePostgres(logger *slog.Logger) (*postgres.Repository, func(), error) {
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

	// Add query hook for SQL logging if enabled
	if viper.GetBool("DB_LOG_QUERIES") {
		queryHook := postgres.NewQueryHook(logger)
		db.AddQueryHook(queryHook)
		logger.Info("SQL query logging enabled")
	}

	// Test connection
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

func ProvideRepository(pg *postgres.Repository) repository.IRepository {
	return repository.New(pg)
}

func ProvideUseCase(repo repository.IRepository, logger *slog.Logger) *usecase.NewsUseCase {
	return usecase.NewNewsUseCase(repo, logger)
}

func ProvideHandler(uc *usecase.NewsUseCase, logger *slog.Logger) *delivery.NewsHandler {
	return delivery.NewNewsHandler(uc, logger)
}

func ProvideEngine(handler *delivery.NewsHandler) http.Handler {
	return handler.RegisterRoutes()
}
