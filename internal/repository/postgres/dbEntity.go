package postgres

import (
	"time"

	"github.com/daniilsolovey/news-portal/internal/domain"
)

type Category struct {
	tableName   struct{} `pg:"categories"`
	CategoryID  int      `pg:"categoryId,pk"`
	Title       string   `pg:"title"`
	OrderNumber int      `pg:"orderNumber"`
	StatusID    int      `pg:"statusId"`
}

type Tag struct {
	tableName struct{} `pg:"tags"`
	TagID     int      `pg:"tagId,pk"`
	Title     string   `pg:"title"`
	StatusID  int      `pg:"statusId"`
}

type News struct {
	tableName   struct{}   `pg:"news"`
	NewsID      int        `pg:"newsId,pk"`
	CategoryID  int        `pg:"categoryId"`
	Title       string     `pg:"title"`
	Content     string     `pg:"content"`
	Author      string     `pg:"author"`
	PublishedAt time.Time  `pg:"publishedAt"`
	UpdatedAt   *time.Time `pg:"updatedAt"`
	StatusID    int        `pg:"statusId"`
	TagIds      []int32    `pg:"tagIds,array"`
	Category    *Category  `pg:"rel:has-one,fk:categoryId"`
}

func (c *Category) toDomain() domain.Category {
	return domain.Category{
		CategoryID:  c.CategoryID,
		Title:       c.Title,
		OrderNumber: c.OrderNumber,
		StatusID:    c.StatusID,
	}
}

func (t *Tag) toDomain() domain.Tag {
	return domain.Tag{
		TagID:    t.TagID,
		Title:    t.Title,
		StatusID: t.StatusID,
	}
}

func (n *News) toDomain() domain.News {
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
		news.Category = n.Category.toDomain()
	}

	return news
}
