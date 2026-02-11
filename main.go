package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/internal/telemetry"
	"github.com/JakeNeyer/ipam/server"
	"github.com/JakeNeyer/ipam/server/middleware"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/crypto/bcrypt"
)

func otelEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_OTEL")))
	return v == "true" || v == "1"
}

// ensureInitialAdmin creates the first admin user when INITIAL_ADMIN_EMAIL and
// INITIAL_ADMIN_PASSWORD are set and no users exist (e.g. for Fly.io deploy without setup UI).
func ensureInitialAdmin(st store.Storer) {
	email := strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL"))
	password := os.Getenv("INITIAL_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return
	}
	users, err := st.ListUsers()
	if err != nil || len(users) > 0 {
		return
	}
	if !validation.ValidateEmail(email) || !validation.ValidatePassword(password) {
		logger.Info("initial admin skipped: invalid email or password (email format, password 8–72 chars)")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("initial admin password hash failed", logger.ErrAttr(err))
		return
	}
	admin := &store.User{
		Email:        strings.TrimSpace(strings.ToLower(email)),
		PasswordHash: string(hash),
		Role:         store.RoleAdmin,
	}
	if err := st.CreateUser(admin); err != nil {
		logger.Error("initial admin create failed", logger.ErrAttr(err))
		return
	}
	logger.Info("initial admin created", slog.String("email", admin.Email))
}

func main() {
	ctx := context.Background()

	// OpenTelemetry: only when ENABLE_OTEL=true
	if otelEnabled() {
		shutdown, err := telemetry.Init(ctx)
		if err != nil {
			logger.Error("telemetry init failed", logger.ErrAttr(err))
			os.Exit(1)
		}
		defer telemetry.Shutdown(ctx, shutdown)
		logger.Info("opentelemetry enabled (stdout traces)")
	}

	var st store.Storer
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		var err error
		st, err = store.NewPostgresStore(ctx, dsn)
		if err != nil {
			logger.Error("postgres store failed", logger.ErrAttr(err))
			os.Exit(1)
		}
		if c, ok := st.(*store.PostgresStore); ok {
			defer c.Close()
		}
		logger.Info("store", slog.String("type", "postgres"))
	} else {
		st = store.NewStore()
		logger.Info("store", slog.String("type", "in_memory"))
	}
	ensureInitialAdmin(st)
	s := server.NewServer(st)

	// Security: headers first, then body limit, then request logging, then panic recovery
	handler := middleware.SecurityHeaders(s)
	handler = middleware.MaxBytes(handler)
	if otelEnabled() {
		handler = middleware.OtelRequestResponseLog(handler)
		handler = otelhttp.NewHandler(handler, "ipam")
	} else {
		handler = middleware.RequestLog(handler)
	}
	handler = middleware.Recover(handler)

	staticDir := resolveStaticDir()
	if staticDir != "" {
		handler = staticHandler(staticDir, handler)
		logger.Info("serving static files", slog.String("dir", staticDir))
	}

	addr := "0.0.0.0"
	if port := os.Getenv("PORT"); port != "" {
		addr = addr + ":" + port
	} else {
		addr = "localhost:8011"
	}
	logger.Info("server listening", slog.String("addr", "http://"+addr), slog.String("docs", "http://"+addr+"/docs"))
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Error("server failed", logger.ErrAttr(err))
		os.Exit(1)
	}
}

// resolveStaticDir returns a directory containing index.html for the SPA, or "" if none found.
// Tries STATIC_DIR, then web/dist relative to CWD or executable, so signup links (GET /) work.
func resolveStaticDir() string {
	if d := os.Getenv("STATIC_DIR"); d != "" {
		if abs, err := filepath.Abs(d); err == nil && staticDirHasIndex(abs) {
			return abs
		}
		return d
	}
	cwd, _ := os.Getwd()
	for _, rel := range []string{"web/dist", "dist", "./web/dist", "./dist"} {
		dir := filepath.Join(cwd, rel)
		if dir = filepath.Clean(dir); staticDirHasIndex(dir) {
			return dir
		}
	}
	// Relative to current binary (e.g. when run from project root)
	for _, rel := range []string{"web/dist", "dist"} {
		if staticDirHasIndex(rel) {
			if abs, err := filepath.Abs(rel); err == nil {
				return abs
			}
			return rel
		}
	}
	if execPath, err := os.Executable(); err == nil {
		base := filepath.Dir(execPath)
		for _, rel := range []string{"web/dist", "dist"} {
			dir := filepath.Join(base, rel)
			if staticDirHasIndex(dir) {
				return dir
			}
		}
	}
	return ""
}

func staticDirHasIndex(dir string) bool {
	f, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && f != nil && !f.IsDir()
}

// staticHandler serves API/docs from next, everything else from dir (SPA fallback to index.html).
func staticHandler(dir string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/docs") {
			next.ServeHTTP(w, r)
			return
		}
		p := filepath.Join(dir, filepath.Clean(strings.TrimPrefix(r.URL.Path, "/")))
		if rel, err := filepath.Rel(dir, p); err != nil || strings.HasPrefix(rel, "..") {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		if f, err := os.Stat(p); err == nil && !f.IsDir() {
			http.ServeFile(w, r, p)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
