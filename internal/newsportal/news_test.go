package newsportal

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/daniilsolovey/news-portal/internal/db"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "failed to begin transaction")

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Errorf("failed to rollback transaction: %v", err)
		}
	})

	repo := tx
	manager := NewNewsManager(repo)
	return tx, ctx, manager
}

// Helper functions for creating test data

func createTestNews(t *testing.T, tx *pg.Tx, ctx context.Context, opts ...newsOption) *db.News {
	t.Helper()

	baseTime := db.BaseTime
	content := "Test content"
	news := &db.News{
		CategoryID:  1,
		Title:       "Test News",
		Content:     &content,
		Author:      "Test Author",
		PublishedAt: baseTime.Add(-24 * time.Hour),
		TagIDs:      []int{1},
		StatusID:    StatusPublished,
	}

	for _, opt := range opts {
		opt(news)
	}

	_, err := tx.ModelContext(ctx, news).Insert()
	require.NoError(t, err, "failed to insert test news")

	return news
}

type newsOption func(*db.News)

func withCategoryID(categoryID int) newsOption {
	return func(n *db.News) {
		n.CategoryID = categoryID
	}
}

func withStatusID(statusID int) newsOption {
	return func(n *db.News) {
		n.StatusID = statusID
	}
}

func withPublishedAt(publishedAt time.Time) newsOption {
	return func(n *db.News) {
		n.PublishedAt = publishedAt
	}
}

func withTitle(title string) newsOption {
	return func(n *db.News) {
		n.Title = title
	}
}

func createTestCategory(t *testing.T, tx *pg.Tx, ctx context.Context, opts ...categoryOption) *db.Category {
	t.Helper()

	category := &db.Category{
		Title:       "Test Category",
		OrderNumber: 99,
		StatusID:    StatusPublished,
	}

	for _, opt := range opts {
		opt(category)
	}

	_, err := tx.ModelContext(ctx, category).Insert()
	require.NoError(t, err, "failed to insert test category")

	return category
}

type categoryOption func(*db.Category)

func withCategoryStatusID(statusID int) categoryOption {
	return func(c *db.Category) {
		c.StatusID = statusID
	}
}

func withCategoryTitle(title string) categoryOption {
	return func(c *db.Category) {
		c.Title = title
	}
}

func createTestTag(t *testing.T, tx *pg.Tx, ctx context.Context, opts ...tagOption) *db.Tag {
	t.Helper()

	tag := &db.Tag{
		Title:    "Test Tag",
		StatusID: StatusPublished,
	}

	for _, opt := range opts {
		opt(tag)
	}

	_, err := tx.ModelContext(ctx, tag).Insert()
	require.NoError(t, err, "failed to insert test tag")

	return tag
}

type tagOption func(*db.Tag)

func withTagStatusID(statusID int) tagOption {
	return func(t *db.Tag) {
		t.StatusID = statusID
	}
}

func withTagTitle(title string) tagOption {
	return func(t *db.Tag) {
		t.Title = title
	}
}

// Helper to check if news has a specific tag
func assertNewsHasTag(t *testing.T, news *News, tagID int) {
	t.Helper()
	hasTag := false
	for _, tag := range news.Tags {
		if tag.ID == tagID {
			hasTag = true
			break
		}
	}
	assert.True(t, hasTag, "news %d (%s) should have tag %d", news.ID, news.Title, tagID)
}

