package aggregator

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/storage"

	"github.com/mmcdole/gofeed"
)

func TestAggregator_GetAvailableTopics(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/tech2"},
			Filters: []string{"AI", "blockchain"},
		},
		"news": {
			URLs:    []string{"http://example.com/news1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	topics := agg.GetAvailableTopics()

	if len(topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(topics))
	}

	// Check if both topics are present
	foundTech := false
	foundNews := false
	for _, topic := range topics {
		if topic == "tech" {
			foundTech = true
		}
		if topic == "news" {
			foundNews = true
		}
	}

	if !foundTech || !foundNews {
		t.Error("Expected topics 'tech' and 'news' not found")
	}
}

func TestAggregator_GetAggregatedFeed_InvalidTopic(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	_, err := agg.GetAggregatedFeed("invalid-topic", nil)
	if err == nil {
		t.Error("Expected error for invalid topic, got nil")
	}
}

func TestAggregator_ApplyODataQuery(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test with nil query
	feed := &models.AggregatedFeed{
		Topic: "test",
		Articles: []models.Article{
			{Title: "Test Article", Author: "Test Author"},
		},
		Count:   1,
		Updated: time.Now(),
	}

	result, err := agg.applyODataQuery(feed, nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(result.Articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(result.Articles))
	}
}

func TestAggregator_FetchFeedsParallel(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/tech2"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles, err := agg.fetchFeedsParallel([]string{"http://example.com/tech1", "http://example.com/tech2"}, "tech")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should handle errors gracefully and return empty articles for invalid URLs
	if len(articles) != 0 {
		t.Errorf("Expected 0 articles for invalid URLs, got %d", len(articles))
	}
}

func TestAggregator_SearchArticles(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles := []models.Article{
		{
			Title:       "AI Revolution in Technology",
			Description: "How AI is changing the world",
			Author:      "John Doe",
		},
		{
			Title:       "Blockchain Basics",
			Description: "Introduction to blockchain technology",
			Author:      "Jane Smith",
		},
	}

	// Test search for "Revolution" (should match first article)
	results := agg.searchArticles(articles, []string{"Revolution"})
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Revolution' search, got %d", len(results))
	}

	// Test search for "blockchain" (should match second article)
	results = agg.searchArticles(articles, []string{"blockchain"})
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'blockchain' search, got %d", len(results))
	}

	// Test search for "technology" (should match both articles)
	results = agg.searchArticles(articles, []string{"technology"})
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'technology' search, got %d", len(results))
	}
}

func TestAggregator_ApplySelectFields(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles := []models.Article{
		{
			Title:       "Test Article",
			Description: "Test Description",
			Author:      "Test Author",
			Source:      "Test Source",
		},
	}

	// Test selecting only title and author
	selectedFields := []string{"title", "author"}
	result := agg.applySelectFields(articles, selectedFields)

	if len(result) != 1 {
		t.Errorf("Expected 1 article, got %d", len(result))
	}

	article := result[0]
	if article.Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", article.Title)
	}
	if article.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", article.Author)
	}
	if article.Description != "" {
		t.Errorf("Expected empty description, got '%s'", article.Description)
	}
	if article.Source != "" {
		t.Errorf("Expected empty source, got '%s'", article.Source)
	}
}

func TestAggregator_ArticleMatchesFilters(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	article := models.Article{
		Title:       "AI Technology News",
		Description: "Latest developments in artificial intelligence",
		Content:     "This article discusses AI breakthroughs",
		Author:      "John Doe",
		Source:      "Tech News",
	}

	// Test with matching filter
	filters := []string{"AI", "artificial intelligence"}
	if !agg.articleMatchesFilters(article, filters) {
		t.Error("Expected article to match AI filter")
	}

	// Test with non-matching filter
	filters = []string{"blockchain", "cryptocurrency"}
	if agg.articleMatchesFilters(article, filters) {
		t.Error("Expected article to not match blockchain filter")
	}

	// Test with empty filters (should not match since no filters means no match)
	filters = []string{}
	if agg.articleMatchesFilters(article, filters) {
		t.Error("Expected article to not match empty filters")
	}
}

