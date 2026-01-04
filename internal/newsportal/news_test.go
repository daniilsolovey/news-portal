package newsportal

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/daniilsolovey/news-portal/internal/db"
	"github.com/go-pg/pg/v10"
)

var (
	testDB      *pg.DB
	testManager *Manager
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
	testManager = NewNewsManager(testRepo)

	code := m.Run()

	if err := testDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close database connection: %v\n", err)
	}

	os.Exit(code)
}

func withTx(t *testing.T) (*pg.Tx, context.Context, *Manager) {
	t.Helper()
	ctx := context.Background()

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Errorf("failed to rollback transaction: %v", err)
		}
	})

	repo := db.New(tx)
	manager := NewNewsManager(repo)
	return tx, ctx, manager
}

func TestManager_NewsByFilter_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	t.Run("WithoutFiltersReturnsAllPublishedNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) == 0 {
			t.Error("expected to get news items, got empty result")
		}
		for i := range news {
			assertNewsBasic(t, &news[i])
			if news[i].Content == "" {
				t.Errorf("news[%d] should have content in NewsByFilter result", i)
			}
		}
		assertNewsSortedByPublishedAt(t, news)
	})

	t.Run("WithCategoryFilterReturnsFilteredNews", func(t *testing.T) {
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, nil, categoryID, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) < 2 {
			t.Fatalf("expected at least 2 news items, got %d", len(news))
		}
		for _, item := range news {
			if item.CategoryID != *categoryID {
				t.Errorf("expected categoryID %d, got %d", *categoryID, item.CategoryID)
			}
			if item.Category.CategoryID != *categoryID {
				t.Errorf("expected category loaded with id %d, got %d", *categoryID, item.Category.CategoryID)
			}
		}
	})

	t.Run("WithTagFilterReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, nil, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) == 0 {
			t.Fatalf("expected at least one news item, got empty result")
		}
		for _, item := range news {
			hasTag := false
			for _, tag := range item.Tags {
				if tag.TagID == *tagID {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("news %d (%s) does not have tag %d", item.NewsID, item.Title, *tagID)
			}
		}
	})

	t.Run("WithBothTagAndCategoryFiltersReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, categoryID, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) < 2 {
			t.Fatalf("expected at least 2 news items, got %d", len(news))
		}
		for _, item := range news {
			if item.CategoryID != *categoryID {
				t.Errorf("expected categoryID %d, got %d", *categoryID, item.CategoryID)
			}
			hasTag := false
			for _, tag := range item.Tags {
				if tag.TagID == *tagID {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("news %d (%s) does not have tag %d", item.NewsID, item.Title, *tagID)
			}
		}
	})

	t.Run("WithPaginationReturnsCorrectPage", func(t *testing.T) {
		page1, err := manager.NewsByFilter(ctx, nil, nil, 1, 3)
		if err != nil {
			t.Fatalf("NewsByFilter page1: %v", err)
		}
		if len(page1) != 3 {
			t.Fatalf("expected 3 items on page1, got %d", len(page1))
		}

		page2, err := manager.NewsByFilter(ctx, nil, nil, 2, 3)
		if err != nil {
			t.Fatalf("NewsByFilter page2: %v", err)
		}
		if len(page2) != 3 {
			t.Fatalf("expected 3 items on page2, got %d", len(page2))
		}

		seen := make(map[int]struct{}, 6)
		for _, n := range page1 {
			seen[n.NewsID] = struct{}{}
		}
		for _, n := range page2 {
			if _, ok := seen[n.NewsID]; ok {
				t.Fatalf("news %d appears on both pages", n.NewsID)
			}
		}
	})

	t.Run("TagsAreAttachedToNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) == 0 {
			t.Fatalf("expected news items, got empty result")
		}

		hasNewsWithTags := false
		for _, item := range news {
			if len(item.Tags) > 0 {
				hasNewsWithTags = true
				for _, tag := range item.Tags {
					if tag.TagID == 0 {
						t.Errorf("news %d has tag with zero TagID", item.NewsID)
					}
					if tag.Title == "" {
						t.Errorf("news %d has tag with empty Title", item.NewsID)
					}
				}
			}
		}
		if !hasNewsWithTags {
			t.Error("expected at least one news item with tags")
		}
	})
}

