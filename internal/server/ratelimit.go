package server

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type clientBucket struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter implements a concurrent per-IP token-bucket rate limiter with a background
// cleanup goroutine coordinated via channels.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rate    float64
	burst   float64
	stopCh  chan struct{}
	once    sync.Once
}

// NewRateLimiter creates a RateLimiter allowing `ratePerSec` requests/sec with `burst` capacity
// and launches a background goroutine to evict inactive IP records.
func NewRateLimiter(ratePerSec, burst float64) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientBucket),
		rate:    ratePerSec,
		burst:   burst,
		stopCh:  make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-3 * time.Minute)
			for ip, b := range rl.clients {
				if b.lastSeen.Before(cutoff) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() {
	rl.once.Do(func() {
		close(rl.stopCh)
	})
}

// Allow checks whether the given client IP has an available token in its bucket.
func (rl *RateLimiter) Allow(ip string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientBucket{
			tokens:   rl.burst - 1,
			lastSeen: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.lastSeen = now
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware wraps an http.Handler with per-IP rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || ip == "" {
			ip = r.RemoteAddr
		}
		if !rl.Allow(ip) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "429 Too Many Requests — rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware sets defensive HTTP response headers on every response.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
