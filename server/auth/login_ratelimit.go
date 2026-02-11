package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultLoginMaxAttempts is the number of failed login attempts before blocking.
	DefaultLoginMaxAttempts = 5
	// DefaultLoginWindow is the time window in which failures are counted; counter resets after.
	DefaultLoginWindow = 15 * time.Minute
)

// LoginAttemptLimiter limits failed login attempts per client IP to mitigate brute-force.
type LoginAttemptLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*attemptEntry
	max      int
	window   time.Duration
}

type attemptEntry struct {
	count int
	until time.Time
}

// NewLoginAttemptLimiter returns a limiter that blocks an IP after maxAttempts
// failed logins within the given window. Pass 0 for max or window to use defaults.
func NewLoginAttemptLimiter(maxAttempts int, window time.Duration) *LoginAttemptLimiter {
	if maxAttempts <= 0 {
		maxAttempts = DefaultLoginMaxAttempts
	}
	if window <= 0 {
		window = DefaultLoginWindow
	}
	return &LoginAttemptLimiter{
		attempts: make(map[string]*attemptEntry),
		max:      maxAttempts,
		window:   window,
	}
}

// ClientIP returns the client IP from the request (X-Forwarded-For or RemoteAddr).
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			xff = strings.TrimSpace(xff[:i])
		} else {
			xff = strings.TrimSpace(xff)
		}
		if xff != "" {
			return xff
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// IsBlocked returns true if the client IP has exceeded the failure limit and is still within the window.
func (l *LoginAttemptLimiter) IsBlocked(ip string) bool {
	if ip == "" {
		return false
	}
	l.mu.RLock()
	e, ok := l.attempts[ip]
	l.mu.RUnlock()
	if !ok || e == nil {
		return false
	}
	if time.Now().After(e.until) {
		return false
	}
	return e.count >= l.max
}

// RecordFailure records a failed login attempt for the IP.
func (l *LoginAttemptLimiter) RecordFailure(ip string) {
	if ip == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.attempts[ip]
	if !ok || e == nil || now.After(e.until) {
		l.attempts[ip] = &attemptEntry{count: 1, until: now.Add(l.window)}
		return
	}
	e.count++
	if e.count >= l.max {
		e.until = now.Add(l.window)
	}
}

// RecordSuccess clears any failure count for the IP (e.g. after successful login).
func (l *LoginAttemptLimiter) RecordSuccess(ip string) {
	if ip == "" {
		return
	}
	l.mu.Lock()
	delete(l.attempts, ip)
	l.mu.Unlock()
}
