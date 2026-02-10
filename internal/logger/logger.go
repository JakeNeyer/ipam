// Package logger provides a standardized structured logger for the IPAM backend.
// Logs to stdout with consistent keys and message formats.
package logger

import (
	"log/slog"
	"os"
)

// Standard keys for structured logs (consistent across the backend).
const (
	KeyPath       = "path"
	KeyMethod     = "method"
	KeyStatus     = "status"
	KeyDuration   = "duration_ms"
	KeyError      = "error"
	KeyUserID     = "user_id"
	KeyEmail      = "email"
	KeyOperation  = "operation"
	KeySetupRequired = "setup_required"
)

// Standard error messages (consistent, non-PII where possible).
const (
	MsgSetupStatusFailed   = "setup status check failed"
	MsgSetupAlreadyDone    = "setup already completed"
	MsgSetupMissingCreds   = "email and password required"
	MsgSetupPasswordFailed = "password setup failed"
	MsgSetupCreateUserFailed = "failed to create user"
	MsgAuthMissingCreds   = "email and password required"
	MsgAuthInvalidCreds   = "invalid email or password"
	MsgAuthPasswordMismatch = "password mismatch"
	MsgStoreError         = "store error"
)

var Log *slog.Logger

func init() {
	Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: false,
	}))
}

// Info logs at Info level with optional key-value pairs.
func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

// Error logs at Error level with optional key-value pairs.
func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

// Warn logs at Warn level with optional key-value pairs.
func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

// Request logs a completed HTTP request (method, path, status, duration_ms).
func Request(method, path string, status int, durationMs int64) {
	Log.Info("request",
		slog.String(KeyMethod, method),
		slog.String(KeyPath, path),
		slog.Int(KeyStatus, status),
		slog.Int64(KeyDuration, durationMs),
	)
}

// ErrAttr returns slog.Any(KeyError, err) or a no-op if err is nil.
func ErrAttr(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any(KeyError, err.Error())
}
