package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daniilsolovey/news-portal/configs"
	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rest"
	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
)

type App struct {
	DB     *db.Repository
	Logger *slog.Logger
	Echo   *echo.Echo
}

func NewApp(cfg *configs.Config) (*App, func(), error) {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo},
		),
	)

	dbConnect := pg.Connect(&cfg.Database)

	ctx := context.Background()
	if err := dbConnect.Ping(ctx); err != nil {
		dbConnect.Close()
		return nil, nil, fmt.Errorf("database not available: %w", err)
	}

	repo := db.New(dbConnect)

	newsManager := newsportal.NewNewsManager(repo)

	handler := rest.NewNewsHandler(newsManager, logger)

	echo := handler.RegisterRoutes()

	cleanup := func() {
		if err := repo.Close(); err != nil {
			logger.Error("error closing database connection", "error", err)
		}
	}

	return &App{
		DB:     repo,
		Logger: logger,
		Echo:   echo,
	}, cleanup, nil
}

func (a *App) Run(ctx context.Context, port int) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	addr := fmt.Sprintf(":%d", port)
	go func() {
		a.Logger.Info("HTTP server started", "port", port)
		if err := a.Echo.Start(addr); err != nil &&
			err != http.ErrServerClosed {
			a.Logger.Error("HTTP server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	a.Logger.Info("service stopping")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.GracefulShutdown(shutdownCtx); err != nil {
		a.Logger.Error("server forced to shutdown", "err", err)
		return err
	}

	return nil
}

func (a *App) GracefulShutdown(ctx context.Context) error {
	return a.Echo.Shutdown(ctx)
}