func TestAggregator_GetFeedHealth(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test getting feed health
	health := agg.GetFeedHealth()
	if health == nil {
		t.Error("Expected feed health to be returned")
	}

	// Should have health info for configured topics
	if len(health) == 0 {
		t.Error("Expected health info for configured topics")
	}
}

func TestAggregator_GetConfig(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{"AI"},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)
	config := agg.GetConfig()

	if len(config) != 1 {
		t.Errorf("Expected 1 topic config, got %d", len(config))
	}

	if config["tech"].URLs[0] != "http://example.com/tech1" {
		t.Errorf("Expected URL http://example.com/tech1, got %s", config["tech"].URLs[0])
	}
}

func TestAggregator_GetAllUniqueFeedURLs(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/tech2"},
			Filters: []string{"AI"},
		},
		"news": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/news1"}, // tech1 is duplicate
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)
	urls := agg.GetAllUniqueFeedURLs()

	// Should have 3 unique URLs: tech1, tech2, news1
	if len(urls) != 3 {
		t.Errorf("Expected 3 unique URLs, got %d", len(urls))
	}

	// Check for specific URLs
	urlSet := make(map[string]bool)
	for _, url := range urls {
		urlSet[url] = true
	}

	expectedURLs := []string{"http://example.com/tech1", "http://example.com/tech2", "http://example.com/news1"}
	for _, expected := range expectedURLs {
		if !urlSet[expected] {
			t.Errorf("Expected URL %s not found", expected)
		}
	}
}

func TestAggregator_GetTopicsForFeed(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/tech2"},
			Filters: []string{"AI"},
		},
		"news": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/news1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test feed that exists in multiple topics
	topics := agg.GetTopicsForFeed("http://example.com/tech1")
	if len(topics) != 2 {
		t.Errorf("Expected 2 topics for tech1, got %d", len(topics))
	}

	// Test feed that exists in only one topic
	topics = agg.GetTopicsForFeed("http://example.com/tech2")
	if len(topics) != 1 {
		t.Errorf("Expected 1 topic for tech2, got %d", len(topics))
	}
	if topics[0] != "tech" {
		t.Errorf("Expected topic 'tech', got %s", topics[0])
	}

	// Test non-existent feed
	topics = agg.GetTopicsForFeed("http://example.com/nonexistent")
	if len(topics) != 0 {
		t.Errorf("Expected 0 topics for non-existent feed, got %d", len(topics))
	}
}

func TestAggregator_GetFeedStatus(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Initially should be empty
	status := agg.GetFeedStatus()
	if len(status) != 0 {
		t.Errorf("Expected empty status initially, got %d entries", len(status))
	}

	// Update status and check
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 5, nil)
	status = agg.GetFeedStatus()
	if len(status) != 1 {
		t.Errorf("Expected 1 status entry, got %d", len(status))
	}
}

func TestAggregator_UpdateFeedStatus(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test successful update
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 5, nil)
	status := agg.GetFeedStatus()
	if len(status) != 1 {
		t.Errorf("Expected 1 status entry, got %d", len(status))
	}

	// Test error update
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))
	status = agg.GetFeedStatus()
	if status["http://example.com/tech1"].ConsecutiveErrors != 1 {
		t.Errorf("Expected 1 consecutive error, got %d", status["http://example.com/tech1"].ConsecutiveErrors)
	}
}

func TestAggregator_ShouldRetryFeed(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test with no errors
	if !agg.ShouldRetryFeed("http://example.com/tech1") {
		t.Error("Expected ShouldRetryFeed to return true for feed with no errors")
	}

	// Test with errors but within retry limit
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))
	if !agg.ShouldRetryFeed("http://example.com/tech1") {
		t.Error("Expected ShouldRetryFeed to return true for feed with errors within limit")
	}

	// Test with too many errors
	for i := 0; i < 10; i++ {
		agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))
	}
	if agg.ShouldRetryFeed("http://example.com/tech1") {
		t.Error("Expected ShouldRetryFeed to return false for feed with too many errors")
	}
}

