package newsportal

import postgres "github.com/daniilsolovey/news-portal/internal/db"

func NewCategory(c *postgres.Category) Category {
	return Category{
		CategoryID:  c.CategoryID,
		Title:       c.Title,
		OrderNumber: c.OrderNumber,
		StatusID:    c.StatusID,
	}
}

func NewTag(t *postgres.Tag) Tag {
	return Tag{
		TagID:    t.TagID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func NewNews(n *postgres.News) News {
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
		news.Category = NewCategory(n.Category)
	}

	if len(n.Tags) > 0 {
		news.Tags = make([]Tag, len(n.Tags))
		for i := range n.Tags {
			news.Tags[i] = NewTag(&n.Tags[i])
		}
	}

	return news
}

func NewNewsSummary(n *postgres.News) News {
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
		summary.Category = NewCategory(n.Category)
	}

	if len(n.Tags) > 0 {
		summary.Tags = make([]Tag, len(n.Tags))
		for i := range n.Tags {
			summary.Tags[i] = NewTag(&n.Tags[i])
		}
	}

	return summary
}
