package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/daniilsolovey/news-portal/internal/domain"
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
	page, pageSize int) ([]domain.News, error) {

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

	var newsEntities []News
	query := r.db.ModelContext(ctx, &newsEntities).
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
		r.log.Error("failed to query news", "error", err, "tagID", tagID, "categoryID", categoryID, "page", page, "pageSize", pageSize)
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	// Convert to domain models and load tags
	newsList := make([]domain.News, 0, len(newsEntities))

	newsList, err = r.attachTagsBatch(ctx, newsEntities)
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
		r.log.Error("failed to get news count", "error", err, "tagID", tagID, "categoryID", categoryID)
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
func (r *Repository) GetNewsByID(ctx context.Context, newsID int) (*domain.News, error) {
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

	// Convert to domain model
	news := newsEntity.toDomain()
	loadTags, err := r.loadTags(ctx, newsEntity.TagIds)
	if err != nil {
		r.log.Error("failed to load tags", "error", err)
		return nil, fmt.Errorf("failed to load tags: %w", err)
	}

	news.Tags = loadTags
	r.log.Info("successfully retrieved news by ID", "newsID", newsID, "title", news.Title)

	return &news, nil
}

// GetAllCategories retrieves all categories ordered by orderNumber
func (r *Repository) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	r.log.Info("getting all categories")

	var categoryEntities []Category
	err := r.db.ModelContext(ctx, &categoryEntities).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"orderNumber" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query categories", "error", err)
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}

	// Convert to domain models
	categories := make([]domain.Category, 0, len(categoryEntities))
	for i := range categoryEntities {
		categories = append(categories, categoryEntities[i].toDomain())
	}

	r.log.Info("successfully retrieved categories", "count", len(categories))

	return categories, nil
}

// GetAllTags retrieves all tags ordered by title
func (r *Repository) GetAllTags(ctx context.Context) ([]domain.Tag, error) {
	r.log.Info("getting all tags")

	var tagEntities []Tag
	err := r.db.ModelContext(ctx, &tagEntities).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query tags", "error", err)
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}

	// Convert to domain models
	tags := make([]domain.Tag, 0, len(tagEntities))
	for i := range tagEntities {
		tags = append(tags, tagEntities[i].toDomain())
	}

	r.log.Info("successfully retrieved tags", "count", len(tags))

	return tags, nil
}

// getTagsByIDs retrieves tags by their IDs
func (r *Repository) getTagsByIDs(ctx context.Context, tagIds []int32) ([]domain.Tag, error) {
	if len(tagIds) == 0 {
		return []domain.Tag{}, nil
	}

	r.log.Debug("getting tags by IDs", "tagIds", tagIds)

	var tagEntities []Tag
	err := r.db.ModelContext(ctx, &tagEntities).
		Where(`"tagId" IN (?)`, pg.In(tagIds)).
		Where(`"statusId" = ?`, StatusPublished).
		OrderExpr(`"title" ASC`).
		Select()

	if err != nil {
		r.log.Error("failed to query tags by ids", "error", err, "tagIds", tagIds)
		return nil, fmt.Errorf("failed to query tags by ids: %w", err)
	}

	// Convert to domain models
	tags := make([]domain.Tag, 0, len(tagEntities))
	for i := range tagEntities {
		tags = append(tags, tagEntities[i].toDomain())
	}

	r.log.Debug("successfully retrieved tags by IDs", "count", len(tags), "tagIds", tagIds)

	return tags, nil
}

// helpers-------------------------------------------------------------------------------------

// attachTagsBatch loads all tags for the given news slice in one query and attaches them back
// preserving the order from each News.TagIds.
func (r *Repository) attachTagsBatch(ctx context.Context, newsEntities []News) ([]domain.News, error) {
	newsList := make([]domain.News, 0, len(newsEntities))

	// Collect tag IDs across the page (set) + keep per-news tagIds to preserve order.
	tagSet := make(map[int32]struct{})
	newsTagIDs := make([][]int32, len(newsEntities))

	for i := range newsEntities {
		ids := newsEntities[i].TagIds
		newsTagIDs[i] = ids
		for _, id := range ids {
			tagSet[id] = struct{}{}
		}
	}

	// Flatten set to slice for IN query.
	allTagIDs := make([]int32, 0, len(tagSet))
	for id := range tagSet {
		allTagIDs = append(allTagIDs, id)
	}

	// Fetch all tags once and index by ID.
	tagsByID := make(map[int32]domain.Tag)
	if len(allTagIDs) > 0 {
		tags, err := r.getTagsByIDs(ctx, allTagIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}

		tagsByID = make(map[int32]domain.Tag, len(tags))
		for _, t := range tags {
			tagsByID[int32(t.TagID)] = t // adjust if TagID is already int32
		}
	}

	// Convert to domain and attach tags back (preserving tagIds order).
	for i := range newsEntities {
		news := newsEntities[i].toDomain()

		ids := newsTagIDs[i]
		if len(ids) == 0 {
			news.Tags = []domain.Tag{}
			newsList = append(newsList, news)
			continue
		}

		tags := make([]domain.Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[id]; ok {
				tags = append(tags, t)
			}
		}
		news.Tags = tags

		newsList = append(newsList, news)
	}

	return newsList, nil
}

func (r *Repository) loadTags(ctx context.Context, tagIDs []int32) ([]domain.Tag, error) {
	if len(tagIDs) == 0 {
		return []domain.Tag{}, nil
	}

	tags, err := r.getTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}
	return tags, nil
}
