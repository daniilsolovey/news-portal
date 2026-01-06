package newsportal

import (
	"context"
	"fmt"

	"github.com/daniilsolovey/news-portal/internal/db"
)

func NewCategory(c db.Category) Category {
	return Category{
		Category: c,
		StatusID: c.StatusID,
	}
}

func Map[From, To any](list []From, converter func(From) To) []To {
	result := make([]To, len(list))
	for i := range list {
		result[i] = converter(list[i])
	}

	return result
}

func NewTag(t db.Tag) Tag {
	return Tag{
		Tag:      t,
		StatusID: t.StatusID,
	}
}

func NewNews(n db.News) News {
	news := News{
		News: n,
	}

	if n.Category != nil {
		news.Category = NewCategory(*n.Category)
	}

	news.Tags = make([]Tag, len(n.TagIDs))
	for i := range n.TagIDs {
		news.Tags[i] = NewTag(db.Tag{
			ID: n.TagIDs[i],
		})
	}

	return news
}

func (u *Manager) fillTags(ctx context.Context, news NewsList) (NewsList, error) {
	if len(news) == 0 {
		return news, nil
	}

	allTagIDs := news.UniqueTagIDs()
	if len(allTagIDs) == 0 {
		return news, nil
	}

	tags, err := u.TagsByIds(ctx, allTagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags by ids: %w", err)
	}

	news.SetTags(tags)

	return news, nil
}
