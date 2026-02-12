package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/swaggest/rest/response"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"golang.org/x/crypto/bcrypt"
)

// UserResponse is the user object returned by auth and admin endpoints.
type UserResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	TourCompleted  bool   `json:"tour_completed"`
	OrganizationID string `json:"organization_id,omitempty"`
}

func userToResponse(u *store.User) UserResponse {
	resp := UserResponse{ID: u.ID.String(), Email: u.Email, Role: u.Role, TourCompleted: u.TourCompleted}
	if u.OrganizationID != uuid.Nil {
		resp.OrganizationID = u.OrganizationID.String()
	}
	return resp
}

// loginInput is the request body for POST /api/auth/login.
type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginOutput is the response for POST /api/auth/login. Embeds response.EmbeddedSetter so the use case can set the session cookie.
type loginOutput struct {
	response.EmbeddedSetter
	User UserResponse `json:"user"`
}

// NewLoginUseCase returns a use case for POST /api/auth/login.
// If limiter is non-nil, failed login attempts per client IP are limited to mitigate brute-force.
// If cfg has any OAuth providers enabled, password login is rejected except when the user is the only one in the system (e.g. right after setup).
func NewLoginUseCase(s store.Storer, limiter *auth.LoginAttemptLimiter, cfg *config.Config) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input loginInput, output *loginOutput) error {
		if cfg != nil && len(cfg.EnabledOAuthProviders()) > 0 {
			allowSetupLogin := false
			if u, err := s.GetUserByEmail(strings.TrimSpace(strings.ToLower(input.Email))); err == nil {
				users, listErr := s.ListUsers(nil)
				if listErr == nil && len(users) == 1 && users[0].ID == u.ID {
					allowSetupLogin = true
				}
			}
			if !allowSetupLogin {
				return status.Wrap(errors.New("password login is disabled; sign in with your provider"), status.PermissionDenied)
			}
		}
		r := auth.RequestFromContext(ctx)
		ip := auth.ClientIP(r)
		if limiter != nil && limiter.IsBlocked(ip) {
			logger.Info("login blocked: too many attempts", logger.KeyOperation, "login", "ip", ip)
			return status.Wrap(errors.New("too many failed login attempts; try again later"), status.ResourceExhausted)
		}
		logger.Info("login request", logger.KeyOperation, "login", logger.KeyEmail, input.Email)
		if !validation.ValidateEmail(input.Email) {
			logger.Info(logger.MsgAuthMissingCreds, logger.KeyOperation, "login")
			if limiter != nil {
				limiter.RecordFailure(ip)
			}
			return status.Wrap(errors.New("valid email required"), status.InvalidArgument)
		}
		if !validation.ValidatePassword(input.Password) {
			logger.Info(logger.MsgAuthMissingCreds, logger.KeyOperation, "login")
			if limiter != nil {
				limiter.RecordFailure(ip)
			}
			return status.Wrap(errors.New("password must be at least 8 characters"), status.InvalidArgument)
		}
		user, err := s.GetUserByEmail(strings.TrimSpace(strings.ToLower(input.Email)))
		if err != nil {
			logger.Error(logger.MsgAuthInvalidCreds, logger.KeyOperation, "login", logger.ErrAttr(err))
			if limiter != nil {
				limiter.RecordFailure(ip)
			}
			return status.Wrap(errors.New("invalid email or password"), status.Unauthenticated)
		}
		if user.PasswordHash == "" {
			logger.Info(logger.MsgAuthInvalidCreds, logger.KeyOperation, "login", logger.KeyEmail, input.Email)
			if limiter != nil {
				limiter.RecordFailure(ip)
			}
			return status.Wrap(errors.New("invalid email or password"), status.Unauthenticated)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			logger.Info(logger.MsgAuthPasswordMismatch, logger.KeyOperation, "login", logger.KeyEmail, input.Email)
			if limiter != nil {
				limiter.RecordFailure(ip)
			}
			return status.Wrap(errors.New("invalid email or password"), status.Unauthenticated)
		}
		if limiter != nil {
			limiter.RecordSuccess(ip)
		}
		sessionID := auth.NewSessionID()
		s.CreateSession(sessionID, user.ID, time.Now().Add(auth.SessionDuration))
		secure := false
		if r := auth.RequestFromContext(ctx); r != nil && r.TLS != nil {
			secure = true
		}
		auth.SetSessionCookie(output.ResponseWriter(), sessionID, secure)
		logger.Info("login success", logger.KeyOperation, "login", logger.KeyUserID, user.ID.String(), logger.KeyEmail, user.Email)
		output.User = userToResponse(user)
		return nil
	})
	u.SetTitle("Login")
	u.SetDescription("Authenticate with email and password; sets session cookie")
	u.SetExpectedErrors(status.InvalidArgument, status.Unauthenticated, status.ResourceExhausted, status.PermissionDenied)
	return u
}

