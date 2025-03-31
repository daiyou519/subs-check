package handler

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/router"
	"github.com/bestruirui/bestsub/web"
	"github.com/gin-gonic/gin"
)

// SystemHandler
type SystemHandler struct {
	config *model.Config
	fsRoot fs.FS
}

// NewSystemHandler Creates system handler instance
func NewSystemHandler(config *model.Config) *SystemHandler {
	// Get sub filesystem from embedded file system
	subFS, err := fs.Sub(web.Web, "out")
	if err != nil {
		logger.Error("Failed to get sub filesystem: %v", err)
	}

	return &SystemHandler{
		config: config,
		fsRoot: subFS,
	}
}

// Groups Returns all route group configurations
func (h *SystemHandler) Groups() []*router.GroupRouter {
	return []*router.GroupRouter{
		h.SystemGroup(),
	}
}

// SystemGroup Returns system related API route group
func (h *SystemHandler) SystemGroup() *router.GroupRouter {
	// 使用链式API创建路由组
	return router.NewGroupRouter("/api").
		AddRoute(
			router.NewRoute("/health", router.GET).
				Handle(h.HealthCheck).
				WithDescription("Health check endpoint"),
		)
}

// HealthCheck godoc
// @Summary Health check
// @Description Get server health status
// @Tags System
// @Produce json
// @Success 200 {object} object{status=string,time=string} "Server is healthy"
// @Router /api/health [get]
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// SetupStaticAssets Sets up frontend static asset handling
func (h *SystemHandler) SetupStaticAssets(router *gin.Engine) {
	if h.fsRoot == nil {
		logger.Error("Static assets not available")
		return
	}

	logger.Info("Setting up static assets...")

	// Create file system HTTP handler
	fileServer := http.FileServer(http.FS(h.fsRoot))

	// Register static asset handling
	router.GET("/", func(c *gin.Context) {
		c.Request.URL.Path = "/index.html"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// Static asset handling
	router.NoRoute(func(c *gin.Context) {
		// If API path, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "API endpoint not found",
			})
			return
		}

		// Handle frontend routing
		ext := path.Ext(c.Request.URL.Path)
		if ext == "" {
			// Path without extension is considered frontend routing, return index.html
			c.Request.URL.Path = "/index.html"
		} else if ext != ".html" {
			// For static resource requests, keep the path unchanged
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	logger.Info("Static assets registered successfully")
}