func TestManager_NewsByFilter_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("WithoutFiltersReturnsAllPublishedNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
		require.NoError(t, err)
		require.NotEmpty(t, news, "expected to get news items, got empty result")
		for i := range news {
			assertNewsBasic(t, &news[i])
			assert.NotEmpty(t, *news[i].Content, "news[%d] should have content in NewsByFilter result", i)
		}
		assertNewsSortedByPublishedAt(t, news)
	})

	t.Run("WithCategoryFilterReturnsFilteredNews", func(t *testing.T) {
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, nil, categoryID, intPtr(1), intPtr(10))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(news), 2, "expected at least 2 news items")
		for _, item := range news {
			assert.Equal(t, *categoryID, item.CategoryID, "expected categoryID to match")
			assert.Equal(t, *categoryID, item.Category.ID, "expected category loaded with id to match")
		}
	})

	t.Run("WithTagFilterReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, nil, intPtr(1), intPtr(10))
		require.NoError(t, err)
		assert.NotEmpty(t, news, "expected at least one news item, got empty result")
		for _, item := range news {
			assertNewsHasTag(t, &item, *tagID)
		}
	})

	t.Run("WithBothTagAndCategoryFiltersReturnsFilteredNews", func(t *testing.T) {
		tagID := intPtr(1)
		categoryID := intPtr(1)
		news, err := manager.NewsByFilter(ctx, tagID, categoryID, intPtr(1), intPtr(10))
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(news), 2, "expected at least 2 news items")
		for _, item := range news {
			assert.Equal(t, *categoryID, item.CategoryID, "expected categoryID to match")
			assertNewsHasTag(t, &item, *tagID)
		}
	})

	t.Run("WithPaginationReturnsCorrectPage", func(t *testing.T) {
		page1, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(3))
		require.NoError(t, err)
		require.Len(t, page1, 3, "expected 3 items on page1")

		page2, err := manager.NewsByFilter(ctx, nil, nil, intPtr(2), intPtr(3))
		require.NoError(t, err)
		require.Len(t, page2, 3, "expected 3 items on page2")

		seen := make(map[int]struct{}, 6)
		for _, n := range page1 {
			seen[n.ID] = struct{}{}
		}
		for _, n := range page2 {
			_, ok := seen[n.ID]
			assert.False(t, ok, "news %d appears on both pages", n.ID)
		}
	})

	t.Run("TagsAreAttachedToNews", func(t *testing.T) {
		news, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
		require.NoError(t, err)
		require.NotEmpty(t, news, "expected news items, got empty result")

		hasNewsWithTags := false
		for _, item := range news {
			if len(item.Tags) > 0 {
				hasNewsWithTags = true
				for _, tag := range item.Tags {
					assert.NotZero(t, tag.ID, "news %d has tag with zero TagID", item.ID)
					assert.NotEmpty(t, tag.Title, "news %d has tag with empty Title", item.ID)
					assert.Equal(t, StatusPublished, tag.StatusID, "news %d has tag %d with invalid StatusID", item.ID, tag.ID)
				}
			}
		}
		require.True(t, hasNewsWithTags, "expected at least one news item with tags")
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
				assert.Error(t, err, "expected error for invalid pagination")
			})
		}
	})

	t.Run("ExcludesNewsWithUnpublishedCategory", func(t *testing.T) {
		unpublishedCategory := createTestCategory(t, tx, ctx,
			withCategoryStatusID(2),
			withCategoryTitle("Unpublished Category"),
		)

		newsInUnpublishedCategory := createTestNews(t, tx, ctx,
			withCategoryID(unpublishedCategory.ID),
			withTitle("News in Unpublished Category"),
		)

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		require.NoError(t, err)

		for _, item := range allNews {
			assert.NotEqual(t, newsInUnpublishedCategory.ID, item.ID, "news should not be returned (unpublished category)")
			assert.Equal(t, StatusPublished, item.Category.StatusID, "returned news %d has category status", item.ID)
		}
	})

	t.Run("ExcludesNewsWithUnpublishedStatus", func(t *testing.T) {
		unpublishedNews := createTestNews(t, tx, ctx,
			withStatusID(2),
			withTitle("Unpublished News"),
		)

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		require.NoError(t, err)

		for _, item := range allNews {
			assert.NotEqual(t, unpublishedNews.ID, item.ID, "news should not be returned (unpublished status)")
			assert.Equal(t, StatusPublished, item.StatusID, "returned news %d has status", item.ID)
		}
	})

	t.Run("ReturnsOnlyNewsWithPublishedStatus", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		require.NoError(t, err)
		require.NotEmpty(t, allNews, "expected at least one news item, got empty result")

		for _, item := range allNews {
			assert.Equal(t, StatusPublished, item.StatusID, "returned news %d (title: %q) has status", item.ID, item.Title)
		}
	})

	t.Run("ExcludesNewsWithFuturePublishedAt", func(t *testing.T) {
		now := time.Now()
		futureNews := createTestNews(t, tx, ctx,
			withPublishedAt(now.Add(24*time.Hour)),
			withTitle("Future News"),
		)

		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(100))
		require.NoError(t, err)

		for _, item := range allNews {
			assert.NotEqual(t, futureNews.ID, item.ID, "news should not be returned (publishedAt in future)")
			assert.True(t, item.PublishedAt.Before(now), "returned news %d has publishedAt which is not in the past", item.ID)
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
			require.NoError(t, err)
			assert.GreaterOrEqual(t, count, tt.minCount, "expected at least %d news items", tt.minCount)
		})
	}
}

