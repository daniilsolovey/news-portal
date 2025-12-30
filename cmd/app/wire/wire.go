//go:build wireinject
// +build wireinject

package wire

import (
	"log/slog"

	postgres "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/google/wire"
	"github.com/labstack/echo/v4"
)

type Service struct {
	Postgres *postgres.Repository
	Logger   *slog.Logger
	Engine   *echo.Echo
}

func Initialize() (*Service, func(), error) {
	wire.Build(
		ProvideLogger,
		ProvideDB,
		ProvideNewsPortal,
		ProvideHandler,
		ProvideEngine,
		wire.Struct(new(Service), "*"),
	)
	return nil, nil, nil
}
