package delivery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daniilsolovey/news-portal/internal/usecase"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
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
func (h *NewsHandler) GetAllNews(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	tagID, err := parseOptionalInt(q.Get("tagId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tagId")
		return
	}

	categoryID, err := parseOptionalInt(q.Get("categoryId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid categoryId")
		return
	}

	page, err := parsePositiveIntOrDefault(q.Get("page"), defaultPage)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid page")
		return
	}

	pageSize, err := parsePositiveIntOrDefault(q.Get("pageSize"), defaultPageSize)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid pageSize")
		return
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	summaries, err := h.uc.GetAllNews(r.Context(), tagID, categoryID, page, pageSize)
	if err != nil {
		h.log.Error("failed to get all news", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := writeJSON(w, http.StatusOK, summaries); err != nil {
		h.log.Warn("failed to write json response", "error", err)
	}
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
func (h *NewsHandler) GetNewsCount(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	tagID, err := parseOptionalInt(q.Get("tagId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tagId")
		return
	}

	categoryID, err := parseOptionalInt(q.Get("categoryId"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid categoryId")
		return
	}

	count, err := h.uc.GetNewsCount(r.Context(), tagID, categoryID)
	if err != nil {
		h.log.Error("failed to get news count", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := writeJSON(w, http.StatusOK, count); err != nil {
		h.log.Warn("failed to write json response", "error", err)
	}
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
func (h *NewsHandler) GetNewsByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path /api/v1/news/{id}
	idStr := r.PathValue("id")
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	news, err := h.uc.GetNewsByID(r.Context(), id)
	if err != nil {
		h.log.Error("failed to get news by ID", "error", err, "id", id)
		// TODO:Check if not found record error
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := writeJSON(w, http.StatusOK, news); err != nil {
		h.log.Warn("failed to write json response", "error", err)
	}
}

// GetAllCategories handles GET /api/v1/categories
// @Summary Get all categories
// @Description Retrieves all categories ordered by orderNumber
// @Tags categories
// @Produce json
// @Success 200 {array} domain.Category
// @Failure 500 {object} map[string]string
// @Router /api/v1/categories [get]
func (h *NewsHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.uc.GetAllCategories(r.Context())
	if err != nil {
		h.log.Error("failed to get all categories", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := writeJSON(w, http.StatusOK, categories); err != nil {
		h.log.Warn("failed to write json response", "error", err)
	}
}

// GetAllTags handles GET /api/v1/tags
// @Summary Get all tags
// @Description Retrieves all tags ordered by title
// @Tags tags
// @Produce json
// @Success 200 {array} domain.Tag
// @Failure 500 {object} map[string]string
// @Router /api/v1/tags [get]
func (h *NewsHandler) GetAllTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.uc.GetAllTags(r.Context())
	if err != nil {
		h.log.Error("failed to get all tags", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := writeJSON(w, http.StatusOK, tags); err != nil {
		h.log.Warn("failed to write json response", "error", err)
	}
}

// helpers: ------------------------------------------------------------------

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

func writeJSON(w http.ResponseWriter, status int, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_, err = w.Write(data)
	return err
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
