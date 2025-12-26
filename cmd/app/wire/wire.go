//go:build wireinject
// +build wireinject

package wire

import (
	"log/slog"
	"net/http"

	postgres "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/google/wire"
)

type Service struct {
	Postgres *postgres.Repository
	Logger   *slog.Logger
	Engine   http.Handler
}

func Initialize() (*Service, func(), error) {
	wire.Build(
		ProvideLogger,
		ProvidePostgres,
		ProvideUseCase,
		ProvideHandler,
		ProvideEngine,
		wire.Struct(new(Service), "*"),
	)
	return nil, nil, nil
}
