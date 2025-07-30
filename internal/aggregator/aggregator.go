package aggregator

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/odata"
	"gorssag/internal/storage"

	"github.com/mmcdole/gofeed"
)

// Common User-Agents to test
var userAgentsToTest = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/120.0.0.0",
	"", // Default/empty User-Agent
}

// Aggregator manages RSS feed aggregation
type Aggregator struct {
	feeds        map[string]config.TopicConfig
	storage      storage.Storage
	cacheManager *cache.Manager
	feedStatus   map[string]*models.FeedStatus // Track feed status
	mu           sync.RWMutex
	parser       *gofeed.Parser
	filterParser *odata.FilterParser
}

func New(cacheManager *cache.Manager, storage storage.Storage, feeds map[string]config.TopicConfig) *Aggregator {
	return &Aggregator{
		cacheManager: cacheManager,
		storage:      storage,
		feeds:        feeds,
		parser:       gofeed.NewParser(),
		filterParser: odata.NewFilterParser(),
		feedStatus:   make(map[string]*models.FeedStatus),
	}
}

func (a *Aggregator) GetAvailableTopics() []string {
	var topics []string
	for topic := range a.feeds {
		topics = append(topics, topic)
	}
	return topics
}

// GetConfig returns the current feed configuration
func (a *Aggregator) GetConfig() map[string]config.TopicConfig {
	return a.feeds
}

// GetFeedStatus returns the status of all feeds
func (a *Aggregator) GetFeedStatus() map[string]*models.FeedStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := make(map[string]*models.FeedStatus)
	for url, feedStatus := range a.feedStatus {
		status[url] = feedStatus
	}
	return status
}

// TestUserAgentForFeed tests different User-Agents to find one that works
func (a *Aggregator) TestUserAgentForFeed(url string) (string, error) {
	a.mu.Lock()
	status, exists := a.feedStatus[url]
	if !exists {
		status = &models.FeedStatus{
			URL:              url,
			TestedUserAgents: []string{},
		}
		a.feedStatus[url] = status
	}
	a.mu.Unlock()

	// Test each User-Agent
	for _, userAgent := range userAgentsToTest {
		// Skip if already tested
		if a.isUserAgentTested(status, userAgent) {
			continue
		}

		log.Printf("Testing User-Agent for %s: %s", url, userAgent)

		// Test the User-Agent
		feed, err := a.testFeedWithUserAgent(url, userAgent)
		if err == nil && feed != nil && len(feed.Items) > 0 {
			// Check content quality
			hasValidContent := false
			for _, item := range feed.Items {
				if item.Title != "" && (item.Content != "" || item.Description != "") {
					hasValidContent = true
					break
				}
			}

			if hasValidContent {
				log.Printf("Found working User-Agent for %s: %s", url, userAgent)
				return userAgent, nil
			}
		}

		// Mark as tested
		a.markUserAgentTested(status, userAgent)
	}

	return "", fmt.Errorf("no working User-Agent found for %s", url)
}

// testFeedWithUserAgent tests a feed with a specific User-Agent
func (a *Aggregator) testFeedWithUserAgent(url, userAgent string) (*gofeed.Feed, error) {
	// Create a custom HTTP client with the User-Agent
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Parse the feed
	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return feed, nil
}

// isUserAgentTested checks if a User-Agent has been tested
func (a *Aggregator) isUserAgentTested(status *models.FeedStatus, userAgent string) bool {
	for _, tested := range status.TestedUserAgents {
		if tested == userAgent {
			return true
		}
	}
	return false
}

// markUserAgentTested marks a User-Agent as tested
func (a *Aggregator) markUserAgentTested(status *models.FeedStatus, userAgent string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	status.TestedUserAgents = append(status.TestedUserAgents, userAgent)
}

