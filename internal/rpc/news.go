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
//zenrpc:tagId optional tag filter
//zenrpc:categoryId optional category filter
//zenrpc:page=1 page number (1-based)
//zenrpc:pageSize=10 items per page
//zenrpc:return list of news summaries
//zenrpc:500 internal server error
func (s *NewsService) List(ctx context.Context, filter NewsFilter) (NewsSummaries, error) {
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
//zenrpc:tagId optional tag filter
//zenrpc:categoryId optional category filter
//zenrpc:return count of news items
//zenrpc:500 internal server error
func (s *NewsService) Count(ctx context.Context, filter NewsCountRequest) (int, error) {
	count, err := s.manager.NewsCount(ctx, filter.TagID, filter.CategoryID)
	return count, err
}

// ByID retrieves a single news item by ID with full content, category and tags.
//
//zenrpc:id news numeric ID
//zenrpc:return news with full content
//zenrpc:400 id must be positive
//zenrpc:404 news not found
//zenrpc:500 internal server error
func (s *NewsService) ByID(ctx context.Context, req NewsByIDRequest) (*News, error) {
	if req.ID <= 0 {
		return nil, zenrpc.NewStringError(400, "id must be positive")
	}

	newsportalNews, err := s.manager.NewsByID(ctx, req.ID)
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
//zenrpc:return list of categories
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
//zenrpc:return list of tags
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