func TestAggregator_RefreshFeed(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test refresh with valid topic
	err := agg.RefreshFeed("tech")
	if err != nil {
		t.Errorf("Expected no error for valid topic refresh, got %v", err)
	}

	// Test refresh with invalid topic
	err = agg.RefreshFeed("invalid-topic")
	if err == nil {
		t.Error("Expected error for invalid topic refresh, got nil")
	}
}

func TestAggregator_GetFeedInfo(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test with valid topic - should return error since no feed data exists yet
	_, err := agg.GetFeedInfo("tech")
	if err == nil {
		t.Error("Expected error for topic with no feed data, got nil")
	}

	// Test with invalid topic
	_, err = agg.GetFeedInfo("invalid-topic")
	if err == nil {
		t.Error("Expected error for invalid topic, got nil")
	}
}

func TestAggregator_InitializeFeeds(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// This should not panic
	agg.InitializeFeeds()
}

func TestAggregator_GetStorageStats(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	stats, err := agg.GetStorageStats()
	if err != nil {
		t.Errorf("Expected no error getting storage stats, got %v", err)
	}
	if stats == nil {
		t.Error("Expected storage stats, got nil")
	}
}

func TestAggregator_GetFeedStats(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	stats, err := agg.GetFeedStats()
	if err != nil {
		t.Errorf("Expected no error getting feed stats, got %v", err)
	}
	if stats == nil {
		t.Error("Expected feed stats, got nil")
	}
}

func TestAggregator_SortArticles(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles := []models.Article{
		{
			Title:       "Article B",
			Link:        "http://example.com/b",
			PublishedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Title:       "Article A",
			Link:        "http://example.com/a",
			PublishedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	// Test sorting by title (currently returns unchanged articles)
	sorted := agg.sortArticles(articles, "title")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(sorted))
	}
	// Since sortArticles is a stub, it returns articles unchanged
	if sorted[0].Title != "Article B" {
		t.Errorf("Expected first article to be 'Article B' (unchanged), got %s", sorted[0].Title)
	}

	// Test sorting by published date (currently returns unchanged articles)
	sorted = agg.sortArticles(articles, "publishedAt")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(sorted))
	}
	// Since sortArticles is a stub, it returns articles unchanged
	if sorted[0].Title != "Article B" {
		t.Errorf("Expected first article to be 'Article B' (unchanged), got %s", sorted[0].Title)
	}

	// Test invalid sort field
	sorted = agg.sortArticles(articles, "invalid")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles for invalid sort field, got %d", len(sorted))
	}
}

func TestAggregator_ArticleContains(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	article := models.Article{
		Title:       "Test Article",
		Description: "This is a test article about technology",
		Content:     "The content contains the word technology multiple times",
	}

	// Test matching term
	if !agg.articleContains(article, "technology") {
		t.Error("Expected article to contain 'technology'")
	}

	// Test non-matching term
	if agg.articleContains(article, "nonexistent") {
		t.Error("Expected article to not contain 'nonexistent'")
	}

	// Test case insensitive matching
	if !agg.articleContains(article, "TECHNOLOGY") {
		t.Error("Expected article to contain 'TECHNOLOGY' (case insensitive)")
	}
}

