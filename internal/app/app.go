package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/daniilsolovey/news-portal/internal/rpc"
	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/zenrpc/v2"
)

type App struct {
	DB      db.DB
	Logger  *slog.Logger
	Server  *http.Server
	Handler http.Handler
	Config  Config
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
	a.Server = &http.Server{
		Addr:    addr,
		Handler: a.Handler,
	}
	return a.Server.ListenAndServe()
}

func (a *App) GracefulShutdown(ctx context.Context) error {
	err := a.Server.Shutdown(ctx)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (a *App) setupRoutes(rpcServer *zenrpc.Server) {
	mux := http.NewServeMux()

	mux.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) { rpcServer.ServeHTTP(w, r) })
	mux.HandleFunc("/doc/", zenrpc.SMDBoxHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./frontend"))))
	mux.HandleFunc("/", a.handleFrontend)

	a.Handler = mux
}

func (a *App) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p := r.URL.Path
	if p == "/" || p == "/index.html" {
		p = "index.html"
	}

	p = strings.TrimPrefix(p, "/")
	filePath := filepath.Join("./frontend", p)

	http.ServeFile(w, r, filePath)
}
