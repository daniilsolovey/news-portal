package newsportal

import (
	"github.com/daniilsolovey/news-portal/internal/db"
)

type Category struct {
	db.Category
	StatusID int
}

type Tag struct {
	db.Tag
	StatusID int
}

type News struct {
	db.News
	Category Category
	Tags     []Tag
}