// Test helper functions
func TestConvertHTMLToMarkdown(t *testing.T) {
	// Test basic HTML conversion
	html := "<p>This is a <strong>test</strong> paragraph.</p>"
	expected := "This is a **test** paragraph."
	result := convertHTMLToMarkdown(html)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test CDATA removal
	html = "<![CDATA[<p>CDATA content</p>]]>"
	expected = "CDATA content"
	result = convertHTMLToMarkdown(html)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test HTML entities
	html = "&amp; &lt; &gt; &quot;"
	expected = "&  \""
	result = convertHTMLToMarkdown(html)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test empty input
	result = convertHTMLToMarkdown("")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test image tag removal
	html = "<p>Text <img src='test.jpg' alt='test'> more text</p>"
	expected = "Text test more text"
	result = convertHTMLToMarkdown(html)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test link conversion
	html = "<a href='http://example.com'>Example</a>"
	expected = "[Example](http://example.com)"
	result = convertHTMLToMarkdown(html)
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestGenerateArticleID(t *testing.T) {
	title := "Test Article"
	link := "http://example.com/test"
	publishedAt := time.Now()

	id1 := generateArticleID(title, link, &publishedAt)
	id2 := generateArticleID(title, link, &publishedAt)

	// Should generate same deterministic IDs for same input
	if id1 != id2 {
		t.Error("Expected same deterministic IDs for same input")
	}

	// Should be UUID format (36 characters) - deterministic hash formatted as UUID
	if len(id1) != 36 {
		t.Errorf("Expected UUID format length 36, got %d", len(id1))
	}

	// Test with different input produces different ID
	id3 := generateArticleID("Different Title", link, &publishedAt)
	if id1 == id3 {
		t.Error("Expected different IDs for different input")
	}

	// Test with nil publishedAt
	id4 := generateArticleID(title, link, nil)
	if len(id4) != 36 {
		t.Errorf("Expected UUID format length 36 for nil publishedAt, got %d", len(id4))
	}

	// Test with different link produces different ID
	id5 := generateArticleID(title, "http://different.com/test", &publishedAt)
	if id1 == id5 {
		t.Error("Expected different IDs for different link")
	}

	// Test deterministic behavior - same inputs should always produce same result
	id6 := generateArticleID(title, link, &publishedAt)
	if id1 != id6 {
		t.Error("Expected deterministic behavior - same inputs should produce same result")
	}
}

func TestGetAuthorName(t *testing.T) {
	// Test with author
	item := &gofeed.Item{
		Author: &gofeed.Person{
			Name: "John Doe",
		},
	}
	author := getAuthorName(item)
	if author != "John Doe" {
		t.Errorf("Expected 'John Doe', got '%s'", author)
	}

	// Test with nil author
	item = &gofeed.Item{}
	author = getAuthorName(item)
	if author != "Unknown" {
		t.Errorf("Expected 'Unknown', got '%s'", author)
	}

	// Test with empty author name
	item = &gofeed.Item{
		Author: &gofeed.Person{
			Name: "",
		},
	}
	author = getAuthorName(item)
	if author != "Unknown" {
		t.Errorf("Expected 'Unknown', got '%s'", author)
	}
}

func TestGetPublishedTime(t *testing.T) {
	now := time.Now()
	publishedAt := now.Add(-1 * time.Hour)
	updatedAt := now.Add(-30 * time.Minute)

	// Test with PublishedParsed
	item := &gofeed.Item{
		PublishedParsed: &publishedAt,
		UpdatedParsed:   &updatedAt,
	}
	result := getPublishedTime(item)
	if !result.Equal(publishedAt) {
		t.Errorf("Expected published time, got %v", result)
	}

	// Test with only UpdatedParsed
	item = &gofeed.Item{
		UpdatedParsed: &updatedAt,
	}
	result = getPublishedTime(item)
	if !result.Equal(updatedAt) {
		t.Errorf("Expected updated time, got %v", result)
	}

	// Test with no dates
	item = &gofeed.Item{}
	result = getPublishedTime(item)
	if result.IsZero() {
		t.Error("Expected non-zero time when no dates provided")
	}
}

func TestAggregator_TestUserAgentForFeed(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test with invalid URL
	_, err := agg.TestUserAgentForFeed("http://invalid-url-that-does-not-exist.com/feed")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestAggregator_SetUserAgentForFeed(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test setting user agent
	agg.SetUserAgentForFeed("http://example.com/tech1", "Test User Agent")

	// Verify it was set
	status := agg.GetFeedStatus()
	if status["http://example.com/tech1"].UserAgent != "Test User Agent" {
		t.Error("Expected user agent to be set")
	}
}

func TestAggregator_CalculateBackoff(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test backoff calculation
	backoff1 := agg.calculateBackoff(1)
	if backoff1 != 5 {
		t.Errorf("Expected backoff 5 for 1 error, got %d", backoff1)
	}

	backoff2 := agg.calculateBackoff(2)
	if backoff2 != 10 {
		t.Errorf("Expected backoff 10 for 2 errors, got %d", backoff2)
	}

	backoff3 := agg.calculateBackoff(3)
	if backoff3 != 20 {
		t.Errorf("Expected backoff 20 for 3 errors, got %d", backoff3)
	}

	// Test max backoff
	backoffMax := agg.calculateBackoff(10)
	if backoffMax != 60 {
		t.Errorf("Expected max backoff 60, got %d", backoffMax)
	}
}

func TestAggregator_GetSpecificErrorReason(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test various error types
	testCases := []struct {
		errorMsg string
		expected string
	}{
		{"404 Not Found", "Feed URL not found (404) - feed may have been moved or discontinued"},
		{"403 Forbidden", "Access forbidden (403) - feed may require authentication or be blocked"},
		{"401 Unauthorized", "Unauthorized (401) - feed requires authentication"},
		{"500 Internal Server Error", "Server error (500) - feed server is experiencing issues"},
		{"timeout", "Connection timeout - feed server is slow or unresponsive"},
		{"connection refused", "Connection refused - feed server is not accepting connections"},
		{"no such host", "DNS resolution failed - feed domain does not exist or is unreachable"},
		{"eof", "Connection closed unexpectedly (EOF) - feed server terminated connection"},
		{"ssl error", "SSL/TLS error - feed has certificate issues"},
		{"certificate error", "Certificate error - feed has invalid or expired SSL certificate"},
		{"parse error", "Feed parsing error - feed format is invalid or corrupted"},
		{"no content and no description", "Content quality issue - feed provides no readable content"},
		{"unknown error", "Error: unknown error"},
	}

	for _, tc := range testCases {
		result := agg.getSpecificErrorReason(tc.errorMsg, 1)
		if result != tc.expected {
			t.Errorf("For error '%s', expected '%s', got '%s'", tc.errorMsg, tc.expected, result)
		}
	}

	// Test with consecutive errors
	result := agg.getSpecificErrorReason("test error", 3)
	if !strings.Contains(result, "3 consecutive failures") {
		t.Errorf("Expected consecutive failures message, got '%s'", result)
	}
}

func TestAggregator_FilterArticlesForTopic(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles := []models.Article{
		{
			Title:       "AI Technology News",
			Description: "Latest developments in artificial intelligence",
			Content:     "This article discusses AI breakthroughs",
		},
		{
			Title:       "Weather Report",
			Description: "Today's weather is sunny",
			Content:     "No artificial intelligence or distributed ledger content here",
		},
	}

	// Test with no filters
	filtered := agg.filterArticlesForTopic(articles, []string{})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 articles with no filters, got %d", len(filtered))
	}

	// Test with matching filters
	filtered = agg.filterArticlesForTopic(articles, []string{"AI"})
	if len(filtered) != 1 {
		t.Errorf("Expected 1 article with AI filter, got %d", len(filtered))
	}

	// Test with non-matching filters
	filtered = agg.filterArticlesForTopic(articles, []string{"blockchain"})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 articles with blockchain filter, got %d", len(filtered))
	}
}

