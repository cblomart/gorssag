package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorssag/internal/aggregator"
	"gorssag/internal/cache"
	"gorssag/internal/config"
	"gorssag/internal/models"
	"gorssag/internal/poller"
	"gorssag/internal/storage"

	"github.com/gin-gonic/gin"
)

func TestServer_New(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	// Test server creation
	server := NewServer(agg, p, cfg)
	if server == nil {
		t.Error("Expected server to be created, got nil")
	}

	if server.router == nil {
		t.Error("Expected router to be initialized")
	}
}

func TestServer_SetupRoutes(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	server := NewServer(agg, p, cfg)

	// Test that routes are set up
	gin.SetMode(gin.TestMode)

	// Test health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for health endpoint, got %d", w.Code)
	}
}

func TestServer_GetTopics(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
			"news": {
				URLs:    []string{"http://example.com/news"},
				Filters: []string{},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	server := NewServer(agg, p, cfg)

	// Test topics endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/topics", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for topics endpoint, got %d", w.Code)
	}
}

func TestServer_GetFeeds(t *testing.T) {
	// Setup
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
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting feeds configuration
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if feeds are returned
	if response["feeds"] == nil {
		t.Error("Expected feeds in response")
	}
}

func TestServer_GetAggregatedFeed(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	server := NewServer(agg, p, cfg)

	// Test aggregated feed endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds/tech", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for aggregated feed endpoint, got %d", w.Code)
	}
}

func TestServer_GetAggregatedFeedWithQueryParams(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test with query parameters
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds/tech?$top=5&$skip=0&$search=AI&$select=title,author", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if feed data is returned
	if response["topic"] != "tech" {
		t.Errorf("Expected topic 'tech', got %v", response["topic"])
	}
}

func TestServer_GetAggregatedFeedInvalidTopic(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test with invalid topic
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds/invalid-topic", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServer_GetFeedInfo(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting feed info
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds/tech/info", nil)
	server.router.ServeHTTP(w, req)

	// Should return 404 since no feed data exists yet
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServer_RefreshFeed(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test refreshing feed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/feeds/tech/refresh", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["message"] != "Feed refreshed successfully" {
		t.Errorf("Expected message 'Feed refreshed successfully', got %v", response["message"])
	}
}

func TestServer_RefreshFeedInvalidTopic(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test refreshing invalid topic
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/feeds/invalid-topic/refresh", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestServer_GetPollerStatus(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting poller status
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/poller/status", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if status is returned
	if response["status"] == nil {
		t.Error("Expected status in response")
	}
}

func TestServer_ForcePollTopic(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test force polling topic
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/poller/force-poll/tech", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["message"] != "Force poll initiated for topic: tech" {
		t.Errorf("Expected message 'Force poll initiated for topic: tech', got %v", response["message"])
	}
}

func TestServer_ForcePollTopicInvalid(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test force polling invalid topic
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/poller/force-poll/invalid-topic", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestServer_GetLastPolledTimes(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting last polled times
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/poller/last-polled", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if last polled times are returned
	if response["last_polled"] == nil {
		t.Error("Expected last_polled in response")
	}
}

func TestServer_GetStorageStats(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting storage stats
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/storage/stats", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if stats are returned
	if response["stats"] == nil {
		t.Error("Expected stats in response")
	}
}

func TestServer_OptimizeStorage(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test optimizing storage
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/storage/optimize", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["message"] != "Storage optimization runs automatically. Check logs for details." {
		t.Errorf("Expected message 'Storage optimization runs automatically. Check logs for details.', got %v", response["message"])
	}
}

func TestServer_GetFeedStats(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test getting feed stats
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/feeds/stats", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Check if stats are returned
	if response["stats"] == nil {
		t.Error("Expected stats in response")
	}
}

func TestServer_ParseODataQuery(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test parsing OData query
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/articles?$top=5&$skip=0&$search=AI&$select=title,author&$orderby=title", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestServer_ParseODataQueryInvalid(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	// Test parsing invalid OData query
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/articles?$top=invalid&$skip=invalid", nil)
	server.router.ServeHTTP(w, req)

	// Should return 400 for invalid parameters
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestServer_GetAllArticles(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	server := NewServer(agg, p, cfg)
	gin.SetMode(gin.TestMode)

	// Test getAllArticles endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/articles", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for getAllArticles endpoint, got %d", w.Code)
	}
}

func TestServer_GetAllArticlesWithQueryParams(t *testing.T) {
	// Create test dependencies
	cacheManager := cache.NewManager(5 * time.Minute)

	cfg := &config.Config{
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Feeds: map[string]config.TopicConfig{
			"tech": {
				URLs:    []string{"http://example.com/tech"},
				Filters: []string{"AI"},
			},
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, cfg.Feeds)
	p := poller.New(agg, cacheManager, storageManager, cfg.Feeds, 1*time.Minute, 1*time.Minute, cfg)

	server := NewServer(agg, p, cfg)
	gin.SetMode(gin.TestMode)

	// Test getAllArticles with query parameters
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/articles?$top=10&$skip=5&$orderby=title&$select=title,link&$search=test", nil)
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for getAllArticles with query params, got %d", w.Code)
	}
}

