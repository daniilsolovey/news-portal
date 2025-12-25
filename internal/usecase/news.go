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

	summaries := make([]domain.NewsSummary, len(newsList))
	for i := range newsList {
		domainNews := newsList[i].ToDomain()
		summaries[i] = domain.NewsSummary{
			NewsID:      domainNews.NewsID,
			CategoryID:  domainNews.CategoryID,
			Title:       domainNews.Title,
			Author:      domainNews.Author,
			PublishedAt: domainNews.PublishedAt,
			UpdatedAt:   domainNews.UpdatedAt,
			StatusID:    domainNews.StatusID,
			Category:    domainNews.Category,
			Tags:        domainNews.Tags,
		}
	}

	return summaries, nil
}

func (u *NewsUseCase) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	u.log.Info("receiving news count", "tagID", tagID, "categoryID", categoryID)

	count, err := u.repo.Postgres().GetNewsCount(ctx, tagID, categoryID)
	if err != nil {
		u.log.Error("failed to get news count", "error", err)
		return 0, err
	}

	return count, nil
}

func (u *NewsUseCase) GetNewsByID(ctx context.Context, newsID int) (*domain.News, error) {
	u.log.Info("receiving news by ID", "newsID", newsID)

	news, err := u.repo.Postgres().GetNewsByID(ctx, newsID)
	if err != nil {
		u.log.Error("failed to get news by ID", "error", err, "newsID", newsID)
		return nil, err
	}

	domainNews := news.ToDomain()
	return &domainNews, nil
}

func (u *NewsUseCase) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	u.log.Info("receiving all categories")

	categories, err := u.repo.Postgres().GetAllCategories(ctx)
	if err != nil {
		u.log.Error("failed to get all categories", "error", err)
		return nil, err
	}

	domainCategories := make([]domain.Category, len(categories))
	for i := range categories {
		domainCategories[i] = categories[i].ToDomain()
	}

	return domainCategories, nil
}

func (u *NewsUseCase) GetAllTags(ctx context.Context) ([]domain.Tag, error) {
	u.log.Info("receiving all tags")

	tags, err := u.repo.Postgres().GetAllTags(ctx)
	if err != nil {
		u.log.Error("failed to get all tags", "error", err)
		return nil, err
	}

	domainTags := make([]domain.Tag, len(tags))
	for i := range tags {
		domainTags[i] = tags[i].ToDomain()
	}

	return domainTags, nil
}
