package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		api.GET("/articles", s.getAllArticles) // New endpoint for all articles
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

func (s *Server) getAllArticles(c *gin.Context) {
	log.Printf("DEBUG: getAllArticles called")

	// Parse OData query parameters
	query, err := s.parseODataQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("DEBUG: Parsed query - Filter: '%s', Search: %v, OrderBy: '%s', Top: %d, Skip: %d",
		query.Filter, query.Search, query.OrderBy, query.Top, query.Skip)

	// Check if there's a topic filter
	var targetTopic string
	if query.Filter != "" {
		// Parse the filter to check for topic filter
		if strings.Contains(strings.ToLower(query.Filter), "topic") {
			// Extract topic from filter like "topic eq 'tech'"
			if strings.Contains(query.Filter, " eq ") {
				parts := strings.Split(query.Filter, " eq ")
				if len(parts) == 2 {
					topicValue := strings.Trim(parts[1], "'\"")
					targetTopic = topicValue
					log.Printf("DEBUG: Topic filter detected: %s", targetTopic)
				}
			}
		}
	}

	var allArticles []models.Article
	var totalCount int

	if targetTopic != "" {
		// If topic filter is specified, only get articles from that topic
		log.Printf("DEBUG: Getting articles for specific topic: %s", targetTopic)

		// Create a topic-specific query without the topic filter (since we're already filtering by topic)
		topicQuery := &models.ODataQuery{
			OrderBy: query.OrderBy,
			Select:  query.Select,
			Search:  query.Search,
			Top:     query.Top,
			Skip:    query.Skip,
		}

		feed, err := s.aggregator.GetAggregatedFeed(targetTopic, topicQuery)
		if err != nil {
			log.Printf("Warning: failed to get feed for topic '%s': %v", targetTopic, err)
			// Return empty result instead of error
			c.JSON(http.StatusOK, gin.H{
				"articles":    []models.Article{},
				"count":       0,
				"total_count": 0,
				"skip":        query.Skip,
				"top":         query.Top,
				"has_more":    false,
			})
			return
		}
		if feed == nil {
			log.Printf("Warning: feed is nil for topic '%s'", targetTopic)
			c.JSON(http.StatusOK, gin.H{
				"articles":    []models.Article{},
				"count":       0,
				"total_count": 0,
				"skip":        query.Skip,
				"top":         query.Top,
				"has_more":    false,
			})
			return
		}

		// Add topic information to each article
		for i := range feed.Articles {
			feed.Articles[i].Topic = targetTopic
		}

		allArticles = feed.Articles
		totalCount = len(feed.Articles)

		log.Printf("DEBUG: Got %d articles for topic %s", len(allArticles), targetTopic)
	} else {
		// No topic filter - get articles from all topics
		log.Printf("DEBUG: Getting articles from all topics")

		topics := s.aggregator.GetAvailableTopics()
		for _, topic := range topics {
			// Create a topic-specific query
			topicQuery := &models.ODataQuery{
				Filter:  query.Filter,
				OrderBy: query.OrderBy,
				Select:  query.Select,
				Search:  query.Search,
				// Don't apply pagination at topic level, we'll do it globally
			}

			feed, err := s.aggregator.GetAggregatedFeed(topic, topicQuery)
			if err != nil {
				log.Printf("Warning: failed to get feed for topic '%s': %v", topic, err)
				continue
			}
			if feed == nil {
				log.Printf("Warning: feed is nil for topic '%s'", topic)
				continue
			}

			// Add topic information to each article
			for i := range feed.Articles {
				feed.Articles[i].Topic = topic
			}

			log.Printf("DEBUG: Got %d articles for topic %s", len(feed.Articles), topic)
			allArticles = append(allArticles, feed.Articles...)
			totalCount += len(feed.Articles)
		}

		log.Printf("DEBUG: Total articles collected: %d", len(allArticles))

		// Apply advanced filtering
		allArticles = s.applyAdvancedFilters(allArticles, query)

		// Apply search filtering
		if len(query.Search) > 0 {
			allArticles = searchArticles(allArticles, query.Search)
		}

		// Apply sorting
		if query.OrderBy != "" {
			allArticles = sortArticles(allArticles, query.OrderBy)
		}

		// Apply field selection
		if len(query.Select) > 0 {
			allArticles = applySelectFields(allArticles, query.Select)
		}
	}

	// Apply pagination (always executed)
	totalCount = len(allArticles)
	start := query.Skip
	end := start + query.Top
	if end > totalCount {
		end = totalCount
	}
	if start < totalCount {
		allArticles = allArticles[start:end]
	} else {
		allArticles = []models.Article{}
	}

	log.Printf("DEBUG: Final result: %d articles", len(allArticles))

	c.JSON(http.StatusOK, gin.H{
		"articles":    allArticles,
		"count":       len(allArticles),
		"total_count": totalCount,
		"skip":        query.Skip,
		"top":         query.Top,
		"has_more":    end < totalCount,
	})
}

