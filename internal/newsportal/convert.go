package newsportal

import (
	"context"
	"fmt"
	"sort"

	"github.com/daniilsolovey/news-portal/internal/db"
)

func NewCategory(c db.Category) Category {
	return Category{
		CategoryID:  c.ID,
		Title:       c.Title,
		OrderNumber: c.OrderNumber,
		StatusID:    c.StatusID,
	}
}

func NewCategories(list []db.Category) []Category {
	categories := make([]Category, len(list))
	for i := range list {
		categories[i] = NewCategory(list[i])
	}

	return categories
}

func NewTag(t db.Tag) Tag {
	return Tag{
		TagID:    t.ID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func NewTags(list []db.Tag) []Tag {
	tags := make([]Tag, len(list))
	for i := range list {
		tags[i] = NewTag(list[i])
	}

	return tags
}

func NewNewsList(list []db.News) []News {
	news := make([]News, len(list))
	for i := range list {
		news[i] = NewNews(list[i])
	}

	return news
}

func NewNews(n db.News) News {
	news := News{
		NewsID:      n.ID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Content:     *n.Content,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		UpdatedAt:   n.UpdatedAt,
		StatusID:    n.StatusID,
	}

	if n.Category != nil {
		news.Category = NewCategory(*n.Category)
	}

	if len(n.TagIDs) > 0 {
		news.Tags = make([]Tag, len(n.TagIDs))
		for i := range n.TagIDs {
			news.Tags[i] = NewTag(db.Tag{
				ID:       n.TagIDs[i],
				StatusID: n.StatusID,
				Title:    n.Title,
			})
		}
	}

	return news
}

func NewNewsSummary(n db.News) News {
	summary := News{
		NewsID:      n.ID,
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

	if len(n.TagIDs) > 0 {
		summary.Tags = make([]Tag, len(n.TagIDs))
		for i := range n.TagIDs {
			summary.Tags[i] = NewTag(db.Tag{
				ID:       n.TagIDs[i],
				StatusID: n.StatusID,
				Title:    n.Title,
			})
		}
	}

	return summary
}

func (u *Manager) attachTagsBatch(ctx context.Context, news []News) ([]News, error) {
	if len(news) == 0 {
		return news, nil
	}

	tagSet := make(map[int]struct{})
	for i := range news {
		for _, tag := range news[i].Tags {
			tagSet[tag.TagID] = struct{}{}
		}
	}

	if len(tagSet) == 0 {
		for i := range news {
			news[i].Tags = []Tag{}
		}
		return news, nil
	}

	allTagIDs := make([]int32, 0, len(tagSet))
	for id := range tagSet {
		allTagIDs = append(allTagIDs, int32(id))
	}

	tags, err := u.db.Tags(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tags by ids: %w", err)
	}

	tagsByID := make(map[int32]db.Tag, len(tags))
	for i := range tags {
		t := tags[i]
		tagsByID[int32(t.ID)] = t
	}

	for i := range news {
		ids := news[i].Tags
		if len(ids) == 0 {
			news[i].Tags = []Tag{}
			continue
		}

		out := make([]db.Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[int32(id.TagID)]; ok {
				out = append(out, t)
			}
		}

		sort.Slice(out, func(i, j int) bool {
			return out[i].Title < out[j].Title
		})
		news[i].Tags = make([]Tag, len(out))
		for j := range out {
			news[i].Tags[j] = NewTag(out[j])
		}
	}

	return news, nil
}
