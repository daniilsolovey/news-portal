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

func Map[From, To any](list []From, converter func(From) To) []To {
	result := make([]To, len(list))
	for i := range list {
		result[i] = converter(list[i])
	}

	return result
}

func NewCategories(list []db.Category) []Category {
	return Map(list, NewCategory)
}

func NewTag(t db.Tag) Tag {
	return Tag{
		TagID:    t.ID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func NewTags(list []db.Tag) []Tag {
	return Map(list, NewTag)
}

func NewNewsList(list []db.News) []News {
	return Map(list, NewNews)
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

	news.Tags = make([]Tag, len(n.TagIDs))
	for i := range n.TagIDs {
		news.Tags[i] = NewTag(db.Tag{
			ID: n.TagIDs[i],
		})
	}

	return news
}

func (u *Manager) fillTags(ctx context.Context, news []News) ([]News, error) {
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

	tags, err := u.TagsByIds(ctx, allTagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags by ids: %w", err)
	}

	tagsByID := make(map[int]Tag, len(tags))
	for i := range tags {
		t := tags[i]
		tagsByID[t.TagID] = t
	}

	for i := range news {
		ids := news[i].Tags
		if len(ids) == 0 {
			news[i].Tags = []Tag{}
			continue
		}

		out := make([]Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[id.TagID]; ok {
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
