package rpc

import (
	"log/slog"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
	middleware "github.com/vmkteam/zenrpc-middleware"
	"github.com/vmkteam/zenrpc/v2"
)

func New(logger *slog.Logger, newsManager *newsportal.Manager) *zenrpc.Server {

	rpcService := NewNewsService(newsManager)
	rpcServer := zenrpc.NewServer(zenrpc.Options{ExposeSMD: true})
	rpcServer.Register("news", rpcService)
	rpcServer.Use(middleware.WithSLog(logger.InfoContext, "news-portal", nil))

	return rpcServer
}
