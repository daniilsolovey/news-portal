package rest

import "github.com/daniilsolovey/news-portal/internal/newsportal"

func Map[From, To any](list []From, converter func(From) To) []To {
	result := make([]To, len(list))
	for i := range list {
		result[i] = converter(list[i])
	}
	return result
}

func NewNews(n newsportal.News) News {
	news := News{
		NewsID:      n.ID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Content:     *n.Content,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		Category:    NewCategory(n.Category),
		Tags:        NewTags(n.Tags),
	}

	return news
}

func NewNewsSummary(n newsportal.News) News {
	summary := News{
		NewsID:      n.ID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		Content:     *n.Content,
		Category:    NewCategory(n.Category),
		Tags:        NewTags(n.Tags),
	}

	return summary
}

func NewCategory(c newsportal.Category) Category {
	return Category{
		CategoryID: c.ID,
		Title:      c.Title,
	}
}

func NewTag(t newsportal.Tag) Tag {
	return Tag{
		TagID:    t.ID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}
