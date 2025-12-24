package postgres

import "github.com/daniilsolovey/news-portal/internal/domain"

func (c *Category) ToDomain() domain.Category {
	return domain.Category{
		CategoryID:  c.CategoryID,
		Title:       c.Title,
		OrderNumber: c.OrderNumber,
		StatusID:    c.StatusID,
	}
}

func (t *Tag) ToDomain() domain.Tag {
	return domain.Tag{
		TagID:    t.TagID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func (n *News) ToDomain() domain.News {
	news := domain.News{
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
		news.Category = n.Category.ToDomain()
	}

	if len(n.Tags) > 0 {
		news.Tags = make([]domain.Tag, len(n.Tags))
		for i := range n.Tags {
			news.Tags[i] = n.Tags[i].ToDomain()
		}
	}

	return news
}
