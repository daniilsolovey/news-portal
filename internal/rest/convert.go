package rest

import "github.com/daniilsolovey/news-portal/internal/newsportal"

func newCategory(c newsportal.Category) Category {
	return Category{
		CategoryID: c.CategoryID,
		Title:      c.Title,
	}
}

func newTag(t newsportal.Tag) Tag {
	return Tag{
		TagID:    t.TagID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func newTags(tags []newsportal.Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}
	result := make([]Tag, len(tags))
	for i := range tags {
		result[i] = newTag(tags[i])
	}
	return result
}

func NewNews(n newsportal.News) News {
	news := News{
		NewsID:      n.NewsID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Content:     n.Content,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		Category:    newCategory(n.Category),
		Tags:        newTags(n.Tags),
	}

	return news
}

func NewNewsSummary(n newsportal.News) News {
	summary := News{
		NewsID:      n.NewsID,
		CategoryID:  n.CategoryID,
		Title:       n.Title,
		Author:      n.Author,
		PublishedAt: n.PublishedAt,
		Content:     n.Content,
		Category:    newCategory(n.Category),
		Tags:        newTags(n.Tags),
	}

	return summary
}

func NewCategory(c newsportal.Category) Category {
	return newCategory(c)
}

func NewTag(t newsportal.Tag) Tag {
	return newTag(t)
}
