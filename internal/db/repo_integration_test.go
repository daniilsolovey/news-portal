package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
)

var (
	testDB     *pg.DB
	testRepo   *Repository
	testLogger *slog.Logger
	baseTime   = time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC)
)

const (
	testDBURL       = "postgres://test_user:test_password@localhost:5433/news_portal_test?sslmode=disable"
	migrationsDir   = "../../migrations"
	statusPublished = StatusPublished
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	opt, err := pg.ParseURL(testDBURL)
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

	if err := resetPublicSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset schema: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := runMigrations(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := ensureTablesExist(ctx, testDB, []string{"statuses", "categories", "tags", "news"}); err != nil {
		fmt.Fprintf(os.Stderr, "schema verification failed: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	if err := loadTestData(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load test data: %v\n", err)
		_ = testDB.Close()
		os.Exit(1)
	}

	testRepo = New(testDB, testLogger)

	code := m.Run()

	if err := testDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close database connection: %v\n", err)
	}

	os.Exit(code)
}

func TestGetAllNews_Integration(t *testing.T) {
	tx, ctx, repo := withTx(t)

	filterTests := []struct {
		name       string
		tagID      *int
		categoryID *int
		minCount   int
		validate   func(t *testing.T, news []News)
	}{
		{
			name:       "WithoutFiltersReturnsAllPublishedNews",
			tagID:      nil,
			categoryID: nil,
			minCount:   1,
			validate: func(t *testing.T, news []News) {
				t.Helper()
				if len(news) == 0 {
					t.Error("expected to get news items, got empty result")
				}
				for i := range news {
					assertNewsRowBasic(t, &news[i])
				}
				assertNewsSortedByPublishedAt(t, news)
			},
		},
		{
			name:       "WithCategoryFilterReturnsFilteredNews",
			tagID:      nil,
			categoryID: intPtr(1),
			minCount:   2,
			validate: func(t *testing.T, news []News) {
				t.Helper()
				wantCategoryID := 1
				for _, item := range news {
					if item.CategoryID != wantCategoryID {
						t.Errorf("expected categoryID %d, got %d", wantCategoryID, item.CategoryID)
					}
					if item.Category == nil || item.Category.CategoryID != wantCategoryID {
						got := 0
						if item.Category != nil {
							got = item.Category.CategoryID
						}
						t.Errorf("expected category loaded with id %d, got %d", wantCategoryID, got)
					}
				}
			},
		},
		{
			name:       "WithTagFilterReturnsFilteredNews",
			tagID:      intPtr(1),
			categoryID: nil,
			minCount:   1,
			validate: func(t *testing.T, news []News) {
				t.Helper()
				wantTagID := int32(1)
				for _, item := range news {
					if !hasTagID(item.TagIds, wantTagID) {
						t.Errorf("news %d (%s) does not have tag %d in TagIds", item.NewsID, item.Title, wantTagID)
					}
				}
			},
		},
		{
			name:       "WithBothTagAndCategoryFiltersReturnsFilteredNews",
			tagID:      intPtr(1),
			categoryID: intPtr(1),
			minCount:   2,
			validate: func(t *testing.T, news []News) {
				t.Helper()
				wantTagID := int32(1)
				wantCategoryID := 1
				for _, item := range news {
					if item.CategoryID != wantCategoryID {
						t.Errorf("expected categoryID %d, got %d", wantCategoryID, item.CategoryID)
					}
					if !hasTagID(item.TagIds, wantTagID) {
						t.Errorf("news %d (%s) does not have tag %d in TagIds", item.NewsID, item.Title, wantTagID)
					}
				}
			},
		},
	}

	for _, tt := range filterTests {
		t.Run(tt.name, func(t *testing.T) {
			news, err := repo.GetAllNews(ctx, tt.tagID, tt.categoryID, 1, 10)
			if err != nil {
				t.Fatalf("GetAllNews failed: %v", err)
			}
			if len(news) < tt.minCount {
				t.Fatalf("expected at least %d news items, got %d", tt.minCount, len(news))
			}
			if tt.validate != nil {
				tt.validate(t, news)
			}
		})
	}

	t.Run("WithPaginationReturnsCorrectPage", func(t *testing.T) {
		page1, err := repo.GetAllNews(ctx, nil, nil, 1, 3)
		if err != nil {
			t.Fatalf("GetAllNews page1: %v", err)
		}
		if len(page1) != 3 {
			t.Fatalf("expected 3 items on page1, got %d", len(page1))
		}

		page2, err := repo.GetAllNews(ctx, nil, nil, 2, 3)
		if err != nil {
			t.Fatalf("GetAllNews page2: %v", err)
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

	t.Run("WithInvalidPaginationReturnsError", func(t *testing.T) {
		cases := []struct {
			name     string
			page     int
			pageSize int
		}{
			{"page=0", 0, 10},
			{"pageSize=0", 1, 0},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := repo.GetAllNews(ctx, nil, nil, tc.page, tc.pageSize)
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			})
		}
	})

	t.Run("ExcludesNewsWithUnpublishedCategory", func(t *testing.T) {
		unpublishedCategory := Category{
			Title:       "Unpublished Category",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		newsInUnpublishedCategory := News{
			CategoryID:  unpublishedCategory.CategoryID,
			Title:       "News in Unpublished Category",
			Content:     "This news is in an unpublished category",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsInUnpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert news in unpublished category: %v", err)
		}

		allNews, err := repo.GetAllNews(ctx, nil, nil, 1, 100)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}

		for _, item := range allNews {
			if item.NewsID == newsInUnpublishedCategory.NewsID {
				t.Fatalf("news %d should not be returned (unpublished category)", item.NewsID)
			}
			if item.Category != nil && item.Category.StatusID != statusPublished {
				t.Fatalf("returned news %d has category status=%d, want %d", item.NewsID, item.Category.StatusID, statusPublished)
			}
		}
	})

	t.Run("ExcludesNewsWithUnpublishedStatus", func(t *testing.T) {
		unpublishedNews := News{
			CategoryID:  1,
			Title:       "Unpublished News",
			Content:     "This news is not published",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedNews).Insert(); err != nil {
			t.Fatalf("insert unpublished news: %v", err)
		}

		allNews, err := repo.GetAllNews(ctx, nil, nil, 1, 100)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}

		for _, item := range allNews {
			if item.NewsID == unpublishedNews.NewsID {
				t.Fatalf("news %d should not be returned (unpublished status)", item.NewsID)
			}
			if item.StatusID != statusPublished {
				t.Fatalf("returned news %d has status=%d, want %d", item.NewsID, item.StatusID, statusPublished)
			}
		}
	})

	t.Run("ReturnsOnlyNewsWithPublishedStatus", func(t *testing.T) {
		allNews, err := repo.GetAllNews(ctx, nil, nil, 1, 100)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}

		if len(allNews) == 0 {
			t.Fatalf("expected at least one news item, got empty result")
		}

		for _, item := range allNews {
			if item.StatusID != statusPublished {
				t.Fatalf("returned news %d (title: %q) has status=%d, want %d (published)",
					item.NewsID, item.Title, item.StatusID, statusPublished)
			}
		}
	})

	t.Run("LoadsCategoryViaRelation", func(t *testing.T) {
		news, err := repo.GetAllNews(ctx, nil, nil, 1, 10)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}
		if len(news) == 0 {
			t.Fatalf("expected news, got empty")
		}

		for i := range news {
			if news[i].Category == nil || news[i].Category.CategoryID == 0 {
				t.Fatalf("news[%d] category not loaded", i)
			}
			if news[i].Category.CategoryID != news[i].CategoryID {
				t.Fatalf("news[%d] category mismatch: %d != %d", i, news[i].Category.CategoryID, news[i].CategoryID)
			}
		}
	})

	t.Run("ExcludesNewsWithFuturePublishedAt", func(t *testing.T) {
		now := time.Now()
		futureNews := News{
			CategoryID:  1,
			Title:       "Future News",
			Content:     "This news is scheduled for the future",
			Author:      "Test Author",
			PublishedAt: now.Add(24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &futureNews).Insert(); err != nil {
			t.Fatalf("insert future news: %v", err)
		}

		allNews, err := repo.GetAllNews(ctx, nil, nil, 1, 100)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}

		for _, item := range allNews {
			if item.NewsID == futureNews.NewsID {
				t.Fatalf("news %d should not be returned (publishedAt in future)", item.NewsID)
			}
			if !item.PublishedAt.Before(now) {
				t.Fatalf("returned news %d has publishedAt=%v which is not in the past (now=%v)",
					item.NewsID, item.PublishedAt, now,
				)
			}
		}
	})
}

