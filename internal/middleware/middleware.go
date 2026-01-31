package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"mcontrolpanel/internal/database"
)

// Rate limiter storage
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientInfo
	limit    int
	window   time.Duration
}

type clientInfo struct {
	count     int
	firstSeen time.Time
}

var (
	rateLimiter     *RateLimiter
	loginLimiter    *RateLimiter
	rateLimiterOnce sync.Once
)

// Initialize rate limiters
func initRateLimiters() {
	rateLimiter = &RateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    60, // 60 requests per minute
		window:   time.Minute,
	}
	loginLimiter = &RateLimiter{
		requests: make(map[string]*clientInfo),
		limit:    5, // 5 login attempts per minute
		window:   time.Minute,
	}

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			rateLimiter.cleanup()
			loginLimiter.cleanup()
		}
	}()
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, info := range rl.requests {
		if now.Sub(info.firstSeen) > rl.window {
			delete(rl.requests, ip)
		}
	}
}

func (rl *RateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	info, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = &clientInfo{count: 1, firstSeen: now}
		return true
	}

	// Reset if window has passed
	if now.Sub(info.firstSeen) > rl.window {
		rl.requests[ip] = &clientInfo{count: 1, firstSeen: now}
		return true
	}

	// Check limit
	if info.count >= rl.limit {
		return false
	}

	info.count++
	return true
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.Printf("%s %s %d %v", c.Request.Method, path, status, latency)
	}
}

// RateLimit middleware - ป้องกัน brute force
func RateLimit() gin.HandlerFunc {
	rateLimiterOnce.Do(initRateLimiters)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rateLimiter.isAllowed(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "คำขอมากเกินไป กรุณารอสักครู่",
				"retry_after": 60,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRateLimit - จำกัดการ login
func LoginRateLimit() gin.HandlerFunc {
	rateLimiterOnce.Do(initRateLimiters)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !loginLimiter.isAllowed(ip) {
			c.HTML(http.StatusTooManyRequests, "login.html", gin.H{
				"error": "พยายาม login มากเกินไป กรุณารอ 1 นาที",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func Auth(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check session cookie
		cookie, err := c.Cookie("session")
		if err != nil || cookie == "" {
			// Check if it's an API request
			if c.Request.URL.Path[:4] == "/api" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Validate session (simple approach: cookie contains user ID)
		// In production, use proper session management
		var userID int64
		if _, err := parseSessionCookie(cookie, &userID); err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := db.GetUser(userID)
		if err != nil {
			c.SetCookie("session", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

func parseSessionCookie(cookie string, userID *int64) (bool, error) {
	// Simple session format: "user_<id>"
	// In production, use signed/encrypted sessions
	var id int64
	_, err := parseFromCookie(cookie, &id)
	if err != nil {
		return false, err
	}
	*userID = id
	return true, nil
}

func parseFromCookie(cookie string, id *int64) (bool, error) {
	// Parse "user_123" format
	if len(cookie) < 6 || cookie[:5] != "user_" {
		return false, nil
	}
	
	var parsed int64
	for _, c := range cookie[5:] {
		if c < '0' || c > '9' {
			return false, nil
		}
		parsed = parsed*10 + int64(c-'0')
	}
	*id = parsed
	return true, nil
}