// UpdateFeedStatus updates the status of a specific feed
func (a *Aggregator) UpdateFeedStatus(url, topic string, articlesCount int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	status, exists := a.feedStatus[url]
	if !exists {
		status = &models.FeedStatus{
			URL:        url,
			Topic:      topic,
			IsDisabled: false,
		}
		a.feedStatus[url] = status
	}

	status.LastPolled = time.Now()
	status.ArticlesCount = articlesCount

	if err != nil {
		status.LastError = err.Error()
		status.ErrorCount++
		status.ConsecutiveErrors++

		// Check if this is a content quality issue
		if strings.Contains(err.Error(), "no content and no description") {
			status.IsDisabled = true
			status.IsContentIssue = true
			status.DisabledReason = "Feed provides no content or description - disabled permanently"
			status.NextRetry = time.Time{} // No retry for content issues
		} else {
			// Technical error - implement retry logic
			status.IsContentIssue = false

			// Calculate next retry time with exponential backoff
			backoffMinutes := a.calculateBackoff(status.ConsecutiveErrors)
			status.NextRetry = time.Now().Add(time.Duration(backoffMinutes) * time.Minute)
			status.RetryCount++

			// Only disable after many consecutive errors (e.g., 10+ errors)
			if status.ConsecutiveErrors >= 10 {
				status.IsDisabled = true
				status.DisabledReason = fmt.Sprintf("Disabled due to %d consecutive technical errors. Will retry every %d minutes.",
					status.ConsecutiveErrors, backoffMinutes)
			} else {
				status.IsDisabled = false
				status.DisabledReason = ""
			}
		}
	} else {
		// Success - reset error counters
		status.LastError = ""
		status.ConsecutiveErrors = 0
		status.LastSuccess = time.Now()
		status.IsDisabled = false
		status.IsContentIssue = false
		status.DisabledReason = ""
		status.NextRetry = time.Time{}
		status.RetryCount = 0
	}
}

// SetUserAgentForFeed sets the working User-Agent for a feed
func (a *Aggregator) SetUserAgentForFeed(url, userAgent string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	status, exists := a.feedStatus[url]
	if !exists {
		status = &models.FeedStatus{
			URL: url,
		}
		a.feedStatus[url] = status
	}

	status.UserAgent = userAgent
	log.Printf("Set User-Agent for %s: %s", url, userAgent)
}

// calculateBackoff calculates retry backoff in minutes
func (a *Aggregator) calculateBackoff(consecutiveErrors int) int {
	// Exponential backoff: 5, 10, 20, 40, 60, 60, 60... minutes
	baseBackoff := 5
	maxBackoff := 60

	backoff := baseBackoff * (1 << (consecutiveErrors - 1))
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}

// ShouldRetryFeed checks if a disabled feed should be retried
func (a *Aggregator) ShouldRetryFeed(url string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status, exists := a.feedStatus[url]
	if !exists {
		return true // New feed, should try
	}

	// Never retry content quality issues
	if status.IsContentIssue {
		return false
	}

	// Check if it's time to retry
	if status.IsDisabled && !status.NextRetry.IsZero() && time.Now().After(status.NextRetry) {
		return true
	}

	return !status.IsDisabled
}

