package aggregator

import (
	"testing"
	"time"

	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/storage"
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
	storageManager, _ := storage.NewStorage("./testdata")
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
	storageManager, _ := storage.NewStorage("./testdata")
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
	storageManager, _ := storage.NewStorage("./testdata")
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
	storageManager, _ := storage.NewStorage("./testdata")
	defer storageManager.Close()

	agg := New(cacheManager, storageManager, feeds)

	articles, err := agg.fetchFeedsParallel([]string{"http://example.com/tech1", "http://example.com/tech2"})
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
	storageManager, _ := storage.NewStorage("./testdata")
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
	storageManager, _ := storage.NewStorage("./testdata")
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
