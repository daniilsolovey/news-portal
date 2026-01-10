package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/daniilsolovey/news-portal/docs"
	"github.com/daniilsolovey/news-portal/internal/app"
	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rpc"
	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
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

	dbConnection := initializeDatabase()
	defer dbConnection.Close()

	ctx := context.Background()

	rpcServer := initializeRPCServer(dbConnection)

	service := setupRoutes(dbConnection, rpcServer)

	runServer(ctx, service)
}

func loadConfig() {
	_, err := toml.DecodeFile(*flConfig, &cfg)
	exitOnError(err)
}

func initializeDatabase() *pg.DB {
	dbConnection := pg.Connect(&cfg.Database)
	err := dbConnection.Ping(context.Background())
	exitOnError(err)
	return dbConnection
}

func initializeRPCServer(dbConnection *pg.DB) *zenrpc.Server {
	database := db.New(dbConnection)
	newsManager := newsportal.NewNewsManager(database)
	rpcService := rpc.NewNewsService(newsManager)

	rpcServer := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true})
	rpcServer.Register("news", *rpcService)
	rpcServer.Register("", *rpcService) // public
	rpcServer.Use(zenrpc.Logger(log.New(os.Stderr, "", log.LstdFlags)))

	return rpcServer
}

func setupRoutes(dbConnection *pg.DB, rpcServer *zenrpc.Server) *app.App {
	service := app.New(cfg, dbConnection, lg)

	service.Echo.Any("/rpc", echo.WrapHandler(rpcServer))
	service.Echo.Any("/doc/", echo.WrapHandler(http.HandlerFunc(zenrpc.SMDBoxHandler)))

	// Frontend static files
	service.Echo.Static("/static", "./frontend")
	service.Echo.GET("/*", handleFrontend)

	return service
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

func handleFrontend(c echo.Context) error {
	if c.Request().Method != http.MethodGet {
		return echo.ErrMethodNotAllowed
	}

	p := c.Request().URL.Path
	if p == "/" || p == "/index.html" {
		p = "index.html"
	}

	p = strings.TrimPrefix(p, "/")
	filePath := filepath.Join("./frontend", p)

	return c.File(filePath)
}
