package postgres

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

var ErrNewsNotFound = errors.New("news not found")

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
		Where(`"news"."statusId" = ?`, StatusPublished).
		Where(`"category"."statusId" = ?`, StatusPublished).
		Where(`"news"."publishedAt" < ?`, now)

	if categoryID != nil {
		query = query.Where(`"news"."categoryId" = ?`, *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("news"."tagIds")`, *tagID)
	}

	err := query.
		OrderExpr(`"news"."publishedAt" DESC`).
		Limit(pageSize).
		Offset(offset).
		Select()

	if err != nil {
		r.log.Error("failed to query news", "error", err, "tagID",
			tagID, "categoryID", categoryID, "page", page, "pageSize", pageSize,
		)
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	newsList, err := r.attachTagsBatch(ctx, news)
	if err != nil {
		r.log.Error("failed to attach tags to news", "error", err)
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	r.log.Info("successfully retrieved news",
		"count", len(newsList),
		"tagID", tagID,
		"categoryID", categoryID,
		"page", page,
		"pageSize", pageSize,
	)

	return newsList, nil
}

// GetNewsCount returns the count of news matching the optional tagID and categoryID filters
func (r *Repository) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	r.log.Info("getting news count",
		"tagID", tagID,
		"categoryID", categoryID,
	)

	query := r.db.ModelContext(ctx, (*News)(nil))

	if categoryID != nil {
		query = query.Where(`"categoryId" = ?`, *categoryID)
	}

	if tagID != nil {
		query = query.Where(`? = ANY("tagIds")`, *tagID)
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

// GetNewsByID retrieves a single news item by ID with full content, category and tags
func (r *Repository) GetNewsByID(ctx context.Context, newsID int) (*News, error) {
	r.log.Info("getting news by ID", "newsID", newsID)
	now := time.Now()
	newsEntity := &News{NewsID: newsID}
	err := r.db.ModelContext(ctx, newsEntity).
		Relation("Category").
		Where(`"news"."statusId" = ?`, StatusPublished).
		Where(`"category"."statusId" = ?`, StatusPublished).
		Where(`"news"."publishedAt" < ?`, now).
		WherePK().
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			r.log.Warn("news not found", "newsID", newsID)
			return nil, fmt.Errorf("get news by id %d: %w", newsID, ErrNewsNotFound)

		}
		r.log.Error("failed to get news by id", "error", err, "newsID", newsID)
		return nil, fmt.Errorf("failed to get news by id: %w", err)
	}

	// Load tags
	loadTags, err := r.loadTags(ctx, newsEntity.TagIds)
	if err != nil {
		r.log.Error("failed to load tags", "error", err)
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	// Attach tags to news entity
	newsEntity.Tags = loadTags
	r.log.Info("successfully retrieved news by ID", "newsID", newsID,
		"title", newsEntity.Title,
	)

	return newsEntity, nil
}

// GetAllCategories retrieves all categories ordered by orderNumber
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

// GetAllTags retrieves all tags ordered by title
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

// getTagsByIDs retrieves tags by their IDs
func (r *Repository) getTagsByIDs(ctx context.Context, tagIds []int32) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	r.log.Debug("getting tags by IDs", "tagIds", tagIds)

	var tags []Tag
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
