package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daniilsolovey/news-portal/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNewsUseCase is a manual stub implementation of INewsUseCase for testing
type mockNewsUseCase struct {
	getAllNewsFunc       func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error)
	getNewsCountFunc     func(ctx context.Context, tagID, categoryID *int) (int, error)
	getNewsByIDFunc      func(ctx context.Context, newsID int) (*domain.News, error)
	getAllCategoriesFunc func(ctx context.Context) ([]domain.Category, error)
	getAllTagsFunc       func(ctx context.Context) ([]domain.Tag, error)
}

func (m *mockNewsUseCase) GetAllNews(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
	if m.getAllNewsFunc != nil {
		return m.getAllNewsFunc(ctx, tagID, categoryID, page, pageSize)
	}
	return nil, nil
}

func (m *mockNewsUseCase) GetNewsCount(ctx context.Context, tagID, categoryID *int) (int, error) {
	if m.getNewsCountFunc != nil {
		return m.getNewsCountFunc(ctx, tagID, categoryID)
	}
	return 0, nil
}

func (m *mockNewsUseCase) GetNewsByID(ctx context.Context, newsID int) (*domain.News, error) {
	if m.getNewsByIDFunc != nil {
		return m.getNewsByIDFunc(ctx, newsID)
	}
	return nil, nil
}

func (m *mockNewsUseCase) GetAllCategories(ctx context.Context) ([]domain.Category, error) {
	if m.getAllCategoriesFunc != nil {
		return m.getAllCategoriesFunc(ctx)
	}
	return nil, nil
}

func (m *mockNewsUseCase) GetAllTags(ctx context.Context) ([]domain.Tag, error) {
	if m.getAllTagsFunc != nil {
		return m.getAllTagsFunc(ctx)
	}
	return nil, nil
}

func setupTestRouter(handler *NewsHandler) http.Handler {
	return handler.RegisterRoutes()
}

func TestNewsHandler_GetAllNews(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		queryParams    string
		mockFunc       func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:        "success without filters",
			queryParams: "",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				assert.Nil(t, tagID)
				assert.Nil(t, categoryID)
				assert.Equal(t, 1, page)
				assert.Equal(t, 10, pageSize)
				return []domain.NewsSummary{
					{
						NewsID:      1,
						CategoryID:  1,
						Title:       "Test News",
						Author:      "Author",
						PublishedAt: time.Now(),
						StatusID:    1,
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success with tagId filter",
			queryParams: "?tagId=5",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				require.NotNil(t, tagID)
				assert.Equal(t, 5, *tagID)
				assert.Nil(t, categoryID)
				return []domain.NewsSummary{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success with categoryId filter",
			queryParams: "?categoryId=3",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				assert.Nil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 3, *categoryID)
				return []domain.NewsSummary{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success with pagination",
			queryParams: "?page=2&pageSize=20",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				assert.Equal(t, 2, page)
				assert.Equal(t, 20, pageSize)
				return []domain.NewsSummary{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "success with all filters",
			queryParams: "?tagId=1&categoryId=2&page=3&pageSize=15",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				require.NotNil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 1, *tagID)
				assert.Equal(t, 2, *categoryID)
				assert.Equal(t, 3, page)
				assert.Equal(t, 15, pageSize)
				return []domain.NewsSummary{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid tagId",
			queryParams:    "?tagId=abc",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid tagId"},
		},
		{
			name:           "invalid categoryId",
			queryParams:    "?categoryId=xyz",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid categoryId"},
		},
		{
			name:           "invalid page",
			queryParams:    "?page=0",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid page"},
		},
		{
			name:           "invalid pageSize",
			queryParams:    "?pageSize=-1",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid pageSize"},
		},
		{
			name:        "pageSize capped at 100",
			queryParams: "?pageSize=200",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				assert.Equal(t, 100, pageSize)
				return []domain.NewsSummary{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "usecase error",
			queryParams: "",
			mockFunc: func(ctx context.Context, tagID, categoryID *int, page, pageSize int) ([]domain.NewsSummary, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockNewsUseCase{
				getAllNewsFunc: tt.mockFunc,
			}
			handler := NewNewsHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/all_news"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.EqualValues(t, tt.expectedBody, response)
			} else if tt.expectedStatus == http.StatusOK {
				// Verify JSON is valid
				var summaries []domain.NewsSummary
				err := json.Unmarshal(w.Body.Bytes(), &summaries)
				require.NoError(t, err)
			}
		})
	}
}

func TestNewsHandler_GetNewsCount(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		queryParams    string
		mockFunc       func(ctx context.Context, tagID, categoryID *int) (int, error)
		expectedStatus int
		expectedCount  int
		expectedBody   interface{}
	}{
		{
			name:        "success without filters",
			queryParams: "",
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				assert.Nil(t, tagID)
				assert.Nil(t, categoryID)
				return 42, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  42,
		},
		{
			name:        "success with tagId",
			queryParams: "?tagId=5",
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				require.NotNil(t, tagID)
				assert.Equal(t, 5, *tagID)
				return 10, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  10,
		},
		{
			name:        "success with categoryId",
			queryParams: "?categoryId=3",
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				require.NotNil(t, categoryID)
				assert.Equal(t, 3, *categoryID)
				return 7, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  7,
		},
		{
			name:        "success with both filters",
			queryParams: "?tagId=1&categoryId=2",
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				require.NotNil(t, tagID)
				require.NotNil(t, categoryID)
				assert.Equal(t, 1, *tagID)
				assert.Equal(t, 2, *categoryID)
				return 3, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
		},
		{
			name:           "invalid tagId",
			queryParams:    "?tagId=abc",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid tagId"},
		},
		{
			name:           "invalid categoryId",
			queryParams:    "?categoryId=xyz",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid categoryId"},
		},
		{
			name:        "usecase error",
			queryParams: "",
			mockFunc: func(ctx context.Context, tagID, categoryID *int) (int, error) {
				return 0, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockNewsUseCase{
				getNewsCountFunc: tt.mockFunc,
			}
			handler := NewNewsHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/count"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.EqualValues(t, tt.expectedBody, response)
			} else if tt.expectedStatus == http.StatusOK {
				var count int
				err := json.Unmarshal(w.Body.Bytes(), &count)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}
		})
	}
}

