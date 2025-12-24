package postgres

import (
	"context"
	"log/slog"

	"github.com/go-pg/pg/v10"
)

type IRepository interface {
	Close() error
	Ping(ctx context.Context) error
	GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]News, error)
	GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error)
	GetNewsByID(ctx context.Context, newsID int) (*News, error)
	GetAllCategories(ctx context.Context) ([]Category, error)
	GetAllTags(ctx context.Context) ([]Tag, error)
}

type Repository struct {
	db  *pg.DB
	log *slog.Logger
}

// New creates a new Repository with a go-pg DB connection
func New(db *pg.DB, logger *slog.Logger) *Repository {
	return &Repository{
		db:  db,
		log: logger,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	r.log.Info("pinging database")
	if err := r.db.Ping(ctx); err != nil {
		r.log.Error("database ping failed", "error", err)
		return err
	}
	r.log.Info("database ping successful")
	return nil
}

func (r *Repository) Close() error {
	r.log.Info("closing database connection pool")
	if err := r.db.Close(); err != nil {
		r.log.Error("error closing database connection", "error", err)
		return err
	}
	r.log.Info("database connection pool closed")
	return nil
}
