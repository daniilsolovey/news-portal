package usecase

import (
	"context"
	"log/slog"

	"github.com/daniilsolovey/news-portal/internal/domain"
	"github.com/daniilsolovey/news-portal/internal/repository"
)

// NewsUseCase represents business logic layer
type NewsUseCase struct {
	repo repository.IRepository
	log  *slog.Logger
}

// NewNewsUseCase creates a new instance of NewsUseCase
func NewNewsUseCase(repo repository.IRepository, log *slog.Logger) *NewsUseCase {
	return &NewsUseCase{
		repo: repo,
		log:  log,
	}
}

// GetAllNews retrieves news with optional filtering by tagID and categoryID, with pagination
// Returns NewsSummary (without content) sorted by publishedAt DESC
func (u *NewsUseCase) GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
	u.log.Info("receiving all news", "tagID", tagID, "categoryID",
		categoryID, "page", page, "pageSize", pageSize)

	newsList, err := u.repo.Postgres().GetAllNews(ctx, tagID, categoryID,
		page, pageSize)
	if err != nil {
		u.log.Error("failed to get all news", "error", err)
		return nil, err
	}

	// Convert News to NewsSummary (remove content)
	summaries := make([]domain.NewsSummary, len(newsList))
	for i, news := range newsList {
		summaries[i] = domain.NewsSummary{
			NewsID:      news.NewsID,
			CategoryID:  news.CategoryID,
			Title:       news.Title,
			Author:      news.Author,
			PublishedAt: news.PublishedAt,
			UpdatedAt:   news.UpdatedAt,
			StatusID:    news.StatusID,
			Category:    news.Category,
			Tags:        news.Tags,
		}
	}

	return summaries, nil
}

// GetNewsCount returns the count of news matching the optional tagID and categoryID filters
func (u *NewsUseCase) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	u.log.Info("receiving news count", "tagID", tagID, "categoryID", categoryID)

	count, err := u.repo.Postgres().GetNewsCount(ctx, tagID, categoryID)
	if err != nil {
		u.log.Error("failed to get news count", "error", err)
		return 0, err
	}

	return count, nil
}

// GetNewsByID retrieves a single news item by ID with full content, category and tags
func (u *NewsUseCase) GetNewsByID(ctx context.Context, newsID int) (*domain.News, error) {
	u.log.Info("receiving news by ID", "newsID", newsID)

	news, err := u.repo.Postgres().GetNewsByID(ctx, newsID)
	if err != nil {
		u.log.Error("failed to get news by ID", "error", err, "newsID", newsID)
		return nil, err
	}

	return news, nil
}

// GetAllCategories retrieves all categories ordered by orderNumber
func (u *NewsUseCase) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	u.log.Info("receiving all categories")

	categories, err := u.repo.Postgres().GetAllCategories(ctx)
	if err != nil {
		u.log.Error("failed to get all categories", "error", err)
		return nil, err
	}

	return categories, nil
}

// GetAllTags retrieves all tags ordered by title
func (u *NewsUseCase) GetAllTags(ctx context.Context) ([]domain.Tag, error) {
	u.log.Info("receiving all tags")

	tags, err := u.repo.Postgres().GetAllTags(ctx)
	if err != nil {
		u.log.Error("failed to get all tags", "error", err)
		return nil, err
	}

	return tags, nil
}
