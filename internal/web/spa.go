package web

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// SPAServer handles serving the Single Page Application
type SPAServer struct {
	enabled bool
}

// NewSPAServer creates a new SPA server instance
func NewSPAServer(enabled bool) *SPAServer {
	if enabled {
		log.Println("SPA Server enabled")
	}
	return &SPAServer{enabled: enabled}
}

// RegisterRoutes registers the SPA routes with the Gin router
func (s *SPAServer) RegisterRoutes(router *gin.Engine) {
	if !s.enabled {
		log.Println("SPA Server is disabled")
		return
	}

	log.Println("Registering SPA routes...")

	// SPA routes
	router.GET("/", s.serveSPA)
	router.GET("/config", s.serveConfig) // New configuration page

	// Get the current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: could not get working directory: %v", err)
		wd = "."
	}

	// Serve static assets from filesystem with absolute path
	staticPath := filepath.Join(wd, "internal", "web", "static")
	log.Printf("Serving static files from: %s", staticPath)
	router.Static("/static", staticPath)

	log.Println("SPA routes registered successfully")
}

// serveSPA serves the main SPA HTML page
func (s *SPAServer) serveSPA(c *gin.Context) {
	c.HTML(http.StatusOK, "spa.html", gin.H{
		"title":     "RSS Aggregator",
		"timestamp": time.Now().Unix(), // Add timestamp for cache busting
	})
}

func (s *SPAServer) serveConfig(c *gin.Context) {
	c.HTML(http.StatusOK, "config.html", gin.H{
		"title":     "Feed Configuration",
		"timestamp": time.Now().Unix(),
	})
}
