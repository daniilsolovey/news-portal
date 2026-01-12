package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/daniilsolovey/news-portal/internal/db"
	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB      *pg.DB
	testHandler *NewsHandler
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	opt, err := pg.ParseURL(db.TestDBURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse database URL: %v\n", err)
		os.Exit(1)
	}

	testDB = pg.Connect(opt)

	if err := testDB.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "failed to connect to test database. Make sure PostgreSQL is running:")
		fmt.Fprintln(os.Stderr, "  docker-compose -f docker-compose.test.yml up -d")
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := db.ResetPublicSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset schema: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := db.RunMigrations(ctx, db.MigrationsDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := db.EnsureTablesExist(ctx, testDB, []string{"statuses", "categories", "tags", "news"}); err != nil {
		fmt.Fprintf(os.Stderr, "schema verification failed: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := db.LoadTestData(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load test data: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	testRepo := db.New(testDB)
	testManager := newsportal.NewNewsManager(testRepo)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testHandler = NewNewsHandler(testManager, logger)

	code := m.Run()

	if err := testDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close database connection: %v\n", err)
	}

	os.Exit(code)
}

func TestNewsHandler_News_Integration(t *testing.T) {
	t.Run("SuccessWithoutFilters", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var summaries []News
		err := json.Unmarshal(rec.Body.Bytes(), &summaries)
		require.NoError(t, err, "failed to unmarshal response")

		require.NotEmpty(t, summaries, "expected news items, got empty result")

		for _, summary := range summaries {
			assert.NotZero(t, summary.NewsID, "invalid NewsID")
			assert.NotEmpty(t, summary.Title, "empty Title")
			assert.NotZero(t, summary.CategoryID, "invalid CategoryID")
		}
	})

	t.Run("SuccessWithTagIdFilter", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?tagId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var summaries []News
		err := json.Unmarshal(rec.Body.Bytes(), &summaries)
		require.NoError(t, err, "failed to unmarshal response")

		assert.NotEmpty(t, summaries, "expected news items with tag 1, got empty result")
	})

	t.Run("SuccessWithCategoryIdFilter", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?categoryId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var summaries []News
		err := json.Unmarshal(rec.Body.Bytes(), &summaries)
		require.NoError(t, err, "failed to unmarshal response")

		require.GreaterOrEqual(t, len(summaries), 2, "expected at least 2 news items")

		for _, summary := range summaries {
			assert.Equal(t, 1, summary.CategoryID, "expected categoryID to match")
		}
	})

	t.Run("SuccessWithPagination", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=1&pageSize=3", nil)
		rec1 := httptest.NewRecorder()
		e.ServeHTTP(rec1, req1)

		require.Equal(t, http.StatusOK, rec1.Code, "expected status 200 for page1")

		var page1 []News
		err := json.Unmarshal(rec1.Body.Bytes(), &page1)
		require.NoError(t, err, "failed to unmarshal page1")
		require.Len(t, page1, 3, "expected 3 items on page1")

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=2&pageSize=3", nil)
		rec2 := httptest.NewRecorder()
		e.ServeHTTP(rec2, req2)

		require.Equal(t, http.StatusOK, rec2.Code, "expected status 200 for page2")

		var page2 []News
		err = json.Unmarshal(rec2.Body.Bytes(), &page2)
		require.NoError(t, err, "failed to unmarshal page2")
		require.Len(t, page2, 3, "expected 3 items on page2")

		seen := make(map[int]struct{})
		for _, n := range page1 {
			seen[n.NewsID] = struct{}{}
		}
		for _, n := range page2 {
			_, ok := seen[n.NewsID]
			assert.False(t, ok, "news %d appears on both pages", n.NewsID)
		}
	})

	t.Run("InvalidTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?tagId=abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "expected status 400")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "invalid request parameters", response["error"], "expected error message to match")
	})

	t.Run("InvalidCategoryId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?categoryId=xyz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "expected status 400")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "invalid request parameters", response["error"], "expected error message to match")
	})

	t.Run("InvalidPage", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=0", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code, "expected status 500 for invalid page")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "internal error", response["error"], "expected error message to match")
	})

	t.Run("InvalidPageSize", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?pageSize=-1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code, "expected status 500 for invalid pageSize")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "internal error", response["error"], "expected error message to match")
	})

	t.Run("PageSizeCappedAt100", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?pageSize=200", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200")

		// The pageSize should be capped at 100, but we can't directly verify this
		// without checking the actual query. We just verify it doesn't error.
	})
}

