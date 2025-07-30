package aggregator

import (
	"fmt"
	"log"
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

type Aggregator struct {
	cacheManager *cache.Manager
	storage      storage.Storage
	feeds        map[string]config.TopicConfig
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
	}
}

func (a *Aggregator) GetAvailableTopics() []string {
	var topics []string
	for topic := range a.feeds {
		topics = append(topics, topic)
	}
	return topics
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
	articles, err := a.fetchFeedsParallel(topicConfig.URLs)
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

func (a *Aggregator) fetchFeedsParallel(feedURLs []string) ([]models.Article, error) {
	var wg sync.WaitGroup
	results := make(chan FeedResult, len(feedURLs))

	// Start goroutines for each feed URL
	for _, url := range feedURLs {
		wg.Add(1)
		go func(feedURL string) {
			defer wg.Done()
			articles, err := a.fetchFeed(feedURL)
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

func (a *Aggregator) fetchFeed(url string) ([]models.Article, error) {
	feed, err := a.parser.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %v", err)
	}

	var articles []models.Article
	for _, item := range feed.Items {
		// Safely get author name
		authorName := ""
		if item.Author != nil {
			authorName = item.Author.Name
		}

		article := models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
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
