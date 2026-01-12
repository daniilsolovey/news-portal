package rest

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daniilsolovey/news-portal/internal/newsportal"
	"github.com/labstack/echo/v4"
)

type NewsRequest struct {
	TagID      *int `query:"tagId"`
	CategoryID *int `query:"categoryId"`
	Page       *int `query:"page"`
	PageSize   *int `query:"pageSize"`
}

type NewsCountRequest struct {
	TagID      *int `query:"tagId"`
	CategoryID *int `query:"categoryId"`
}

type NewsHandler struct {
	uc  *newsportal.Manager
	log *slog.Logger
}

func NewNewsHandler(uc *newsportal.Manager, log *slog.Logger) *NewsHandler {
	return &NewsHandler{
		uc:  uc,
		log: log,
	}
}

func (h *NewsHandler) handleError(c echo.Context, err error, statusCode int, message string) error {
	h.log.Error("handleError", "error", err, "statusCode", statusCode, "message", message)
	return c.JSON(statusCode, map[string]string{"error": message})
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
		return h.handleError(c, err, http.StatusBadRequest, "invalid request parameters")
	}

	newsportalSummaries, err := h.uc.NewsByFilter(
		c.Request().Context(), req.TagID, req.CategoryID, req.Page, req.PageSize,
	)
	if err != nil {
		return h.handleError(c, err, http.StatusInternalServerError, "internal error")
	}

	summaries := NewNewsSummaries(newsportalSummaries)

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
		return h.handleError(c, err, http.StatusBadRequest, "invalid request parameters")
	}

	count, err := h.uc.NewsCount(c.Request().Context(), req.TagID, req.CategoryID)
	if err != nil {
		return h.handleError(c, err, http.StatusInternalServerError, "internal error")
	}

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
	if idStr == "" {
		return h.handleError(c, nil, http.StatusBadRequest, "invalid id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return h.handleError(c, err, http.StatusBadRequest, "invalid id")
	}

	newsportalNews, err := h.uc.NewsByID(c.Request().Context(), id)
	if err != nil {
		return h.handleError(c, err, http.StatusInternalServerError, "internal error")
	}
	if newsportalNews == nil {
		return c.String(http.StatusNotFound, "news not found")
	}

	return c.JSON(http.StatusOK, NewNews(*newsportalNews))
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
	categories, err := h.uc.Categories(c.Request().Context())
	if err != nil {
		return h.handleError(c, err, http.StatusInternalServerError, "internal error")
	}

	result := NewCategories(categories)
	return c.JSON(http.StatusOK, result)
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
	tags, err := h.uc.Tags(c.Request().Context())
	if err != nil {
		return h.handleError(c, err, http.StatusInternalServerError, "internal error")
	}

	result := NewTags(tags)
	return c.JSON(http.StatusOK, result)
}
