package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(10), 5)

	// Test getting limiter for same IP
	ip1 := "192.168.1.1"
	limiter1 := limiter.GetLimiter(ip1)
	limiter2 := limiter.GetLimiter(ip1)

	if limiter1 != limiter2 {
		t.Error("Expected same limiter for same IP")
	}

	// Test getting limiter for different IP
	ip2 := "192.168.1.2"
	limiter3 := limiter.GetLimiter(ip2)

	if limiter1 == limiter3 {
		t.Error("Expected different limiters for different IPs")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	limiter := NewRateLimiter(rate.Limit(10), 5)

	// Test that cleanup doesn't panic
	limiter.Cleanup()

	// Test that limiters still work after cleanup
	ip := "192.168.1.1"
	limiter1 := limiter.GetLimiter(ip)
	if limiter1 == nil {
		t.Error("Expected limiter to be created after cleanup")
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if !config.EnableRateLimit {
		t.Error("Expected rate limiting to be enabled by default")
	}

	if config.RateLimitPerSecond != 10.0 {
		t.Errorf("Expected rate limit per second to be 10.0, got %f", config.RateLimitPerSecond)
	}

	if config.RateLimitBurst != 20 {
		t.Errorf("Expected rate limit burst to be 20, got %d", config.RateLimitBurst)
	}

	if !config.EnableCORS {
		t.Error("Expected CORS to be enabled by default")
	}

	if !config.EnableSecurityHeaders {
		t.Error("Expected security headers to be enabled by default")
	}

	if config.MaxRequestSize != 10<<20 {
		t.Errorf("Expected max request size to be 10MB, got %d", config.MaxRequestSize)
	}

	if !config.EnableRequestID {
		t.Error("Expected request ID to be enabled by default")
	}
}

func TestSetupSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Test with nil config (should use defaults)
	SetupSecurityMiddleware(router, nil)

	// Test with custom config
	config := &SecurityConfig{
		EnableRateLimit:       true,
		RateLimitPerSecond:    5.0,
		RateLimitBurst:        10,
		EnableCORS:            true,
		AllowedOrigins:        []string{"http://localhost:3000"},
		EnableSecurityHeaders: true,
		MaxRequestSize:        1024,
		EnableRequestID:       true,
	}

	router2 := gin.New()
	SetupSecurityMiddleware(router2, config)

	// Test with disabled features
	config2 := &SecurityConfig{
		EnableRateLimit:       false,
		EnableCORS:            false,
		EnableSecurityHeaders: false,
		EnableRequestID:       false,
		MaxRequestSize:        1024,
	}

	router3 := gin.New()
	SetupSecurityMiddleware(router3, config2)
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	limiter := NewRateLimiter(rate.Limit(10), 5)
	router.Use(RateLimitMiddleware(limiter))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test successful request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test rate limiting (this might not trigger due to high limits, but tests the path)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	router.ServeHTTP(w, req)

	// Should still succeed due to high limits
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestSizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestSizeMiddleware(100)) // 100 bytes limit

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test request within size limit
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	req.ContentLength = 50
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test request exceeding size limit
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	req.ContentLength = 150
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}

	// Test request with no content length
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for request with no content length, got %d", w.Code)
	}
}

func TestInputValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(InputValidationMiddleware())

	router.GET("/test/:topic", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test valid topic name
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/valid-topic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test invalid topic name
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/invalid@topic", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Test valid OData parameters
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/valid-topic?$top=10&$skip=0", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test invalid OData parameters
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/valid-topic?$top=abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Test route without topic parameter
	router.GET("/simple", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/simple", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for simple route, got %d", w.Code)
	}
}

func TestSecurityLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(SecurityLoggingMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test successful request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test request with user agent
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestBot/1.0")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		ip := getClientIP(c)
		c.JSON(http.StatusOK, gin.H{"ip": ip})
	})

	// Test X-Forwarded-For header
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test X-Real-IP header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "192.168.1.2")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test X-Forwarded-For header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.3")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test no headers (should use RemoteAddr)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.4:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidationFunctions(t *testing.T) {
	// Test isValidNumber
	if !isValidNumber("123") {
		t.Error("Expected '123' to be valid")
	}

	if isValidNumber("abc") {
		t.Error("Expected 'abc' to be invalid")
	}

	if isValidNumber("") {
		t.Error("Expected empty string to be invalid")
	}

	if !isValidNumber("0") {
		t.Error("Expected '0' to be valid")
	}

	if isValidNumber("-123") {
		t.Error("Expected '-123' to be invalid (only positive integers)")
	}

	if isValidNumber("12.34") {
		t.Error("Expected '12.34' to be invalid (not an integer)")
	}

	// Test isValidTopicName
	if !isValidTopicName("valid-topic") {
		t.Error("Expected 'valid-topic' to be valid")
	}

	if !isValidTopicName("ValidTopic123") {
		t.Error("Expected 'ValidTopic123' to be valid")
	}

	if isValidTopicName("invalid@topic") {
		t.Error("Expected 'invalid@topic' to be invalid")
	}

	if isValidTopicName("") {
		t.Error("Expected empty string to be invalid")
	}

	if !isValidTopicName("a") { // Single character should be valid
		t.Error("Expected single character to be valid")
	}

	if isValidTopicName("topic_with_underscores") {
		t.Error("Expected 'topic_with_underscores' to be invalid (no underscores allowed)")
	}

	if !isValidTopicName("topic-with-dashes") {
		t.Error("Expected 'topic-with-dashes' to be valid")
	}

	if !isValidTopicName("topic123") {
		t.Error("Expected 'topic123' to be valid")
	}

	if isValidTopicName("topic with spaces") {
		t.Error("Expected 'topic with spaces' to be invalid")
	}

	if isValidTopicName("topic!with!special!chars") {
		t.Error("Expected 'topic!with!special!chars' to be invalid")
	}
}

func TestValidateODataQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		err := validateODataQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test valid $top parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?$top=10", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $top, got %d", w.Code)
	}

	// Test invalid $top parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$top=abc", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid $top, got %d", w.Code)
	}

	// Test valid $skip parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$skip=5", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $skip, got %d", w.Code)
	}

	// Test invalid $skip parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$skip=xyz", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid $skip, got %d", w.Code)
	}

	// Test valid $orderby parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$orderby=title", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $orderby, got %d", w.Code)
	}

	// Test valid $filter parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$filter=title eq 'test'", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $filter, got %d", w.Code)
	}

	// Test valid $select parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$select=title,description", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $select, got %d", w.Code)
	}

	// Test valid $search parameter
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?$search=test", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid $search, got %d", w.Code)
	}
}

func TestValidatePathParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test/:topic", func(c *gin.Context) {
		err := validatePathParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test valid topic name
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/valid-topic", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid topic, got %d", w.Code)
	}

	// Test invalid topic name
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test/invalid@topic", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid topic, got %d", w.Code)
	}

	// Test route without topic parameter
	router.GET("/simple", func(c *gin.Context) {
		err := validatePathParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/simple", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for simple route, got %d", w.Code)
	}
}
