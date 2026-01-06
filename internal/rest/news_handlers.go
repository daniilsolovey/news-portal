package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/labstack/echo/v4"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// NewsRequest represents query parameters for News endpoint
type NewsRequest struct {
	TagID      *int `query:"tagId"`
	CategoryID *int `query:"categoryId"`
	Page       *int `query:"page"`
	PageSize   *int `query:"pageSize"`
}

// NewsCountRequest represents query parameters for NewsCount endpoint
type NewsCountRequest struct {
	TagID      *int `query:"tagId"`
	CategoryID *int `query:"categoryId"`
}

// NewsHandler handles HTTP requests
type NewsHandler struct {
	uc  *newsportal.Manager
	log *slog.Logger
}

// NewNewsHandler creates a new instance of NewsHandler
func NewNewsHandler(uc *newsportal.Manager, log *slog.Logger) *NewsHandler {
	return &NewsHandler{
		uc:  uc,
		log: log,
	}
}

// News handles GET /api/v1/all_news
// @Summary Get all news
// @Description Retrieves news with optional filtering by tagId and categoryId, with pagination. Returns NewsSummary (without content) sorted by publishedAt DESC
// @Tags news
// @Produce json
// @Param tagId query int false "Filter by tag ID"
// @Param categoryId query int false "Filter by category ID"
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Page size (default: 10)"
// @Success 200 {array} rest.NewsSummary
// @Failure 400,500 {object} map[string]string
// @Router /api/v1/all_news [get]
func (h *NewsHandler) News(c echo.Context) error {
	var req NewsRequest
	if err := c.Bind(&req); err != nil {
		h.log.Warn("News: failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request parameters"})
	}

	h.log.Info("News request",
		"tagId", req.TagID,
		"categoryId", req.CategoryID,
		"page", req.Page,
		"pageSize", req.PageSize,
	)

	page := defaultPage
	if req.Page != nil {
		if *req.Page <= 0 {
			h.log.Warn("News: invalid page", "page", *req.Page)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid page"})
		}
		page = *req.Page
	}

	pageSize := defaultPageSize
	if req.PageSize != nil {
		if *req.PageSize <= 0 {
			h.log.Warn("News: invalid pageSize", "pageSize", *req.PageSize)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid pageSize"})
		}
		pageSize = *req.PageSize
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
	}

	newsportalSummaries, err := h.uc.NewsByFilter(c.Request().Context(), req.TagID,
		req.CategoryID, page, pageSize,
	)
	if err != nil {
		h.log.Error("News: failed to get all news",
			"error", err,
			"tagId", req.TagID,
			"categoryId", req.CategoryID,
			"page", page,
			"pageSize", pageSize,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	summaries := make([]News, len(newsportalSummaries))
	for i := range newsportalSummaries {
		summaries[i] = NewNewsSummary(newsportalSummaries[i])
	}

	h.log.Info("News: success",
		"count", len(summaries),
		"tagId", req.TagID,
		"categoryId", req.CategoryID,
		"page", page,
		"pageSize", pageSize,
	)

	return c.JSON(http.StatusOK, summaries)
}

// NewsCount handles GET /api/v1/count
// @Summary Get news count
// @Description Returns the count of news matching the optional tagId and categoryId filters
// @Tags news
// @Produce json
// @Param tagId query int false "Filter by tag ID"
// @Param categoryId query int false "Filter by category ID"
// @Success 200 {integer} int
// @Failure 400,500 {object} map[string]string
// @Router /api/v1/count [get]
func (h *NewsHandler) NewsCount(c echo.Context) error {
	var req NewsCountRequest
	if err := c.Bind(&req); err != nil {
		h.log.Warn("NewsCount: failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request parameters"})
	}

	h.log.Info("NewsCount request",
		"tagId", req.TagID,
		"categoryId", req.CategoryID,
	)

	count, err := h.uc.NewsCount(c.Request().Context(), req.TagID, req.CategoryID)
	if err != nil {
		h.log.Error("NewsCount: failed to get news count",
			"error", err,
			"tagId", req.TagID,
			"categoryId", req.CategoryID,
		)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	h.log.Info("NewsCount: success",
		"count", count,
		"tagId", req.TagID,
		"categoryId", req.CategoryID,
	)

	return c.JSON(http.StatusOK, count)
}

// NewsByID handles GET /api/v1/news/:id
// @Summary Get news by ID
// @Description Retrieves a single news item by ID with full content, category and tags
// @Tags news
// @Produce json
// @Param id path int true "News ID"
// @Success 200 {object} rest.News
// @Failure 400,404,500 {object} map[string]string
// @Router /api/v1/news/{id} [get]
func (h *NewsHandler) NewsByID(c echo.Context) error {
	idStr := c.Param("id")
	h.log.Info("NewsByID request", "id", idStr)

	if idStr == "" {
		h.log.Warn("NewsByID: empty id")
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid id"},
		)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.log.Warn("NewsByID: invalid id format", "id", idStr, "error", err)
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid id"},
		)
	}

	newsportalNews, err := h.uc.NewsByID(c.Request().Context(), id)
	if err != nil {
		h.log.Error("NewsByID: failed to get news by ID",
			"error", err,
			"id", id,
		)
		return err
	} else if newsportalNews == nil {
		h.log.Info("NewsByID: news not found", "id", id)
		return c.String(http.StatusNotFound, "news not found")
	}

	news := NewNews(*newsportalNews)

	h.log.Info("NewsByID: success", "id", id, "title", newsportalNews.Title)

	return c.JSON(http.StatusOK, news)
}

// Categories handles GET /api/v1/categories
// @Summary Get all categories
// @Description Retrieves all categories ordered by orderNumber
// @Tags categories
// @Produce json
// @Success 200 {array} rest.Category
// @Failure 500 {object} map[string]string
// @Router /api/v1/categories [get]
func (h *NewsHandler) Categories(c echo.Context) error {
	h.log.Info("Categories request")

	newsportalCategories, err := h.uc.Categories(c.Request().Context())
	if err != nil {
		h.log.Error("Categories: failed to get all categories", "error", err)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	categories := make([]Category, len(newsportalCategories))
	for i := range newsportalCategories {
		categories[i] = NewCategory(newsportalCategories[i])
	}

	h.log.Info("Categories: success", "count", len(categories))

	return c.JSON(http.StatusOK, categories)
}

// Tags handles GET /api/v1/tags
// @Summary Get all tags
// @Description Retrieves all tags ordered by title
// @Tags tags
// @Produce json
// @Success 200 {array} rest.Tag
// @Failure 500 {object} map[string]string
// @Router /api/v1/tags [get]
func (h *NewsHandler) Tags(c echo.Context) error {
	h.log.Info("Tags request")

	newsportalTags, err := h.uc.Tags(c.Request().Context())
	if err != nil {
		h.log.Error("Tags: failed to get all tags", "error", err)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	tags := make([]Tag, len(newsportalTags))
	for i := range newsportalTags {
		tags[i] = NewTag(newsportalTags[i])
	}

	h.log.Info("Tags: success", "count", len(tags))

	return c.JSON(http.StatusOK, tags)
}
