package newsportal

import (
	"context"
	"errors"
	"fmt"
	"time"

	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/go-pg/pg/v10/orm"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
	StatusPublished = 1
)

type Manager struct {
	repo db.NewsRepo
}

func NewNewsManager(dbc orm.DB) *Manager {
	return &Manager{
		repo: db.NewNewsRepo(dbc),
	}
}

func withTagIDFilter(tagID *int) db.OpFunc {
	return func(query *orm.Query) {
		if tagID != nil {
			db.Filter{Field: db.Columns.News.TagIDs, Value: *tagID, SearchType: db.SearchTypeArrayContains}.Apply(query)
		}
	}
}

// NewsByFilter retrieves news with optional filtering by tagID and categoryID, with pagination
// Returns NewsSummary (without content) sorted by publishedAt DESC
func (u *Manager) NewsByFilter(ctx context.Context, tagID, categoryID *int, page, pageSize *int) ([]News, error) {
	p, ps, err := validatePagination(page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("invalid pagination parameters: %w", err)
	}

	dbNews, err := u.repo.NewsByFilters(ctx, &db.NewsSearch{CategoryID: categoryID},
		db.NewPager(p, ps),
		db.WithRelations(db.Columns.News.Category), db.EnabledOnly(),
		db.WithPublishedBefore(time.Now()),
		db.WithCategoryEnabled(),
		db.WithSort(db.NewSortField(db.Columns.News.PublishedAt, true)),
		withTagIDFilter(tagID),
	)

	newsList := NewNewsList(dbNews)

	err = u.fillTags(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	return newsList, nil
}

func (u *Manager) NewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	count, err := u.repo.CountNews(ctx, &db.NewsSearch{CategoryID: categoryID},
		db.WithRelations(db.Columns.News.Category), db.EnabledOnly(),
		db.WithPublishedBefore(time.Now()), db.WithCategoryEnabled(),
		withTagIDFilter(tagID),
	)
	if err != nil {
		return 0, fmt.Errorf("db get news count: %w", err)
	}

	return count, nil
}

func (u *Manager) NewsByID(ctx context.Context, newsID int) (*News, error) {
	dbNews, err := u.repo.NewsByID(ctx, newsID,
		db.WithRelations(db.Columns.News.Category), db.EnabledOnly(),
		db.WithPublishedBefore(time.Now()), db.WithCategoryEnabled(),
	)
	if err != nil {
		return nil, fmt.Errorf("db get news by id: %w", err)
	} else if dbNews == nil {
		return nil, nil
	}

	newsList := NewNewsList([]db.News{*dbNews})

	err = u.fillTags(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	return &newsList[0], nil
}

func (u *Manager) Categories(ctx context.Context) ([]Category, error) {
	list, err := u.repo.CategoriesByFilters(ctx, nil, db.PagerNoLimit, db.EnabledOnly())

	return NewCategories(list), err
}

func (u *Manager) Tags(ctx context.Context) ([]Tag, error) {
	list, err := u.repo.TagsByFilters(ctx, nil, db.PagerNoLimit,
		db.WithSort(db.NewSortField(db.Columns.Tag.Title, false)), db.EnabledOnly(),
	)

	return NewTags(list), err
}

func (u *Manager) TagsByIds(ctx context.Context, tagIds []int) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	list, err := u.repo.TagsByFilters(ctx, &db.TagSearch{IDs: tagIds}, db.PagerNoLimit,
		db.EnabledOnly(),
	)

	return NewTags(list), err
}

func validatePagination(page, pageSize *int) (int, int, error) {
	p := defaultPage
	if page != nil {
		if *page <= 0 {
			return 0, 0, errors.New("invalid page")
		}
		p = *page
	}

	ps := defaultPageSize
	if pageSize != nil {
		if *pageSize <= 0 {
			return 0, 0, errors.New("invalid pageSize")
		}
		ps = *pageSize
		if ps > maxPageSize {
			ps = maxPageSize
		}
	}

	return p, ps, nil
}
