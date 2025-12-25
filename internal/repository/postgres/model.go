package postgres

import (
	"time"
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
	Tags        []Tag      `pg:"-"`
}