func TestGetNewsCount_Integration(t *testing.T) {
	_, ctx, repo := withTx(t)

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
			count, err := repo.GetNewsCount(ctx, tt.tagID, tt.categoryID)
			if err != nil {
				t.Fatalf("GetNewsCount: %v", err)
			}
			if count < tt.minCount {
				t.Fatalf("expected at least %d, got %d", tt.minCount, count)
			}
		})
	}
}

func TestGetNewsByID_Integration(t *testing.T) {
	tx, ctx, repo := withTx(t)

	t.Run("WithValidIDReturnsNews", func(t *testing.T) {
		allNews, err := repo.GetAllNews(ctx, nil, nil, 1, 1)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}
		if len(allNews) == 0 {
			t.Fatalf("no news items available for testing")
		}

		newsID := allNews[0].NewsID
		news, err := repo.GetNewsByID(ctx, newsID)
		if err != nil {
			t.Fatalf("GetNewsByID: %v", err)
		}
		assertNewsValid(t, news, newsID)
	})

	t.Run("WithInvalidIDReturnsError", func(t *testing.T) {
		invalidID := 99999
		news, err := repo.GetNewsByID(ctx, invalidID)
		if err == nil {
			t.Fatalf("expected error for invalid news ID, got nil")
		}
		if news != nil {
			t.Fatalf("expected nil news for invalid ID, got %+v", news)
		}
		if !errors.Is(err, ErrNewsNotFound) && !contains(err.Error(), "news not found") {
			t.Fatalf("expected ErrNewsNotFound, got: %v", err)
		}
	})

	t.Run("WithUnpublishedStatusReturnsError", func(t *testing.T) {
		unpublishedNews := News{
			CategoryID:  1,
			Title:       "Unpublished News",
			Content:     "This news is not published",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedNews).Insert(); err != nil {
			t.Fatalf("insert unpublished news: %v", err)
		}

		got, err := repo.GetNewsByID(ctx, unpublishedNews.NewsID)
		if err == nil {
			t.Fatalf("expected error for unpublished news, got nil (news=%+v)", got)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
	})

	t.Run("WithUnpublishedCategoryReturnsError", func(t *testing.T) {
		unpublishedCategory := Category{
			Title:       "Unpublished Category for GetNewsByID",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		newsInUnpublishedCategory := News{
			CategoryID:  unpublishedCategory.CategoryID,
			Title:       "News in Unpublished Category",
			Content:     "This news is in an unpublished category",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsInUnpublishedCategory).Insert(); err != nil {
			t.Fatalf("insert news in unpublished category: %v", err)
		}

		got, err := repo.GetNewsByID(ctx, newsInUnpublishedCategory.NewsID)
		if err == nil {
			t.Fatalf("expected error for news with unpublished category, got nil (news=%+v)", got)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
	})

	t.Run("WithFuturePublishedAtReturnsError", func(t *testing.T) {
		now := time.Now()
		futureNews := News{
			CategoryID:  1,
			Title:       "Future News for GetNewsByID",
			Content:     "This news is scheduled for the future",
			Author:      "Test Author",
			PublishedAt: now.Add(24 * time.Hour),
			TagIds:      []int32{1},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &futureNews).Insert(); err != nil {
			t.Fatalf("insert future news: %v", err)
		}

		got, err := repo.GetNewsByID(ctx, futureNews.NewsID)
		if err == nil {
			t.Fatalf("expected error for news with future publishedAt, got nil (news=%+v)", got)
		}
		if got != nil {
			t.Fatalf("expected nil news, got %+v", got)
		}
		if !errors.Is(err, ErrNewsNotFound) && !contains(err.Error(), "news not found") {
			t.Fatalf("expected ErrNewsNotFound, got: %v", err)
		}
	})
}

func TestGetAllCategories_Integration(t *testing.T) {
	tx, ctx, repo := withTx(t)

	t.Run("ReturnsAllPublishedCategories", func(t *testing.T) {
		categories, err := repo.GetAllCategories(ctx)
		if err != nil {
			t.Fatalf("GetAllCategories: %v", err)
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
		unpublishedCat := Category{
			Title:       "Unpublished Category",
			OrderNumber: 99,
			StatusID:    2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedCat).Insert(); err != nil {
			t.Fatalf("insert unpublished category: %v", err)
		}

		categories, err := repo.GetAllCategories(ctx)
		if err != nil {
			t.Fatalf("GetAllCategories: %v", err)
		}

		for _, cat := range categories {
			if cat.CategoryID == unpublishedCat.CategoryID {
				t.Fatalf("unpublished category should not be returned")
			}
		}
	})
}

func TestGetAllTags_Integration(t *testing.T) {
	tx, ctx, repo := withTx(t)

	t.Run("ReturnsAllPublishedTags", func(t *testing.T) {
		tags, err := repo.GetAllTags(ctx)
		if err != nil {
			t.Fatalf("GetAllTags: %v", err)
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
		unpublishedTag := Tag{
			Title:    "Unpublished Tag",
			StatusID: 2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedTag).Insert(); err != nil {
			t.Fatalf("insert unpublished tag: %v", err)
		}

		tags, err := repo.GetAllTags(ctx)
		if err != nil {
			t.Fatalf("GetAllTags: %v", err)
		}

		for _, tag := range tags {
			if tag.TagID == unpublishedTag.TagID {
				t.Fatalf("unpublished tag should not be returned")
			}
		}
	})
}

func TestGetTagsByIDs_Integration(t *testing.T) {
	tx, ctx, repo := withTx(t)

	t.Run("ReturnsTagIdsInGetAllNews", func(t *testing.T) {
		news, err := repo.GetAllNews(ctx, nil, nil, 1, 10)
		if err != nil {
			t.Fatalf("GetAllNews: %v", err)
		}
		if len(news) == 0 {
			t.Fatalf("no news items available")
		}

		for i := range news {
			item := news[i]

			// GetAllNews in db layer should return TagIds but not load Tags
			// Tags should be empty as they are loaded in newsportal layer
			if len(item.TagIds) > 0 {
				// If TagIds exist, Tags should be empty (not loaded in db layer)
				if len(item.Tags) != 0 {
					t.Fatalf("news %d has TagIds but Tags should be empty in db layer (Tags are loaded in newsportal layer)", item.NewsID)
				}
			} else {
				// If no TagIds, Tags should also be empty
				if len(item.Tags) != 0 {
					t.Fatalf("news %d has no TagIds but has Tags", item.NewsID)
				}
			}

			// Verify that TagIds are valid (non-negative)
			for _, tagID := range item.TagIds {
				if tagID <= 0 {
					t.Fatalf("news %d has invalid TagID: %d", item.NewsID, tagID)
				}
			}
		}
	})

	t.Run("ExcludesUnpublishedTags", func(t *testing.T) {
		unpublishedTag := Tag{
			Title:    "Unpublished Tag for News",
			StatusID: 2,
		}
		if _, err := tx.ModelContext(ctx, &unpublishedTag).Insert(); err != nil {
			t.Fatalf("insert unpublished tag: %v", err)
		}

		mixedTagIDs := []int32{1, int32(unpublishedTag.TagID)}
		tags, err := repo.GetTagsByIDs(ctx, mixedTagIDs)
		if err != nil {
			t.Fatalf("GetTagsByIDs: %v", err)
		}

		for _, tag := range tags {
			if tag.TagID == unpublishedTag.TagID {
				t.Fatalf("unpublished tag %d should not be loaded", unpublishedTag.TagID)
			}
		}

		if len(tags) != 1 || tags[0].TagID != 1 {
			t.Fatalf("expected only published tag 1, got %+v", tags)
		}
	})

	t.Run("HandlesEmptyTagIds", func(t *testing.T) {
		newsWithoutTags := News{
			CategoryID:  1,
			Title:       "News without Tags",
			Content:     "This news has no tags",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsWithoutTags).Insert(); err != nil {
			t.Fatalf("insert news without tags: %v", err)
		}
		if newsWithoutTags.NewsID == 0 {
			t.Fatalf("NewsID was not set after insert")
		}

		got, err := repo.GetTagsByIDs(ctx, nil)
		if err != nil {
			t.Fatalf("GetTagsByIDs empty: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("expected empty slice, got %+v", got)
		}
	})

	t.Run("HandlesNonExistentTagIds", func(t *testing.T) {
		newsWithNonExistentTags := News{
			CategoryID:  1,
			Title:       "News with Non-Existent Tags",
			Content:     "This news references tags that don't exist",
			Author:      "Test Author",
			PublishedAt: baseTime.Add(-24 * time.Hour),
			TagIds:      []int32{99999, 99998},
			StatusID:    statusPublished,
		}
		if _, err := tx.ModelContext(ctx, &newsWithNonExistentTags).Insert(); err != nil {
			t.Fatalf("insert news with non-existent tags: %v", err)
		}
		if newsWithNonExistentTags.NewsID == 0 {
			t.Fatalf("NewsID was not set after insert")
		}

		got, err := repo.GetTagsByIDs(ctx, newsWithNonExistentTags.TagIds)
		if err != nil {
			t.Fatalf("GetTagsByIDs non-existent: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("expected empty slice, got %+v", got)
		}
	})
}

func intPtr(i int) *int { return &i }

func hasTagID(tagIds []int32, id int32) bool {
	for _, tagID := range tagIds {
		if tagID == id {
			return true
		}
	}
	return false
}

func assertNewsRowBasic(t *testing.T, item *News) {
	t.Helper()

	if item.NewsID == 0 {
		t.Fatalf("invalid NewsID")
	}
	if item.Title == "" {
		t.Fatalf("empty Title")
	}
	if item.CategoryID == 0 {
		t.Fatalf("invalid CategoryID")
	}
	if item.Category == nil || item.Category.CategoryID == 0 {
		t.Fatalf("category not loaded")
	}
	if item.PublishedAt.After(baseTime.Add(365 * 24 * time.Hour)) {
		t.Fatalf("publishedAt is unexpectedly in the future: %v", item.PublishedAt)
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
	if news.Category == nil || news.Category.CategoryID == 0 {
		t.Fatalf("category not loaded")
	}
	if len(news.TagIds) > 0 && len(news.Tags) != 0 {
		t.Fatalf("Tags should be empty in db layer (Tags are loaded in newsportal layer)")
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
	if category.StatusID != statusPublished {
		t.Fatalf("invalid StatusID: got %d want %d", category.StatusID, statusPublished)
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
	if tag.StatusID != statusPublished {
		t.Fatalf("invalid StatusID: got %d want %d", tag.StatusID, statusPublished)
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
