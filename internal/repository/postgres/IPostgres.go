package postgres

import (
	"context"
	"log/slog"

	"github.com/daniilsolovey/news-portal/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IRepository interface {
	Close()
	Ping(ctx context.Context) error
	GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.News, error)
	GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error)
	GetNewsByID(ctx context.Context, newsID int) (*domain.News, error)
	GetAllCategories(ctx context.Context) ([]domain.Category, error)
	GetAllTags(ctx context.Context) ([]domain.Tag, error)
}

type Repository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func New(pool *pgxpool.Pool, logger *slog.Logger) *Repository {
	return &Repository{
		pool: pool,
		log:  logger,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	r.log.Info("pinging database")
	if err := r.pool.Ping(ctx); err != nil {
		r.log.Error("database ping failed", "error", err)
		return err
	}
	r.log.Info("database ping successful")
	return nil
}

func (r *Repository) Close() {
	r.log.Info("closing database connection pool")
	r.pool.Close()
	r.log.Info("database connection pool closed")
}
