package rest

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const (
	// API paths
	apiV1Prefix = "/api/v1"

	allNewsPath    = apiV1Prefix + "/all_news"
	countPath      = apiV1Prefix + "/count"
	newsByIDPath   = apiV1Prefix + "/news/{id}"
	categoriesPath = apiV1Prefix + "/categories"
	tagsPath       = apiV1Prefix + "/tags"

	// Health check paths
	staticPathPrefix = "/static/"
	healthPath       = "/health"

	// Frontend paths
	frontendDir     = "./frontend"
	indexHTML       = "index.html"
	contentTypeJSON = "application/json"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// RegisterRoutes registers all routes for the handler
func (h *NewsHandler) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	h.registerAPIRoutes(mux)

	h.registerHealthCheck(mux)

	h.registerStaticRoutes(mux)

	return h.loggingMiddleware(mux)
}

func (h *NewsHandler) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc(allNewsPath, h.requireMethod(http.MethodGet, h.GetAllNews))
	mux.HandleFunc(countPath, h.requireMethod(http.MethodGet, h.GetNewsCount))
	mux.HandleFunc(newsByIDPath, h.requireMethod(http.MethodGet, h.GetNewsByID))
	mux.HandleFunc(categoriesPath, h.requireMethod(http.MethodGet, h.GetAllCategories))
	mux.HandleFunc(tagsPath, h.requireMethod(http.MethodGet, h.GetAllTags))
}

func (h *NewsHandler) registerHealthCheck(mux *http.ServeMux) {
	mux.HandleFunc(healthPath, h.requireMethod(http.MethodGet, h.handleHealth))
}

func (h *NewsHandler) registerStaticRoutes(mux *http.ServeMux) {
	staticFS := http.Dir(frontendDir)
	fileServer := http.FileServer(staticFS)
	mux.Handle(staticPathPrefix, http.StripPrefix(staticPathPrefix, fileServer))
	mux.HandleFunc("/", h.handleFrontend)
}

func (h *NewsHandler) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p := r.URL.Path
	if p == "/" || p == "/index.html" {
		p = indexHTML
	}

	p = strings.TrimPrefix(p, "/")
	filePath := filepath.Join(frontendDir, p)

	http.ServeFile(w, r, filePath)
}

func (h *NewsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentTypeJSON)
	if err := json.NewEncoder(w).Encode(
		map[string]string{"status": "ok"},
	); err != nil {
		h.log.Error("failed to encode health response", "error", err)
	}
}

// helpers TODO: move to separate file

func (h *NewsHandler) requireMethod(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handler(w, r)
	}
}

func (h *NewsHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		status := rw.statusCode
		if status == 0 {
			status = http.StatusOK
		}

		h.log.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
