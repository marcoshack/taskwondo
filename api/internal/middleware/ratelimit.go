package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns middleware that enforces per-IP rate limiting using a token
// bucket algorithm. rps is the sustained rate in requests per second; burst is
// the maximum number of requests allowed in a single burst.
func RateLimit(rps rate.Limit, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*ipLimiter)

	// Clean up stale entries every 5 minutes.
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > 10*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		if l, ok := limiters[ip]; ok {
			l.lastSeen = time.Now()
			return l.limiter
		}
		l := rate.NewLimiter(rps, burst)
		limiters[ip] = &ipLimiter{limiter: l, lastSeen: time.Now()}
		return l
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			limiter := getLimiter(ip)

			if !limiter.Allow() {
				retryAfter := int(math.Ceil(1 / float64(rps)))
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "too many requests, try again later",
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request, checking proxy headers
// first, then falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; take the first (client).
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
