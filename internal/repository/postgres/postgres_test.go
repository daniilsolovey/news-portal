// +build ignore

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestRepository_GetAllNews_InvalidPagination(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantErr  bool
	}{
		{
			name:     "zero page",
			page:     0,
			pageSize: 10,
			wantErr:  true,
		},
		{
			name:     "zero pageSize",
			page:     1,
			pageSize: 0,
			wantErr:  true,
		},
		{
			name:     "negative page",
			page:     -1,
			pageSize: 10,
			wantErr:  true,
		},
		{
			name:     "negative pageSize",
			page:     1,
			pageSize: -5,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			repo := New(mock, getTestLogger())

			_, err = repo.GetAllNews(context.Background(), nil, nil, tt.page, tt.pageSize)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must be greater than 0")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_GetAllNews_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	publishedAt := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	updatedAt := time.Now().Add(-12 * time.Hour).Truncate(time.Second)

	// Mock news query
	newsRows := mock.NewRows([]string{
		"newsId", "categoryId", "title", "content", "author", "publishedAt",
		"updatedAt", "statusId", "tagIds", "category_categoryId", "category_title",
		"category_orderNumber", "category_statusId",
	}).AddRow(
		int32(1),        // newsId
		int32(1),        // categoryId
		"Test News",     // title
		"Test Content",  // content
		"Test Author",   // author
		publishedAt,     // publishedAt
		updatedAt,       // updatedAt
		int32(1),        // statusId
		[]int32{1, 2},   // tagIds
		int32(1),        // category_categoryId
		"Test Category", // category_title
		int32(1),        // category_orderNumber
		int32(1),        // category_statusId
	)

	// Mock tags query
	tagRows := mock.NewRows([]string{"tagId", "title", "statusId"}).
		AddRow(int32(1), "Tag 1", int32(1)).
		AddRow(int32(2), "Tag 2", int32(1))

	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), 10, 0).
		WillReturnRows(newsRows)

	mock.ExpectQuery(`SELECT`).
		WithArgs([]int32{1, 2}).
		WillReturnRows(tagRows)

	repo := New(mock, getTestLogger())

	news, err := repo.GetAllNews(context.Background(), nil, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, news, 1)
	assert.Equal(t, 1, news[0].NewsID)
	assert.Equal(t, "Test News", news[0].Title)
	assert.Equal(t, "Test Content", news[0].Content)
	assert.Equal(t, "Test Author", news[0].Author)
	assert.Equal(t, 1, news[0].Category.CategoryID)
	assert.Equal(t, "Test Category", news[0].Category.Title)
	assert.Len(t, news[0].Tags, 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllNews_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), 10, 0).
		WillReturnError(errors.New("database error"))

	repo := New(mock, getTestLogger())

	_, err = repo.GetAllNews(context.Background(), nil, nil, 1, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query news")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetNewsCount(t *testing.T) {
	tests := []struct {
		name        string
		tagID       *int
		categoryID  *int
		returnCount int
		returnErr   error
		wantErr     bool
		wantCount   int
	}{
		{
			name:        "success without filters",
			tagID:       nil,
			categoryID:  nil,
			returnCount: 10,
			returnErr:   nil,
			wantErr:     false,
			wantCount:   10,
		},
		{
			name:        "success with tag filter",
			tagID:       intPtr(1),
			categoryID:  nil,
			returnCount: 5,
			returnErr:   nil,
			wantErr:     false,
			wantCount:   5,
		},
		{
			name:        "success with category filter",
			tagID:       nil,
			categoryID:  intPtr(2),
			returnCount: 3,
			returnErr:   nil,
			wantErr:     false,
			wantCount:   3,
		},
		{
			name:        "database error",
			tagID:       nil,
			categoryID:  nil,
			returnCount: 0,
			returnErr:   errors.New("database error"),
			wantErr:     true,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			row := mock.NewRows([]string{"count"}).AddRow(tt.returnCount)

			expectQuery := mock.ExpectQuery(`SELECT COUNT`).
				WithArgs(tt.categoryID, tt.tagID)

			if tt.returnErr != nil {
				expectQuery.WillReturnError(tt.returnErr)
			} else {
				expectQuery.WillReturnRows(row)
			}

			repo := New(mock, getTestLogger())

			count, err := repo.GetNewsCount(context.Background(), tt.tagID, tt.categoryID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetNewsByID_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	publishedAt := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	updatedAt := time.Now().Add(-12 * time.Hour).Truncate(time.Second)

	// Mock news query (uses QueryRow, but pgxmock uses ExpectQuery)
	newsRows := mock.NewRows([]string{
		"newsId", "categoryId", "title", "content", "author", "publishedAt",
		"updatedAt", "statusId", "tagIds", "category_categoryId", "category_title",
		"category_orderNumber", "category_statusId",
	}).AddRow(
		int32(1),        // newsId
		int32(1),        // categoryId
		"Test News",     // title
		"Test Content",  // content
		"Test Author",   // author
		publishedAt,     // publishedAt
		updatedAt,       // updatedAt
		int32(1),        // statusId
		[]int32{1, 2},   // tagIds
		int32(1),        // category_categoryId
		"Test Category", // category_title
		int32(1),        // category_orderNumber
		int32(1),        // category_statusId
	)

	// Mock tags query
	tagRows := mock.NewRows([]string{"tagId", "title", "statusId"}).
		AddRow(int32(1), "Tag 1", int32(1)).
		AddRow(int32(2), "Tag 2", int32(1))

	mock.ExpectQuery(`SELECT`).
		WithArgs(1).
		WillReturnRows(newsRows)

	mock.ExpectQuery(`SELECT`).
		WithArgs([]int32{1, 2}).
		WillReturnRows(tagRows)

	repo := New(mock, getTestLogger())

	news, err := repo.GetNewsByID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, news)
	assert.Equal(t, 1, news.NewsID)
	assert.Equal(t, "Test News", news.Title)
	assert.Equal(t, "Test Content", news.Content)
	assert.NotNil(t, news.UpdatedAt)
	assert.WithinDuration(t, updatedAt, *news.UpdatedAt, time.Second)
	assert.Len(t, news.Tags, 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetNewsByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).
		WithArgs(999).
		WillReturnError(pgx.ErrNoRows)

	repo := New(mock, getTestLogger())

	news, err := repo.GetNewsByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Nil(t, news)
	assert.Contains(t, err.Error(), "not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := mock.NewRows([]string{"categoryId", "title", "orderNumber", "statusId"}).
		AddRow(int32(1), "Category 1", int32(1), int32(1)).
		AddRow(int32(2), "Category 2", int32(2), int32(1))

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(rows)

	repo := New(mock, getTestLogger())

	categories, err := repo.GetAllCategories(context.Background())
	require.NoError(t, err)
	require.Len(t, categories, 2)
	assert.Equal(t, 1, categories[0].CategoryID)
	assert.Equal(t, "Category 1", categories[0].Title)
	assert.Equal(t, 2, categories[1].CategoryID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).
		WillReturnError(errors.New("database error"))

	repo := New(mock, getTestLogger())

	_, err = repo.GetAllCategories(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query categories")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllTags_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := mock.NewRows([]string{"tagId", "title", "statusId"}).
		AddRow(int32(1), "Tag A", int32(1)).
		AddRow(int32(2), "Tag B", int32(1))

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(rows)

	repo := New(mock, getTestLogger())

	tags, err := repo.GetAllTags(context.Background())
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, 1, tags[0].TagID)
	assert.Equal(t, "Tag A", tags[0].Title)
	assert.Equal(t, 2, tags[1].TagID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllTags_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery(`SELECT`).
		WillReturnError(errors.New("database error"))

	repo := New(mock, getTestLogger())

	_, err = repo.GetAllTags(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query tags")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTagsByIDs_EmptySlice(t *testing.T) {
	// This tests the private method indirectly through GetAllNews
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	publishedAt := time.Now().Truncate(time.Second)

	newsRows := mock.NewRows([]string{
		"newsId", "categoryId", "title", "content", "author", "publishedAt",
		"updatedAt", "statusId", "tagIds", "category_categoryId", "category_title",
		"category_orderNumber", "category_statusId",
	}).AddRow(
		int32(1),                   // newsId
		int32(1),                   // categoryId
		"Test News",                // title
		"Test Content",             // content
		"Test Author",              // author
		publishedAt,                // publishedAt
		sql.NullTime{Valid: false}, // updatedAt
		int32(1),                   // statusId
		[]int32{},                  // tagIds - empty
		int32(1),                   // category_categoryId
		"Test Category",            // category_title
		int32(1),                   // category_orderNumber
		int32(1),                   // category_statusId
	)

	// Only news query, no tags query since tagIds is empty
	mock.ExpectQuery(`SELECT`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), 10, 0).
		WillReturnRows(newsRows)

	repo := New(mock, getTestLogger())

	news, err := repo.GetAllNews(context.Background(), nil, nil, 1, 10)
	require.NoError(t, err)
	require.Len(t, news, 1)
	assert.Empty(t, news[0].Tags)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Helper function
func intPtr(i int) *int {
	return &i
}