func TestNewsHandler_GetNewsByID(t *testing.T) {
	logger := slog.Default()
	testTime := time.Now()

	tests := []struct {
		name           string
		pathID         string
		mockFunc       func(ctx context.Context, newsID int) (*domain.News, error)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "success",
			pathID: "1",
			mockFunc: func(ctx context.Context, newsID int) (*domain.News, error) {
				assert.Equal(t, 1, newsID)
				return &domain.News{
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
					Tags: []domain.Tag{},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid id format",
			pathID:         "abc",
			mockFunc:       nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]interface{}{"error": "invalid id"},
		},
		{
			name:   "not found",
			pathID: "999",
			mockFunc: func(ctx context.Context, newsID int) (*domain.News, error) {
				return nil, errors.New("news with id 999 not found")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
		{
			name:   "usecase error",
			pathID: "1",
			mockFunc: func(ctx context.Context, newsID int) (*domain.News, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockNewsUseCase{
				getNewsByIDFunc: tt.mockFunc,
			}
			handler := NewNewsHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/news/"+tt.pathID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.EqualValues(t, tt.expectedBody, response)
			} else if tt.expectedStatus == http.StatusOK {
				var news domain.News
				err := json.Unmarshal(w.Body.Bytes(), &news)
				require.NoError(t, err)
				assert.Equal(t, 1, news.NewsID)
			}
		})
	}
}

func TestNewsHandler_GetAllCategories(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		mockFunc       func(ctx context.Context) ([]domain.Category, error)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) ([]domain.Category, error) {
				return []domain.Category{
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
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context) ([]domain.Category, error) {
				return []domain.Category{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "usecase error",
			mockFunc: func(ctx context.Context) ([]domain.Category, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockNewsUseCase{
				getAllCategoriesFunc: tt.mockFunc,
			}
			handler := NewNewsHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.EqualValues(t, tt.expectedBody, response)
			} else if tt.expectedStatus == http.StatusOK {
				var categories []domain.Category
				err := json.Unmarshal(w.Body.Bytes(), &categories)
				require.NoError(t, err)
				if tt.name == "success" {
					assert.Len(t, categories, 2)
				} else {
					assert.Len(t, categories, 0)
				}
			}
		})
	}
}

func TestNewsHandler_GetAllTags(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		mockFunc       func(ctx context.Context) ([]domain.Tag, error)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "success",
			mockFunc: func(ctx context.Context) ([]domain.Tag, error) {
				return []domain.Tag{
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
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty list",
			mockFunc: func(ctx context.Context) ([]domain.Tag, error) {
				return []domain.Tag{}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "usecase error",
			mockFunc: func(ctx context.Context) ([]domain.Tag, error) {
				return nil, errors.New("database error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]interface{}{"error": "internal error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := &mockNewsUseCase{
				getAllTagsFunc: tt.mockFunc,
			}
			handler := NewNewsHandler(mockUC, logger)
			router := setupTestRouter(handler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.EqualValues(t, tt.expectedBody, response)
			} else if tt.expectedStatus == http.StatusOK {
				var tags []domain.Tag
				err := json.Unmarshal(w.Body.Bytes(), &tags)
				require.NoError(t, err)
				if tt.name == "success" {
					assert.Len(t, tags, 2)
				} else {
					assert.Len(t, tags, 0)
				}
			}
		})
	}
}
