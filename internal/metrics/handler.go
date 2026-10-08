package metrics

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

// Path is the HTTP path Prometheus scrapes.
const Path = "/metrics"

const (
	// cacheTTL is how long one store walk is reused. Several scrapers (HA
	// Prometheus pairs, an agent plus a sidecar) hitting the endpoint inside
	// this window share a single collection instead of each walking the store.
	cacheTTL = 5 * time.Second
	// gatherTimeout bounds one scrape, including time spent waiting for an
	// in-flight collection. promhttp answers 503 when it is exceeded.
	gatherTimeout = 15 * time.Second
)

// Handler serves Prometheus text metrics from the current IPAM store.
// When token is non-empty, requests must send Authorization: Bearer <token>.
func Handler(s store.Storer, token string) http.Handler {
	return newHandler(s, token, cacheTTL, time.Now)
}

func newHandler(s store.Storer, token string, ttl time.Duration, now func() time.Time) http.Handler {
	reg := prometheus.NewRegistry()
	reg.MustRegister(NewCollector(s))
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	h := promhttp.HandlerFor(&cachedGatherer{g: reg, ttl: ttl, now: now}, promhttp.HandlerOpts{
		// A bad row (for example two resources that still collide on every
		// label) must not blank the whole scrape: serve what gathered cleanly
		// and log the rest.
		ErrorHandling: promhttp.ContinueOnError,
		ErrorLog:      gatherLogger{},
		Timeout:       gatherTimeout,
	})
	token = strings.TrimSpace(token)
	if token == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !bearerOK(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Wrap serves metricsHandler at Path and passes every other request to next.
func Wrap(next, metricsHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == Path {
			metricsHandler.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cachedGatherer serializes collection and reuses the last result for ttl.
// Concurrent scrapes block on mu and then read the fresh snapshot rather than
// each walking the store.
type cachedGatherer struct {
	g   prometheus.Gatherer
	ttl time.Duration
	now func() time.Time

	mu  sync.Mutex
	at  time.Time
	mfs []*dto.MetricFamily
	err error
}

func (c *cachedGatherer) Gather() ([]*dto.MetricFamily, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mfs != nil && c.ttl > 0 && c.now().Sub(c.at) < c.ttl {
		return c.mfs, c.err
	}
	mfs, err := c.g.Gather()
	c.mfs, c.err, c.at = mfs, err, c.now()
	return mfs, err
}

// gatherLogger adapts promhttp's Println-style error log to the app logger.
type gatherLogger struct{}

func (gatherLogger) Println(v ...interface{}) {
	logger.Warn("metrics gather error", slog.String("err", strings.TrimSpace(fmt.Sprintln(v...))))
}

func bearerOK(r *http.Request, token string) bool {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if len(got) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
