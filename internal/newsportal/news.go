package newsportal

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/daniilsolovey/news-portal/internal/db"
)

type Manager struct {
	db  *db.Repository
	log *slog.Logger
}

func NewNewsUseCase(repo *db.Repository, log *slog.Logger) *Manager {
	return &Manager{
		db:  repo,
		log: log,
	}
}

// GetAllNews retrieves news with optional filtering by tagID and categoryID, with pagination
// Returns NewsSummary (without content) sorted by publishedAt DESC
func (u *Manager) GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]News, error) {
	u.log.Info("receiving all news", "tagID", tagID, "categoryID",
		categoryID, "page", page, "pageSize", pageSize)

	dbNews, err := u.db.GetAllNews(ctx, tagID, categoryID,
		page, pageSize)
	if err != nil {
		u.log.Error("failed to get all news", "error", err)
		return nil, err
	}

	dbNewsWithTags, err := u.attachTagsBatch(ctx, dbNews)
	if err != nil {
		u.log.Error("failed to attach tags to news", "error", err)
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	news := make([]News, len(dbNewsWithTags))
	for i := range dbNewsWithTags {
		news[i] = NewNewsSummary(dbNewsWithTags[i])
	}

	return news, nil
}

func (u *Manager) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	u.log.Info("receiving news count", "tagID", tagID, "categoryID", categoryID)

	count, err := u.db.GetNewsCount(ctx, tagID, categoryID)
	if err != nil {
		u.log.Error("failed to get news count", "error", err)
		return 0, err
	}

	return count, nil
}

func (u *Manager) GetNewsByID(ctx context.Context, newsID int) (*News, error) {
	u.log.Info("receiving news by ID", "newsID", newsID)

	dbNews, err := u.db.GetNewsByID(ctx, newsID)
	if err != nil {
		u.log.Error("failed to get news by ID", "error", err, "newsID", newsID)
		return nil, err
	}

	dbNewsWithTags, err := u.attachTagsBatch(ctx, []db.News{*dbNews})
	if err != nil {
		u.log.Error("failed to attach tags to news", "error", err)
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	news := NewNews(dbNewsWithTags[0])
	return &news, nil
}

func (u *Manager) GetAllCategories(ctx context.Context) ([]Category, error) {
	u.log.Info("receiving all categories")

	dbCategories, err := u.db.GetAllCategories(ctx)
	if err != nil {
		u.log.Error("failed to get all categories", "error", err)
		return nil, err
	}

	categories := make([]Category, len(dbCategories))
	for i := range dbCategories {
		categories[i] = NewCategory(dbCategories[i])
	}

	return categories, nil
}

func (u *Manager) GetAllTags(ctx context.Context) ([]Tag, error) {
	u.log.Info("receiving all tags")

	dbTags, err := u.db.GetAllTags(ctx)
	if err != nil {
		u.log.Error("failed to get all tags", "error", err)
		return nil, err
	}

	tags := make([]Tag, len(dbTags))
	for i := range dbTags {
		tags[i] = NewTag(dbTags[i])
	}

	return tags, nil
}
