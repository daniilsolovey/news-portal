package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-pg/pg/v10"
)

// QueryHook implements pg.QueryHook interface for logging SQL queries. TODO: remove this functionality
type QueryHook struct {
	logger *slog.Logger
}

func NewQueryHook(logger *slog.Logger) *QueryHook {
	return &QueryHook{
		logger: logger,
	}
}

func (h *QueryHook) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

func (h *QueryHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	query, err := event.FormattedQuery()
	if err != nil {
		h.logger.Error("failed to format query", "error", err)
		return nil
	}

	duration := time.Since(event.StartTime)
	h.logger.Info("SQL query executed",
		"query", query,
		"duration", duration,
		"error", event.Err,
	)

	return nil
}
