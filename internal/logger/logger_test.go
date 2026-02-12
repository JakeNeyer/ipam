package logger

import (
	"errors"
	"log/slog"
	"testing"
)

func TestErrAttr(t *testing.T) {
	// nil error returns empty attr (no-op for slog)
	got := ErrAttr(nil)
	if !got.Equal(slog.Attr{}) {
		t.Errorf("ErrAttr(nil) should equal empty slog.Attr, got Key=%q", got.Key)
	}
	err := errors.New("test error")
	got = ErrAttr(err)
	if got.Key != KeyError {
		t.Errorf("ErrAttr(err).Key = %q, want %q", got.Key, KeyError)
	}
	if v, ok := got.Value.Any().(string); !ok || v != "test error" {
		t.Errorf("ErrAttr(err).Value = %v", got.Value.Any())
	}
}