func (a *Aggregator) GetAggregatedFeed(topic string, query *models.ODataQuery) (*models.AggregatedFeed, error) {
	// Check if topic exists
	topicConfig, exists := a.feeds[topic]
	if !exists {
		return nil, fmt.Errorf("topic '%s' not found", topic)
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("feed:%s", topic)
	if cached, found := a.cacheManager.Get(cacheKey); found {
		if feed, ok := cached.(*models.AggregatedFeed); ok {
			// Apply OData query to cached data
			return a.applyODataQuery(feed, query)
		}
	}

	// Try to load from storage
	if feed, err := a.storage.LoadFeed(topic); err == nil {
		// Cache the loaded feed
		a.cacheManager.Set(cacheKey, feed, 0)
		// Apply OData query
		return a.applyODataQuery(feed, query)
	}

	// Fetch from RSS feeds
	articles, err := a.fetchFeedsParallel(topicConfig.URLs, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feeds for topic '%s': %v", topic, err)
	}

	// Filter articles based on topic configuration
	filteredArticles := a.filterArticlesForTopic(articles, topicConfig.Filters)

	if len(filteredArticles) == 0 {
		return nil, fmt.Errorf("no articles found for topic '%s' after filtering", topic)
	}

	feed := &models.AggregatedFeed{
		Topic:    topic,
		Articles: filteredArticles,
		Count:    len(filteredArticles),
		Updated:  time.Now(),
	}

	// Save to storage
	if err := a.storage.SaveFeed(topic, feed); err != nil {
		log.Printf("Warning: failed to save feed for topic '%s': %v", topic, err)
	}

	// Cache the feed
	a.cacheManager.Set(cacheKey, feed, 0)

	// Apply OData query
	return a.applyODataQuery(feed, query)
}

func (a *Aggregator) filterArticlesForTopic(articles []models.Article, filters []string) []models.Article {
	if len(filters) == 0 {
		// No filters specified, return all articles
		return articles
	}

	var filteredArticles []models.Article

	for _, article := range articles {
		if a.articleMatchesFilters(article, filters) {
			filteredArticles = append(filteredArticles, article)
		}
	}

	return filteredArticles
}

func (a *Aggregator) articleMatchesFilters(article models.Article, filters []string) bool {
	// Create a combined text field for searching
	articleText := strings.ToLower(strings.Join([]string{
		article.Title,
		article.Description,
		article.Content,
		article.Author,
		strings.Join(article.Categories, " "),
	}, " "))

	// Check if any filter term matches (OR logic)
	for _, filter := range filters {
		if strings.Contains(articleText, strings.ToLower(filter)) {
			return true
		}
	}

	return false
}

func (a *Aggregator) fetchFeedsParallel(feedURLs []string, topic string) ([]models.Article, error) {
	var wg sync.WaitGroup
	results := make(chan FeedResult, len(feedURLs))

	// Start goroutines for each feed URL
	for _, url := range feedURLs {
		wg.Add(1)
		go func(feedURL string) {
			defer wg.Done()
			articles, err := a.fetchFeed(feedURL, topic)
			results <- FeedResult{
				URL:      feedURL,
				Articles: articles,
				Error:    err,
			}
		}(url)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with timeout
	timeout := time.After(30 * time.Second)
	var allArticles []models.Article

	for {
		select {
		case result, ok := <-results:
			if !ok {
				return allArticles, nil
			}
			if result.Error != nil {
				log.Printf("Error fetching feed %s: %v", result.URL, result.Error)
			} else {
				allArticles = append(allArticles, result.Articles...)
			}
		case <-timeout:
			log.Printf("Timeout waiting for feed results")
			return allArticles, nil
		}
	}
}

func (a *Aggregator) fetchFeed(url string, topic string) ([]models.Article, error) {
	// Check if feed should be retried
	if !a.ShouldRetryFeed(url) {
		a.mu.RLock()
		status, exists := a.feedStatus[url]
		if exists && status.IsDisabled {
			a.mu.RUnlock()
			return nil, fmt.Errorf("feed is disabled: %s", status.DisabledReason)
		}
		a.mu.RUnlock()
	}

	// Get stored User-Agent for this feed
	a.mu.RLock()
	status, exists := a.feedStatus[url]
	var userAgent string
	if exists && status.UserAgent != "" {
		userAgent = status.UserAgent
	}
	a.mu.RUnlock()

	// Try to fetch with stored User-Agent first
	var feed *gofeed.Feed
	var err error

	if userAgent != "" {
		feed, err = a.testFeedWithUserAgent(url, userAgent)
		if err != nil {
			log.Printf("Failed to fetch %s with stored User-Agent: %v", url, err)
		}
	}

	// If no stored User-Agent or it failed, try to find a working one
	if feed == nil || err != nil {
		log.Printf("Testing User-Agents for %s", url)
		workingUserAgent, uaErr := a.TestUserAgentForFeed(url)
		if uaErr != nil {
			// Update status with error
			a.UpdateFeedStatus(url, topic, 0, uaErr)
			return nil, fmt.Errorf("failed to find working User-Agent for %s: %v", url, uaErr)
		}

		// Set the working User-Agent
		a.SetUserAgentForFeed(url, workingUserAgent)
		userAgent = workingUserAgent

		// Fetch with the working User-Agent
		feed, err = a.testFeedWithUserAgent(url, userAgent)
		if err != nil {
			a.UpdateFeedStatus(url, topic, 0, err)
			return nil, fmt.Errorf("failed to parse feed: %v", err)
		}
	}

	var articles []models.Article
	for _, item := range feed.Items {
		// Check content quality - require title and either content or description
		if item.Title == "" {
			log.Printf("WARNING: Feed '%s' has item with no title - skipping", feed.Title)
			continue
		}

		content := item.Content
		description := item.Description

		// If no content, use description
		if content == "" && description != "" {
			content = description
		}

		// If still no content, this is a content quality issue
		if content == "" {
			log.Printf("WARNING: Feed '%s' has item '%s' with no content and no description - skipping", feed.Title, item.Title)
			continue
		}

		// Safely get author name
		authorName := ""
		if item.Author != nil {
			authorName = item.Author.Name
		}

		article := models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     content,
			Author:      authorName,
			Source:      feed.Title,
			Categories:  []string{},
			PublishedAt: time.Now(),
		}

		// Extract categories
		for _, category := range item.Categories {
			article.Categories = append(article.Categories, category)
		}

		// Parse published date
		if item.PublishedParsed != nil {
			article.PublishedAt = *item.PublishedParsed
		}

		articles = append(articles, article)
	}

	// Update status with success
	a.UpdateFeedStatus(url, topic, len(articles), nil)
	return articles, nil
}

func (a *Aggregator) applyODataQuery(feed *models.AggregatedFeed, query *models.ODataQuery) (*models.AggregatedFeed, error) {
	if query == nil {
		return feed, nil
	}

	articles := feed.Articles

	// Apply search if specified
	if len(query.Search) > 0 {
		articles = a.searchArticles(articles, query.Search)
	}

	// Apply filter if specified
	if query.Filter != "" {
		filterExpr, err := a.filterParser.Parse(query.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter expression: %v", err)
		}

		var filteredArticles []models.Article
		for _, article := range articles {
			matches, err := a.filterParser.Evaluate(filterExpr, article)
			if err != nil {
				return nil, fmt.Errorf("filter evaluation error: %v", err)
			}
			if matches {
				filteredArticles = append(filteredArticles, article)
			}
		}
		articles = filteredArticles
	}

	// Apply sorting
	if query.OrderBy != "" {
		articles = a.sortArticles(articles, query.OrderBy)
	}

	// Apply pagination
	if query.Skip > 0 {
		if query.Skip >= len(articles) {
			articles = []models.Article{}
		} else {
			articles = articles[query.Skip:]
		}
	}

	if query.Top > 0 && query.Top < len(articles) {
		articles = articles[:query.Top]
	}

	// Apply field selection
	if len(query.Select) > 0 {
		articles = a.applySelectFields(articles, query.Select)
	}

	return &models.AggregatedFeed{
		Topic:    feed.Topic,
		Articles: articles,
		Count:    len(articles),
		Updated:  feed.Updated,
	}, nil
}

func (a *Aggregator) searchArticles(articles []models.Article, searchTerms []string) []models.Article {
	var results []models.Article

	for _, article := range articles {
		for _, term := range searchTerms {
			if a.articleContains(article, term) {
				results = append(results, article)
				break
			}
		}
	}

	return results
}

func (a *Aggregator) articleContains(article models.Article, term string) bool {
	term = strings.ToLower(term)

	fields := []string{
		article.Title,
		article.Description,
		article.Content,
		article.Author,
		article.Source,
		strings.Join(article.Categories, " "),
	}

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), term) {
			return true
		}
	}

	return false
}

