package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-pg/pg/v10"
)

const (
	StatusPublished = 1
)

var ErrNewsNotFound = errors.New("news not found")

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

// GetAllNews retrieves news with optional filtering by tagID and categoryID, with pagination
// Results are sorted by publishedAt DESC and include full category and tags information
// Content field is not included in the result (empty string)
func (r *Repository) GetAllNews(ctx context.Context, tagID, categoryID *int,
	page, pageSize int) ([]News, error) {

	r.log.Info("getting all news",
		"tagID", tagID,
		"categoryID", categoryID,
		"page", page,
		"pageSize", pageSize,
	)

	if page < 1 || pageSize < 1 {
		r.log.Error("invalid pagination parameters", "page", page, "pageSize", pageSize)
		return nil, fmt.Errorf(
			"page or pageSize must be greater than 0: page=%d, pageSize=%d",
			page, pageSize,
		)
	}

	offset := (page - 1) * pageSize

	now := time.Now()

	var news []News
	query := r.db.ModelContext(ctx, &news).
		Relation("Category").
		Where(`"t"."statusId" = ?`, StatusPublished).
		Where(`"category"."statusId" = ?`, StatusPublished).
		Where(`"t"."publishedAt" < ?`, now)

	if categoryID != nil {
		query = query.Where(`"t"."categoryId" = ?`, *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("t"."tagIds")`, *tagID)
	}

	err := query.
		OrderExpr(`"t"."publishedAt" DESC`).
		Limit(pageSize).
		Offset(offset).
		Select()

	if err != nil {
		r.log.Error("failed to query news", "error", err, "tagID",
			tagID, "categoryID", categoryID, "page", page, "pageSize", pageSize,
		)
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	r.log.Info("successfully retrieved news",
		"count", len(news),
		"tagID", tagID,
		"categoryID", categoryID,
		"page", page,
		"pageSize", pageSize,
	)

	return news, nil
}

func (r *Repository) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	r.log.Info("getting news count",
		"tagID", tagID,
		"categoryID", categoryID,
	)

	query := r.db.ModelContext(ctx, (*News)(nil))

	if categoryID != nil {
		query = query.Where(`"t"."categoryId" = ?`, *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("t"."tagIds")`, *tagID)
	}

	count, err := query.Count()
	if err != nil {
		r.log.Error("failed to get news count", "error", err, "tagID",
			tagID, "categoryID", categoryID,
		)
		return 0, fmt.Errorf("failed to get news count: %w", err)
	}

	r.log.Info("successfully retrieved news count",
		"count", count,
		"tagID", tagID,
		"categoryID", categoryID,
	)

	return count, nil
}

func (r *Repository) GetNewsByID(ctx context.Context, newsID int) (*News, error) {
	r.log.Info("getting news by ID", "newsID", newsID)
	now := time.Now()
	news := &News{}
	err := r.db.ModelContext(ctx, news).
		Relation("Category").
		Where(`"t"."statusId" = ?`, StatusPublished).
		Where(`"category"."statusId" = ?`, StatusPublished).
		Where(`"t"."publishedAt" < ?`, now).
		Where(`"t"."newsId" = ?`, newsID).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			r.log.Warn("news not found", "newsID", newsID)
			return nil, fmt.Errorf("get news by id %d: %w", newsID, ErrNewsNotFound)

		}
		r.log.Error("failed to get news by id", "error", err, "newsID", newsID)
		return nil, fmt.Errorf("failed to get news by id: %w", err)
	}

	r.log.Info("successfully retrieved news by ID", "newsID", newsID,
		"title", news.Title,
	)

	return news, nil
}

func (r *Repository) GetAllCategories(ctx context.Context) ([]Category, error) {
	r.log.Info("getting all categories")

	var category []Category
	err := r.db.ModelContext(ctx, &category).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"orderNumber" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query categories", "error", err)
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	r.log.Info("successfully retrieved categories", "count", len(category))

	return category, nil
}

func (r *Repository) GetAllTags(ctx context.Context) ([]Tag, error) {
	r.log.Info("getting all tags")

	var tags []Tag
	err := r.db.ModelContext(ctx, &tags).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query tags", "error", err)
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	r.log.Info("successfully retrieved tags", "count", len(tags))

	return tags, nil
}

func (r *Repository) GetTagsByIDs(ctx context.Context, tagIds []int32) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	r.log.Debug("getting tags by IDs", "tagIds", tagIds)

	tags := []Tag{}
	err := r.db.ModelContext(ctx, &tags).
		Where(`"tagId" IN (?)`, pg.In(tagIds)).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query tags by ids", "error", err, "tagIds", tagIds)
		return nil, fmt.Errorf("failed to query tags by ids: %w", err)
	}

	r.log.Debug("successfully retrieved tags by IDs", "count", len(tags), "tagIds", tagIds)

	return tags, nil
}
