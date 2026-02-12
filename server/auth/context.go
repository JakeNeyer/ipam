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
const effectiveOrgContextKey contextKey = "effective_organization"

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

// WithEffectiveOrganization sets the effective organization for this request (e.g. from an org-scoped API token).
// When set, the request is limited to that org even if the user is global admin.
func WithEffectiveOrganization(ctx context.Context, orgID uuid.UUID) context.Context {
	return context.WithValue(ctx, effectiveOrgContextKey, orgID)
}

// EffectiveOrganizationID returns the effective organization for this request, or uuid.Nil if not set.
// When set (e.g. org-scoped API token), handlers should filter by this org and not treat the user as global admin for scope.
func EffectiveOrganizationID(ctx context.Context) uuid.UUID {
	v, _ := ctx.Value(effectiveOrgContextKey).(uuid.UUID)
	return v
}

// ResolveOrgID returns the organization ID to use for list/create: effective org from token if set,
// else user's org (or optional input org for global admin). Used by env/block/alloc/reserved handlers.
func ResolveOrgID(ctx context.Context, user *store.User, inputOrgID uuid.UUID) *uuid.UUID {
	if effective := EffectiveOrganizationID(ctx); effective != uuid.Nil {
		return &effective
	}
	if user == nil {
		return nil
	}
	if !IsGlobalAdmin(user) {
		return &user.OrganizationID
	}
	if inputOrgID != uuid.Nil {
		return &inputOrgID
	}
	return nil
}

// UserOrgForAccess returns the organization ID to use for access checks (get/update/delete).
// When effective org is set (org-scoped token), returns that; else returns user.OrganizationID (Nil for global admin).
func UserOrgForAccess(ctx context.Context, user *store.User) uuid.UUID {
	if effective := EffectiveOrganizationID(ctx); effective != uuid.Nil {
		return effective
	}
	if user == nil {
		return uuid.Nil
	}
	return user.OrganizationID
}

// IsGlobalAdmin returns true if the user is the global admin (no organization).
// Global admin can create organizations and access all org-scoped resources.
// OrganizationID == uuid.Nil is the global-admin sentinel; it must never be assignable by non-global-admin.
// When EffectiveOrganizationID(ctx) is set (org-scoped token), the request is not treated as global admin for scope.
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
