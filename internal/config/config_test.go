package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Test default configuration
	cfg := Load()

	// Check default values
	if cfg.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Port)
	}

	if cfg.CacheTTL != 15*time.Minute {
		t.Errorf("Expected default cache TTL 15m, got %v", cfg.CacheTTL)
	}

	if cfg.PollInterval != 15*time.Minute {
		t.Errorf("Expected default poll interval 15m, got %v", cfg.PollInterval)
	}

	if cfg.ArticleRetention != 30*24*time.Hour {
		t.Errorf("Expected default article retention 30 days, got %v", cfg.ArticleRetention)
	}

	if !cfg.EnableSPA {
		t.Error("Expected default EnableSPA to be true")
	}

	if !cfg.EnableSwagger {
		t.Error("Expected default EnableSwagger to be true")
	}

	if !cfg.EnableContentCompression {
		t.Error("Expected default EnableContentCompression to be true")
	}

	if cfg.MaxContentLength != 50000 {
		t.Errorf("Expected default MaxContentLength 50000, got %d", cfg.MaxContentLength)
	}

	if !cfg.EnableDuplicateRemoval {
		t.Error("Expected default EnableDuplicateRemoval to be true")
	}

	if cfg.DatabaseOptimizeInterval != 24*time.Hour {
		t.Errorf("Expected default DatabaseOptimizeInterval 24h, got %v", cfg.DatabaseOptimizeInterval)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("CACHE_TTL", "30m")
	os.Setenv("POLL_INTERVAL", "30m")
	os.Setenv("ARTICLE_RETENTION", "48h")
	os.Setenv("ENABLE_SPA", "false")
	os.Setenv("ENABLE_SWAGGER", "false")
	os.Setenv("ENABLE_CONTENT_COMPRESSION", "false")
	os.Setenv("MAX_CONTENT_LENGTH", "5000")
	os.Setenv("ENABLE_DUPLICATE_REMOVAL", "false")
	os.Setenv("DATABASE_OPTIMIZE_INTERVAL", "1800")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("CACHE_TTL")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("ARTICLE_RETENTION")
		os.Unsetenv("ENABLE_SPA")
		os.Unsetenv("ENABLE_SWAGGER")
		os.Unsetenv("ENABLE_CONTENT_COMPRESSION")
		os.Unsetenv("MAX_CONTENT_LENGTH")
		os.Unsetenv("ENABLE_DUPLICATE_REMOVAL")
		os.Unsetenv("DATABASE_OPTIMIZE_INTERVAL")
	}()

	cfg := Load()

	// Check that environment variables are respected
	if cfg.Port != 9090 {
		t.Errorf("Expected port 9090 from env, got %d", cfg.Port)
	}

	if cfg.CacheTTL != 30*time.Minute {
		t.Errorf("Expected cache TTL 30m from env, got %v", cfg.CacheTTL)
	}

	if cfg.PollInterval != 30*time.Minute {
		t.Errorf("Expected poll interval 30m from env, got %v", cfg.PollInterval)
	}

	if cfg.ArticleRetention != 48*time.Hour {
		t.Errorf("Expected article retention 48h from env, got %v", cfg.ArticleRetention)
	}

	if cfg.EnableSPA {
		t.Error("Expected EnableSPA false from env")
	}

	if cfg.EnableSwagger {
		t.Error("Expected EnableSwagger false from env")
	}

	if cfg.EnableContentCompression {
		t.Error("Expected EnableContentCompression false from env")
	}

	if cfg.MaxContentLength != 5000 {
		t.Errorf("Expected MaxContentLength 5000 from env, got %d", cfg.MaxContentLength)
	}

	if cfg.EnableDuplicateRemoval {
		t.Error("Expected EnableDuplicateRemoval false from env")
	}

	if cfg.DatabaseOptimizeInterval != 24*time.Hour {
		t.Errorf("Expected DatabaseOptimizeInterval 24h from env, got %v", cfg.DatabaseOptimizeInterval)
	}
}

func TestLoadConfig_FeedTopics(t *testing.T) {
	// Set feed topic environment variables
	os.Setenv("FEED_TOPIC_TECH", "https://example.com/tech1,https://example.com/tech2|AI,blockchain")
	os.Setenv("FEED_TOPIC_NEWS", "https://example.com/news1|technology,innovation")

	defer func() {
		os.Unsetenv("FEED_TOPIC_TECH")
		os.Unsetenv("FEED_TOPIC_NEWS")
	}()

	cfg := Load()

	// Check that feed topics are loaded
	if len(cfg.Feeds) != 2 {
		t.Errorf("Expected 2 feed topics, got %d", len(cfg.Feeds))
	}

	// Check tech topic
	techConfig, exists := cfg.Feeds["tech"]
	if !exists {
		t.Error("Expected tech topic to exist")
	}

	if len(techConfig.URLs) != 2 {
		t.Errorf("Expected 2 URLs for tech topic, got %d", len(techConfig.URLs))
	}

	if len(techConfig.Filters) != 2 {
		t.Errorf("Expected 2 filters for tech topic, got %d", len(techConfig.Filters))
	}

	// Check news topic
	newsConfig, exists := cfg.Feeds["news"]
	if !exists {
		t.Error("Expected news topic to exist")
	}

	if len(newsConfig.URLs) != 1 {
		t.Errorf("Expected 1 URL for news topic, got %d", len(newsConfig.URLs))
	}

	if len(newsConfig.Filters) != 2 {
		t.Errorf("Expected 2 filters for news topic, got %d", len(newsConfig.Filters))
	}
}

func TestParseFeedTopicConfig(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected TopicConfig
	}{
		{
			name:  "single URL with filters",
			value: "https://example.com/feed|AI,blockchain",
			expected: TopicConfig{
				URLs:    []string{"https://example.com/feed"},
				Filters: []string{"AI", "blockchain"},
			},
		},
		{
			name:  "multiple URLs with filters",
			value: "https://example.com/feed1,https://example.com/feed2|tech,innovation",
			expected: TopicConfig{
				URLs:    []string{"https://example.com/feed1", "https://example.com/feed2"},
				Filters: []string{"tech", "innovation"},
			},
		},
		{
			name:  "single URL without filters",
			value: "https://example.com/feed",
			expected: TopicConfig{
				URLs:    []string{"https://example.com/feed"},
				Filters: []string{},
			},
		},
		{
			name:  "multiple URLs without filters",
			value: "https://example.com/feed1,https://example.com/feed2",
			expected: TopicConfig{
				URLs:    []string{"https://example.com/feed1", "https://example.com/feed2"},
				Filters: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, filters := parseTopicValue(tt.value)

			if len(urls) != len(tt.expected.URLs) {
				t.Errorf("Expected %d URLs, got %d", len(tt.expected.URLs), len(urls))
			}

			if len(filters) != len(tt.expected.Filters) {
				t.Errorf("Expected %d filters, got %d", len(tt.expected.Filters), len(filters))
			}

			// Check URLs
			for i, url := range tt.expected.URLs {
				if urls[i] != url {
					t.Errorf("Expected URL %s, got %s", url, urls[i])
				}
			}

			// Check filters
			for i, filter := range tt.expected.Filters {
				if filters[i] != filter {
					t.Errorf("Expected filter %s, got %s", filter, filters[i])
				}
			}
		})
	}
}
