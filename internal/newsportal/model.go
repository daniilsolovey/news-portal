package newsportal

import "time"

type Category struct {
	CategoryID  int
	Title       string
	OrderNumber int
	StatusID    int
}

type Tag struct {
	TagID    int
	Title    string
	StatusID int
}

type News struct {
	NewsID      int
	CategoryID  int
	Title       string
	Content     string
	Author      string
	PublishedAt time.Time
	UpdatedAt   *time.Time
	StatusID    int
	Category    Category
	Tags        []Tag
}
