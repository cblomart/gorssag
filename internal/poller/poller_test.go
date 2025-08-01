package poller

import (
	"testing"
	"time"

	"gorssag/internal/aggregator"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/storage"
)

func TestPoller_New(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{"AI", "blockchain"},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	if p == nil {
		t.Error("Expected poller to be created, got nil")
	}
}

func TestPoller_IsPolling(t *testing.T) {
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
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Initially should not be polling
	if p.IsPolling() {
		t.Error("Expected poller to not be polling initially")
	}

	// Start polling
	p.Start()
	if !p.IsPolling() {
		t.Error("Expected poller to be polling after start")
	}

	// Stop polling
	p.Stop()
	if p.IsPolling() {
		t.Error("Expected poller to not be polling after stop")
	}
}

func TestPoller_GetLastPolledTime(t *testing.T) {
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
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Initially should be empty
	lastPolled := p.GetLastPolledTime()
	if len(lastPolled) != 0 {
		t.Errorf("Expected empty last polled times, got %d", len(lastPolled))
	}

	// Force poll a topic
	err := p.ForcePoll("tech")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should now have a last polled time
	lastPolled = p.GetLastPolledTime()
	if len(lastPolled) == 0 {
		t.Error("Expected non-zero last polled times after force poll")
	}

	// Check if the time is recent
	if lastPolled["tech"].IsZero() {
		t.Error("Expected last polled time to be recent")
	}
}

func TestPoller_ForcePoll_InvalidTopic(t *testing.T) {
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
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Try to force poll an invalid topic
	err := p.ForcePoll("invalid-topic")
	if err == nil {
		t.Error("Expected error for invalid topic, got nil")
	}
}

func TestPoller_ForcePoll_ValidTopic(t *testing.T) {
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
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Test force poll with valid topic
	err := p.ForcePoll("tech")
	if err != nil {
		t.Errorf("Expected no error for valid topic force poll, got %v", err)
	}
}

func TestPoller_StartStopMultipleTimes(t *testing.T) {
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
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Test multiple start/stop cycles
	p.Start()
	if !p.IsPolling() {
		t.Error("Expected poller to be polling after first start")
	}

	p.Stop()
	if p.IsPolling() {
		t.Error("Expected poller to not be polling after first stop")
	}

	p.Start()
	if !p.IsPolling() {
		t.Error("Expected poller to be polling after second start")
	}

	p.Stop()
	if p.IsPolling() {
		t.Error("Expected poller to not be polling after second stop")
	}
}

func TestPoller_ArticleMatchesFilters(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{"AI", "blockchain"},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Test article that matches filters
	article := models.Article{
		Title:       "AI Breakthrough in Blockchain Technology",
		Description: "This article discusses AI and blockchain",
		Content:     "The content contains both AI and blockchain keywords",
	}

	if !p.articleMatchesFilters(article, []string{"AI", "blockchain"}) {
		t.Error("Expected article to match filters")
	}

	// Test article that doesn't match filters
	article = models.Article{
		Title:       "Weather Report",
		Description: "Today's weather is sunny",
		Content:     "No artificial intelligence or distributed ledger content here",
	}

	matches := p.articleMatchesFilters(article, []string{"AI", "blockchain"})
	if matches {
		t.Error("Expected article to not match filters")
	}

	// Test with empty filters - should return false when no filters are specified
	if p.articleMatchesFilters(article, []string{}) {
		t.Error("Expected article to not match when no filters are specified")
	}
}

func TestPoller_FilterArticlesForTopic(t *testing.T) {
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{"AI", "blockchain"},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		ArticleRetention:         24 * time.Hour,
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	p := New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)

	articles := []models.Article{
		{
			Title:       "AI Breakthrough in Blockchain Technology",
			Description: "This article discusses AI and blockchain",
			Content:     "The content contains both AI and blockchain keywords",
		},
		{
			Title:       "Weather Report",
			Description: "Today's weather is sunny",
			Content:     "No artificial intelligence or distributed ledger content here",
		},
		{
			Title:       "Machine Learning Advances",
			Description: "AI and ML technologies",
			Content:     "Contains AI keyword",
		},
	}

	filtered := p.filterArticlesForTopic(articles, []string{"AI", "blockchain"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered articles, got %d", len(filtered))
	}

	// Test with no filters - should return all articles
	filtered = p.filterArticlesForTopic(articles, []string{})
	if len(filtered) != 3 {
		t.Errorf("Expected 3 articles when no filters, got %d", len(filtered))
	}
}

func TestPoller_ConvertHTMLToMarkdown(t *testing.T) {
	// Test HTML to Markdown conversion
	html := "<h1>Title</h1><p>This is a <strong>bold</strong> paragraph with a <a href='http://example.com'>link</a>.</p>"
	markdown := convertHTMLToMarkdown(html)

	if markdown == "" {
		t.Error("Expected non-empty markdown output")
	}

	// Test with empty HTML
	markdown = convertHTMLToMarkdown("")
	if markdown != "" {
		t.Error("Expected empty markdown for empty HTML input")
	}

	// Test with plain text
	markdown = convertHTMLToMarkdown("Plain text without HTML")
	if markdown == "" {
		t.Error("Expected markdown output for plain text")
	}
}

func TestPoller_GenerateArticleID(t *testing.T) {
	// Test article ID generation
	title := "Test Article"
	link := "http://example.com/article"
	publishedAt := time.Now()

	id := generateArticleID(title, link, &publishedAt)
	if id == "" {
		t.Error("Expected non-empty article ID")
	}

	// Test with nil publishedAt
	id = generateArticleID(title, link, nil)
	if id == "" {
		t.Error("Expected non-empty article ID with nil publishedAt")
	}

	// Test with empty title and link
	id = generateArticleID("", "", nil)
	if id == "" {
		t.Error("Expected non-empty article ID with empty inputs")
	}
}
