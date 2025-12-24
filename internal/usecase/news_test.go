package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/daniilsolovey/news-portal/internal/domain"
	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noOpLogger creates a logger that discards all output for tests
func noOpLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError + 1, // Set level above Error to suppress all logs
	}))
}

// mockPostgresRepository is a manual stub implementation of postgres.IRepository
type mockPostgresRepository struct {
	getAllNewsFunc       func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error)
	getNewsCountFunc     func(ctx context.Context, tagID, categoryID *int) (int, error)
	getNewsByIDFunc      func(ctx context.Context, newsID int) (*postgres.News, error)
	getAllCategoriesFunc func(ctx context.Context) ([]postgres.Category, error)
	getAllTagsFunc       func(ctx context.Context) ([]postgres.Tag, error)
}

func (m *mockPostgresRepository) Close() error {
	return nil
}
func (m *mockPostgresRepository) Ping(ctx context.Context) error {
	return nil
}

func (m *mockPostgresRepository) GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
	if m.getAllNewsFunc != nil {
		return m.getAllNewsFunc(ctx, tagID, categoryID, page, pageSize)
	}
	return nil, nil
}

func (m *mockPostgresRepository) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	if m.getNewsCountFunc != nil {
		return m.getNewsCountFunc(ctx, tagID, categoryID)
	}
	return 0, nil
}

func (m *mockPostgresRepository) GetNewsByID(ctx context.Context, newsID int) (*postgres.News, error) {
	if m.getNewsByIDFunc != nil {
		return m.getNewsByIDFunc(ctx, newsID)
	}
	return nil, nil
}

func (m *mockPostgresRepository) GetAllCategories(ctx context.Context) ([]postgres.Category, error) {
	if m.getAllCategoriesFunc != nil {
		return m.getAllCategoriesFunc(ctx)
	}
	return nil, nil
}

func (m *mockPostgresRepository) GetAllTags(ctx context.Context) ([]postgres.Tag, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(ctx)
	}
	return nil, nil
}

// mockRepository is a manual stub implementation of repository.IRepository
type mockRepository struct {
	postgresRepo postgres.IRepository
}

func (m *mockRepository) Postgres() postgres.IRepository {
	return m.postgresRepo
}

