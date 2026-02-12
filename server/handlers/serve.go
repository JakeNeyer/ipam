package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Unauthorized returns 401 Unauthorized with a simple HTML body for non-API, non-docs requests (used when APP_ORIGIN is set so the app is served from another origin).
func Unauthorized(appOrigin string, next http.Handler) http.Handler {
	origin := strings.TrimSuffix(appOrigin, "/")
	body := "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>Unauthorized</title></head><body><h1>Unauthorized</h1><p>This is the API server. Use the app at <a href=\"" + origin + "\">" + origin + "</a>.</p></body></html>"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/docs") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(body))
	})
}

// Static serves API/docs from next, everything else from dir (SPA fallback to index.html).
func Static(dir string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || strings.HasPrefix(r.URL.Path, "/docs") {
			next.ServeHTTP(w, r)
			return
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		p := filepath.Join(dir, filepath.Clean(strings.TrimPrefix(r.URL.Path, "/")))
		absP, err := filepath.Abs(p)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		rel, err := filepath.Rel(absDir, absP)
		if err != nil || strings.Contains(rel, "..") {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		// p is safe: absP was verified under absDir via Rel above
		if f, err := os.Stat(p); err == nil && !f.IsDir() { // #nosec G703
			http.ServeFile(w, r, p)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

// ResolveStaticDir returns a directory containing index.html for the SPA, or "" if none found.
// Tries STATIC_DIR, then web/dist relative to CWD or executable, so signup links (GET /) work.
func ResolveStaticDir() string {
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
	cleanDir := filepath.Clean(dir)
	// dir is from ResolveStaticDir (env or known rel paths); we only join with constant "index.html"
	f, err := os.Stat(filepath.Join(cleanDir, "index.html")) // #nosec G703
	return err == nil && f != nil && !f.IsDir()
}
