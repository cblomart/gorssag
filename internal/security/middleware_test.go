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
}
