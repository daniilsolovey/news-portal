package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/daniilsolovey/news-portal/internal/domain"
	"github.com/jackc/pgx/v5"
)

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
	// may interfere with the use of indexes because:
	// request plan one and condition always contains OR
	query := `
		SELECT 
			n."newsId",
			n."categoryId",
			n."title",
			n."content",
			n."author",
			n."publishedAt",
			n."updatedAt",
			n."statusId",
			n."tagIds",
			c."categoryId" as category_categoryId,
			c."title" as category_title,
			c."orderNumber" as category_orderNumber,
			c."statusId" as category_statusId
		FROM "news" n
		JOIN "categories" c ON n."categoryId" = c."categoryId"
		WHERE 
			($1::int IS NULL OR n."categoryId" = $1::int)
			AND ($2::int IS NULL OR $2::int = ANY(n."tagIds"))
		ORDER BY n."publishedAt" DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, categoryID, tagID, pageSize, offset)
	if err != nil {
		r.log.Error("failed to query news", "error", err, "tagID", tagID, "categoryID", categoryID, "page", page, "pageSize", pageSize)
		return nil, fmt.Errorf("failed to query news: %w", err)
	}

	defer rows.Close()

	var newsList []domain.News
	for rows.Next() {
		var news domain.News
		var updatedAt sql.NullTime
		var tagIds []int32

		err := rows.Scan(
			&news.NewsID,
			&news.CategoryID,
			&news.Title,
			&news.Content,
			&news.Author,
			&news.PublishedAt,
			&updatedAt,
			&news.StatusID,
			&tagIds,
			&news.Category.CategoryID,
			&news.Category.Title,
			&news.Category.OrderNumber,
			&news.Category.StatusID,
		)
		if err != nil {
			r.log.Error("failed to scan news row", "error", err, "newsID", news.NewsID)
			return nil, fmt.Errorf("failed to scan news row: %w", err)
		}

		// Get tags information by tagIds
		// TODO: transaction tx
		if len(tagIds) > 0 {
			tags, err := r.getTagsByIDs(ctx, tagIds)
			if err != nil {
				r.log.Error("failed to get tags for news", "error", err, "newsID", news.NewsID, "tagIds", tagIds)
				return nil, fmt.Errorf("failed to get tags: %w", err)
			}
			news.Tags = tags
		} else {
			news.Tags = []domain.Tag{}
		}

		newsList = append(newsList, news)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("error iterating news rows", "error", err)
		return nil, fmt.Errorf("error iterating news rows: %w", err)
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

	// may interfere with the use of indexes because:
	// request plan one and condition always contains OR
	query := `
		SELECT COUNT(*) as count
		FROM "news" n
		WHERE 
			($1::int IS NULL OR n."categoryId" = $1::int)
			AND ($2::int IS NULL OR $2::int = ANY(n."tagIds"))
	`

	var count int
	err := r.pool.QueryRow(ctx, query, categoryID, tagID).Scan(&count)
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

	query := `
		SELECT 
			n."newsId",
			n."categoryId",
			n."title",
			n."content",
			n."author",
			n."publishedAt",
			n."updatedAt",
			n."statusId",
			n."tagIds",
			c."categoryId" as category_categoryId,
			c."title" as category_title,
			c."orderNumber" as category_orderNumber,
			c."statusId" as category_statusId
		FROM "news" n
		JOIN "categories" c ON n."categoryId" = c."categoryId"
		WHERE n."newsId" = $1
	`

	var news domain.News
	var updatedAt sql.NullTime
	var tagIds []int32

	err := r.pool.QueryRow(ctx, query, newsID).Scan(
		&news.NewsID,
		&news.CategoryID,
		&news.Title,
		&news.Content,
		&news.Author,
		&news.PublishedAt,
		&updatedAt,
		&news.StatusID,
		&tagIds,
		&news.Category.CategoryID,
		&news.Category.Title,
		&news.Category.OrderNumber,
		&news.Category.StatusID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			r.log.Warn("news not found", "newsID", newsID)
			return nil, fmt.Errorf("news with id %d not found", newsID)
		}
		r.log.Error("failed to get news by id", "error", err, "newsID", newsID)
		return nil, fmt.Errorf("failed to get news by id: %w", err)
	}

	if updatedAt.Valid {
		news.UpdatedAt = &updatedAt.Time
	}

	// Get tags information by tagIds
	// TODO: transaction tx
	if len(tagIds) > 0 {
		tags, err := r.getTagsByIDs(ctx, tagIds)
		if err != nil {
			r.log.Error("failed to get tags for news", "error", err, "newsID", newsID, "tagIds", tagIds)
			return nil, fmt.Errorf("failed to get tags: %w", err)
		}
		news.Tags = tags
	} else {
		news.Tags = []domain.Tag{}
	}

	r.log.Info("successfully retrieved news by ID", "newsID", newsID, "title", news.Title)

	return &news, nil
}

// GetAllCategories retrieves all categories ordered by orderNumber
func (r *Repository) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	r.log.Info("getting all categories")

	query := `
		SELECT 
			"categoryId",
			"title",
			"orderNumber",
			"statusId"
		FROM "categories"
		ORDER BY "orderNumber" ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.log.Error("failed to query categories", "error", err)
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		var category domain.Category
		err := rows.Scan(
			&category.CategoryID,
			&category.Title,
			&category.OrderNumber,
			&category.StatusID,
		)
		if err != nil {
			r.log.Error("failed to scan category row", "error", err)
			return nil, fmt.Errorf("failed to scan category row: %w", err)
		}
		categories = append(categories, category)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("error iterating category rows", "error", err)
		return nil, fmt.Errorf("error iterating category rows: %w", err)
	}

	r.log.Info("successfully retrieved categories", "count", len(categories))

	return categories, nil
}

// GetAllTags retrieves all tags ordered by title
func (r *Repository) GetAllTags(ctx context.Context) ([]domain.Tag, error) {
	r.log.Info("getting all tags")

	query := `
		SELECT 
			"tagId",
			"title",
			"statusId"
		FROM "tags"
		ORDER BY "title" ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.log.Error("failed to query tags", "error", err)
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var tag domain.Tag
		err := rows.Scan(
			&tag.TagID,
			&tag.Title,
			&tag.StatusID,
		)
		if err != nil {
			r.log.Error("failed to scan tag row", "error", err)
			return nil, fmt.Errorf("failed to scan tag row: %w", err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("error iterating tag rows", "error", err)
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
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

	query := `
		SELECT 
			"tagId",
			"title",
			"statusId"
		FROM "tags"
		WHERE "tagId" = ANY($1)
		ORDER BY "title" ASC
	`

	rows, err := r.pool.Query(ctx, query, tagIds)
	if err != nil {
		r.log.Error("failed to query tags by ids", "error", err, "tagIds", tagIds)
		return nil, fmt.Errorf("failed to query tags by ids: %w", err)
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var tag domain.Tag
		err := rows.Scan(&tag.TagID, &tag.Title, &tag.StatusID)
		if err != nil {
			r.log.Error("failed to scan tag row", "error", err)
			return nil, fmt.Errorf("failed to scan tag row: %w", err)
		}
		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("error iterating tag rows", "error", err)
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	r.log.Debug("successfully retrieved tags by IDs", "count", len(tags), "tagIds", tagIds)

	return tags, nil
}