func (a *Aggregator) sortArticles(articles []models.Article, orderBy string) []models.Article {
	// Implementation of sorting logic
	// This would sort articles based on the orderBy parameter
	return articles
}

func (a *Aggregator) applySelectFields(articles []models.Article, selectedFields []string) []models.Article {
	// Create a map of valid fields for quick lookup
	validFields := make(map[string]bool)
	for _, field := range selectedFields {
		validFields[strings.ToLower(strings.TrimSpace(field))] = true
	}

	// If no valid fields selected, return all fields (default behavior)
	if len(validFields) == 0 {
		return articles
	}

	// Create new articles with only selected fields
	result := make([]models.Article, len(articles))
	for i, article := range articles {
		newArticle := models.Article{}

		// Only copy selected fields
		if validFields["title"] {
			newArticle.Title = article.Title
		}
		if validFields["link"] {
			newArticle.Link = article.Link
		}
		if validFields["description"] {
			newArticle.Description = article.Description
		}
		if validFields["content"] {
			newArticle.Content = article.Content
		}
		if validFields["author"] {
			newArticle.Author = article.Author
		}
		if validFields["source"] {
			newArticle.Source = article.Source
		}
		if validFields["categories"] {
			newArticle.Categories = article.Categories
		}
		if validFields["published_at"] {
			newArticle.PublishedAt = article.PublishedAt
		}

		result[i] = newArticle
	}

	return result
}

func (a *Aggregator) RefreshFeed(topic string) error {
	// Remove from cache to force refresh
	cacheKey := fmt.Sprintf("feed:%s", topic)
	a.cacheManager.Delete(cacheKey)

	// Fetch fresh data
	_, err := a.GetAggregatedFeed(topic, nil)
	return err
}

func (a *Aggregator) GetFeedInfo(topic string) (*models.FeedInfo, error) {
	return a.storage.GetFeedInfo(topic)
}

type FeedResult struct {
	URL      string
	Articles []models.Article
	Error    error
}

// InitializeFeeds performs initial polling of all feeds to establish their status
func (a *Aggregator) InitializeFeeds() {
	log.Printf("Starting initial feed polling...")

	for topic, topicConfig := range a.feeds {
		log.Printf("Initializing feeds for topic: %s", topic)

		for _, url := range topicConfig.URLs {
			log.Printf("Testing feed: %s", url)

			// Test the feed to establish initial status
			articles, err := a.fetchFeed(url, topic)
			if err != nil {
				log.Printf("Initial test failed for %s: %v", url, err)
			} else {
				log.Printf("Initial test successful for %s: %d articles", url, len(articles))
			}
		}
	}

	log.Printf("Initial feed polling completed")
}

