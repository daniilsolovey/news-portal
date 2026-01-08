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

func applyPublishedNewsFilters(query *pg.Query) *pg.Query {
	now := time.Now()
	return query.
		Where(`"t".? = ?`, pg.Ident(Columns.News.StatusID), StatusPublished).
		Where(`"category".? = ?`, pg.Ident(Columns.Category.StatusID), StatusPublished).
		Where(`"t".? < ?`, pg.Ident(Columns.News.PublishedAt), now)
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

	var news []News
	query := r.db.ModelContext(ctx, &news).
		Relation(Columns.News.Category)
	query = applyPublishedNewsFilters(query)

	if categoryID != nil {
		query = query.Where(`"t".? = ?`, pg.Ident(Columns.News.CategoryID), *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("t".?)`, *tagID, pg.Ident(Columns.News.TagIDs))
	}

	err := query.
		OrderExpr(`"t".? DESC`, pg.Ident(Columns.News.PublishedAt)).
		Limit(pageSize).
		Offset(offset).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	return news, nil
}

func (r *Repository) NewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	var news []News

	query := r.db.ModelContext(ctx, &news).
		Relation(Columns.News.Category)
	query = applyPublishedNewsFilters(query)

	if categoryID != nil {
		query = query.Where(`"t".? = ?`, pg.Ident(Columns.News.CategoryID), *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("t".?)`, *tagID, pg.Ident(Columns.News.TagIDs))
	}

	count, err := query.Count()
	if err != nil {
		return 0, fmt.Errorf("failed to get news count: %w", err)
	}

	return count, nil
}

func (r *Repository) NewsByID(ctx context.Context, newsID int) (*News, error) {
	news := &News{}
	query := r.db.ModelContext(ctx, news).
		Relation("Category")
	query = applyPublishedNewsFilters(query)
	err := query.
		Where(`"t".? = ?`, pg.Ident(Columns.News.ID), newsID).
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
		Where(`"t".? = ?`, pg.Ident(Columns.Category.StatusID), StatusPublished).
		OrderExpr(`"t".? ASC`, pg.Ident(Columns.Category.OrderNumber)).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	return category, nil
}

func (r *Repository) Tags(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	err := r.db.ModelContext(ctx, &tags).
		Where(`"t".? = ?`, pg.Ident(Columns.Tag.StatusID), StatusPublished).
		OrderExpr(`"t".? ASC`, pg.Ident(Columns.Tag.Title)).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	return tags, nil
}

func (r *Repository) TagsByIDs(ctx context.Context, tagIds []int) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	tags := []Tag{}
	err := r.db.ModelContext(ctx, &tags).
		Where(`"t".? IN (?)`, pg.Ident(Columns.Tag.ID), pg.In(tagIds)).
		Where(`"t".? = ?`, pg.Ident(Columns.Tag.StatusID), StatusPublished).
		OrderExpr(`"t".? ASC`, pg.Ident(Columns.Tag.Title)).
		Select()

	if err != nil {
		return nil, fmt.Errorf("failed to query tags by ids: %w", err)
	}

	return tags, nil
}
