package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/daniilsolovey/news-portal/docs"
	"github.com/daniilsolovey/news-portal/internal/app"
	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/go-pg/pg/v10"
)

var (
	flConfig = flag.String("config", "config.toml", "path to TOML configuration file")
	flDebug  = flag.Bool("debug", false, "enable debug mode")
	cfg      app.Config
	lg       *slog.Logger
)

// @title News Portal API
// @version 1.0
// @description Template API for news portal
// @host localhost:3000
// @BasePath /

func main() {
	flag.Parse()

	lg = newLogger(*flDebug)
	loadConfig()
	ctx := context.Background()

	dbConnection := pg.Connect(&cfg.Database)
	err := dbConnection.Ping(ctx)
	exitOnError(err)
	defer dbConnection.Close()
	database := db.New(dbConnection)

	service := app.New(cfg, database, lg)
	runServer(ctx, service)
}

func loadConfig() {
	_, err := toml.DecodeFile(*flConfig, &cfg)
	exitOnError(err)
}

func runServer(ctx context.Context, service *app.App) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		err := service.Run(ctx, cfg.App.Port)
		if err != nil && err != http.ErrServerClosed {
			lg.Error("service run failed", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	lg.Info("service started", "port", cfg.App.Port)

	<-quit
	lg.Info("service stopping")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := service.GracefulShutdown(shutdownCtx)
	if err != nil {
		lg.Error("service graceful shutdown failed", "error", err)
	}
}

func newLogger(debug bool) *slog.Logger {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

func exitOnError(err error) {
	if err != nil {
		lg.Error("app init failed", "error", err)
		os.Exit(1)
	}
}
