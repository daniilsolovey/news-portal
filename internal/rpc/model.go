package rpc

import (
	"time"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
)

type NewsFilter struct {
	TagID      *int `json:"tagId,omitempty"`
	CategoryID *int `json:"categoryId,omitempty"`
	Page       *int `json:"page,omitempty"`
	PageSize   *int `json:"pageSize,omitempty"`
}

func (f NewsFilter) ToModel() *newsportal.NewsFilter {
	return &newsportal.NewsFilter{
		TagID:      f.TagID,
		CategoryID: f.CategoryID,
	}
}

type NewsCountRequest struct {
	TagID      *int `json:"tagId,omitempty"`
	CategoryID *int `json:"categoryId,omitempty"`
}

type NewsByIDRequest struct {
	ID int `json:"id"`
}

type Category struct {
	CategoryID int    `json:"categoryId"`
	Title      string `json:"title"`
}

type Tag struct {
	TagID    int    `json:"tagId"`
	Title    string `json:"title"`
	StatusID int    `json:"statusId"`
}

type News struct {
	NewsID      int       `json:"newsId"`
	CategoryID  int       `json:"categoryId"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"publishedAt"`
	Category    Category  `json:"category"`
	Tags        []Tag     `json:"tags"`
}

type NewsSummary struct {
	NewsID      int       `json:"newsId"`
	CategoryID  int       `json:"categoryId"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	PublishedAt time.Time `json:"publishedAt"`
	Category    Category  `json:"category"`
	Tags        []Tag     `json:"tags"`
}
