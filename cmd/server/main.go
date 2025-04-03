package main

import (
	"flag"
	"os"
	"path/filepath"

	_ "github.com/bestruirui/bestsub/docs"
	"github.com/bestruirui/bestsub/internal/config"
	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/server"
	"github.com/gin-gonic/gin"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	Author    = "bestruirui"
)

// @title BestSub API
// @version 1.0
// @description BestSub API server
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 请在值前加上 "Bearer " 前缀，例如："Bearer abcde12345"
func main() {
	configPath := flag.String("f", "", "Configuration file path, default is ./data/config.json")
	version := flag.Bool("version", false, "Display version information")
	port := flag.Int("port", 0, "Specify server port, overrides config file")
	flag.Parse()

	if *version {
		server.PrintVersion(Version, BuildTime, Author)
		return
	}

	if *configPath == "" {
		execPath, err := os.Executable()
		if err != nil {
			logger.Error("Failed to get program path: %v", err)
		}
		execDir := filepath.Dir(execPath)
		*configPath = filepath.Join(execDir, "data", "config.json")
	}

	server.PrintVersion(Version, BuildTime, Author)

	if os.Getenv("GIN_MODE") == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Configuration loading failed: %s", err)
	}

	if _, err := os.Stat(*configPath); err == nil {
		absPath, _ := filepath.Abs(*configPath)
		logger.Info("Using configuration file: %s", absPath)
	}

	if *port > 0 {
		cfg.Server.Port = *port
		logger.Info("Using command line specified port: %d", *port)
	}

	srv := server.NewServer(cfg)
	if err := srv.Start(); err != nil {
		logger.Error("Server startup failed: %s", err)
	}
}
