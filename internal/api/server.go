package api

import (
	"net/http"
	"strconv"
	"strings"

	"gorssag/internal/aggregator"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/poller"
	"gorssag/internal/security"
	"gorssag/internal/web"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router        *gin.Engine
	aggregator    *aggregator.Aggregator
	poller        *poller.Poller
	port          int
	spaServer     *web.SPAServer
	swaggerServer *web.SwaggerServer
}

func NewServer(agg *aggregator.Aggregator, poller *poller.Poller, cfg *config.Config) *Server {
	router := gin.Default()

	// Load HTML templates from filesystem (only if SPA is enabled)
	if cfg.EnableSPA {
		router.LoadHTMLGlob("internal/web/templates/*")
	}

	// Setup security middleware
	securityConfig := &security.SecurityConfig{
		EnableRateLimit:       cfg.Security.EnableRateLimit,
		RateLimitPerSecond:    cfg.Security.RateLimitPerSecond,
		RateLimitBurst:        cfg.Security.RateLimitBurst,
		EnableCORS:            cfg.Security.EnableCORS,
		AllowedOrigins:        cfg.Security.AllowedOrigins,
		EnableSecurityHeaders: cfg.Security.EnableSecurityHeaders,
		MaxRequestSize:        cfg.Security.MaxRequestSize,
		EnableRequestID:       cfg.Security.EnableRequestID,
	}
	security.SetupSecurityMiddleware(router, securityConfig)

	// Create web servers
	spaServer := web.NewSPAServer(cfg.EnableSPA)
	swaggerServer := web.NewSwaggerServer(cfg.EnableSwagger)

	server := &Server{
		router:        router,
		aggregator:    agg,
		poller:        poller,
		port:          cfg.Port,
		spaServer:     spaServer,
		swaggerServer: swaggerServer,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthCheck)

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.GET("/topics", s.getTopics)
		api.GET("/feeds/:topic", s.getAggregatedFeed)
		api.GET("/feeds/:topic/info", s.getFeedInfo)
		api.POST("/feeds/:topic/refresh", s.refreshFeed)

		// Poller control endpoints
		api.GET("/poller/status", s.getPollerStatus)
		api.POST("/poller/force-poll/:topic", s.forcePollTopic)
		api.GET("/poller/last-polled", s.getLastPolledTimes)
	}

	// Register web interfaces
	s.spaServer.RegisterRoutes(s.router)
	s.swaggerServer.RegisterRoutes(s.router)
}

func (s *Server) Start() error {
	return s.router.Run(":" + strconv.Itoa(s.port))
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":        "healthy",
		"service":       "rss-aggregator",
		"poller_active": s.poller.IsPolling(),
	})
}

func (s *Server) getTopics(c *gin.Context) {
	topics := s.aggregator.GetAvailableTopics()
	c.JSON(http.StatusOK, gin.H{
		"topics": topics,
		"count":  len(topics),
	})
}

func (s *Server) getAggregatedFeed(c *gin.Context) {
	topic := c.Param("topic")

	// Parse OData query parameters
	query := &models.ODataQuery{
		Filter:  c.Query("$filter"),
		OrderBy: c.Query("$orderby"),
		Select:  parseSelectFields(c.Query("$select")),
	}

	// Parse search terms (comma-separated)
	if searchStr := c.Query("$search"); searchStr != "" {
		searchTerms := strings.Split(searchStr, ",")
		for i, term := range searchTerms {
			searchTerms[i] = strings.TrimSpace(term)
		}
		query.Search = searchTerms
	}

	if topStr := c.Query("$top"); topStr != "" {
		if top, err := strconv.Atoi(topStr); err == nil {
			query.Top = top
		}
	}

	if skipStr := c.Query("$skip"); skipStr != "" {
		if skip, err := strconv.Atoi(skipStr); err == nil {
			query.Skip = skip
		}
	}

	feed, err := s.aggregator.GetAggregatedFeed(topic, query)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, feed)
}

// parseSelectFields parses the $select parameter and returns a slice of field names
func parseSelectFields(selectStr string) []string {
	if selectStr == "" {
		return nil
	}

	// Split by comma and trim whitespace
	fields := strings.Split(selectStr, ",")
	result := make([]string, 0, len(fields))

	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func (s *Server) getFeedInfo(c *gin.Context) {
	topic := c.Param("topic")

	info, err := s.aggregator.GetFeedInfo(topic)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (s *Server) refreshFeed(c *gin.Context) {
	topic := c.Param("topic")

	if err := s.aggregator.RefreshFeed(topic); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Feed refreshed successfully",
		"topic":   topic,
	})
}

func (s *Server) getPollerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"is_polling": s.poller.IsPolling(),
		"status":     "active",
	})
}

func (s *Server) forcePollTopic(c *gin.Context) {
	topic := c.Param("topic")

	if err := s.poller.ForcePoll(topic); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Force poll initiated successfully",
		"topic":   topic,
	})
}

func (s *Server) getLastPolledTimes(c *gin.Context) {
	lastPolled := s.poller.GetLastPolledTime()
	c.JSON(http.StatusOK, lastPolled)
}
