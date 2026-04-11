package http_api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipRateLimiter struct {
	enabled bool
	limit   rate.Limit
	burst   int
	mu      sync.Mutex
	entries map[string]*ipRateEntry
}

type ipRateEntry struct {
	lim  *rate.Limiter
	seen time.Time
}

func newIPRateLimiter(enabled bool, rps float64, burst int) *ipRateLimiter {
	if !enabled || rps <= 0 {
		return &ipRateLimiter{enabled: false}
	}
	if burst <= 0 {
		burst = 1
	}
	return &ipRateLimiter{
		enabled: true,
		limit:   rate.Limit(rps),
		burst:   burst,
		entries: make(map[string]*ipRateEntry),
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	if !l.enabled {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok || time.Since(e.seen) > 20*time.Minute {
		e = &ipRateEntry{lim: rate.NewLimiter(l.limit, l.burst)}
		l.entries[ip] = e
	}
	if len(l.entries) > 8000 {
		for k := range l.entries {
			delete(l.entries, k)
			break
		}
	}
	e.seen = time.Now()
	return e.lim.Allow()
}

func rateLimitMiddleware(lim *ipRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !lim.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"detail": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
