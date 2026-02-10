package auth

import (
	"context"
	"net/http"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "user"
const requestContextKey contextKey = "request"

// WithUser returns a context with the user attached.
func WithUser(ctx context.Context, user *store.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the user from the context, or nil if not set.
func UserFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(userContextKey).(*store.User)
	return u
}

// UserIDFromContext returns the current user's ID, or uuid.Nil if not set.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	u := UserFromContext(ctx)
	if u == nil {
		return uuid.Nil
	}
	return u.ID
}

// WithRequest returns a context with the request attached (for use cases that need cookies etc.).
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, r)
}

// RequestFromContext returns the request from the context, or nil if not set.
func RequestFromContext(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestContextKey).(*http.Request)
	return r
}
