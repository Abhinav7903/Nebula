package api

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Middleware struct {
	logger      *slog.Logger
	apiKeys     map[string]bool
	requireKey  bool
	rateLimiter *RateLimiter
}

type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*rate.Limiter
	burst    int
	rps      rate.Limit
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		burst:    burst,
		rps:      rate.Limit(rps),
	}
}

func (rl *RateLimiter) Get(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, ok := rl.visitors[ip]
	rl.mu.RUnlock()
	if ok {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	limiter = rate.NewLimiter(rl.rps, rl.burst)
	rl.visitors[ip] = limiter
	return limiter
}

func NewMiddleware(logger *slog.Logger, keys []string, requireKey bool, rpm, burst int) *Middleware {
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	return &Middleware{
		logger:      logger,
		apiKeys:     keyMap,
		requireKey:  requireKey,
		rateLimiter: NewRateLimiter(rpm, burst),
	}
}

func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				m.logger.Error("panic recovered",
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.requireKey {
			next.ServeHTTP(w, r)
			return
		}
		key := r.Header.Get("X-API-Key")
		if key == "" {
			writeProblem(w, http.StatusUnauthorized, "missing_api_key", "X-API-Key header required")
			return
		}
		if !m.apiKeys[key] {
			writeProblem(w, http.StatusForbidden, "invalid_api_key", "Invalid API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		limiter := m.rateLimiter.Get(ip)
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, http.StatusTooManyRequests, "rate_limited", "Rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		m.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.status = code
	l.ResponseWriter.WriteHeader(code)
}

func writeProblem(w http.ResponseWriter, status int, typ, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":   typ,
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
