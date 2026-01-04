package main

import (
	"context"
	"fmt"
	"os"

	"github.com/daniilsolovey/news-portal/config"
	_ "github.com/daniilsolovey/news-portal/docs"
	"github.com/daniilsolovey/news-portal/internal/app"
)

// @title News Portal API
// @version 1.0
// @description Template API for news portal
// @host localhost:3000
// @BasePath /

func main() {
	cfg, err := config.Init()
	if err != nil {
		panic(fmt.Errorf("failed to initialize config: %w", err))
	}

	service, cleanup, err := app.NewApp(cfg)
	if err != nil {
		panic(err)
	}

	defer cleanup()

	ctx := context.Background()

	if err := service.Run(ctx, cfg.Port); err != nil {
		service.Logger.Error("failed to run server", "error", err)
		os.Exit(1)
	}
}
