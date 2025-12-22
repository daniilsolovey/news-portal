package delivery

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daniilsolovey/news-portal/internal/usecase"
	"github.com/gin-gonic/gin"
)

// NewsHandler handles HTTP requests
type NewsHandler struct {
	uc  usecase.INewsUseCase
	log *slog.Logger
}

// NewNewsHandler creates a new instance of NewsHandler
func NewNewsHandler(uc usecase.INewsUseCase, log *slog.Logger) *NewsHandler {
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
// @Success 200 {array} domain.NewsSummary
// @Failure 400,500 {object} map[string]string
// @Router /api/v1/all_news [get]
func (h *NewsHandler) GetAllNews(c *gin.Context) {
	var tagID, categoryID *int
	var page, pageSize int = 1, 10

	// Parse optional tagId
	if tagIDStr := c.Query("tagId"); tagIDStr != "" {
		if id, err := strconv.Atoi(tagIDStr); err == nil {
			tagID = &id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tagId"})
			return
		}
	}

	// Parse optional categoryId
	if categoryIDStr := c.Query("categoryId"); categoryIDStr != "" {
		if id, err := strconv.Atoi(categoryIDStr); err == nil {
			categoryID = &id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid categoryId"})
			return
		}
	}

	// Parse optional page
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
	}

	// Parse optional pageSize
	if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			if ps > 100 {
				ps = 100
			}

			pageSize = ps
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pageSize"})
			return
		}
	}

	summaries, err := h.uc.GetAllNews(c.Request.Context(), tagID, categoryID, page, pageSize)
	if err != nil {
		h.log.Error("failed to get all news", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
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
func (h *NewsHandler) GetNewsCount(c *gin.Context) {
	var tagID, categoryID *int

	// Parse optional tagId
	if tagIDStr := c.Query("tagId"); tagIDStr != "" {
		if id, err := strconv.Atoi(tagIDStr); err == nil {
			tagID = &id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tagId"})
			return
		}
	}

	// Parse optional categoryId
	if categoryIDStr := c.Query("categoryId"); categoryIDStr != "" {
		if id, err := strconv.Atoi(categoryIDStr); err == nil {
			categoryID = &id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid categoryId"})
			return
		}
	}

	count, err := h.uc.GetNewsCount(c.Request.Context(), tagID, categoryID)
	if err != nil {
		h.log.Error("failed to get news count", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, count)
}

// GetNewsByID handles GET /api/v1/news/:id
// @Summary Get news by ID
// @Description Retrieves a single news item by ID with full content, category and tags
// @Tags news
// @Produce json
// @Param id path int true "News ID"
// @Success 200 {object} domain.News
// @Failure 400,404,500 {object} map[string]string
// @Router /api/v1/news/{id} [get]
func (h *NewsHandler) GetNewsByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	news, err := h.uc.GetNewsByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get news by ID", "error", err, "id", id)
		// Check if it's a "not found" error
		if err.Error() == "news with id "+idStr+" not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, news)
}

// GetAllCategories handles GET /api/v1/categories
// @Summary Get all categories
// @Description Retrieves all categories ordered by orderNumber
// @Tags categories
// @Produce json
// @Success 200 {array} domain.Category
// @Failure 500 {object} map[string]string
// @Router /api/v1/categories [get]
func (h *NewsHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.uc.GetAllCategories(c.Request.Context())
	if err != nil {
		h.log.Error("failed to get all categories", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// GetAllTags handles GET /api/v1/tags
// @Summary Get all tags
// @Description Retrieves all tags ordered by title
// @Tags tags
// @Produce json
// @Success 200 {array} domain.Tag
// @Failure 500 {object} map[string]string
// @Router /api/v1/tags [get]
func (h *NewsHandler) GetAllTags(c *gin.Context) {
	tags, err := h.uc.GetAllTags(c.Request.Context())
	if err != nil {
		h.log.Error("failed to get all tags", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tags)
}
