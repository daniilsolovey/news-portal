package rpc

import (
	"context"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/vmkteam/zenrpc/v2"
)

//go:generate zenrpc

// NewsService provides RPC methods for news operations.
type NewsService struct {
	zenrpc.Service
	manager *newsportal.Manager
}

func NewNewsService(manager *newsportal.Manager) *NewsService {
	return &NewsService{manager: manager}
}

// List retrieves news with optional filtering by tagId and categoryId, with pagination.
// Returns NewsSummary (without content) sorted by publishedAt DESC.
//
//zenrpc:500 internal server error
func (s *NewsService) List(ctx context.Context, filter NewsFilter) ([]NewsSummary, error) {
	newsportalSummaries, err := s.manager.NewsByFilter(
		ctx,
		filter.TagID,
		filter.CategoryID,
		filter.Page,
		filter.PageSize,
	)

	return NewNewsSummaries(newsportalSummaries), err
}

// Count returns the count of news matching the optional tagId and categoryId filters.
//
//zenrpc:return count of news items
//zenrpc:500 internal server error
func (s *NewsService) Count(ctx context.Context, filter NewsFilter) (int, error) {
	count, err := s.manager.NewsCount(ctx, filter.TagID, filter.CategoryID)
	return count, err
}

// ByID retrieves a single news item by ID with full content, category and tags.
//
//zenrpc:id news numeric ID
//zenrpc:400 id must be positive
//zenrpc:404 news not found
//zenrpc:500 internal server error
func (s *NewsService) ByID(ctx context.Context, id int) (*News, error) {
	if id <= 0 {
		return nil, zenrpc.NewStringError(400, "id must be positive")
	}

	newsportalNews, err := s.manager.NewsByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if newsportalNews == nil {
		return nil, zenrpc.NewStringError(404, "news not found")
	}

	news := NewNews(*newsportalNews)
	return &news, nil
}

// Categories retrieves all categories ordered by orderNumber.
//
//zenrpc:404 categories not found
//zenrpc:500 internal server error
func (s *NewsService) Categories(ctx context.Context) (Categories, error) {
	categories, err := s.manager.Categories(ctx)
	if err != nil {
		return nil, err
	}

	if len(categories) == 0 {
		return nil, zenrpc.NewStringError(404, "categories not found")
	}

	return NewCategories(categories), nil
}

// Tags retrieves all tags ordered by title.
//
//zenrpc:404 tags not found
//zenrpc:500 internal server error
func (s *NewsService) Tags(ctx context.Context) (Tags, error) {
	tags, err := s.manager.Tags(ctx)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, zenrpc.NewStringError(404, "tags not found")
	}

	return NewTags(tags), nil
}
