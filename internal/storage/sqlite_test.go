package storage

import (
	"testing"
	"time"

	"gorssag/internal/models"
)

func TestSQLiteStorage_BasicOperations(t *testing.T) {
	// Use a temporary directory for testing
	tempDir := t.TempDir()

	// Create SQLite storage
	storage, err := NewSQLiteStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test data
	articles := []models.Article{
		{
			Title:       "Test Article 1",
			Link:        "https://example.com/1",
			Description: "Test description 1",
			Content:     "Test content 1",
			Author:      "John Doe",
			Source:      "Test Source",
			Categories:  []string{"tech", "test"},
			PublishedAt: time.Now(),
		},
		{
			Title:       "Test Article 2",
			Link:        "https://example.com/2",
			Description: "Test description 2",
			Content:     "Test content 2",
			Author:      "Jane Smith",
			Source:      "Test Source 2",
			Categories:  []string{"news", "test"},
			PublishedAt: time.Now().Add(-time.Hour),
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "test-topic",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Test SaveFeed
	err = storage.SaveFeed("test-topic", feed)
	if err != nil {
		t.Fatalf("Failed to save feed: %v", err)
	}

	// Test LoadFeed
	loadedFeed, err := storage.LoadFeed("test-topic")
	if err != nil {
		t.Fatalf("Failed to load feed: %v", err)
	}

	if loadedFeed.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", loadedFeed.Topic)
	}

	if len(loadedFeed.Articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(loadedFeed.Articles))
	}

	// Test ListTopics
	topics, err := storage.ListTopics()
	if err != nil {
		t.Fatalf("Failed to list topics: %v", err)
	}

	if len(topics) != 1 {
		t.Errorf("Expected 1 topic, got %d", len(topics))
	}

	if topics[0] != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", topics[0])
	}

	// Test GetFeedInfo
	info, err := storage.GetFeedInfo("test-topic")
	if err != nil {
		t.Fatalf("Failed to get feed info: %v", err)
	}

	if info.Topic != "test-topic" {
		t.Errorf("Expected topic 'test-topic', got '%s'", info.Topic)
	}

	if info.ArticleCount != 2 {
		t.Errorf("Expected 2 articles, got %d", info.ArticleCount)
	}
}

func TestSQLiteStorage_ODataQueries(t *testing.T) {
	tempDir := t.TempDir()

	storage, err := NewSQLiteStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test data
	articles := []models.Article{
		{
			Title:       "AI Technology News",
			Link:        "https://example.com/ai",
			Description: "Latest developments in AI",
			Content:     "AI is transforming technology",
			Author:      "John Doe",
			Source:      "Tech News",
			Categories:  []string{"AI", "technology"},
			PublishedAt: time.Now(),
		},
		{
			Title:       "Blockchain Revolution",
			Link:        "https://example.com/blockchain",
			Description: "Cryptocurrency and blockchain",
			Content:     "Blockchain is changing finance",
			Author:      "Jane Smith",
			Source:      "Crypto News",
			Categories:  []string{"blockchain", "cryptocurrency"},
			PublishedAt: time.Now().Add(-time.Hour),
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "tech",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("tech", feed)
	if err != nil {
		t.Fatalf("Failed to save feed: %v", err)
	}

	// Test search query
	query := &models.ODataQuery{
		Search: []string{"Blockchain"},
	}

	results, err := storage.QueryArticles("tech", query)
	if err != nil {
		t.Fatalf("Failed to query articles: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Blockchain' search, got %d", len(results))
	}

	if results[0].Title != "Blockchain Revolution" {
		t.Errorf("Expected 'Blockchain Revolution', got '%s'", results[0].Title)
	}

	// Test top limit
	query = &models.ODataQuery{
		Top: 1,
	}

	results, err = storage.QueryArticles("tech", query)
	if err != nil {
		t.Fatalf("Failed to query articles with top limit: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result with top=1, got %d", len(results))
	}
}

func TestSQLiteStorage_StorageFactory(t *testing.T) {
	tempDir := t.TempDir()

	// Test SQLite storage creation via factory
	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage via factory: %v", err)
	}
	defer storage.Close()

	// Verify it's a SQLite storage instance
	if storage == nil {
		t.Error("Storage should not be nil")
	}
}
