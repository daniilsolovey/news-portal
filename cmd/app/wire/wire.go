//go:build wireinject
// +build wireinject

package wire

import (
	"log/slog"

	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

type Service struct {
	Postgres *postgres.Repository
	Logger   *slog.Logger
	Engine   *gin.Engine
}

func Initialize() (*Service, func(), error) {
	wire.Build(
		ProvideLogger,
		ProvidePostgres,
		ProvideRepository,
		ProvideUseCase,
		ProvideHandler,
		ProvideEngine,
		wire.Struct(new(Service), "*"),
	)
	return nil, nil, nil
}
