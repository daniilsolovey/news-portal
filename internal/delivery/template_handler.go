package delivery

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/daniilsolovey/news-portal/internal/usecase"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// TemplateHandler handles HTTP requests
// Replace TemplateHandler with your actual handler name
type TemplateHandler struct {
	uc  *usecase.TemplateUseCase
	log *slog.Logger
}

// NewTemplateHandler creates a new instance of TemplateHandler
// Replace NewTemplateHandler with your actual constructor name
func NewTemplateHandler(uc *usecase.TemplateUseCase, log *slog.Logger) *TemplateHandler {
	return &TemplateHandler{
		uc:  uc,
		log: log,
	}
}

// RegisterRoutes registers all routes for the handler
func (h *TemplateHandler) RegisterRoutes() *gin.Engine {
	r := gin.Default()

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Replace "entities" with your actual resource name
		entities := api.Group("/entities")
		{
			entities.POST("", h.CreateEntity)
			entities.GET("/:id", h.GetEntity)
			entities.GET("", h.ListEntities)
		}
	}

	return r
}

// CreateEntity handles POST /api/v1/entities
// @Summary Create entity
// @Description Creates a new entity
// @Tags entities
// @Accept json
// @Produce json
// @Param request body map[string]string true "Entity data"
// @Success 201 {object} map[string]interface{}
// @Failure 400,500 {object} map[string]string
// @Router /api/v1/entities [post]
func (h *TemplateHandler) CreateEntity(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Extract data from request
	name, ok := req["name"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := h.uc.CreateEntity(c.Request.Context(), name); err != nil {
		h.log.Error("failed to create entity", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "entity created"})
}

// GetEntity handles GET /api/v1/entities/:id
// @Summary Get entity
// @Description Retrieves an entity by ID
// @Tags entities
// @Produce json
// @Param id path int true "Entity ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400,404,500 {object} map[string]string
// @Router /api/v1/entities/{id} [get]
func (h *TemplateHandler) GetEntity(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	entity, err := h.uc.GetEntity(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get entity", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if entity == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
		return
	}

	c.JSON(http.StatusOK, entity)
}

// ListEntities handles GET /api/v1/entities
// @Summary List entities
// @Description Retrieves all entities
// @Tags entities
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /api/v1/entities [get]
func (h *TemplateHandler) ListEntities(c *gin.Context) {
	entities, err := h.uc.ListEntities(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list entities", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": entities})
}

