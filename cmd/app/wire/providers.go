package wire

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/daniilsolovey/news-portal/internal/delivery"
	"github.com/daniilsolovey/news-portal/internal/repository"
	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
	"github.com/daniilsolovey/news-portal/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

func ProvidePostgres(logger *slog.Logger) (*postgres.Repository, error) {
	ctx := context.Background()
	url := viper.GetString("DATABASE_URL")

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		logger.Error("failed to parse database URL", "error", err)
		return nil, err
	}

	cfg.MaxConns = int32(viper.GetInt("DB_MAX_CONNS"))
	lifetimeStr := viper.GetString("DB_MAX_CONN_LIFETIME")
	lifetime, err := time.ParseDuration(lifetimeStr)
	if err != nil {
		logger.Error("failed to parse DB_MAX_CONN_LIFETIME", "error", err, "value", lifetimeStr)
		return nil, err
	}
	cfg.MaxConnLifetime = lifetime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Error("failed to create postgres connection pool", "error", err)
		return nil, err
	}

	return postgres.New(pool, logger), nil
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

func ProvideUseCase(repo repository.IRepository, logger *slog.Logger) *usecase.TemplateUseCase {
	return usecase.NewTemplateUseCase(repo, logger)
}

func ProvideHandler(uc *usecase.TemplateUseCase, logger *slog.Logger) *delivery.TemplateHandler {
	return delivery.NewTemplateHandler(uc, logger)
}

func ProvideEngine(handler *delivery.TemplateHandler) *gin.Engine {
	return handler.RegisterRoutes()
}
