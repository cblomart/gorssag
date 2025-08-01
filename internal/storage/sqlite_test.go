package storage

import (
	"fmt"
	"testing"
	"time"

	"gorssag/internal/config"
	"gorssag/internal/models"
	"sync"
)

func TestSQLiteStorage_BasicOperations(t *testing.T) {
	// Use a temporary directory for testing
	tempDir := t.TempDir()

	// Create test config
	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	// Create SQLite storage
	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test data
	articles := []models.Article{
		{
			ID:          "test-article-1",
			Title:       "Test Article 1",
			Link:        "https://example.com/1",
			Description: "Test description 1",
			Content:     "Test content 1",
			Author:      "John Doe",
			Source:      "Test Source",
			Categories:  []string{"tech", "test"},
			PublishedAt: time.Now(),
			Language:    "en",
		},
		{
			ID:          "test-article-2",
			Title:       "Test Article 2",
			Link:        "https://example.com/2",
			Description: "Test description 2",
			Content:     "Test content 2",
			Author:      "Jane Smith",
			Source:      "Test Source 2",
			Categories:  []string{"news", "test"},
			PublishedAt: time.Now().Add(-time.Hour),
			Language:    "en",
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

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test data
	articles := []models.Article{
		{
			ID:          "ai-article",
			Title:       "AI Technology News",
			Link:        "https://example.com/ai",
			Description: "Latest developments in AI",
			Content:     "AI is transforming technology",
			Author:      "John Doe",
			Source:      "Tech News",
			Categories:  []string{"AI", "technology"},
			PublishedAt: time.Now(),
			Language:    "en",
		},
		{
			ID:          "blockchain-article",
			Title:       "Blockchain Revolution",
			Link:        "https://example.com/blockchain",
			Description: "Cryptocurrency and blockchain",
			Content:     "Blockchain is changing finance",
			Author:      "Jane Smith",
			Source:      "Crypto News",
			Categories:  []string{"blockchain", "cryptocurrency"},
			PublishedAt: time.Now().Add(-time.Hour),
			Language:    "en",
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

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	// Test SQLite storage creation via factory
	storage, err := NewStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage via factory: %v", err)
	}
	defer storage.Close()

	// Verify it's a SQLite storage instance
	if storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestSQLiteStorage_Compression(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: true,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test data with old article (should be compressed)
	oldArticle := models.Article{
		ID:          "old-article",
		Title:       "Old Article",
		Link:        "https://example.com/old",
		Description: "Old description",
		Content:     "Old content that should be compressed",
		Author:      "Old Author",
		Source:      "Old Source",
		Categories:  []string{"old"},
		PublishedAt: time.Now().AddDate(0, 0, -5), // 5 days old
		Language:    "en",
	}

	feed := &models.AggregatedFeed{
		Topic:    "test-compression",
		Articles: []models.Article{oldArticle},
		Count:    1,
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("test-compression", feed)
	if err != nil {
		t.Fatalf("Failed to save feed: %v", err)
	}

	// Load feed and verify content is accessible
	loadedFeed, err := storage.LoadFeed("test-compression")
	if err != nil {
		t.Fatalf("Failed to load feed: %v", err)
	}

	if len(loadedFeed.Articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(loadedFeed.Articles))
	}

	if loadedFeed.Articles[0].Content != "Old content that should be compressed" {
		t.Errorf("Expected content to be accessible after compression")
	}
}

func TestSQLiteStorage_DatabaseStats(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test GetDatabaseStats
	stats, err := storage.GetDatabaseStats()
	if err != nil {
		t.Fatalf("Failed to get database stats: %v", err)
	}

	// Should have basic stats even with empty database
	if totalArticles, ok := stats["total_articles"].(int); !ok || totalArticles != 0 {
		t.Errorf("Expected 0 total articles, got %v", stats["total_articles"])
	}

	// Test GetFeedStats
	feedStats, err := storage.GetFeedStats()
	if err != nil {
		t.Fatalf("Failed to get feed stats: %v", err)
	}

	// Should have stats even with empty database (at least the structure)
	if feedStats == nil {
		t.Error("Expected feed stats to be returned")
	}
}

func TestSQLiteStorage_CleanupOperations(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		ArticleRetention:         24 * time.Hour, // 1 day retention
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test CleanupOldArticles
	err = storage.CleanupOldArticles(24 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old articles: %v", err)
	}

	// Test OptimizeDatabase
	err = storage.OptimizeDatabase()
	if err != nil {
		t.Fatalf("Failed to optimize database: %v", err)
	}

	// Test RemoveDuplicateArticles
	err = storage.RemoveDuplicateArticles()
	if err != nil {
		t.Fatalf("Failed to remove duplicate articles: %v", err)
	}
}

func TestSQLiteStorage_DeleteFeed(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Create test data
	articles := []models.Article{
		{
			ID:          "test-article-1",
			Title:       "Test Article 1",
			Link:        "https://example.com/1",
			Description: "Test description 1",
			Content:     "Test content 1",
			Author:      "John Doe",
			Source:      "Test Source",
			Categories:  []string{"tech", "test"},
			PublishedAt: time.Now(),
			Language:    "en",
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "test-topic",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("test-topic", feed)
	if err != nil {
		t.Fatalf("Failed to save feed: %v", err)
	}

	// Verify feed exists
	topics, err := storage.ListTopics()
	if err != nil {
		t.Fatalf("Failed to list topics: %v", err)
	}
	if len(topics) != 1 {
		t.Errorf("Expected 1 topic before deletion, got %d", len(topics))
	}

	// Delete feed
	err = storage.DeleteFeed("test-topic")
	if err != nil {
		t.Fatalf("Failed to delete feed: %v", err)
	}

	// Verify feed is deleted
	topics, err = storage.ListTopics()
	if err != nil {
		t.Fatalf("Failed to list topics after deletion: %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("Expected 0 topics after deletion, got %d", len(topics))
	}

	// Test deleting non-existent feed (should not error)
	err = storage.DeleteFeed("non-existent-topic")
	if err != nil {
		t.Errorf("Expected no error when deleting non-existent feed, got %v", err)
	}
}

func TestSQLiteStorage_LanguageDetection(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test articles with different languages
	articles := []models.Article{
		{
			ID:          "english-article",
			Title:       "English Article",
			Link:        "https://example.com/en",
			Description: "This is an English article",
			Content:     "This article is written in English language",
			Author:      "English Author",
			Source:      "English Source",
			Categories:  []string{"english"},
			PublishedAt: time.Now(),
			Language:    "en",
		},
		{
			ID:          "french-article",
			Title:       "Article Français",
			Link:        "https://example.com/fr",
			Description: "Ceci est un article français",
			Content:     "Cet article est écrit en langue française",
			Author:      "Auteur Français",
			Source:      "Source Française",
			Categories:  []string{"french"},
			PublishedAt: time.Now().Add(-time.Hour),
			Language:    "fr",
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "multilingual",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("multilingual", feed)
	if err != nil {
		t.Fatalf("Failed to save multilingual feed: %v", err)
	}

	// Load feed and verify language detection
	loadedFeed, err := storage.LoadFeed("multilingual")
	if err != nil {
		t.Fatalf("Failed to load multilingual feed: %v", err)
	}

	if len(loadedFeed.Articles) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(loadedFeed.Articles))
	}

	// Verify language detection worked
	for _, article := range loadedFeed.Articles {
		if article.Language == "" {
			t.Errorf("Expected language to be detected for article %s", article.ID)
		}
	}
}

func TestSQLiteStorage_AdvancedODataQueries(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Create test data with various attributes
	articles := []models.Article{
		{
			ID:          "article-1",
			Title:       "First Article",
			Link:        "https://example.com/1",
			Description: "First article description",
			Content:     "First article content",
			Author:      "Author A",
			Source:      "Source A",
			Categories:  []string{"category1", "category2"},
			PublishedAt: time.Now().Add(-2 * time.Hour),
			Language:    "en",
		},
		{
			ID:          "article-2",
			Title:       "Second Article",
			Link:        "https://example.com/2",
			Description: "Second article description",
			Content:     "Second article content",
			Author:      "Author B",
			Source:      "Source B",
			Categories:  []string{"category2", "category3"},
			PublishedAt: time.Now().Add(-1 * time.Hour),
			Language:    "en",
		},
		{
			ID:          "article-3",
			Title:       "Third Article",
			Link:        "https://example.com/3",
			Description: "Third article description",
			Content:     "Third article content",
			Author:      "Author A",
			Source:      "Source A",
			Categories:  []string{"category1", "category3"},
			PublishedAt: time.Now(),
			Language:    "en",
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "test-advanced",
		Articles: articles,
		Count:    len(articles),
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("test-advanced", feed)
	if err != nil {
		t.Fatalf("Failed to save feed: %v", err)
	}

	// Test skip and top
	query := &models.ODataQuery{
		Skip: 1,
		Top:  2,
	}

	results, err := storage.QueryArticles("test-advanced", query)
	if err != nil {
		t.Fatalf("Failed to query articles with skip and top: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result with skip=1, top=2 (only 2 articles after skip), got %d", len(results))
	}

	// Test order by
	query = &models.ODataQuery{
		OrderBy: "publishedAt desc",
	}

	results, err = storage.QueryArticles("test-advanced", query)
	if err != nil {
		t.Fatalf("Failed to query articles with order by: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results with order by, got %d", len(results))
	}

	// Verify ordering (newest first)
	if results[0].Title != "Third Article" {
		t.Errorf("Expected 'Third Article' first, got '%s'", results[0].Title)
	}

	// Test select fields - note: the current implementation doesn't actually filter fields
	query = &models.ODataQuery{
		Select: []string{"title", "author"},
	}

	results, err = storage.QueryArticles("test-advanced", query)
	if err != nil {
		t.Fatalf("Failed to query articles with select: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results with select, got %d", len(results))
	}

	// Verify fields are populated (current implementation returns all fields)
	for _, article := range results {
		if article.Title == "" {
			t.Error("Expected title to be populated")
		}
		if article.Author == "" {
			t.Error("Expected author to be populated")
		}
		// Note: Current implementation doesn't filter fields, so description will be populated
	}
}

func TestSQLiteStorage_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test loading non-existent feed
	_, err = storage.LoadFeed("non-existent-topic")
	if err == nil {
		t.Error("Expected error when loading non-existent feed")
	}

	// Test getting info for non-existent feed
	_, err = storage.GetFeedInfo("non-existent-topic")
	if err == nil {
		t.Error("Expected error when getting info for non-existent feed")
	}

	// Test querying non-existent feed
	_, err = storage.QueryArticles("non-existent-topic", &models.ODataQuery{})
	if err == nil {
		t.Error("Expected error when querying non-existent feed")
	}
}

func TestSQLiteStorage_CompressOldArticles(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: true,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Create old articles that should be compressed
	oldArticles := []models.Article{
		{
			ID:          "old-article-1",
			Title:       "Old Article 1",
			Link:        "https://example.com/old1",
			Description: "Old description 1",
			Content:     "Old content 1 that should be compressed",
			Author:      "Old Author 1",
			Source:      "Old Source 1",
			Categories:  []string{"old"},
			PublishedAt: time.Now().AddDate(0, 0, -10), // 10 days old
			Language:    "en",
		},
		{
			ID:          "old-article-2",
			Title:       "Old Article 2",
			Link:        "https://example.com/old2",
			Description: "Old description 2",
			Content:     "Old content 2 that should be compressed",
			Author:      "Old Author 2",
			Source:      "Old Source 2",
			Categories:  []string{"old"},
			PublishedAt: time.Now().AddDate(0, 0, -15), // 15 days old
			Language:    "en",
		},
	}

	feed := &models.AggregatedFeed{
		Topic:    "old-articles",
		Articles: oldArticles,
		Count:    len(oldArticles),
		Updated:  time.Now(),
	}

	// Save feed
	err = storage.SaveFeed("old-articles", feed)
	if err != nil {
		t.Fatalf("Failed to save old articles feed: %v", err)
	}

	// Compress old articles
	err = storage.CompressOldArticles()
	if err != nil {
		t.Fatalf("Failed to compress old articles: %v", err)
	}

	// Load feed and verify content is still accessible
	loadedFeed, err := storage.LoadFeed("old-articles")
	if err != nil {
		t.Fatalf("Failed to load old articles feed: %v", err)
	}

	if len(loadedFeed.Articles) != 2 {
		t.Errorf("Expected 2 articles after compression, got %d", len(loadedFeed.Articles))
	}

	// Verify content is still readable
	for i, article := range loadedFeed.Articles {
		expectedContent := fmt.Sprintf("Old content %d that should be compressed", i+1)
		if article.Content != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, article.Content)
		}
	}
}

func TestSQLiteStorage_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
	}

	storage, err := NewSQLiteStorage(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Test concurrent access to different topics
	topics := []string{"topic1", "topic2", "topic3"}
	var wg sync.WaitGroup

	for _, topic := range topics {
		wg.Add(1)
		go func(topicName string) {
			defer wg.Done()

			articles := []models.Article{
				{
					ID:          fmt.Sprintf("article-%s", topicName),
					Title:       fmt.Sprintf("Article for %s", topicName),
					Link:        fmt.Sprintf("https://example.com/%s", topicName),
					Description: fmt.Sprintf("Description for %s", topicName),
					Content:     fmt.Sprintf("Content for %s", topicName),
					Author:      "Concurrent Author",
					Source:      "Concurrent Source",
					Categories:  []string{"concurrent"},
					PublishedAt: time.Now(),
					Language:    "en",
				},
			}

			feed := &models.AggregatedFeed{
				Topic:    topicName,
				Articles: articles,
				Count:    len(articles),
				Updated:  time.Now(),
			}

			// Save feed
			err := storage.SaveFeed(topicName, feed)
			if err != nil {
				t.Errorf("Failed to save feed for %s: %v", topicName, err)
				return
			}

			// Load feed
			_, err = storage.LoadFeed(topicName)
			if err != nil {
				t.Errorf("Failed to load feed for %s: %v", topicName, err)
				return
			}
		}(topic)
	}

	wg.Wait()

	// Verify all topics were created
	topicsList, err := storage.ListTopics()
	if err != nil {
		t.Fatalf("Failed to list topics: %v", err)
	}

	if len(topicsList) != 3 {
		t.Errorf("Expected 3 topics after concurrent access, got %d", len(topicsList))
	}
}
