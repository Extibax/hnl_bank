package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

type bucket struct {
	count int
	start time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

// RateLimit limits requests per client IP within a fixed window.
// It is an in-memory limiter suitable for a single instance.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{buckets: map[string]*bucket{}, limit: limit, window: window}
	return rl.middleware
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientKey(r)
		now := time.Now()
		rl.mu.Lock()
		b, ok := rl.buckets[key]
		if !ok || now.Sub(b.start) > rl.window {
			b = &bucket{count: 1, start: now}
			rl.buckets[key] = b
		} else {
			b.count++
		}
		allowed := b.count <= rl.limit
		rl.mu.Unlock()

		if !allowed {
			w.Header().Set("Retry-After", "1")
			response.Error(w, http.StatusTooManyRequests, "too many requests", "rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}
// clientKey returns a stable per-client key (the IP without port).
func clientKey(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
