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

func (u *Manager) attachTagsBatch(ctx context.Context, news []db.News) ([]db.News, error) {
	if len(news) == 0 {
		return news, nil
	}

	tagSet := make(map[int32]struct{})
	for i := range news {
		for _, id := range news[i].TagIDs {
			tagSet[int32(id)] = struct{}{}
		}
	}

	if len(tagSet) == 0 {
		for i := range news {
			news[i].TagIDs = []int{}
		}
		return news, nil
	}

	allTagIDs := make([]int32, 0, len(tagSet))
	for id := range tagSet {
		allTagIDs = append(allTagIDs, id)
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
		ids := news[i].TagIDs
		if len(ids) == 0 {
			news[i].TagIDs = []int{}
			continue
		}

		out := make([]db.Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[int32(id)]; ok {
				out = append(out, t)
			}
		}

		sort.Slice(out, func(i, j int) bool {
			return out[i].Title < out[j].Title
		})
		news[i].TagIDs = make([]int, len(out))
		for i := range out {
			news[i].TagIDs[i] = int(out[i].ID)
		}
	}

	return news, nil
}
