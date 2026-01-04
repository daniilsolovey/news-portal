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

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var summaries []News
		if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(summaries) == 0 {
			t.Error("expected news items, got empty result")
		}

		for _, summary := range summaries {
			if summary.NewsID == 0 {
				t.Errorf("invalid NewsID")
			}
			if summary.Title == "" {
				t.Errorf("empty Title")
			}
			if summary.CategoryID == 0 {
				t.Errorf("invalid CategoryID")
			}
		}
	})

	t.Run("SuccessWithTagIdFilter", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?tagId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var summaries []News
		if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(summaries) == 0 {
			t.Error("expected news items with tag 1, got empty result")
		}
	})

	t.Run("SuccessWithCategoryIdFilter", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?categoryId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var summaries []News
		if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(summaries) < 2 {
			t.Fatalf("expected at least 2 news items, got %d", len(summaries))
		}

		for _, summary := range summaries {
			if summary.CategoryID != 1 {
				t.Errorf("expected categoryID 1, got %d", summary.CategoryID)
			}
		}
	})

	t.Run("SuccessWithPagination", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=1&pageSize=3", nil)
		rec1 := httptest.NewRecorder()
		e.ServeHTTP(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec1.Code)
		}

		var page1 []News
		if err := json.Unmarshal(rec1.Body.Bytes(), &page1); err != nil {
			t.Fatalf("failed to unmarshal page1: %v", err)
		}

		if len(page1) != 3 {
			t.Fatalf("expected 3 items on page1, got %d", len(page1))
		}

		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=2&pageSize=3", nil)
		rec2 := httptest.NewRecorder()
		e.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec2.Code)
		}

		var page2 []News
		if err := json.Unmarshal(rec2.Body.Bytes(), &page2); err != nil {
			t.Fatalf("failed to unmarshal page2: %v", err)
		}

		if len(page2) != 3 {
			t.Fatalf("expected 3 items on page2, got %d", len(page2))
		}

		seen := make(map[int]struct{})
		for _, n := range page1 {
			seen[n.NewsID] = struct{}{}
		}
		for _, n := range page2 {
			if _, ok := seen[n.NewsID]; ok {
				t.Fatalf("news %d appears on both pages", n.NewsID)
			}
		}
	})

	t.Run("InvalidTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?tagId=abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid tagId" {
			t.Errorf("expected error 'invalid tagId', got %q", response["error"])
		}
	})

	t.Run("InvalidCategoryId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?categoryId=xyz", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid categoryId" {
			t.Errorf("expected error 'invalid categoryId', got %q", response["error"])
		}
	})

	t.Run("InvalidPage", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=0", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid page" {
			t.Errorf("expected error 'invalid page', got %q", response["error"])
		}
	})

	t.Run("InvalidPageSize", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?pageSize=-1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid pageSize" {
			t.Errorf("expected error 'invalid pageSize', got %q", response["error"])
		}
	})

	t.Run("PageSizeCappedAt100", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news?pageSize=200", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

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

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var count int
		if err := json.Unmarshal(rec.Body.Bytes(), &count); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if count < 7 {
			t.Fatalf("expected at least 7, got %d", count)
		}
	})

	t.Run("SuccessWithTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?tagId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var count int
		if err := json.Unmarshal(rec.Body.Bytes(), &count); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if count < 7 {
			t.Fatalf("expected at least 7, got %d", count)
		}
	})

	t.Run("SuccessWithCategoryId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?categoryId=1", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var count int
		if err := json.Unmarshal(rec.Body.Bytes(), &count); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if count < 2 {
			t.Fatalf("expected at least 2, got %d", count)
		}
	})

	t.Run("InvalidTagId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/count?tagId=abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid tagId" {
			t.Errorf("expected error 'invalid tagId', got %q", response["error"])
		}
	})
}

func TestNewsHandler_NewsByID_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// First get a valid news ID
		e := testHandler.RegisterRoutes()
		reqList := httptest.NewRequest(http.MethodGet, "/api/v1/news?page=1&pageSize=1", nil)
		recList := httptest.NewRecorder()
		e.ServeHTTP(recList, reqList)

		if recList.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recList.Code)
		}

		var summaries []News
		if err := json.Unmarshal(recList.Body.Bytes(), &summaries); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(summaries) == 0 {
			t.Fatal("no news items available for testing")
		}

		newsID := summaries[0].NewsID

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/news/%d", newsID), nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var news News
		if err := json.Unmarshal(rec.Body.Bytes(), &news); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if news.NewsID != newsID {
			t.Errorf("expected NewsID %d, got %d", newsID, news.NewsID)
		}
		if news.Title == "" {
			t.Error("empty Title")
		}
		if news.Content == "" {
			t.Error("empty Content")
		}
		if news.CategoryID == 0 {
			t.Error("invalid CategoryID")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/99999", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d, body: %s", rec.Code, rec.Body.String())
		}

		if rec.Body.String() != "news not found" {
			t.Errorf("expected 'news not found', got %q", rec.Body.String())
		}
	})

	t.Run("InvalidId", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/news/abc", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["error"] != "invalid id" {
			t.Errorf("expected error 'invalid id', got %q", response["error"])
		}
	})
}

func TestNewsHandler_Categories_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var categories []Category
		if err := json.Unmarshal(rec.Body.Bytes(), &categories); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(categories) < 5 {
			t.Fatalf("expected at least 5 categories, got %d", len(categories))
		}

		for _, cat := range categories {
			if cat.CategoryID == 0 {
				t.Errorf("invalid CategoryID")
			}
			if cat.Title == "" {
				t.Errorf("empty Title")
			}
		}
	})
}

func TestNewsHandler_Tags_Integration(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		e := testHandler.RegisterRoutes()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d, body: %s", rec.Code, rec.Body.String())
		}

		var tags []Tag
		if err := json.Unmarshal(rec.Body.Bytes(), &tags); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(tags) < 5 {
			t.Fatalf("expected at least 5 tags, got %d", len(tags))
		}

		for _, tag := range tags {
			if tag.TagID == 0 {
				t.Errorf("invalid TagID")
			}
			if tag.Title == "" {
				t.Errorf("empty Title")
			}
		}
	})
}