func TestManager_NewsCount_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	tests := []struct {
		name       string
		tagID      *int
		categoryID *int
		minCount   int
	}{
		{"WithoutFiltersReturnsTotalCount", nil, nil, 7},
		{"WithCategoryFilterReturnsFilteredCount", nil, intPtr(1), 2},
		{"WithTagFilterReturnsFilteredCount", intPtr(1), nil, 7},
		{"WithBothFiltersReturnsFilteredCount", intPtr(1), intPtr(1), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := manager.NewsCount(ctx, tt.tagID, tt.categoryID)
			if err != nil {
				t.Fatalf("NewsCount: %v", err)
			}
			if count < tt.minCount {
				t.Fatalf("expected at least %d, got %d", tt.minCount, count)
			}
		})
	}
}

func TestManager_NewsByID_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	t.Run("WithValidIDReturnsNews", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, 1, 1)
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}
		if len(allNews) == 0 {
			t.Fatalf("no news items available for testing")
		}

		newsID := allNews[0].NewsID
		news, err := manager.NewsByID(ctx, newsID)
		if err != nil {
			t.Fatalf("NewsByID: %v", err)
		}
		if news == nil {
			t.Fatalf("expected news, got nil")
		}
		assertNewsValid(t, news, newsID)
		if news.Content == "" {
			t.Error("expected content to be present")
		}
	})

	t.Run("WithInvalidIDReturnsNil", func(t *testing.T) {
		invalidID := 99999
		news, err := manager.NewsByID(ctx, invalidID)
		if err != nil {
			t.Fatalf("expected nil error for invalid news ID, got: %v", err)
		}
		if news != nil {
			t.Fatalf("expected nil news for invalid ID, got %+v", news)
		}
	})

	t.Run("TagsAreAttachedToNews", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, 1, 10)
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}
		if len(allNews) == 0 {
			t.Fatalf("no news items available")
		}

		// Find a news item with tags
		var newsWithTags *News
		for i := range allNews {
			if len(allNews[i].Tags) > 0 {
				newsWithTags = &allNews[i]
				break
			}
		}

		if newsWithTags == nil {
			t.Skip("no news with tags found for testing")
		}

		news, err := manager.NewsByID(ctx, newsWithTags.NewsID)
		if err != nil {
			t.Fatalf("NewsByID: %v", err)
		}
		if news == nil {
			t.Fatalf("expected news, got nil")
		}

		if len(news.Tags) == 0 {
			t.Error("expected tags to be attached")
		}
		for _, tag := range news.Tags {
			if tag.TagID == 0 {
				t.Errorf("tag has zero TagID")
			}
			if tag.Title == "" {
				t.Errorf("tag has empty Title")
			}
		}
	})
}

func TestManager_Categories_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	t.Run("ReturnsAllPublishedCategories", func(t *testing.T) {
		categories, err := manager.Categories(ctx)
		if err != nil {
			t.Fatalf("Categories: %v", err)
		}
		if len(categories) < 5 {
			t.Fatalf("expected at least 5 categories, got %d", len(categories))
		}
		for _, cat := range categories {
			assertCategoryValid(t, cat)
		}
		for i := 0; i < len(categories)-1; i++ {
			if categories[i].OrderNumber > categories[i+1].OrderNumber {
				t.Fatalf("categories not sorted by orderNumber ASC")
			}
		}
	})
}

func TestManager_Tags_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	t.Run("ReturnsAllPublishedTags", func(t *testing.T) {
		tags, err := manager.Tags(ctx)
		if err != nil {
			t.Fatalf("Tags: %v", err)
		}
		if len(tags) < 5 {
			t.Fatalf("expected at least 5 tags, got %d", len(tags))
		}
		for _, tag := range tags {
			assertTagValid(t, tag)
		}
		for i := 0; i < len(tags)-1; i++ {
			if tags[i].Title > tags[i+1].Title {
				t.Fatalf("tags not sorted by title ASC")
			}
		}
	})
}

