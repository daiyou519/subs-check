package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bestruirui/bestsub/docs"
	"github.com/bestruirui/bestsub/internal/database"
	"github.com/bestruirui/bestsub/internal/handler"
	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/middleware"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/router"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server Wraps HTTP server and dependent components
type Server struct {
	config     *model.Config
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer Creates and configures server instance
// Uses dependency injection mode to receive configuration
func NewServer(cfg *model.Config) *Server {
	router := gin.New()

	router.Use(gin.Recovery())

	if gin.Mode() == gin.ReleaseMode {
		router.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	} else {
		router.SetTrustedProxies(nil)
	}

	router.Use(middleware.Cors())
	router.Use(middleware.RequestLogger())

	return &Server{
		config: cfg,
		router: router,
		httpServer: &http.Server{
			Handler: router,
		},
	}
}

// initDatabase Initializes database connection and schema
func (s *Server) initDatabase() error {
	logger.Info("Initializing database connection...")
	err := database.InitDatabase(s.config.Database.Path)
	if err != nil {
		return fmt.Errorf("database initialization failed: %v", err)
	}
	logger.Info("Database initialized successfully")
	return nil
}

// setupRoutes Registers all HTTP routes and handlers
func (s *Server) setupRoutes() {
	logger.Info("Setting up API routes...")

	userHandler := handler.NewUserHandler(database.DB, s.config)
	systemHandler := handler.NewSystemHandler(s.config)

	router.MustRegisterGroup(s.router, userHandler)
	router.MustRegisterGroup(s.router, systemHandler)

	_ = docs.SwaggerInfo.ReadDoc()

	s.router.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api/swagger/doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.DocExpansion("list"),
		ginSwagger.InstanceName("swagger"),
	))
	logger.Info("Swagger documentation available at /swagger/index.html")

	systemHandler.SetupStaticAssets(s.router)

	logger.Info("Routes registered successfully")
}

// Start Starts HTTP server and handles graceful shutdown
func (s *Server) Start() error {
	if err := s.initDatabase(); err != nil {
		return err
	}

	s.setupRoutes()

	serverAddr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer.Addr = serverAddr

	s.httpServer.ReadTimeout = 10 * time.Second
	s.httpServer.WriteTimeout = 30 * time.Second
	s.httpServer.IdleTimeout = 120 * time.Second

	go s.gracefulShutdown()

	logger.Info("Server started, listening on: %s", serverAddr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

// gracefulShutdown Handles graceful shutdown of server
func (s *Server) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	if err := database.Close(); err != nil {
		logger.Error("Error closing database connection: %v", err)
	}

	logger.Info("Server shutdown completed")
}

// PrintVersion Formats and prints service version information
func PrintVersion(version, buildTime, author string) {
	fmt.Printf(`
 BestSub Backend Service
------------------------
  Version:    %s
  Build Time: %s
  Author:     %s
------------------------

`, version, buildTime, author)
}
