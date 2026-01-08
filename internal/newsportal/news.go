package newsportal

import (
	"context"
	"fmt"

	db "github.com/daniilsolovey/news-portal/internal/db"
	"github.com/go-pg/pg/v10/orm"
)

type Manager struct {
	// db *db.Repository
	repo db.NewsRepo
}

func NewNewsManager(dbc orm.DB) *Manager {
	return &Manager{
		// db: repo,
		repo: db.NewNewsRepo(dbc),
	}
}

// NewsByFilter retrieves news with optional filtering by tagID and categoryID, with pagination
// Returns NewsSummary (without content) sorted by publishedAt DESC
func (u *Manager) NewsByFilter(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]News, error) {
	// dbNews, err := u.db.News(ctx, tagID, categoryID,
	// 	page, pageSize)
	// if err != nil {
	// 	return nil, fmt.Errorf("db get news: %w", err)
	// }

	dbNews, err := u.repo.NewsByFilters(ctx, &db.NewsSearch{CategoryID: categoryID},
		db.NewPager(page, pageSize),
		db.WithRelations(db.Columns.News.Category),
		db.WithSort(db.NewSortField(db.Columns.News.PublishedAt, true)),
	)

	newsList := NewNewsList(dbNews)

	err = u.fillTags(ctx, newsList)
	if err != nil {
		return nil, fmt.Errorf("failed to attach tags to news: %w", err)
	}

	return newsList, nil
}

func (u *Manager) NewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	count, err := u.repo.CountNews(ctx, &db.NewsSearch{CategoryID: categoryID})
	if err != nil {
		return 0, fmt.Errorf("db get news count: %w", err)
	}

	return count, nil
}

func (u *Manager) NewsByID(ctx context.Context, newsID int) (*News, error) {
	dbNews, err := u.repo.NewsByID(ctx, newsID,
		db.WithRelations(db.Columns.News.Category),
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
	list, err := u.repo.CategoriesByFilters(ctx, nil, db.PagerNoLimit)

	return NewCategories(list), err
}

func (u *Manager) Tags(ctx context.Context) ([]Tag, error) {
	list, err := u.repo.TagsByFilters(ctx, nil, db.PagerNoLimit)

	return NewTags(list), err
}

func (u *Manager) TagsByIds(ctx context.Context, tagIds []int) ([]Tag, error) {
	if len(tagIds) == 0 {
		return []Tag{}, nil
	}

	list, err := u.repo.TagsByFilters(ctx, &db.TagSearch{IDs: tagIds}, db.PagerNoLimit)

	return NewTags(list), err
}
