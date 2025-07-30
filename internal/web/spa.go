package web

import (
	"log"
	"net/http"
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

	// Serve the main SPA page
	router.GET("/", s.serveSPA)

	// Serve static assets from filesystem
	router.Static("/static", "./internal/web/static")

	log.Println("SPA routes registered successfully")
}

// serveSPA serves the main SPA HTML page
func (s *SPAServer) serveSPA(c *gin.Context) {
	c.HTML(http.StatusOK, "spa.html", gin.H{
		"title":     "RSS Aggregator",
		"timestamp": time.Now().Unix(),
	})
}
