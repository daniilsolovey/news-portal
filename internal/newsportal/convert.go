package newsportal

import (
	"context"
	"fmt"
	"sort"

	postgres "github.com/daniilsolovey/news-portal/internal/db"
)

func NewCategory(c postgres.Category) Category {
	return Category{
		CategoryID:  c.CategoryID,
		Title:       c.Title,
		OrderNumber: c.OrderNumber,
		StatusID:    c.StatusID,
	}
}

func NewTag(t postgres.Tag) Tag {
	return Tag{
		TagID:    t.TagID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func NewNews(n postgres.News) News {
	news := News{
		NewsID:      n.NewsID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Content:     n.Content,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		UpdatedAt:   n.UpdatedAt,
		StatusID:    n.StatusID,
	}

	if n.Category != nil {
		news.Category = NewCategory(*n.Category)
	}

	if len(n.Tags) > 0 {
		news.Tags = make([]Tag, len(n.Tags))
		for i := range n.Tags {
			news.Tags[i] = NewTag(n.Tags[i])
		}
	}

	return news
}

func NewNewsSummary(n postgres.News) News {
	summary := News{
		NewsID:      n.NewsID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		UpdatedAt:   n.UpdatedAt,
		StatusID:    n.StatusID,
	}

	if n.Category != nil {
		summary.Category = NewCategory(*n.Category)
	}

	if len(n.Tags) > 0 {
		summary.Tags = make([]Tag, len(n.Tags))
		for i := range n.Tags {
			summary.Tags[i] = NewTag(n.Tags[i])
		}
	}

	return summary
}

func (u *Manager) attachTagsBatch(ctx context.Context,
	news []postgres.News) ([]postgres.News, error) {
	if len(news) == 0 {
		return news, nil
	}

	tagSet := make(map[int32]struct{})
	for i := range news {
		for _, id := range news[i].TagIds {
			tagSet[id] = struct{}{}
		}
	}

	if len(tagSet) == 0 {
		for i := range news {
			news[i].Tags = []postgres.Tag{}
		}
		return news, nil
	}

	allTagIDs := make([]int32, 0, len(tagSet))
	for id := range tagSet {
		allTagIDs = append(allTagIDs, id)
	}

	tags, err := u.db.GetTagsByIDs(ctx, allTagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags by ids: %w", err)
	}

	tagsByID := make(map[int32]postgres.Tag, len(tags))
	for i := range tags {
		t := tags[i]
		tagsByID[int32(t.TagID)] = t
	}

	for i := range news {
		ids := news[i].TagIds
		if len(ids) == 0 {
			news[i].Tags = []postgres.Tag{}
			continue
		}

		out := make([]postgres.Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[id]; ok {
				out = append(out, t)
			}
		}

		sort.Slice(out, func(i, j int) bool {
			return out[i].Title < out[j].Title
		})
		news[i].Tags = out
	}

	return news, nil
}
