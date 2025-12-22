package usecase

import (
	"context"

	"github.com/daniilsolovey/news-portal/internal/domain"
)

// INewsUseCase defines the interface for news use case operations
type INewsUseCase interface {
	GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error)
	GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error)
	GetNewsByID(ctx context.Context, newsID int) (*domain.News, error)
	GetAllCategories(ctx context.Context) ([]domain.Category, error)
	GetAllTags(ctx context.Context) ([]domain.Tag, error)
}