func TestAggregator_PollAllFeeds(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1", "http://example.com/tech2"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test polling all feeds (should handle errors gracefully)
	err := agg.PollAllFeeds()
	if err == nil {
		t.Error("Expected error when polling invalid feeds")
	}
}

func TestAggregator_ApplyODataQueryWithFilter(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	feed := &models.AggregatedFeed{
		Topic: "test",
		Articles: []models.Article{
			{Title: "Test Article", Author: "Test Author"},
		},
		Count:   1,
		Updated: time.Now(),
	}

	// Test with invalid filter expression
	query := &models.ODataQuery{
		Filter: "invalid filter expression",
	}

	_, err := agg.applyODataQuery(feed, query)
	if err == nil {
		t.Error("Expected error for invalid filter expression")
	}
}

func TestAggregator_ApplyODataQueryWithPagination(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	feed := &models.AggregatedFeed{
		Topic: "test",
		Articles: []models.Article{
			{Title: "Article 1"},
			{Title: "Article 2"},
			{Title: "Article 3"},
		},
		Count:   3,
		Updated: time.Now(),
	}

	// Test skip
	query := &models.ODataQuery{
		Skip: 1,
	}
	result, err := agg.applyODataQuery(feed, query)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result.Articles) != 2 {
		t.Errorf("Expected 2 articles after skip, got %d", len(result.Articles))
	}

	// Test top
	query = &models.ODataQuery{
		Top: 2,
	}
	result, err = agg.applyODataQuery(feed, query)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result.Articles) != 2 {
		t.Errorf("Expected 2 articles with top=2, got %d", len(result.Articles))
	}

	// Test skip beyond available articles
	query = &models.ODataQuery{
		Skip: 10,
	}
	result, err = agg.applyODataQuery(feed, query)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result.Articles) != 0 {
		t.Errorf("Expected 0 articles after skip beyond available, got %d", len(result.Articles))
	}
}

