package security

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limit information per IP
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit
	b        int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

// GetLimiter returns the rate limiter for the given key (IP address)
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.limiters[key] = limiter
	}

	return limiter
}

// Cleanup removes old limiters to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// In a production environment, you might want to implement
	// a more sophisticated cleanup strategy based on time
	// For now, we'll keep all limiters as they're lightweight
}

// SecurityConfig holds security configuration
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

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EnableRateLimit:       true,
		RateLimitPerSecond:    10.0, // 10 requests per second
		RateLimitBurst:        20,   // Allow bursts up to 20 requests
		EnableCORS:            true,
		AllowedOrigins:        []string{"*"}, // Allow all origins by default
		EnableSecurityHeaders: true,
		MaxRequestSize:        10 << 20, // 10MB
		EnableRequestID:       true,
	}
}

// SetupSecurityMiddleware configures all security middleware
func SetupSecurityMiddleware(router *gin.Engine, config *SecurityConfig) {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	// Add request ID middleware
	if config.EnableRequestID {
		router.Use(requestid.New())
	}

	// Add security headers middleware
	if config.EnableSecurityHeaders {
		router.Use(secure.New(secure.Config{
			SSLRedirect:           false, // Set to true in production with HTTPS
			STSSeconds:            31536000,
			STSIncludeSubdomains:  true,
			FrameDeny:             true,
			ContentTypeNosniff:    true,
			BrowserXssFilter:      true,
			ContentSecurityPolicy: "default-src 'self'",
			ReferrerPolicy:        "strict-origin-when-cross-origin",
		}))
	}

	// Add CORS middleware
	if config.EnableCORS {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = config.AllowedOrigins
		corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
		corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}
		corsConfig.ExposeHeaders = []string{"X-Request-ID"}
		router.Use(cors.New(corsConfig))
	}

	// Add rate limiting middleware
	if config.EnableRateLimit {
		limiter := NewRateLimiter(rate.Limit(config.RateLimitPerSecond), config.RateLimitBurst)
		router.Use(RateLimitMiddleware(limiter))
	}

	// Add request size limiting middleware
	router.Use(RequestSizeMiddleware(config.MaxRequestSize))

	// Add input validation middleware
	router.Use(InputValidationMiddleware())

	// Add logging middleware
	router.Use(SecurityLoggingMiddleware())
}

// RateLimitMiddleware implements rate limiting per IP
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)
		limiter := limiter.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestSizeMiddleware limits request body size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request too large",
				"message": "Request body exceeds maximum allowed size",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// InputValidationMiddleware validates and sanitizes input
func InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate OData query parameters
		if err := validateODataQuery(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate path parameters
		if err := validatePathParams(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid path parameters",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecurityLoggingMiddleware logs security-relevant information
func SecurityLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Log security-relevant information
		securityInfo := []string{
			"ip=" + param.ClientIP,
			"method=" + param.Method,
			"path=" + param.Path,
			"status=" + fmt.Sprintf("%d", param.StatusCode),
			"latency=" + param.Latency.String(),
			"user_agent=" + param.Request.UserAgent(),
		}

		if param.StatusCode >= 400 {
			// Log errors with more detail
			securityInfo = append(securityInfo, "error=true")
		}

		return strings.Join(securityInfo, " ") + "\n"
	})
}

// validateODataQuery validates OData query parameters
func validateODataQuery(c *gin.Context) error {
	// Validate $top parameter
	if top := c.Query("$top"); top != "" {
		if !isValidNumber(top) {
			return fmt.Errorf("invalid $top parameter: must be a positive integer")
		}
	}

	// Validate $skip parameter
	if skip := c.Query("$skip"); skip != "" {
		if !isValidNumber(skip) {
			return fmt.Errorf("invalid $skip parameter: must be a non-negative integer")
		}
	}

	// Validate $filter parameter length
	if filter := c.Query("$filter"); filter != "" {
		if len(filter) > 1000 {
			return fmt.Errorf("$filter parameter too long: maximum 1000 characters")
		}
	}

	// Validate $search parameter length
	if search := c.Query("$search"); search != "" {
		if len(search) > 500 {
			return fmt.Errorf("$search parameter too long: maximum 500 characters")
		}
	}

	// Validate $select parameter
	if selectParam := c.Query("$select"); selectParam != "" {
		if len(selectParam) > 200 {
			return fmt.Errorf("$select parameter too long: maximum 200 characters")
		}
	}

	return nil
}

// validatePathParams validates path parameters
func validatePathParams(c *gin.Context) error {
	// Validate topic parameter
	if topic := c.Param("topic"); topic != "" {
		if !isValidTopicName(topic) {
			return fmt.Errorf("invalid topic name: must contain only alphanumeric characters and hyphens")
		}
	}

	return nil
}

// getClientIP extracts the real client IP address
func getClientIP(c *gin.Context) string {
	// Check for forwarded headers (when behind proxy/load balancer)
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if commaIndex := strings.Index(ip, ","); commaIndex != -1 {
			return strings.TrimSpace(ip[:commaIndex])
		}
		return strings.TrimSpace(ip)
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	if ip := c.GetHeader("X-Client-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Fallback to remote address
	return c.ClientIP()
}

// isValidNumber checks if a string is a valid positive integer
func isValidNumber(s string) bool {
	if s == "" {
		return false
	}

	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// isValidTopicName checks if a topic name is valid
func isValidTopicName(s string) bool {
	if s == "" || len(s) > 50 {
		return false
	}

	for _, char := range s {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return false
		}
	}

	return true
}