// FeedHealth represents the health status of a feed
type FeedHealth struct {
	URL           string `json:"url"`
	Topic         string `json:"topic"`
	Status        string `json:"status"` // "healthy", "warning", "error", "disabled"
	Reason        string `json:"reason,omitempty"`
	ArticlesCount int    `json:"articles_count"`
	LastPolled    string `json:"last_polled,omitempty"`
}

// GetFeedHealth returns health status for all feeds
func (a *Aggregator) GetFeedHealth() map[string][]FeedHealth {
	a.mu.RLock()
	defer a.mu.RUnlock()

	health := make(map[string][]FeedHealth)

	for topic, topicConfig := range a.feeds {
		var topicHealth []FeedHealth

		for _, url := range topicConfig.URLs {
			status, exists := a.feedStatus[url]
			feedHealth := FeedHealth{
				URL:   url,
				Topic: topic,
			}

			if !exists {
				// Feed hasn't been polled yet
				feedHealth.Status = "unknown"
				feedHealth.Reason = "Not yet polled"
			} else {
				feedHealth.ArticlesCount = status.ArticlesCount
				feedHealth.LastPolled = status.LastPolled.Format("2006-01-02 15:04:05")

				if status.IsDisabled {
					if status.IsContentIssue {
						feedHealth.Status = "disabled"
						feedHealth.Reason = "Content quality issue - no title, content, or description"
					} else {
						feedHealth.Status = "error"
						feedHealth.Reason = fmt.Sprintf("Disabled due to %d consecutive errors", status.ConsecutiveErrors)
					}
				} else if status.LastError != "" {
					// Show specific error details
					errorReason := a.getSpecificErrorReason(status.LastError, status.ConsecutiveErrors)

					if status.ConsecutiveErrors >= 5 {
						feedHealth.Status = "warning"
						feedHealth.Reason = fmt.Sprintf("%s (%d consecutive failures)", errorReason, status.ConsecutiveErrors)
					} else {
						feedHealth.Status = "warning"
						feedHealth.Reason = errorReason
					}
				} else if status.ArticlesCount > 0 {
					feedHealth.Status = "healthy"
					feedHealth.Reason = fmt.Sprintf("%d articles available", status.ArticlesCount)
				} else {
					feedHealth.Status = "warning"
					feedHealth.Reason = "No articles found in feed"
				}
			}

			topicHealth = append(topicHealth, feedHealth)
		}

		health[topic] = topicHealth
	}

	return health
}

// getSpecificErrorReason provides detailed error explanations
func (a *Aggregator) getSpecificErrorReason(errorMsg string, consecutiveErrors int) string {
	errorMsg = strings.ToLower(errorMsg)

	switch {
	case strings.Contains(errorMsg, "404"):
		return "Feed URL not found (404) - feed may have been moved or discontinued"
	case strings.Contains(errorMsg, "403"):
		return "Access forbidden (403) - feed may require authentication or be blocked"
	case strings.Contains(errorMsg, "401"):
		return "Unauthorized (401) - feed requires authentication"
	case strings.Contains(errorMsg, "500"):
		return "Server error (500) - feed server is experiencing issues"
	case strings.Contains(errorMsg, "502"):
		return "Bad gateway (502) - feed server is temporarily unavailable"
	case strings.Contains(errorMsg, "503"):
		return "Service unavailable (503) - feed server is overloaded"
	case strings.Contains(errorMsg, "timeout"):
		return "Connection timeout - feed server is slow or unresponsive"
	case strings.Contains(errorMsg, "connection refused"):
		return "Connection refused - feed server is not accepting connections"
	case strings.Contains(errorMsg, "no such host"):
		return "DNS resolution failed - feed domain does not exist or is unreachable"
	case strings.Contains(errorMsg, "eof"):
		return "Connection closed unexpectedly (EOF) - feed server terminated connection"
	case strings.Contains(errorMsg, "ssl"):
		return "SSL/TLS error - feed has certificate issues"
	case strings.Contains(errorMsg, "certificate"):
		return "Certificate error - feed has invalid or expired SSL certificate"
	case strings.Contains(errorMsg, "parse"):
		return "Feed parsing error - feed format is invalid or corrupted"
	case strings.Contains(errorMsg, "no content and no description"):
		return "Content quality issue - feed provides no readable content"
	default:
		if consecutiveErrors > 1 {
			return fmt.Sprintf("Connection error (%d consecutive failures): %s", consecutiveErrors, errorMsg)
		}
		return fmt.Sprintf("Error: %s", errorMsg)
	}
}
