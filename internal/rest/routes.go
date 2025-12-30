package rest

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// Frontend paths
	frontendDir = "./frontend"
	indexHTML   = "index.html"
)

func (h *NewsHandler) RegisterRoutes() *echo.Echo {
	e := echo.New()

	// Middleware
	e.Use(h.loggingMiddleware)
	e.Use(middleware.Recover())

	// API routes
	h.registerAPIRoutes(e)

	// Health check
	h.registerHealthCheck(e)

	// Frontend routes
	h.registerStaticRoutes(e)

	return e
}

func (h *NewsHandler) registerAPIRoutes(e *echo.Echo) {
	e.GET("/api/v1/all_news", h.GetAllNews)
	e.GET("/api/v1/count", h.GetNewsCount)
	e.GET("/api/v1/news/:id", h.GetNewsByID)
	e.GET("/api/v1/categories", h.GetAllCategories)
	e.GET("/api/v1/tags", h.GetAllTags)
}

func (h *NewsHandler) registerHealthCheck(e *echo.Echo) {
	e.GET("/health", h.handleHealth)
}

func (h *NewsHandler) registerStaticRoutes(e *echo.Echo) {
	e.Static("/static", frontendDir)
	e.GET("/*", h.handleFrontend)
}

func (h *NewsHandler) handleFrontend(c echo.Context) error {
	if c.Request().Method != http.MethodGet {
		return echo.ErrMethodNotAllowed
	}

	p := c.Request().URL.Path
	if p == "/" || p == "/index.html" {
		p = indexHTML
	}

	p = strings.TrimPrefix(p, "/")
	filePath := filepath.Join(frontendDir, p)

	return c.File(filePath)
}

func (h *NewsHandler) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *NewsHandler) loggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		err := next(c)

		duration := time.Since(start)
		status := c.Response().Status
		if status == 0 {
			status = http.StatusOK
		}

		h.log.Info("HTTP request",
			"method", c.Request().Method,
			"path", c.Request().URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", c.Request().RemoteAddr,
		)

		return err
	}
}