// logoutOutput holds the response writer so the use case can clear the session cookie. No body (204).
type logoutOutput struct {
	response.EmbeddedSetter
}

// NewLogoutUseCase returns a use case for POST /api/auth/logout.
func NewLogoutUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *logoutOutput) error {
		r := auth.RequestFromContext(ctx)
		secure := false
		if r != nil {
			if r.TLS != nil {
				secure = true
			}
			if c, err := r.Cookie(auth.SessionCookieName); err == nil && c != nil && c.Value != "" {
				s.DeleteSession(c.Value)
			}
		}
		auth.ClearSessionCookie(output.ResponseWriter(), secure)
		return nil
	})
	u.SetTitle("Logout")
	u.SetDescription("Clear session cookie")
	return u
}

// meOutput is the response for GET /api/auth/me.
type meOutput struct {
	User UserResponse `json:"user"`
}

// NewMeUseCase returns a use case for GET /api/auth/me.
func NewMeUseCase() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *meOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		output.User = userToResponse(user)
		return nil
	})
	u.SetTitle("Get current user")
	u.SetDescription("Returns the current authenticated user")
	u.SetExpectedErrors(status.Unauthenticated)
	return u
}

// NewTourCompletedUseCase returns a use case for POST /api/auth/me/tour-completed.
func NewTourCompletedUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *struct{}) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		if err := s.SetUserTourCompleted(user.ID, true); err != nil {
			return status.Wrap(err, status.Internal)
		}
		return nil
	})
	u.SetTitle("Mark tour completed")
	u.SetDescription("Marks the onboarding tour as completed for the current user")
	u.SetExpectedErrors(status.Unauthenticated, status.Internal)
	return u
}

// API token response types
type apiTokenResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	CreatedAt      string  `json:"created_at"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	OrganizationID string  `json:"organization_id,omitempty"` // when set, token is scoped to this org
}

type createTokenResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Token     string  `json:"token"`
	CreatedAt      string  `json:"created_at"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	OrganizationID string  `json:"organization_id,omitempty"`
}

