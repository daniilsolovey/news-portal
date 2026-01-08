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

	repo := tx
	manager := NewNewsManager(repo)
	return tx, ctx, manager
}

func TestManager_NewsByFilter_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("WithoutFiltersReturnsAllPublishedNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) == 0 {
			t.Error("expected to get news items, got empty result")
		}
		for i := range news {
			assertNewsBasic(t, &news[i])
			if *news[i].Content == "" {
				t.Errorf("news[%d] should have content in NewsByFilter result", i)
			}
		}
		assertNewsSortedByPublishedAt(t, news)
	})

	t.Run("WithCategoryFilterReturnsFilteredNews", func(t *testing.T) {
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, nil, categoryID, intPtr(1), intPtr(10))
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
			if item.Category.ID != *categoryID {
				t.Errorf("expected category loaded with id %d, got %d", *categoryID, item.Category.ID)
			}
		}
	})

	t.Run("WithTagFilterReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, nil, intPtr(1), intPtr(10))
		if err != nil {
			t.Fatalf("NewsByFilter failed: %v", err)
		}
		if len(news) == 0 {
			t.Fatalf("expected at least one news item, got empty result")
		}
		for _, item := range news {
			hasTag := false
			for _, tag := range item.Tags {
				if tag.ID == *tagID {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("news %d (%s) does not have tag %d", item.ID, item.Title, *tagID)
			}
		}
	})

	t.Run("WithBothTagAndCategoryFiltersReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, categoryID, intPtr(1), intPtr(10))
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
				if tag.ID == *tagID {
					hasTag = true
					break
				}
			}
			if !hasTag {
				t.Errorf("news %d (%s) does not have tag %d", item.ID, item.Title, *tagID)
			}
		}
	})

	t.Run("WithPaginationReturnsCorrectPage", func(t *testing.T) {
		page1, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(3))
		if err != nil {
			t.Fatalf("NewsByFilter page1: %v", err)
		}
		if len(page1) != 3 {
			t.Fatalf("expected 3 items on page1, got %d", len(page1))
		}

		page2, err := manager.NewsByFilter(ctx, nil, nil, intPtr(2), intPtr(3))
		if err != nil {
			t.Fatalf("NewsByFilter page2: %v", err)
		}
		if len(page2) != 3 {
			t.Fatalf("expected 3 items on page2, got %d", len(page2))
		}

		seen := make(map[int]struct{}, 6)
		for _, n := range page1 {
			seen[n.ID] = struct{}{}
		}
		for _, n := range page2 {
			if _, ok := seen[n.ID]; ok {
				t.Fatalf("news %d appears on both pages", n.ID)
			}
		}
	})

	t.Run("TagsAreAttachedToNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
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
					if tag.ID == 0 {
						t.Errorf("news %d has tag with zero TagID", item.ID)
					}
					if tag.Title == "" {
						t.Errorf("news %d has tag with empty Title", item.ID)
					}
					if tag.StatusID != StatusPublished {
						t.Errorf("news %d has tag %d with invalid StatusID: got %d want %d (published)", item.ID, tag.ID, tag.StatusID, StatusPublished)
					}
				}
			}
		}
		if !hasNewsWithTags {
			t.Error("expected at least one news item with tags")
		}
	})

	t.Run("WithInvalidPaginationReturnsError", func(t *testing.T) {
		cases := []struct {
			name     string
			page     *int
			pageSize *int
		}{
			{"page=0", intPtr(0), intPtr(10)},
			{"pageSize=0", intPtr(1), intPtr(0)},
			{"page=nil, pageSize=0", nil, intPtr(0)},
			{"page=0, pageSize=nil", intPtr(0), nil},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := manager.NewsByFilter(ctx, nil, nil, tc.page, tc.pageSize)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			})
		}
	})

	t.Run("ExcludesNewsWithUnpublishedCategory", func(t *testing.T) {
		baseTime := db.BaseTime

		unpublishedCategory := db.Category{
			Title:       "Unpublished Category",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		contentUnpubCat := "This news is in an unpublished category"
		newsInUnpublishedCategory := db.News{
			CategoryID:  unpublishedCategory.ID,
			Title:       "News in Unpublished Category",
			Content:     &contentUnpubCat,
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    StatusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsInUnpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert news in unpublished category: %v", err)
		}

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}

		for _, item := range allNews {
			if item.ID == newsInUnpublishedCategory.ID {
				t.Fatalf("news %d should not be returned (unpublished category)", item.ID)
			}
			if item.Category.StatusID != StatusPublished {
				t.Fatalf("returned news %d has category status=%d, want %d", item.ID, item.Category.StatusID, StatusPublished)
			}
		}
	})

	t.Run("ExcludesNewsWithUnpublishedStatus", func(t *testing.T) {
		baseTime := db.BaseTime

		contentUnpub := "This news is not published"
		unpublishedNews := db.News{
			CategoryID:  1,
			Title:       "Unpublished News",
			Content:     &contentUnpub,
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedNews).Insert(); err != nil {
			t.Fatalf("insert unpublished news: %v", err)
		}

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}

		for _, item := range allNews {
			if item.ID == unpublishedNews.ID {
				t.Fatalf("news %d should not be returned (unpublished status)", item.ID)
			}
			if item.StatusID != StatusPublished {
				t.Fatalf("returned news %d has status=%d, want %d", item.ID, item.StatusID, StatusPublished)
			}
		}
	})

	t.Run("ReturnsOnlyNewsWithPublishedStatus", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}

		if len(allNews) == 0 {
			t.Fatalf("expected at least one news item, got empty result")
		}

		for _, item := range allNews {
			if item.StatusID != StatusPublished {
				t.Fatalf("returned news %d (title: %q) has status=%d, want %d (published)",
					item.ID, item.Title, item.StatusID, StatusPublished)
			}
		}
	})

	t.Run("ExcludesNewsWithFuturePublishedAt", func(t *testing.T) {
		now := time.Now()

		content3 := "This news is scheduled for the future"
		futureNews := db.News{
			CategoryID:  1,
			Title:       "Future News",
			Content:     &content3,
			Author:      "Test Author",
			PublishedAt: now.Add(24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    StatusPublished,
		}
		if _, err := tx.ModelContext(ctx, &futureNews).Insert(); err != nil {
			t.Fatalf("insert future news: %v", err)
		}

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}

		for _, item := range allNews {
			if item.ID == futureNews.ID {
				t.Fatalf("news %d should not be returned (publishedAt in future)", item.ID)
			}
			if !item.PublishedAt.Before(now) {
				t.Fatalf("returned news %d has publishedAt=%v which is not in the past (now=%v)",
					item.ID, item.PublishedAt, now,
				)
			}
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
	tx, ctx, manager := withTx(t)

	t.Run("WithValidIDReturnsNews", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(1))
		if err != nil {
			t.Fatalf("NewsByFilter: %v", err)
		}
		if len(allNews) == 0 {
			t.Fatalf("no news items available for testing")
		}

		newsID := allNews[0].ID
		news, err := manager.NewsByID(ctx, newsID)
		if err != nil {
			t.Fatalf("NewsByID: %v", err)
		}
		if news == nil {
			t.Fatalf("expected news, got nil")
		}
		assertNewsValid(t, news, newsID)
		if *news.Content == "" {
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
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
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

		news, err := manager.NewsByID(ctx, newsWithTags.ID)
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
			if tag.ID == 0 {
				t.Errorf("tag has zero TagID")
			}
			if tag.Title == "" {
				t.Errorf("tag has empty Title")
			}
			if tag.StatusID != StatusPublished {
				t.Errorf("tag %d has invalid StatusID: got %d want %d (published)", tag.ID, tag.StatusID, StatusPublished)
			}
		}
	})

	t.Run("WithUnpublishedStatusReturnsNil", func(t *testing.T) {
		baseTime := db.BaseTime

		contentUnpub2 := "This news is not published"
		unpublishedNews := db.News{
			CategoryID:  1,
			Title:       "Unpublished News",
			Content:     &contentUnpub2,
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedNews).Insert(); err != nil {
			t.Fatalf("insert unpublished news: %v", err)
		}

		got, err := manager.NewsByID(ctx, unpublishedNews.ID)
		if err != nil {
			t.Fatalf("expected nil error for unpublished news, got: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
	})

	t.Run("WithUnpublishedCategoryReturnsNil", func(t *testing.T) {
		baseTime := db.BaseTime

		unpublishedCategory := db.Category{
			Title:       "Unpublished Category for GetNewsByID",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		contentUnpubCat2 := "This news is in an unpublished category"
		newsInUnpublishedCategory := db.News{
			CategoryID:  unpublishedCategory.ID,
			Title:       "News in Unpublished Category",
			Content:     &contentUnpubCat2,
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    StatusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsInUnpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert news in unpublished category: %v", err)
		}

		got, err := manager.NewsByID(ctx, newsInUnpublishedCategory.ID)
		if err != nil {
			t.Fatalf("expected nil error for news with unpublished category, got: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
	})

	t.Run("WithFuturePublishedAtReturnsNil", func(t *testing.T) {
		now := time.Now()

		content6 := "This news is scheduled for the future"
		futureNews := db.News{
			CategoryID:  1,
			Title:       "Future News for GetNewsByID",
			Content:     &content6,
			Author:      "Test Author",
			PublishedAt: now.Add(24 * time.Hour),
			TagIDs:      []int{1},
			StatusID:    StatusPublished,
		}
		if _, err := tx.ModelContext(ctx, &futureNews).Insert(); err != nil {
			t.Fatalf("insert future news: %v", err)
		}

		got, err := manager.NewsByID(ctx, futureNews.ID)
		if err != nil {
			t.Fatalf("expected nil error for news with future publishedAt, got: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
	})
}

func TestManager_Categories_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

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

	t.Run("OnlyReturnsPublishedCategories", func(t *testing.T) {
		unpublishedCat := db.Category{
			Title:       "Unpublished Category",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCat).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		categories, err := manager.Categories(ctx)
		if err != nil {
			t.Fatalf("Categories: %v", err)
		}

		for _, cat := range categories {
			if cat.ID == unpublishedCat.ID {
				t.Fatalf("unpublished category should not be returned")
			}
		}
	})
}

func TestManager_Tags_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

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

	t.Run("OnlyReturnsPublishedTags", func(t *testing.T) {
		unpublishedTag := db.Tag{
			Title:    "Unpublished Tag",
			StatusID: 2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedTag).Insert(); err != nil {
			t.Fatalf("insert unpublished tag: %v", err)
		}

		tags, err := manager.Tags(ctx)
		if err != nil {
			t.Fatalf("Tags: %v", err)
		}

		for _, tag := range tags {
			if tag.ID == unpublishedTag.ID {
				t.Fatalf("unpublished tag should not be returned")
			}
		}
	})
}

func TestManager_TagsByIds_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("ReturnsTagsForValidIds", func(t *testing.T) {
		tagIDs := []int{1, 2, 3}
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
		tagIDs := []int{99999, 99998}
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

	t.Run("ExcludesUnpublishedTags", func(t *testing.T) {
		unpublishedTag := db.Tag{
			Title:    "Unpublished Tag for News",
			StatusID: 2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedTag).Insert(); err != nil {
			t.Fatalf("insert unpublished tag: %v", err)
		}

		mixedTagIDs := []int{1, unpublishedTag.ID}
		tags, err := manager.TagsByIds(ctx, mixedTagIDs)
		if err != nil {
			t.Fatalf("TagsByIds: %v", err)
		}

		for _, tag := range tags {
			if tag.ID == unpublishedTag.ID {
				t.Fatalf("unpublished tag %d should not be loaded", unpublishedTag.ID)
			}
		}

		if len(tags) != 1 || tags[0].ID != 1 {
			t.Fatalf("expected only published tag 1, got %+v", tags)
		}
	})
}

// Helper functions

func intPtr(i int) *int { return &i }

func assertNewsBasic(t *testing.T, news *News) {
	t.Helper()

	if news.ID == 0 {
		t.Fatalf("invalid NewsID")
	}
	if news.Title == "" {
		t.Fatalf("empty Title")
	}
	if news.StatusID != StatusPublished {
		t.Fatalf("invalid StatusID: got %d want %d (published)", news.StatusID, StatusPublished)
	}
	if news.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if news.Category.ID == 0 {
		t.Fatalf("category not loaded")
	}
	if news.Category.StatusID != StatusPublished {
		t.Fatalf("invalid Category StatusID: got %d want %d (published)", news.Category.StatusID, StatusPublished)
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
	if news.ID != newsID {
		t.Fatalf("expected NewsID %d, got %d", newsID, news.ID)
	}
	if news.StatusID != StatusPublished {
		t.Fatalf("invalid StatusID: got %d want %d (published)", news.StatusID, StatusPublished)
	}
	if news.Title == "" {
		t.Fatalf("empty Title")
	}
	if *news.Content == "" {
		t.Fatalf("empty Content")
	}
	if news.Author == "" {
		t.Fatalf("empty Author")
	}
	if news.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if news.Category.ID == 0 {
		t.Fatalf("category not loaded")
	}
	if news.Category.StatusID != StatusPublished {
		t.Fatalf("invalid Category StatusID: got %d want %d (published)", news.Category.StatusID, StatusPublished)
	}
}

func assertCategoryValid(t *testing.T, category Category) {
	t.Helper()
	if category.ID == 0 {
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
	if tag.ID == 0 {
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
