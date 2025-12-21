package domain

import "time"

type Category struct {
	CategoryID  int    `json:"categoryId"`
	Title       string `json:"title"`
	OrderNumber int    `json:"orderNumber"`
	StatusID    int    `json:"statusId"`
}

type Tag struct {
	TagID    int    `json:"tagId"`
	Title    string `json:"title"`
	StatusID int    `json:"statusId"`
}

type News struct {
	NewsID      int        `json:"newsId"`
	CategoryID  int        `json:"categoryId"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	PublishedAt time.Time  `json:"publishedAt"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	StatusID    int        `json:"statusId"`
	Category    Category   `json:"category"`
	Tags        []Tag      `json:"tags"`
}