// Helper functions for OData operations
func searchArticles(articles []models.Article, searchTerms []string) []models.Article {
	var filtered []models.Article

	for _, article := range articles {
		for _, term := range searchTerms {
			if articleContains(article, term) {
				filtered = append(filtered, article)
				break
			}
		}
	}

	return filtered
}

func articleContains(article models.Article, term string) bool {
	term = strings.ToLower(term)

	// Search in title
	if strings.Contains(strings.ToLower(article.Title), term) {
		return true
	}

	// Search in description
	if strings.Contains(strings.ToLower(article.Description), term) {
		return true
	}

	// Search in content
	if strings.Contains(strings.ToLower(article.Content), term) {
		return true
	}

	// Search in author
	if strings.Contains(strings.ToLower(article.Author), term) {
		return true
	}

	// Search in source
	if strings.Contains(strings.ToLower(article.Source), term) {
		return true
	}

	return false
}

func sortArticles(articles []models.Article, orderBy string) []models.Article {
	// Simple sorting implementation
	// In a real application, you might want to use a more sophisticated sorting library

	// For now, just return the articles as-is
	// You can implement proper sorting based on the orderBy parameter
	return articles
}

func applySelectFields(articles []models.Article, selectedFields []string) []models.Article {
	// Simple field selection implementation
	// In a real application, you might want to use reflection or a more sophisticated approach

	// For now, just return the articles as-is
	// You can implement proper field selection based on the selectedFields parameter
	return articles
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

// parseODataQuery parses OData query parameters from the request
func (s *Server) parseODataQuery(c *gin.Context) (*models.ODataQuery, error) {
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

	// Set default pagination if not specified
	if topStr := c.Query("$top"); topStr != "" {
		if top, err := strconv.Atoi(topStr); err == nil {
			query.Top = top
		}
	} else {
		query.Top = 20 // Default page size
	}

	if skipStr := c.Query("$skip"); skipStr != "" {
		if skip, err := strconv.Atoi(skipStr); err == nil {
			query.Skip = skip
		}
	}

	// Parse advanced filter options
	if filterStr := c.Query("$filter"); filterStr != "" {
		query.Filter = filterStr
	}

	// Parse date range filters
	if dateFromStr := c.Query("$datefrom"); dateFromStr != "" {
		if dateFrom, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			query.DateFrom = &dateFrom
		}
	}

	if dateToStr := c.Query("$dateto"); dateToStr != "" {
		if dateTo, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			query.DateTo = &dateTo
		}
	}

	// Parse source filter
	if sourceStr := c.Query("$source"); sourceStr != "" {
		query.Source = sourceStr
	}

	// Parse author filter
	if authorStr := c.Query("$author"); authorStr != "" {
		query.Author = authorStr
	}

	// Parse category filter
	if categoryStr := c.Query("$category"); categoryStr != "" {
		query.Category = categoryStr
	}

	return query, nil
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
	})
}

func (s *Server) forcePollTopic(c *gin.Context) {
	topic := c.Param("topic")
	if topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic parameter is required"})
		return
	}

	err := s.poller.ForcePoll(topic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Force poll initiated for topic: %s", topic)})
}

func (s *Server) getLastPolledTimes(c *gin.Context) {
	lastPolled := s.poller.GetLastPolledTime()
	c.JSON(http.StatusOK, lastPolled)
}

// applyAdvancedFilters applies advanced filtering options to articles
func (s *Server) applyAdvancedFilters(articles []models.Article, query *models.ODataQuery) []models.Article {
	var filtered []models.Article

	for _, article := range articles {
		// Apply date range filtering
		if query.DateFrom != nil && article.PublishedAt.Before(*query.DateFrom) {
			continue
		}
		if query.DateTo != nil && article.PublishedAt.After(*query.DateTo) {
			continue
		}

		// Apply source filtering
		if query.Source != "" && !strings.Contains(strings.ToLower(article.Source), strings.ToLower(query.Source)) {
			continue
		}

		// Apply author filtering
		if query.Author != "" && !strings.Contains(strings.ToLower(article.Author), strings.ToLower(query.Author)) {
			continue
		}

		// Apply category filtering
		if query.Category != "" {
			found := false
			for _, category := range article.Categories {
				if strings.Contains(strings.ToLower(category), strings.ToLower(query.Category)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, article)
	}

	return filtered
}
