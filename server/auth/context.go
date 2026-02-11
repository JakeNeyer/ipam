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

// IsGlobalAdmin returns true if the user is the global admin (no organization).
// Global admin can create organizations and access all org-scoped resources.
// OrganizationID == uuid.Nil is the global-admin sentinel; it must never be assignable by non-global-admin.
func IsGlobalAdmin(u *store.User) bool {
	return u != nil && u.OrganizationID == uuid.Nil
}

// RequireGlobalAdminForNilOrg returns nil if organizationID is not Nil, or if the user is global admin.
// Otherwise it returns an error so that assigning "global admin" (Nil org) is never allowed for non-global-admin.
// Call this before any operation that could set a user's or invite's organization to Nil.
func RequireGlobalAdminForNilOrg(user *store.User, organizationID uuid.UUID) error {
	if organizationID != uuid.Nil {
		return nil
	}
	if user != nil && IsGlobalAdmin(user) {
		return nil
	}
	return errForbiddenAssignGlobalAdmin
}

var errForbiddenAssignGlobalAdmin = &authErr{code: "forbidden", msg: "only global admin can assign global admin"}

type authErr struct{ code, msg string }

func (e *authErr) Error() string { return e.msg }

// WithRequest returns a context with the request attached (for use cases that need cookies etc.).
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, r)
}

// RequestFromContext returns the request from the context, or nil if not set.
func RequestFromContext(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestContextKey).(*http.Request)
	return r
}