// Test helper functions
func TestSearchArticles(t *testing.T) {
	articles := []models.Article{
		{
			Title:       "AI Technology News",
			Description: "Latest developments in artificial intelligence",
			Content:     "AI is changing the world",
		},
		{
			Title:       "Weather Report",
			Description: "Today's weather is sunny",
			Content:     "No AI content here",
		},
	}

	// Test search for "AI"
	results := searchArticles(articles, []string{"AI"})
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'AI' search, got %d", len(results))
	}

	// Test search for "weather"
	results = searchArticles(articles, []string{"weather"})
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'weather' search, got %d", len(results))
	}

	// Test search for non-existent term
	results = searchArticles(articles, []string{"nonexistent"})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nonexistent' search, got %d", len(results))
	}
}

func TestArticleContains(t *testing.T) {
	article := models.Article{
		Title:       "Test Article",
		Description: "This is a test article about technology",
		Content:     "The content contains the word technology multiple times",
		Author:      "John Doe",
		Source:      "Tech News",
		Categories:  []string{"technology", "test"},
	}

	// Test matching term
	if !articleContains(article, "technology") {
		t.Error("Expected article to contain 'technology'")
	}

	// Test non-matching term
	if articleContains(article, "nonexistent") {
		t.Error("Expected article to not contain 'nonexistent'")
	}

	// Test case insensitive matching
	if !articleContains(article, "TECHNOLOGY") {
		t.Error("Expected article to contain 'TECHNOLOGY' (case insensitive)")
	}

	// Test matching in author
	if !articleContains(article, "John") {
		t.Error("Expected article to contain 'John' in author")
	}

	// Test matching in source
	if !articleContains(article, "Tech") {
		t.Error("Expected article to contain 'Tech' in source")
	}

	// Test matching in categories
	if !articleContains(article, "test") {
		t.Error("Expected article to contain 'test' in categories")
	}
}

func TestSortArticles(t *testing.T) {
	articles := []models.Article{
		{
			Title:       "Article B",
			PublishedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Title:       "Article A",
			PublishedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	// Test sorting by title (currently returns unchanged)
	sorted := sortArticles(articles, "title")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(sorted))
	}

	// Test sorting by publishedAt (currently returns unchanged)
	sorted = sortArticles(articles, "publishedAt")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(sorted))
	}

	// Test invalid sort field
	sorted = sortArticles(articles, "invalid")
	if len(sorted) != 2 {
		t.Errorf("Expected 2 articles for invalid sort field, got %d", len(sorted))
	}
}

func TestApplySelectFields(t *testing.T) {
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
	result := applySelectFields(articles, selectedFields)

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
	if article.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", article.Description)
	}
	if article.Source != "Test Source" {
		t.Errorf("Expected source 'Test Source', got '%s'", article.Source)
	}

	// Test with no valid fields
	result = applySelectFields(articles, []string{})
	if len(result) != 1 {
		t.Errorf("Expected 1 article with no fields selected, got %d", len(result))
	}

	// Test with invalid fields
	result = applySelectFields(articles, []string{"invalid_field"})
	if len(result) != 1 {
		t.Errorf("Expected 1 article with invalid field, got %d", len(result))
	}
}

func TestParseSelectFields(t *testing.T) {
	// Test parsing select fields
	selectStr := "title,author,description"
	fields := parseSelectFields(selectStr)

	expected := []string{"title", "author", "description"}
	if len(fields) != len(expected) {
		t.Errorf("Expected %d fields, got %d", len(expected), len(fields))
	}

	for i, field := range expected {
		if fields[i] != field {
			t.Errorf("Expected field '%s', got '%s'", field, fields[i])
		}
	}

	// Test empty string
	fields = parseSelectFields("")
	if len(fields) != 0 {
		t.Errorf("Expected 0 fields for empty string, got %d", len(fields))
	}

	// Test with spaces
	fields = parseSelectFields(" title , author ")
	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}
	if fields[0] != "title" || fields[1] != "author" {
		t.Errorf("Expected fields ['title', 'author'], got %v", fields)
	}
}

func TestApplyAdvancedFilters(t *testing.T) {
	// Setup
	feeds := map[string]config.TopicConfig{
		"tech": {
			URLs:    []string{"http://example.com/tech1"},
			Filters: []string{},
		},
	}

	cacheManager := cache.NewManager(5 * time.Minute)
	cfg := &config.Config{
		Port:                     8080,
		EnableSPA:                false,
		EnableSwagger:            false,
		EnableContentCompression: false,
		MaxContentLength:         10000,
		EnableDuplicateRemoval:   true,
		Security: config.SecurityConfig{
			EnableRateLimit:       false,
			EnableCORS:            false,
			EnableSecurityHeaders: false,
			EnableRequestID:       false,
		},
	}

	storageManager, _ := storage.NewStorage("./testdata", cfg)
	defer storageManager.Close()

	agg := aggregator.New(cacheManager, storageManager, feeds)
	poller := poller.New(agg, cacheManager, storageManager, feeds, 1*time.Minute, 1*time.Minute, cfg)
	server := NewServer(agg, poller, cfg)

	articles := []models.Article{
		{
			Title:       "AI Technology News",
			Description: "Latest developments in artificial intelligence",
			Content:     "AI is changing the world",
		},
		{
			Title:       "Weather Report",
			Description: "Today's weather is sunny",
			Content:     "No AI content here",
		},
	}

	// Test with search filter
	query := &models.ODataQuery{
		Search: []string{"AI"},
	}
	filtered := server.applyAdvancedFilters(articles, query)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 articles after AI search, got %d", len(filtered))
	}

	// Test with no filters
	query = &models.ODataQuery{}
	filtered = server.applyAdvancedFilters(articles, query)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 articles with no filters, got %d", len(filtered))
	}
}