func TestManager_NewsByID_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("WithValidIDReturnsNews", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(1))
		require.NoError(t, err)
		require.NotEmpty(t, allNews, "no news items available for testing")

		newsID := allNews[0].ID
		news, err := manager.NewsByID(ctx, newsID)
		require.NoError(t, err)
		require.NotNil(t, news, "expected news, got nil")
		assertNewsValid(t, news, newsID)
		assert.NotEmpty(t, *news.Content, "expected content to be present")
	})

	t.Run("WithInvalidIDReturnsNil", func(t *testing.T) {
		invalidID := 99999
		news, err := manager.NewsByID(ctx, invalidID)
		assert.NoError(t, err, "expected nil error for invalid news ID")
		assert.Nil(t, news, "expected nil news for invalid ID")
	})

	t.Run("TagsAreAttachedToNews", func(t *testing.T) {
		allNews, err := manager.NewsByFilter(ctx, nil, nil, intPtr(1), intPtr(10))
		require.NoError(t, err)
		require.NotEmpty(t, allNews, "no news items available")

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
		require.NoError(t, err)
		require.NotNil(t, news, "expected news, got nil")
		require.NotEmpty(t, news.Tags, "expected tags to be attached")
		for _, tag := range news.Tags {
			assert.NotZero(t, tag.ID, "tag has zero TagID")
			assert.NotEmpty(t, tag.Title, "tag has empty Title")
			assert.Equal(t, StatusPublished, tag.StatusID, "tag %d has invalid StatusID", tag.ID)
		}
	})

	t.Run("WithUnpublishedStatusReturnsNil", func(t *testing.T) {
		unpublishedNews := createTestNews(t, tx, ctx,
			withStatusID(2),
			withTitle("Unpublished News"),
		)

		got, err := manager.NewsByID(ctx, unpublishedNews.ID)
		require.NoError(t, err, "expected nil error for unpublished news")
		assert.Nil(t, got, "expected nil news for unpublished status")
	})

	t.Run("WithUnpublishedCategoryReturnsNil", func(t *testing.T) {
		unpublishedCategory := createTestCategory(t, tx, ctx,
			withCategoryStatusID(2),
			withCategoryTitle("Unpublished Category for GetNewsByID"),
		)

		newsInUnpublishedCategory := createTestNews(t, tx, ctx,
			withCategoryID(unpublishedCategory.ID),
			withTitle("News in Unpublished Category"),
		)

		got, err := manager.NewsByID(ctx, newsInUnpublishedCategory.ID)
		require.NoError(t, err, "expected nil error for news with unpublished category")
		assert.Nil(t, got, "expected nil news for unpublished category")
	})

	t.Run("WithFuturePublishedAtReturnsNil", func(t *testing.T) {
		now := time.Now()
		futureNews := createTestNews(t, tx, ctx,
			withPublishedAt(now.Add(24*time.Hour)),
			withTitle("Future News for GetNewsByID"),
		)

		got, err := manager.NewsByID(ctx, futureNews.ID)
		require.NoError(t, err, "expected nil error for news with future publishedAt")
		assert.Nil(t, got, "expected nil news for future publishedAt")
	})
}

func TestManager_Categories_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("ReturnsAllPublishedCategories", func(t *testing.T) {
		categories, err := manager.Categories(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(categories), 5, "expected at least 5 categories")
		for _, cat := range categories {
			assertCategoryValid(t, cat)
		}
		for i := 0; i < len(categories)-1; i++ {
			assert.LessOrEqual(t, categories[i].OrderNumber, categories[i+1].OrderNumber, "categories not sorted by orderNumber ASC")
		}
	})

	t.Run("OnlyReturnsPublishedCategories", func(t *testing.T) {
		unpublishedCat := createTestCategory(t, tx, ctx,
			withCategoryStatusID(2),
			withCategoryTitle("Unpublished Category"),
		)

		categories, err := manager.Categories(ctx)
		require.NoError(t, err)

		for _, cat := range categories {
			assert.NotEqual(t, unpublishedCat.ID, cat.ID, "unpublished category should not be returned")
		}
	})
}

func TestManager_Tags_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("ReturnsAllPublishedTags", func(t *testing.T) {
		tags, err := manager.Tags(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(tags), 5, "expected at least 5 tags")
		for _, tag := range tags {
			assertTagValid(t, tag)
		}
		for i := 0; i < len(tags)-1; i++ {
			assert.LessOrEqual(t, tags[i].Title, tags[i+1].Title, "tags not sorted by title ASC")
		}
	})

	t.Run("OnlyReturnsPublishedTags", func(t *testing.T) {
		unpublishedTag := createTestTag(t, tx, ctx,
			withTagStatusID(2),
			withTagTitle("Unpublished Tag"),
		)

		tags, err := manager.Tags(ctx)
		require.NoError(t, err)

		for _, tag := range tags {
			assert.NotEqual(t, unpublishedTag.ID, tag.ID, "unpublished tag should not be returned")
		}
	})
}

