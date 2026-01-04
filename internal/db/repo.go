package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
)

const (
	StatusPublished = 1
)

type Repository struct {
	db pg.DBI
}

func New(db pg.DBI) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	if db, ok := r.db.(*pg.DB); ok {
		if err := db.Ping(ctx); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func (r *Repository) Close() error {
	if db, ok := r.db.(*pg.DB); ok {
		if err := db.Close(); err != nil {
			return err
		}
		return nil
	}

	return nil
}

// News retrieves news with optional filtering by tagID and categoryID, with pagination
// Results are sorted by publishedAt DESC and include full category and tags information
// Content field is not included in the result (empty string)
func (r *Repository) News(ctx context.Context, tagID, categoryID *int,
	page, pageSize int) ([]News, error) {

	if page < 1 || pageSize < 1 {
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
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	return news, nil
}

func (r *Repository) NewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	query := r.db.ModelContext(ctx, (*News)(nil))

	if categoryID != nil {
		query = query.Where(`"t"."categoryId" = ?`, *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("t"."tagIds")`, *tagID)
	}

	count, err := query.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to get news count: %w", err)
	}

	return count, nil
}

func (r *Repository) NewsByID(ctx context.Context, newsID int) (*News, error) {
	now := time.Now()
	news := &News{}
	err := r.db.ModelContext(ctx, news).
		Relation("Category").
		Where(`"t"."statusId" = ?`, StatusPublished).
		Where(`"category"."statusId" = ?`, StatusPublished).
		Where(`"t"."publishedAt" < ?`, now).
		Where(`"t"."newsId" = ?`, newsID).
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get news by id: %w", err)
	}

	return news, nil
}

func (r *Repository) Categories(ctx context.Context) ([]Category, error) {
	var category []Category
	err := r.db.ModelContext(ctx, &category).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"orderNumber" ASC`).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	return category, nil
}

func (r *Repository) Tags(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	err := r.db.ModelContext(ctx, &tags).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	return tags, nil
}

func (r *Repository) TagsByIDs(ctx context.Context, tagIds []int32) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	tags := []Tag{}
	err := r.db.ModelContext(ctx, &tags).
		Where(`"tagId" IN (?)`, pg.In(tagIds)).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query tags by ids: %w", err)
	}

	return tags, nil
}
