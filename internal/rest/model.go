package rest

import "time"

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