func TestManager_TagsByIds_Integration(t *testing.T) {
	tx, ctx, manager := withTx(t)

	t.Run("ReturnsTagsForValidIds", func(t *testing.T) {
		tagIDs := []int{1, 2, 3}
		tags, err := manager.TagsByIds(ctx, tagIDs)
		require.NoError(t, err)
		require.Len(t, tags, 3, "expected 3 tags")
		for _, tag := range tags {
			assertTagValid(t, tag)
		}
	})

	t.Run("HandlesEmptyTagIds", func(t *testing.T) {
		tags, err := manager.TagsByIds(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, tags, "expected empty slice, got nil")
		assert.Empty(t, tags, "expected empty slice")
	})

	t.Run("HandlesNonExistentTagIds", func(t *testing.T) {
		tagIDs := []int{99999, 99998}
		tags, err := manager.TagsByIds(ctx, tagIDs)
		require.NoError(t, err)
		require.NotNil(t, tags, "expected empty slice, got nil")
		assert.Empty(t, tags, "expected empty slice for non-existent tags")
	})

	t.Run("ExcludesUnpublishedTags", func(t *testing.T) {
		unpublishedTag := createTestTag(t, tx, ctx,
			withTagStatusID(2),
			withTagTitle("Unpublished Tag for News"),
		)

		mixedTagIDs := []int{1, unpublishedTag.ID}
		tags, err := manager.TagsByIds(ctx, mixedTagIDs)
		require.NoError(t, err)

		for _, tag := range tags {
			assert.NotEqual(t, unpublishedTag.ID, tag.ID, "unpublished tag should not be loaded")
		}

		assert.Len(t, tags, 1, "expected only one published tag")
		assert.Equal(t, 1, tags[0].ID, "expected only published tag 1")
	})
}

// Helper functions

func intPtr(i int) *int { return &i }

func assertNewsBasic(t *testing.T, news *News) {
	t.Helper()

	require.NotZero(t, news.ID, "invalid NewsID")
	require.NotEmpty(t, news.Title, "empty Title")
	assert.Equal(t, StatusPublished, news.StatusID, "invalid StatusID")
	require.NotZero(t, news.CategoryID, "invalid CategoryID")
	require.NotZero(t, news.Category.ID, "category not loaded")
	assert.Equal(t, StatusPublished, news.Category.StatusID, "invalid Category StatusID")
	assert.False(t, news.PublishedAt.After(db.BaseTime.Add(365*24*time.Hour)), "publishedAt is unexpectedly in the future: %v", news.PublishedAt)
}

func assertNewsValid(t *testing.T, news *News, newsID int) {
	t.Helper()
	require.NotNil(t, news, "news is nil")
	assert.Equal(t, newsID, news.ID, "expected NewsID to match")
	assert.Equal(t, StatusPublished, news.StatusID, "invalid StatusID")
	require.NotEmpty(t, news.Title, "empty Title")
	require.NotEmpty(t, *news.Content, "empty Content")
	require.NotEmpty(t, news.Author, "empty Author")
	require.NotZero(t, news.CategoryID, "invalid CategoryID")
	require.NotZero(t, news.Category.ID, "category not loaded")
	assert.Equal(t, StatusPublished, news.Category.StatusID, "invalid Category StatusID")
}

func assertCategoryValid(t *testing.T, category Category) {
	t.Helper()
	require.NotZero(t, category.ID, "invalid CategoryID")
	require.NotEmpty(t, category.Title, "empty Title")
	assert.Equal(t, 1, category.StatusID, "invalid StatusID")
}

func assertTagValid(t *testing.T, tag Tag) {
	t.Helper()
	require.NotZero(t, tag.ID, "invalid TagID")
	require.NotEmpty(t, tag.Title, "empty Title")
	assert.Equal(t, 1, tag.StatusID, "invalid StatusID")
}

func assertNewsSortedByPublishedAt(t *testing.T, news []News) {
	t.Helper()
	for i := 0; i < len(news)-1; i++ {
		assert.False(t, news[i].PublishedAt.Before(news[i+1].PublishedAt), "news not sorted by publishedAt desc at %d", i)
	}
}
