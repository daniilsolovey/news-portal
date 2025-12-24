package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-pg/pg/v10"
)

// QueryHook implements pg.QueryHook interface for logging SQL queries
type QueryHook struct {
	logger *slog.Logger
}

// NewQueryHook creates a new QueryHook instance
func NewQueryHook(logger *slog.Logger) *QueryHook {
	return &QueryHook{
		logger: logger,
	}
}

// BeforeQuery is called before executing a query
func (h *QueryHook) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

// AfterQuery is called after executing a query
func (h *QueryHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	query, err := event.FormattedQuery()
	if err != nil {
		h.logger.Error("failed to format query", "error", err)
		return nil
	}

	// Log query with duration
	duration := time.Since(event.StartTime)
	h.logger.Info("SQL query executed",
		"query", query,
		"duration", duration,
		"error", event.Err,
	)

	return nil
}
