package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/daniilsolovey/news-portal/cmd/app/wire"
	"github.com/daniilsolovey/news-portal/configs"
	_ "github.com/daniilsolovey/news-portal/docs"
	"github.com/spf13/viper"
)

// @title News Portal API
// @version 1.0
// @description Template API for news portal
// @host localhost:3000
// @BasePath /

func init() {
	configs.Init()
}

func main() {
	service, cleanup, err := wire.Initialize()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	ctx := context.Background()

	// Check connection PostgreSQL
	if err := service.Postgres.Ping(ctx); err != nil {
		service.Logger.Error("PostgreSQL not available", "error", err)
		os.Exit(1)
	}

	engine := service.Engine
	port := viper.GetInt("HTTP_PORT")

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: engine,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Run HTTP-server
	go func() {
		service.Logger.Info("HTTP server started", "port", port)
		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			service.Logger.Error("HTTP server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	service.Logger.Info("service stopping")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		service.Logger.Error("server forced to shutdown", "err", err)
	}
}
