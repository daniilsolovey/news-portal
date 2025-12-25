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
	db  pg.DBI
	log *slog.Logger
}

func New(db pg.DBI, logger *slog.Logger) *Repository {
	return &Repository{
		db:  db,
		log: logger,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	r.log.Info("pinging database")
	if db, ok := r.db.(*pg.DB); ok {
		if err := db.Ping(ctx); err != nil {
			r.log.Error("database ping failed", "error", err)
			return err
		}
		r.log.Info("database ping successful")
		return nil
	}

	return nil
}

func (r *Repository) Close() error {
	if db, ok := r.db.(*pg.DB); ok {
		r.log.Info("closing database connection pool")
		if err := db.Close(); err != nil {
			r.log.Error("error closing database connection", "error", err)
			return err
		}
		r.log.Info("database connection pool closed")
		return nil
	}

	return nil
}
