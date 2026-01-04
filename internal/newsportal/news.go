package newsportal

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/daniilsolovey/news-portal/internal/db"
)

type Manager struct {
	db *db.Repository
}

func NewNewsUseCase(repo *db.Repository, log *slog.Logger) *Manager {
	return &Manager{
		db: repo,
	}
}

// NewsByFilter retrieves news with optional filtering by tagID and categoryID, with pagination
// Returns NewsSummary (without content) sorted by publishedAt DESC
func (u *Manager) NewsByFilter(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]News, error) {
	dbNews, err := u.db.News(ctx, tagID, categoryID,
		page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("db get news: %w", err)
	}

	newsList := NewNewsList(dbNews)

	result, err := u.attachTagsBatch(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	return result, nil
}

func (u *Manager) NewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	count, err := u.db.NewsCount(ctx, tagID, categoryID)
	if err != nil {
		return 0, fmt.Errorf("db get news count: %w", err)
	}

	return count, nil
}

func (u *Manager) NewsByID(ctx context.Context, newsID int) (*News, error) {
	dbNews, err := u.db.NewsByID(ctx, newsID)
	if err != nil {
		return nil, fmt.Errorf("db get news by id: %w", err)
	} else if dbNews == nil {
		return nil, nil
	}

	newsList := NewNewsList([]db.News{*dbNews})

	result, err := u.attachTagsBatch(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	return &result[0], nil
}

func (u *Manager) Categories(ctx context.Context) ([]Category, error) {
	list, err := u.db.Categories(ctx)

	return NewCategories(list), err
}

func (u *Manager) Tags(ctx context.Context) ([]Tag, error) {
	list, err := u.db.Tags(ctx)

	return NewTags(list), err
}

func (u *Manager) TagsByIds(ctx context.Context, newsIDs []int32) ([]Tag, error) {
	list, err := u.db.TagsByIDs(ctx, newsIDs)
	if err != nil {
		return nil, fmt.Errorf("db get tags by ids: %w", err)
	} else if list == nil {
		return nil, nil
	}

	tags := make([]Tag, len(list))
	for i := range list {
		tags[i] = NewTag(list[i])
	}

	return tags, nil
}