func TestNewsUseCase_GetAllNews(t *testing.T) {
	logger := noOpLogger()
	ctx := context.Background()
	testTime := time.Now()
	updatedTime := testTime.Add(1 * time.Hour)

	tests := []struct {
		name           string
		tagID          *int
		categoryID     *int
		page           int
		pageSize       int
		mockFunc       func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error)
		expectedResult []domain.NewsSummary
		expectedError  error
	}{
		{
			name:       "success without filters",
			tagID:      nil,
			categoryID: nil,
			page:       1,
			pageSize:   10,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				assert.Nil(t, tagID)
				assert.Nil(t, categoryID)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, pageSize)
				return []postgres.News{
					{
						NewsID:      1,
						CategoryID:  1,
						Title:       "News 1",
						Content:     "Content 1",
						Author:      "Author 1",
						PublishedAt: testTime,
						UpdatedAt:   &updatedTime,
						StatusID:    1,
						Category: &postgres.Category{
							CategoryID:  1,
							Title:       "Category 1",
							OrderNumber: 1,
							StatusID:    1,
						},
						Tags: []postgres.Tag{
							{TagID: 1, Title: "Tag 1", StatusID: 1},
						},
					},
					{
						NewsID:      2,
						CategoryID:  2,
						Title:       "News 2",
						Content:     "Content 2",
						Author:      "Author 2",
						PublishedAt: testTime,
						UpdatedAt:   nil,
						StatusID:    1,
						Category: &postgres.Category{
							CategoryID:  2,
							Title:       "Category 2",
							OrderNumber: 2,
							StatusID:    1,
						},
						Tags: []postgres.Tag{},
					},
				}, nil
			},
			expectedResult: []domain.NewsSummary{
				{
					NewsID:      1,
					CategoryID:  1,
					Title:       "News 1",
					Author:      "Author 1",
					PublishedAt: testTime,
					UpdatedAt:   &updatedTime,
					StatusID:    1,
					Category: domain.Category{
						CategoryID:  1,
						Title:       "Category 1",
						OrderNumber: 1,
						StatusID:    1,
					},
					Tags: []domain.Tag{
						{TagID: 1, Title: "Tag 1", StatusID: 1},
					},
				},
				{
					NewsID:      2,
					CategoryID:  2,
					Title:       "News 2",
					Author:      "Author 2",
					PublishedAt: testTime,
					UpdatedAt:   nil,
					StatusID:    1,
					Category: domain.Category{
						CategoryID:  2,
						Title:       "Category 2",
						OrderNumber: 2,
						StatusID:    1,
					},
					Tags: []domain.Tag{},
				},
			},
			expectedError: nil,
		},
		{
			name:       "success with tagID filter",
			tagID:      intPtr(5),
			categoryID: nil,
			page:       2,
			pageSize:   20,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				require.NotNil(t, tagID)
				assert.Equal(t, 5, *tagID)
				assert.Nil(t, categoryID)
				assert.Equal(t, 2, page)
				assert.Equal(t, 20, pageSize)
				return []postgres.News{}, nil
			},
			expectedResult: []domain.NewsSummary{},
			expectedError:  nil,
		},
		{
			name:       "success with categoryID filter",
			tagID:      nil,
			categoryID: intPtr(3),
			page:       1,
			pageSize:   10,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				assert.Nil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 3, *categoryID)
				return []postgres.News{}, nil
			},
			expectedResult: []domain.NewsSummary{},
			expectedError:  nil,
		},
		{
			name:       "success with both filters",
			tagID:      intPtr(1),
			categoryID: intPtr(2),
			page:       3,
			pageSize:   15,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				require.NotNil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 1, *tagID)
				assert.Equal(t, 2, *categoryID)
				assert.Equal(t, 3, page)
				assert.Equal(t, 15, pageSize)
				return []postgres.News{}, nil
			},
			expectedResult: []domain.NewsSummary{},
			expectedError:  nil,
		},
		{
			name:       "repository error",
			tagID:      nil,
			categoryID: nil,
			page:       1,
			pageSize:   10,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				return nil, errors.New("database error")
			},
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
		{
			name:       "empty result",
			tagID:      nil,
			categoryID: nil,
			page:       1,
			pageSize:   10,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				return []postgres.News{}, nil
			},
			expectedResult: []domain.NewsSummary{},
			expectedError:  nil,
		},
		{
			name:       "content field removed in summary",
			tagID:      nil,
			categoryID: nil,
			page:       1,
			pageSize:   10,
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]postgres.News, error) {
				return []postgres.News{
					{
						NewsID:      1,
						CategoryID:  1,
						Title:       "Test News",
						Content:     "This content should not be in summary",
						Author:      "Author",
						PublishedAt: testTime,
						StatusID:    1,
						Category:    &postgres.Category{CategoryID: 1, Title: "Cat", OrderNumber: 1, StatusID: 1},
						Tags:        []postgres.Tag{},
					},
				}, nil
			},
			expectedResult: []domain.NewsSummary{
				{
					NewsID:      1,
					CategoryID:  1,
					Title:       "Test News",
					Author:      "Author",
					PublishedAt: testTime,
					StatusID:    1,
					Category:    domain.Category{CategoryID: 1, Title: "Cat", OrderNumber: 1, StatusID: 1},
					Tags:        []domain.Tag{},
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := &mockPostgresRepository{
				getAllNewsFunc: tt.mockFunc,
			}
			mockRepo := &mockRepository{
				postgresRepo: mockPostgres,
			}

			uc := NewNewsUseCase(mockRepo, logger)
			result, err := uc.GetAllNews(ctx, tt.tagID, tt.categoryID, tt.page, tt.pageSize)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
				// Verify that content field is not present in summaries
				// NewsSummary doesn't have Content field, which is verified by the conversion logic
			}
		})
	}
}