func TestNewsHandler_NewsCount_Integration(t *testing.T) {
	t.Run("SuccessWithoutFilters", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var count int
		err := json.Unmarshal(rec.Body.Bytes(), &count)
		require.NoError(t, err, "failed to unmarshal response")

		assert.GreaterOrEqual(t, count, 7, "expected at least 7 news items")
	})

	t.Run("SuccessWithTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?tagId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200")

		var count int
		err := json.Unmarshal(rec.Body.Bytes(), &count)
		require.NoError(t, err, "failed to unmarshal response")

		assert.GreaterOrEqual(t, count, 7, "expected at least 7 news items")
	})

	t.Run("SuccessWithCategoryId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?categoryId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200")

		var count int
		err := json.Unmarshal(rec.Body.Bytes(), &count)
		require.NoError(t, err, "failed to unmarshal response")

		assert.GreaterOrEqual(t, count, 2, "expected at least 2 news items")
	})

	t.Run("InvalidTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?tagId=abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "expected status 400")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "invalid request parameters", response["error"], "expected error message to match")
	})
}

func TestNewsHandler_NewsByID_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// First get a valid news ID
		e := testHandler.RegisterRoutes()
		reqList := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=1&pageSize=1", nil)
		recList := httptest.NewRecorder()
		e.ServeHTTP(recList, reqList)

		require.Equal(t, http.StatusOK, recList.Code, "expected status 200 for news list")

		var summaries []News
		err := json.Unmarshal(recList.Body.Bytes(), &summaries)
		require.NoError(t, err, "failed to unmarshal news list response")
		require.NotEmpty(t, summaries, "no news items available for testing")

		newsID := summaries[0].NewsID

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/news/%d", newsID), nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var news News
		err = json.Unmarshal(rec.Body.Bytes(), &news)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, newsID, news.NewsID, "expected NewsID to match")
		assert.NotEmpty(t, news.Title, "empty Title")
		assert.NotEmpty(t, news.Content, "empty Content")
		assert.NotZero(t, news.CategoryID, "invalid CategoryID")
	})

	t.Run("NotFound", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/99999", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code, "expected status 404, body: %s", rec.Body.String())
		assert.Equal(t, "news not found", rec.Body.String(), "expected error message to match")
	})

	t.Run("InvalidId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "expected status 400")

		var response map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err, "failed to unmarshal response")

		assert.Equal(t, "invalid id", response["error"], "expected error message to match")
	})
}

func TestNewsHandler_Categories_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var categories []Category
		err := json.Unmarshal(rec.Body.Bytes(), &categories)
		require.NoError(t, err, "failed to unmarshal response")

		require.GreaterOrEqual(t, len(categories), 5, "expected at least 5 categories")

		for _, cat := range categories {
			assert.NotZero(t, cat.CategoryID, "invalid CategoryID")
			assert.NotEmpty(t, cat.Title, "empty Title")
		}
	})
}

func TestNewsHandler_Tags_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "expected status 200, body: %s", rec.Body.String())

		var tags []Tag
		err := json.Unmarshal(rec.Body.Bytes(), &tags)
		require.NoError(t, err, "failed to unmarshal response")

		require.GreaterOrEqual(t, len(tags), 5, "expected at least 5 tags")

		for _, tag := range tags {
			assert.NotZero(t, tag.TagID, "invalid TagID")
			assert.NotEmpty(t, tag.Title, "empty Title")
		}
	})
}
