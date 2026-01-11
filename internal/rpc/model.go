package rpc

import (
	"time"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
)

type NewsFilter struct {
	//tagId optional tag filter
	TagID *int `json:"tagId,omitempty"`
	//categoryId optional category filter
	CategoryID *int `json:"categoryId,omitempty"`
	//page=1 page number (1-based)
	Page *int `json:"page,omitempty"`
	//pageSize=10 items per page
	PageSize *int `json:"pageSize,omitempty"`
}

func (f NewsFilter) ToModel() *newsportal.NewsFilter {
	return &newsportal.NewsFilter{
		TagID:      f.TagID,
		CategoryID: f.CategoryID,
	}
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
