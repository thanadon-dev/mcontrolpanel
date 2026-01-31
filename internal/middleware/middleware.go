package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"mcontrolpanel/internal/database"
)

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
