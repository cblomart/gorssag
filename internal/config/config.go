package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// TopicConfig represents configuration for a single topic
type TopicConfig struct {
	URLs    []string
	Filters []string // Full-text terms to filter articles
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableRateLimit       bool
	RateLimitPerSecond    float64
	RateLimitBurst        int
	EnableCORS            bool
	AllowedOrigins        []string
	EnableSecurityHeaders bool
	MaxRequestSize        int64
	EnableRequestID       bool
}

type Config struct {
	Port          int
	CacheTTL      time.Duration
	DataDir       string
	Feeds         map[string]TopicConfig
	LogLevel      string
	PollInterval  time.Duration
	EnableSPA     bool
	EnableSwagger bool
	Security      SecurityConfig
}

func Load() *Config {
	port := getEnvAsInt("PORT", 8080)
	cacheTTL := getEnvAsDuration("CACHE_TTL", 15*time.Minute)
	dataDir := getEnv("DATA_DIR", "./data")
	logLevel := getEnv("LOG_LEVEL", "info")
	pollInterval := getEnvAsDuration("POLL_INTERVAL", 15*time.Minute)
	enableSPA := getEnvAsBool("ENABLE_SPA", true)
	enableSwagger := getEnvAsBool("ENABLE_SWAGGER", true)

	// Load security configuration
	security := loadSecurityConfig()

	// Load feeds from environment variables
	feeds := loadFeedsFromEnv()

	// If no feeds configured via env, use defaults
	if len(feeds) == 0 {
		feeds = getDefaultFeeds()
	}

	return &Config{
		Port:          port,
		CacheTTL:      cacheTTL,
		DataDir:       dataDir,
		Feeds:         feeds,
		LogLevel:      logLevel,
		PollInterval:  pollInterval,
		EnableSPA:     enableSPA,
		EnableSwagger: enableSwagger,
		Security:      security,
	}
}

func loadSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableRateLimit:       getEnvAsBool("ENABLE_RATE_LIMIT", true),
		RateLimitPerSecond:    getEnvAsFloat("RATE_LIMIT_PER_SECOND", 10.0),
		RateLimitBurst:        getEnvAsInt("RATE_LIMIT_BURST", 20),
		EnableCORS:            getEnvAsBool("ENABLE_CORS", true),
		AllowedOrigins:        getEnvAsStringSlice("ALLOWED_ORIGINS", []string{"*"}),
		EnableSecurityHeaders: getEnvAsBool("ENABLE_SECURITY_HEADERS", true),
		MaxRequestSize:        getEnvAsInt64("MAX_REQUEST_SIZE", 10<<20), // 10MB
		EnableRequestID:       getEnvAsBool("ENABLE_REQUEST_ID", true),
	}
}

func loadFeedsFromEnv() map[string]TopicConfig {
	feeds := make(map[string]TopicConfig)

	// Look for FEED_TOPIC_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "FEED_TOPIC_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}

			// Parse topic name from FEED_TOPIC_<TOPIC_NAME>
			topicName := strings.TrimPrefix(parts[0], "FEED_TOPIC_")
			topicName = strings.ToLower(topicName)

			// Parse URLs and filters from value
			value := parts[1]
			urls, filters := parseTopicValue(value)

			feeds[topicName] = TopicConfig{
				URLs:    urls,
				Filters: filters,
			}
		}
	}

	return feeds
}

func parseTopicValue(value string) ([]string, []string) {
	// Format: "url1,url2,url3|filter1,filter2,filter3"
	// If no filters specified, just URLs: "url1,url2,url3"

	parts := strings.Split(value, "|")
	urls := strings.Split(parts[0], ",")

	// Clean up URLs
	for i, url := range urls {
		urls[i] = strings.TrimSpace(url)
	}

	var filters []string
	if len(parts) > 1 {
		filters = strings.Split(parts[1], ",")
		// Clean up filters
		for i, filter := range filters {
			filters[i] = strings.TrimSpace(filter)
		}
	}

	return urls, filters
}

func getDefaultFeeds() map[string]TopicConfig {
	return map[string]TopicConfig{
		"tech": {
			URLs: []string{
				"https://feeds.feedburner.com/TechCrunch",
				"http://rss.cnn.com/rss/edition_technology.rss",
				"https://feeds.arstechnica.com/arstechnica/index",
			},
			Filters: []string{"AI", "artificial intelligence", "machine learning", "blockchain", "cryptocurrency"},
		},
		"news": {
			URLs: []string{
				"https://feeds.npr.org/1001/rss.xml",
				"https://feeds.feedburner.com/TheHackersNews",
			},
			Filters: []string{"technology", "innovation", "digital", "startup", "cybersecurity"},
		},
		"programming": {
			URLs: []string{
				"https://blog.golang.org/feed.atom",
				"https://feeds.feedburner.com/oreilly/go",
				"https://www.reddit.com/r/golang/.rss",
			},
			Filters: []string{"Go", "golang", "programming", "development", "software"},
		},
	}
}

func getEnv(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsStringSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		origins := strings.Split(val, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		return origins
	}
	return defaultVal
}
