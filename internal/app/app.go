package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rpc"
	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
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

func New(cfg Config, database db.DB, logger *slog.Logger) *App {
	// for rest api:	// handler := rest.NewNewsHandler(newsportal.NewNewsManager(database),logger,)
	newsManager := newsportal.NewNewsManager(database)
	rpcServer := rpc.New(logger, newsManager)

	a := &App{
		DB:     database,
		Logger: logger,
		Config: cfg,
	}

	a.setupRoutes(rpcServer)

	return a
}

func (a *App) Run(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	return a.Echo.Start(addr)
}

func (a *App) GracefulShutdown(ctx context.Context) error {
	a.Logger.Info("shutting down server")
	err := a.Echo.Shutdown(ctx)
	if err != nil {
		a.Logger.Error("failed to shutdown server", "error", err)
		return err
	}

	a.Logger.Info("server shutdown complete")
	return nil
}

func (a *App) setupRoutes(rpcServer *zenrpc.Server) {
	e := echo.New()

	e.Any("/rpc", echo.WrapHandler(rpcServer))

	e.Any("/doc/*", func(c echo.Context) error {
		zenrpc.SMDBoxHandler(c.Response().Writer, c.Request())
		return nil
	})

	e.Static("/static", "./frontend")

	e.GET("/*", a.handleFrontend)

	a.Echo = e
}

func (a *App) handleFrontend(c echo.Context) error {
	p := c.Request().URL.Path
	if p == "/" || p == "/index.html" {
		p = "index.html"
	}

	p = strings.TrimPrefix(p, "/")
	filePath := filepath.Join("./frontend", p)

	return c.File(filePath)
}
