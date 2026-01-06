package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-pg/pg/v10"
	"github.com/namsral/flag"

	"github.com/daniilsolovey/news-portal/config"
	_ "github.com/daniilsolovey/news-portal/docs"
	"github.com/daniilsolovey/news-portal/internal/app"
)

var (
	flConfig = flag.String("config", "config.toml", "path to TOML configuration file")
	flDebug  = flag.Bool("debug", false, "enable debug mode")
	cfg      config.Config
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

	_, err := toml.DecodeFile(*flConfig, &cfg)
	if err != nil {
		exitOnError(err)
	}

	db := pg.Connect(&cfg.Database)
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		exitOnError(err)
	}

	service := app.New(&cfg, db, lg)
	ctx := context.Background()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		err := service.Run(ctx, cfg.App.Port)
		if err != nil {
			lg.Error("service run failed", "error", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	lg.Info("service stopping")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = service.GracefulShutdown(shutdownCtx)
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
