package web

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templates embed.FS

//go:embed static/*
var staticFiles embed.FS

type SPAServer struct {
	enabled bool
}

func NewSPAServer(enabled bool) *SPAServer {
	return &SPAServer{enabled: enabled}
}

func (s *SPAServer) RegisterRoutes(router *gin.Engine) {
	if !s.enabled {
		return
	}

	// Serve the main SPA page
	router.GET("/", s.serveSPA)

	// Serve static assets
	router.GET("/static/*filepath", s.serveStatic)
}

func (s *SPAServer) serveSPA(c *gin.Context) {
	// Only serve SPA for root path
	if c.Request.URL.Path != "/" {
		c.Status(http.StatusNotFound)
		return
	}

	tmpl, err := template.ParseFS(templates, "templates/spa.html")
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "text/html")
	tmpl.Execute(c.Writer, nil)
}

func (s *SPAServer) serveStatic(c *gin.Context) {
	// Get filepath from Gin
	filepath := c.Param("filepath")

	// Read the embedded file
	content, err := staticFiles.ReadFile("static/" + filepath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// Set appropriate content type based on file extension
	switch {
	case strings.HasSuffix(filepath, ".css"):
		c.Header("Content-Type", "text/css")
	case strings.HasSuffix(filepath, ".js"):
		c.Header("Content-Type", "application/javascript")
	case strings.HasSuffix(filepath, ".png"):
		c.Header("Content-Type", "image/png")
	case strings.HasSuffix(filepath, ".jpg"), strings.HasSuffix(filepath, ".jpeg"):
		c.Header("Content-Type", "image/jpeg")
	case strings.HasSuffix(filepath, ".svg"):
		c.Header("Content-Type", "image/svg+xml")
	}

	c.Data(http.StatusOK, c.GetHeader("Content-Type"), content)
}
