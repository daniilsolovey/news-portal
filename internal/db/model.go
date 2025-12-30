// nolint
//
//lint:file-ignore U1000 ignore unused code, it's generated
package db

import (
	"time"
)

var Columns = struct {
	Category struct {
		ID, Title, OrderNumber, StatusID string

		Status string
	}
	GooseDbVersion struct {
		ID, VersionID, IsApplied, Tstamp string
	}
	News struct {
		ID, CategoryID, Title, Content, Author, PublishedAt, UpdatedAt, TagIDs, StatusID string

		Category, Status string
	}
	Status struct {
		ID string
	}
	Tag struct {
		ID, Title, StatusID string

		Status string
	}
}{
	Category: struct {
		ID, Title, OrderNumber, StatusID string

		Status string
	}{
		ID:          "categoryId",
		Title:       "title",
		OrderNumber: "orderNumber",
		StatusID:    "statusId",

		Status: "Status",
	},
	GooseDbVersion: struct {
		ID, VersionID, IsApplied, Tstamp string
	}{
		ID:        "id",
		VersionID: "version_id",
		IsApplied: "is_applied",
		Tstamp:    "tstamp",
	},
	News: struct {
		ID, CategoryID, Title, Content, Author, PublishedAt, UpdatedAt, TagIDs, StatusID string

		Category, Status string
	}{
		ID:          "newsId",
		CategoryID:  "categoryId",
		Title:       "title",
		Content:     "content",
		Author:      "author",
		PublishedAt: "publishedAt",
		UpdatedAt:   "updatedAt",
		TagIDs:      "tagIds",
		StatusID:    "statusId",

		Category: "Category",
		Status:   "Status",
	},
	Status: struct {
		ID string
	}{
		ID: "statusId",
	},
	Tag: struct {
		ID, Title, StatusID string

		Status string
	}{
		ID:       "tagId",
		Title:    "title",
		StatusID: "statusId",

		Status: "Status",
	},
}

var Tables = struct {
	Category struct {
		Name, Alias string
	}
	GooseDbVersion struct {
		Name, Alias string
	}
	News struct {
		Name, Alias string
	}
	Status struct {
		Name, Alias string
	}
	Tag struct {
		Name, Alias string
	}
}{
	Category: struct {
		Name, Alias string
	}{
		Name:  "categories",
		Alias: "t",
	},
	GooseDbVersion: struct {
		Name, Alias string
	}{
		Name:  "goose_db_version",
		Alias: "t",
	},
	News: struct {
		Name, Alias string
	}{
		Name:  "news",
		Alias: "t",
	},
	Status: struct {
		Name, Alias string
	}{
		Name:  "statuses",
		Alias: "t",
	},
	Tag: struct {
		Name, Alias string
	}{
		Name:  "tags",
		Alias: "t",
	},
}

type Category struct {
	tableName struct{} `pg:"categories,alias:t,discard_unknown_columns"`

	ID          int    `pg:"categoryId,pk"`
	Title       string `pg:"title,use_zero"`
	OrderNumber int    `pg:"orderNumber,use_zero"`
	StatusID    int    `pg:"statusId,use_zero"`

	Status *Status `pg:"fk:statusId,rel:has-one"`
}

type GooseDbVersion struct {
	tableName struct{} `pg:"goose_db_version,alias:t,discard_unknown_columns"`

	ID        int       `pg:"id,pk"`
	VersionID int64     `pg:"version_id,use_zero"`
	IsApplied bool      `pg:"is_applied,use_zero"`
	Tstamp    time.Time `pg:"tstamp,use_zero"`
}

type News struct {
	tableName struct{} `pg:"news,alias:t,discard_unknown_columns"`

	ID          int        `pg:"newsId,pk"`
	CategoryID  int        `pg:"categoryId,use_zero"`
	Title       string     `pg:"title,use_zero"`
	Content     *string    `pg:"content"`
	Author      string     `pg:"author,use_zero"`
	PublishedAt time.Time  `pg:"publishedAt,use_zero"`
	UpdatedAt   *time.Time `pg:"updatedAt"`
	TagIDs      []int      `pg:"tagIds,array,use_zero"`
	StatusID    int        `pg:"statusId,use_zero"`

	Category *Category `pg:"fk:categoryId,rel:has-one"`
	Status   *Status   `pg:"fk:statusId,rel:has-one"`
}

type Status struct {
	tableName struct{} `pg:"statuses,alias:t,discard_unknown_columns"`

	ID int `pg:"statusId,pk"`
}

type Tag struct {
	tableName struct{} `pg:"tags,alias:t,discard_unknown_columns"`

	ID       int    `pg:"tagId,pk"`
	Title    string `pg:"title,use_zero"`
	StatusID int    `pg:"statusId,use_zero"`

	Status *Status `pg:"fk:statusId,rel:has-one"`
}
