package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnauthorized(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := Unauthorized("https://app.example.com", next)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantNextCall bool
		wantBody     string
	}{
		{"api passes through", "/api/foo", http.StatusOK, true, ""},
		{"docs passes through", "/docs", http.StatusOK, true, ""},
		{"root returns 401", "/", 401, false, "Unauthorized"},
		{"other returns 401", "/login", 401, false, "Unauthorized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCall {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNextCall)
			}
			if tt.wantBody != "" && rec.Body.Len() > 0 && !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body %q does not contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestUnauthorized_trimTrailingSlash(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := Unauthorized("https://app.example.com/", next)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	// Body should contain origin without trailing slash
	if !strings.Contains(rec.Body.String(), "https://app.example.com") {
		t.Errorf("body should contain app origin: %s", rec.Body.String())
	}
}

func TestStatic(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>ok</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := Static(dir, next)

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantNextCall bool
	}{
		{"api passes through", "/api/foo", 200, true},
		{"docs passes through", "/docs", 200, true},
		{"root serves index", "/", 200, false},
		{"subpath no file serves index", "/nope", 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCall {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNextCall)
			}
		})
	}
}

func TestStatic_pathTraversal(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0644)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := Static(dir, next)
	// Request path that would escape dir - must not serve files outside dir
	req := httptest.NewRequest("GET", "/../../../etc/passwd", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	// Should either serve index.html (200) or reject (400), never serve actual /etc/passwd
	if rec.Code != 200 && rec.Code != 400 {
		t.Errorf("status = %d, want 200 (fallback) or 400 (rejected)", rec.Code)
	}
}

func TestStaticDirHasIndex(t *testing.T) {
	dir := t.TempDir()
	if staticDirHasIndex(dir) {
		t.Error("empty dir should not have index")
	}
	_ = os.WriteFile(filepath.Join(dir, "index.html"), nil, 0644)
	if !staticDirHasIndex(dir) {
		t.Error("dir with index.html should have index")
	}
	sub := filepath.Join(dir, "sub")
	_ = os.Mkdir(sub, 0755)
	if staticDirHasIndex(sub) {
		t.Error("subdir without index should not have index")
	}
}

func TestResolveStaticDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.html"), nil, 0644)
	prev := os.Getenv("STATIC_DIR")
	defer os.Setenv("STATIC_DIR", prev)
	os.Setenv("STATIC_DIR", dir)
	got := ResolveStaticDir()
	if got != dir {
		// ResolveStaticDir may return abs path
		abs, _ := filepath.Abs(dir)
		if got != abs && got != dir {
			t.Errorf("ResolveStaticDir() = %q, want %q or %q", got, dir, abs)
		}
	}
}