func TestNewsUseCase_GetNewsCount(t *testing.T) {
	logger := noOpLogger()
	ctx := context.Background()

	tests := []struct {
		name          string
		tagID         *int
		categoryID    *int
		mockFunc      func(ctx context.Context, tagID, categoryID *int) (int, error)
		expectedCount int
		expectedError error
	}{
		{
			name:       "success without filters",
			tagID:      nil,
			categoryID: nil,
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				assert.Nil(t, tagID)
				assert.Nil(t, categoryID)
				return 42, nil
			},
			expectedCount: 42,
			expectedError: nil,
		},
		{
			name:       "success with tagID",
			tagID:      intPtr(5),
			categoryID: nil,
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				require.NotNil(t, tagID)
				assert.Equal(t, 5, *tagID)
				assert.Nil(t, categoryID)
				return 10, nil
			},
			expectedCount: 10,
			expectedError: nil,
		},
		{
			name:       "success with categoryID",
			tagID:      nil,
			categoryID: intPtr(3),
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				assert.Nil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 3, *categoryID)
				return 7, nil
			},
			expectedCount: 7,
			expectedError: nil,
		},
		{
			name:       "success with both filters",
			tagID:      intPtr(1),
			categoryID: intPtr(2),
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				require.NotNil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 1, *tagID)
				assert.Equal(t, 2, *categoryID)
				return 3, nil
			},
			expectedCount: 3,
			expectedError: nil,
		},
		{
			name:       "repository error",
			tagID:      nil,
			categoryID: nil,
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				return 0, errors.New("database error")
			},
			expectedCount: 0,
			expectedError: errors.New("database error"),
		},
		{
			name:       "zero count",
			tagID:      nil,
			categoryID: nil,
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				return 0, nil
			},
			expectedCount: 0,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := &mockPostgresRepository{
				getNewsCountFunc: tt.mockFunc,
			}
			mockRepo := &mockRepository{
				postgresRepo: mockPostgres,
			}

			uc := NewNewsUseCase(mockRepo, logger)
			count, err := uc.GetNewsCount(ctx, tt.tagID, tt.categoryID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Equal(t, 0, count)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

func TestNewsUseCase_GetNewsByID(t *testing.T) {
	logger := noOpLogger()
	ctx := context.Background()
	testTime := time.Now()

	tests := []struct {
		name           string
		newsID         int
		mockFunc       func(ctx context.Context, newsID int) (*postgres.News, error)
		expectedResult *domain.News
		expectedError  error
	}{
		{
			name:   "success",
			newsID: 1,
			mockFunc: func(ctx context.Context, newsID int) (*postgres.News, error) {
				assert.Equal(t, 1, newsID)
				return &postgres.News{
					NewsID:      1,
					CategoryID:  1,
					Title:       "Test News",
					Content:     "Content",
					Author:      "Author",
					PublishedAt: testTime,
					StatusID:    1,
					Category: &postgres.Category{
						CategoryID:  1,
						Title:       "Category",
						OrderNumber: 1,
						StatusID:    1,
					},
					Tags: []postgres.Tag{
						{TagID: 1, Title: "Tag 1", StatusID: 1},
					},
				}, nil
			},
			expectedResult: &domain.News{
				NewsID:      1,
				CategoryID:  1,
				Title:       "Test News",
				Content:     "Content",
				Author:      "Author",
				PublishedAt: testTime,
				StatusID:    1,
				Category: domain.Category{
					CategoryID:  1,
					Title:       "Category",
					OrderNumber: 1,
					StatusID:    1,
				},
				Tags: []domain.Tag{
					{TagID: 1, Title: "Tag 1", StatusID: 1},
				},
			},
			expectedError: nil,
		},
		{
			name:   "not found",
			newsID: 999,
			mockFunc: func(ctx context.Context, newsID int) (*postgres.News, error) {
				return nil, errors.New("news with id 999 not found")
			},
			expectedResult: nil,
			expectedError:  errors.New("news with id 999 not found"),
		},
		{
			name:   "repository error",
			newsID: 1,
			mockFunc: func(ctx context.Context, newsID int) (*postgres.News, error) {
				return nil, errors.New("database error")
			},
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := &mockPostgresRepository{
				getNewsByIDFunc: tt.mockFunc,
			}
			mockRepo := &mockRepository{
				postgresRepo: mockPostgres,
			}

			uc := NewNewsUseCase(mockRepo, logger)
			result, err := uc.GetNewsByID(ctx, tt.newsID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestNewsUseCase_GetAllCategories(t *testing.T) {
	logger := noOpLogger()
	ctx := context.Background()

	tests := []struct {
		name           string
		mockFunc       func(ctx context.Context) ([]postgres.Category, error)
		expectedResult []domain.Category
		expectedError  error
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) ([]postgres.Category, error) {
				return []postgres.Category{
					{
						CategoryID:  1,
						Title:       "Category 1",
						OrderNumber: 1,
						StatusID:    1,
					},
					{
						CategoryID:  2,
						Title:       "Category 2",
						OrderNumber: 2,
						StatusID:    1,
					},
				}, nil
			},
			expectedResult: []domain.Category{
				{
					CategoryID:  1,
					Title:       "Category 1",
					OrderNumber: 1,
					StatusID:    1,
				},
				{
					CategoryID:  2,
					Title:       "Category 2",
					OrderNumber: 2,
					StatusID:    1,
				},
			},
			expectedError: nil,
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context) ([]postgres.Category, error) {
				return []postgres.Category{}, nil
			},
			expectedResult: []domain.Category{},
			expectedError:  nil,
		},
		{
			name: "repository error",
			mockFunc: func(ctx context.Context) ([]postgres.Category, error) {
				return nil, errors.New("database error")
			},
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := &mockPostgresRepository{
				getAllCategoriesFunc: tt.mockFunc,
			}
			mockRepo := &mockRepository{
				postgresRepo: mockPostgres,
			}

			uc := NewNewsUseCase(mockRepo, logger)
			result, err := uc.GetAllCategories(ctx)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestNewsUseCase_GetAllTags(t *testing.T) {
	logger := noOpLogger()
	ctx := context.Background()

	tests := []struct {
		name           string
		mockFunc       func(ctx context.Context) ([]postgres.Tag, error)
		expectedResult []domain.Tag
		expectedError  error
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) ([]postgres.Tag, error) {
				return []postgres.Tag{
					{
						TagID:    1,
						Title:    "Tag 1",
						StatusID: 1,
					},
					{
						TagID:    2,
						Title:    "Tag 2",
						StatusID: 1,
					},
				}, nil
			},
			expectedResult: []domain.Tag{
				{
					TagID:    1,
					Title:    "Tag 1",
					StatusID: 1,
				},
				{
					TagID:    2,
					Title:    "Tag 2",
					StatusID: 1,
				},
			},
			expectedError: nil,
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context) ([]postgres.Tag, error) {
				return []postgres.Tag{}, nil
			},
			expectedResult: []domain.Tag{},
			expectedError:  nil,
		},
		{
			name: "repository error",
			mockFunc: func(ctx context.Context) ([]postgres.Tag, error) {
				return nil, errors.New("database error")
			},
			expectedResult: nil,
			expectedError:  errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPostgres := &mockPostgresRepository{
				getAllTagsFunc: tt.mockFunc,
			}
			mockRepo := &mockRepository{
				postgresRepo: mockPostgres,
			}

			uc := NewNewsUseCase(mockRepo, logger)
			result, err := uc.GetAllTags(ctx)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