func TestManager_TagsByIds_Integration(t *testing.T) {
	_, ctx, manager := withTx(t)

	t.Run("ReturnsTagsForValidIds", func(t *testing.T) {
		tagIDs := []int32{1, 2, 3}
		tags, err := manager.TagsByIds(ctx, tagIDs)
		if err != nil {
			t.Fatalf("TagsByIds: %v", err)
		}
		if len(tags) != 3 {
			t.Fatalf("expected 3 tags, got %d", len(tags))
		}
		for _, tag := range tags {
			assertTagValid(t, tag)
		}
	})

	t.Run("HandlesEmptyTagIds", func(t *testing.T) {
		tags, err := manager.TagsByIds(ctx, nil)
		if err != nil {
			t.Fatalf("TagsByIds empty: %v", err)
		}
		if tags == nil {
			t.Fatalf("expected empty slice, got nil")
		}
		if len(tags) != 0 {
			t.Fatalf("expected empty slice, got %d items", len(tags))
		}
	})

	t.Run("HandlesNonExistentTagIds", func(t *testing.T) {
		tagIDs := []int32{99999, 99998}
		tags, err := manager.TagsByIds(ctx, tagIDs)
		if err != nil {
			t.Fatalf("TagsByIds non-existent: %v", err)
		}
		if tags == nil {
			t.Fatalf("expected empty slice, got nil")
		}
		if len(tags) != 0 {
			t.Fatalf("expected empty slice for non-existent tags, got %d items", len(tags))
		}
	})
}

// Helper functions

func intPtr(i int) *int { return &i }

func assertNewsBasic(t *testing.T, news *News) {
	t.Helper()

	if news.NewsID == 0 {
		t.Fatalf("invalid NewsID")
	}
	if news.Title == "" {
		t.Fatalf("empty Title")
	}
	if news.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if news.Category.CategoryID == 0 {
		t.Fatalf("category not loaded")
	}
	if news.PublishedAt.After(db.BaseTime.Add(365 * 24 * time.Hour)) {
		t.Fatalf("publishedAt is unexpectedly in the future: %v", news.PublishedAt)
	}
}

func assertNewsValid(t *testing.T, news *News, newsID int) {
	t.Helper()
	if news == nil {
		t.Fatalf("news is nil")
	}
	if news.NewsID != newsID {
		t.Fatalf("expected NewsID %d, got %d", newsID, news.NewsID)
	}
	if news.Title == "" {
		t.Fatalf("empty Title")
	}
	if news.Content == "" {
		t.Fatalf("empty Content")
	}
	if news.Author == "" {
		t.Fatalf("empty Author")
	}
	if news.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if news.Category.CategoryID == 0 {
		t.Fatalf("category not loaded")
	}
}

func assertCategoryValid(t *testing.T, category Category) {
	t.Helper()
	if category.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if category.Title == "" {
		t.Fatalf("empty Title")
	}
	if category.StatusID != 1 {
		t.Fatalf("invalid StatusID: got %d want 1 (published)", category.StatusID)
	}
}

func assertTagValid(t *testing.T, tag Tag) {
	t.Helper()
	if tag.TagID == 0 {
		t.Fatalf("invalid TagID")
	}
	if tag.Title == "" {
		t.Fatalf("empty Title")
	}
	if tag.StatusID != 1 {
		t.Fatalf("invalid StatusID: got %d want 1 (published)", tag.StatusID)
	}
}

func assertNewsSortedByPublishedAt(t *testing.T, news []News) {
	t.Helper()
	for i := 0; i < len(news)-1; i++ {
		if news[i].PublishedAt.Before(news[i+1].PublishedAt) {
			t.Fatalf("news not sorted by publishedAt desc at %d", i)
		}
	}
}
