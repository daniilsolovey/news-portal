package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rest"
	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
)

type App struct {
	DB     db.DB
	Logger *slog.Logger
	Echo   *echo.Echo
	Config Config
}

type Config struct {
	Database pg.Options
	App      struct {
		Host string
		Port int
	}
}

func New(cfg Config, dbConnect *pg.DB, logger *slog.Logger) *App {
	database := db.New(dbConnect)
	handler := rest.NewNewsHandler(
		newsportal.NewNewsManager(database),
		logger,
	)

	return &App{
		DB:     database,
		Logger: logger,
		Echo:   handler.RegisterRoutes(),
		Config: cfg,
	}
}

func (a *App) Run(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	return a.Echo.Start(addr)
}

func (a *App) GracefulShutdown(ctx context.Context) error {
	err := a.Echo.Shutdown(ctx)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
