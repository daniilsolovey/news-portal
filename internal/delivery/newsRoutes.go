package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes registers all routes for the handler
func (h *NewsHandler) RegisterRoutes() *gin.Engine {
	r := gin.Default()

	// Serve static files from frontend directory
	r.Static("/static", "./frontend")
	r.StaticFile("/", "./frontend/index.html")
	r.StaticFile("/index.html", "./frontend/index.html")

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/all_news", h.GetAllNews)
		api.GET("/count", h.GetNewsCount)
		api.GET("/news/:id", h.GetNewsByID)
		api.GET("/categories", h.GetAllCategories)
		api.GET("/tags", h.GetAllTags)
	}

	return r
}