func TestAggregator_ApplyODataQueryWithSearch(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	feed := &models.AggregatedFeed{
		Topic: "test",
		Articles: []models.Article{
			{Title: "AI Technology News", Content: "AI is changing the world"},
			{Title: "Weather Report", Content: "Today is sunny"},
		},
		Count:   2,
		Updated: time.Now(),
	}

	// Test search
	query := &models.ODataQuery{
		Search: []string{"AI"},
	}
	result, err := agg.applyODataQuery(feed, query)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(result.Articles) != 1 {
		t.Errorf("Expected 1 article matching 'AI', got %d", len(result.Articles))
	}
}

func TestAggregator_GetFeedHealthWithErrors(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Add some error states
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("404 Not Found"))
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("404 Not Found"))
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("404 Not Found"))

	health := agg.GetFeedHealth()
	if health == nil {
		t.Error("Expected feed health to be returned")
	}

	// Should have health info for configured topics
	if len(health) == 0 {
		t.Error("Expected health info for configured topics")
	}
}

func TestAggregator_UpdateFeedStatusWithContentIssue(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Test content quality issue
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("no content and no description"))
	status := agg.GetFeedStatus()

	if !status["http://example.com/tech1"].IsContentIssue {
		t.Error("Expected content issue flag to be set")
	}

	if !status["http://example.com/tech1"].IsDisabled {
		t.Error("Expected feed to be disabled for content issue")
	}
}

func TestAggregator_UpdateFeedStatusWithManyErrors(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Add many consecutive errors
	for i := 0; i < 10; i++ {
		agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))
	}

	status := agg.GetFeedStatus()
	if !status["http://example.com/tech1"].IsDisabled {
		t.Error("Expected feed to be disabled after many errors")
	}

	if status["http://example.com/tech1"].ConsecutiveErrors != 10 {
		t.Errorf("Expected 10 consecutive errors, got %d", status["http://example.com/tech1"].ConsecutiveErrors)
	}
}

func TestAggregator_UpdateFeedStatusSuccess(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}
	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	// Add some errors first
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 0, fmt.Errorf("test error"))

	// Then success
	agg.UpdateFeedStatus("http://example.com/tech1", "tech", 5, nil)

	status := agg.GetFeedStatus()
	if status["http://example.com/tech1"].ConsecutiveErrors != 0 {
		t.Errorf("Expected 0 consecutive errors after success, got %d", status["http://example.com/tech1"].ConsecutiveErrors)
	}

	if status["http://example.com/tech1"].IsDisabled {
		t.Error("Expected feed to not be disabled after success")
	}

	if status["http://example.com/tech1"].ArticlesCount != 5 {
		t.Errorf("Expected 5 articles count, got %d", status["http://example.com/tech1"].ArticlesCount)
	}
}