// CreateTokenRequest is the body for POST /api/auth/me/tokens.
type CreateTokenRequest struct {
	Name           string  `json:"name"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	OrganizationID *string `json:"organization_id,omitempty"` // optional; global admin only — scopes token to this org
}

// listTokensOutput is the response for GET /api/auth/me/tokens.
type listTokensOutput struct {
	Tokens []apiTokenResponse `json:"tokens"`
}

// createTokenOutput is the response for POST /api/auth/me/tokens.
type createTokenOutput struct {
	Token createTokenResponse `json:"token"`
}

// NewListTokensUseCase returns a use case for GET /api/auth/me/tokens.
func NewListTokensUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *listTokensOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		tokens, err := s.ListAPITokens(user.ID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		out := make([]apiTokenResponse, 0, len(tokens))
		for _, t := range tokens {
			var expiresAt *string
			if t.ExpiresAt != nil {
				s := t.ExpiresAt.Format(time.RFC3339)
				expiresAt = &s
			}
			orgID := ""
			if t.OrganizationID != uuid.Nil {
				orgID = t.OrganizationID.String()
			}
			out = append(out, apiTokenResponse{
				ID:             t.ID.String(),
				Name:           t.Name,
				CreatedAt:      t.CreatedAt.Format(time.RFC3339),
				ExpiresAt:      expiresAt,
				OrganizationID: orgID,
			})
		}
		output.Tokens = out
		return nil
	})
	u.SetTitle("List API tokens")
	u.SetDescription("List API tokens for the current user")
	u.SetExpectedErrors(status.Unauthenticated, status.Internal)
	return u
}

// NewCreateTokenUseCase returns a use case for POST /api/auth/me/tokens. Only admins can create tokens.
func NewCreateTokenUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input CreateTokenRequest, output *createTokenOutput) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		if user.Role != store.RoleAdmin {
			return status.Wrap(errors.New("only admins can create API tokens"), status.PermissionDenied)
		}
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return status.Wrap(errors.New("name is required"), status.InvalidArgument)
		}
		var orgID *uuid.UUID
		if input.OrganizationID != nil && *input.OrganizationID != "" {
			if !auth.IsGlobalAdmin(user) {
				return status.Wrap(errors.New("only global admins can create org-scoped tokens"), status.PermissionDenied)
			}
			parsed, err := uuid.Parse(*input.OrganizationID)
			if err != nil {
				return status.Wrap(errors.New("organization_id must be a valid UUID"), status.InvalidArgument)
			}
			if _, err := s.GetOrganization(parsed); err != nil {
				return status.Wrap(errors.New("organization not found"), status.NotFound)
			}
			orgID = &parsed
		}
		if auth.IsGlobalAdmin(user) && orgID == nil {
			return status.Wrap(errors.New("organization_id is required for global admin tokens"), status.InvalidArgument)
		}
		var expiresAt *time.Time
		if input.ExpiresAt != nil && *input.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, *input.ExpiresAt)
			if err != nil {
				return status.Wrap(errors.New("expires_at must be RFC3339"), status.InvalidArgument)
			}
			if t.Before(time.Now()) {
				return status.Wrap(errors.New("expires_at must be in the future"), status.InvalidArgument)
			}
			expiresAt = &t
		}
		token, rawToken, err := s.CreateAPIToken(user.ID, name, expiresAt, orgID)
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		var expiresAtStr *string
		if token.ExpiresAt != nil {
			s := token.ExpiresAt.Format(time.RFC3339)
			expiresAtStr = &s
		}
		createOrgID := ""
		if token.OrganizationID != uuid.Nil {
			createOrgID = token.OrganizationID.String()
		}
		output.Token = createTokenResponse{
			ID:             token.ID.String(),
			Name:           token.Name,
			Token:          rawToken,
			CreatedAt:      token.CreatedAt.Format(time.RFC3339),
			ExpiresAt:      expiresAtStr,
			OrganizationID: createOrgID,
		}
		return nil
	})
	u.SetTitle("Create API token")
	u.SetDescription("Create an API token. Only admins can create tokens. The raw token is returned once.")
	u.SetExpectedErrors(status.Unauthenticated, status.PermissionDenied, status.InvalidArgument, status.Internal)
	return u
}

// NewDeleteTokenUseCase returns a use case for DELETE /api/auth/me/tokens/:id.
func NewDeleteTokenUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct {
		ID uuid.UUID `path:"id"`
	}, output *struct{}) error {
		user := auth.UserFromContext(ctx)
		if user == nil {
			return status.Wrap(errors.New("unauthorized"), status.Unauthenticated)
		}
		if err := s.DeleteAPIToken(input.ID, user.ID); err != nil {
			if err.Error() == "token not found" {
				return status.Wrap(err, status.NotFound)
			}
			return status.Wrap(err, status.Internal)
		}
		return nil
	})
	u.SetTitle("Delete API token")
	u.SetDescription("Delete an API token for the current user")
	u.SetExpectedErrors(status.Unauthenticated, status.NotFound, status.Internal)
	return u
}
