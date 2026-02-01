package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
)

type RateLimiter struct {
	mu              sync.Mutex
	limit           int
	duration        time.Duration
	requestCounts   map[string]int
	requestExpiry   map[string]time.Time
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

func NewRateLimiter(duration time.Duration, limit int, cleanupInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:           limit,
		duration:        duration,
		requestCounts:   make(map[string]int),
		requestExpiry:   make(map[string]time.Time),
		lastCleanup:     time.Now(),
		cleanupInterval: cleanupInterval,
	}
}

func (rl *RateLimiter) AllowRequest(clientID string) bool {
	ts := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if ts.Sub(rl.lastCleanup) > rl.cleanupInterval {
		unSafeClearExpiredEntries(ts, rl)
		rl.lastCleanup = ts
	}

	expiry, exists := rl.requestExpiry[clientID]
	if !exists || ts.After(expiry) {
		rl.requestCounts[clientID] = 1
		rl.requestExpiry[clientID] = ts.Add(rl.duration)
		return true
	}

	if rl.requestCounts[clientID] < rl.limit {
		rl.requestCounts[clientID]++
		return true
	}

	return false
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		clientID := c.ClientIP()

		if !rl.AllowRequest(clientID) {
			_ = c.AbortWithError(429, authError.NewAuthError(429, "Too many requests"))
			return
		}

		c.Next()
	}
}

// unSafeClearExpiredEntries Not thread-safe; caller must hold rl.mu lock.
func unSafeClearExpiredEntries(ts time.Time, rl *RateLimiter) {
	for clientID, expiry := range rl.requestExpiry {
		if ts.After(expiry) {
			delete(rl.requestCounts, clientID)
			delete(rl.requestExpiry, clientID)
		}
	}
}
