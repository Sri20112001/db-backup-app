package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter is a per-key sliding-window limiter (brute-force protection
// for login/register). Zero value is usable; callers pass limits explicitly.
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	attempts map[string][]time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{limit: limit, window: window, attempts: make(map[string][]time.Time)}
	go rl.reap()
	return rl
}

// reap drops idle keys every window so the map stays bounded.
func (rl *RateLimiter) reap() {
	t := time.NewTicker(rl.window)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-rl.window)
		rl.mu.Lock()
		for k, times := range rl.attempts {
			kept := times[:0]
			for _, ts := range times {
				if ts.After(cutoff) {
					kept = append(kept, ts)
				}
			}
			if len(kept) == 0 {
				delete(rl.attempts, k)
			} else {
				rl.attempts[k] = kept
			}
		}
		rl.mu.Unlock()
	}
}

// Allow records an attempt for key and reports whether it is within budget.
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	// In-place filter: write index never passes the read index.
	kept := rl.attempts[key][:0]
	for _, ts := range rl.attempts[key] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= rl.limit {
		rl.attempts[key] = kept
		return false
	}
	rl.attempts[key] = append(kept, now)
	return true
}

// LimitAuth aborts with 429 when the client IP exceeds the budget.
func (rl *RateLimiter) LimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try again later"})
			return
		}
		c.Next()
	}
}
