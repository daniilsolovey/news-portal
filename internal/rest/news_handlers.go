package rest

import (
	"fmt"
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

// GetAllNews handles GET /api/v1/all_news
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
func (h *NewsHandler) GetAllNews(c echo.Context) error {
	tagID, err := parseOptionalInt(c.QueryParam("tagId"))
	if err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{"error": "invalid tagId"},
		)
	}

	categoryID, err := parseOptionalInt(c.QueryParam("categoryId"))
	if err != nil {
		return c.JSON(
			http.StatusBadRequest,
			map[string]string{"error": "invalid categoryId"},
		)
	}

	page, err := parsePositiveIntOrDefault(c.QueryParam("page"), defaultPage)
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid page"},
		)
	}

	pageSize, err := parsePositiveIntOrDefault(c.QueryParam("pageSize"), defaultPageSize)
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid pageSize"},
		)
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	newsportalSummaries, err := h.uc.NewsByFilter(c.Request().Context(), tagID,
		categoryID, page, pageSize,
	)
	if err != nil {
		h.log.Error("failed to get all news", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	summaries := make([]News, len(newsportalSummaries))
	for i := range newsportalSummaries {
		summaries[i] = NewNewsSummary(newsportalSummaries[i])
	}

	return c.JSON(http.StatusOK, summaries)
}

// GetNewsCount handles GET /api/v1/count
// @Summary Get news count
// @Description Returns the count of news matching the optional tagId and categoryId filters
// @Tags news
// @Produce json
// @Param tagId query int false "Filter by tag ID"
// @Param categoryId query int false "Filter by category ID"
// @Success 200 {integer} int
// @Failure 400,500 {object} map[string]string
// @Router /api/v1/count [get]
func (h *NewsHandler) GetNewsCount(c echo.Context) error {
	tagID, err := parseOptionalInt(c.QueryParam("tagId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid tagId"},
		)
	}

	categoryID, err := parseOptionalInt(c.QueryParam("categoryId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid categoryId"},
		)
	}

	count, err := h.uc.NewsCount(c.Request().Context(), tagID, categoryID)
	if err != nil {
		h.log.Error("failed to get news count", "error", err)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	return c.JSON(http.StatusOK, count)
}

// GetNewsByID handles GET /api/v1/news/:id
// @Summary Get news by ID
// @Description Retrieves a single news item by ID with full content, category and tags
// @Tags news
// @Produce json
// @Param id path int true "News ID"
// @Success 200 {object} rest.News
// @Failure 400,404,500 {object} map[string]string
// @Router /api/v1/news/{id} [get]
func (h *NewsHandler) GetNewsByID(c echo.Context) error {
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid id"},
		)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest,
			map[string]string{"error": "invalid id"},
		)
	}

	newsportalNews, err := h.uc.NewsByID(c.Request().Context(), id)
	if err != nil {
		h.log.Error("failed to get news by ID", "error", err, "id", id)
		return err
	} else if newsportalNews == nil {
		return c.String(http.StatusNotFound, "news not found")
	}

	news := NewNews(*newsportalNews)

	return c.JSON(http.StatusOK, news)
}

// GetAllCategories handles GET /api/v1/categories
// @Summary Get all categories
// @Description Retrieves all categories ordered by orderNumber
// @Tags categories
// @Produce json
// @Success 200 {array} rest.Category
// @Failure 500 {object} map[string]string
// @Router /api/v1/categories [get]
func (h *NewsHandler) GetAllCategories(c echo.Context) error {
	newsportalCategories, err := h.uc.Categories(c.Request().Context())
	if err != nil {
		h.log.Error("failed to get all categories", "error", err)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	categories := make([]Category, len(newsportalCategories))
	for i := range newsportalCategories {
		categories[i] = NewCategory(newsportalCategories[i])
	}

	return c.JSON(http.StatusOK, categories)
}

// GetAllTags handles GET /api/v1/tags
// @Summary Get all tags
// @Description Retrieves all tags ordered by title
// @Tags tags
// @Produce json
// @Success 200 {array} rest.Tag
// @Failure 500 {object} map[string]string
// @Router /api/v1/tags [get]
func (h *NewsHandler) GetAllTags(c echo.Context) error {
	newsportalTags, err := h.uc.Tags(c.Request().Context())
	if err != nil {
		h.log.Error("failed to get all tags", "error", err)
		return c.JSON(http.StatusInternalServerError,
			map[string]string{"error": "internal error"},
		)
	}

	tags := make([]Tag, len(newsportalTags))
	for i := range newsportalTags {
		tags[i] = NewTag(newsportalTags[i])
	}

	return c.JSON(http.StatusOK, tags)
}

func parseOptionalInt(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func parsePositiveIntOrDefault(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("must be positive int")
	}
	return v, nil
}
