package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateSignupInviteRequest is the body for POST /api/admin/signup-invites.
type CreateSignupInviteRequest struct {
	ExpiresInHours int        `json:"expires_in_hours"`
	OrganizationID uuid.UUID  `json:"organization_id,omitempty"`
	Role           string     `json:"role,omitempty"`
}

// CreateSignupInviteResponse is the response for POST /api/admin/signup-invites.
type CreateSignupInviteResponse struct {
	InviteURL string    `json:"invite_url"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ValidateSignupInviteResponse is the response for GET /api/signup/validate.
type ValidateSignupInviteResponse struct {
	Valid     bool      `json:"valid"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RegisterWithInviteRequest is the body for POST /api/signup/register.
type RegisterWithInviteRequest struct {
	Token    string `json:"token"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignupInviteResponse is one invite in the list for GET /api/admin/signup-invites.
type SignupInviteResponse struct {
	ID          string     `json:"id"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	UsedAt      *time.Time `json:"used_at,omitempty"`
	UsedByEmail string     `json:"used_by_email,omitempty"`
}

// AdminSignupInvitesHandler handles GET (list) and POST (create) /api/admin/signup-invites. Admin only.
func AdminSignupInvitesHandler(s store.Storer, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || user.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			listSignupInvites(s, user.ID, w)
		case http.MethodPost:
			createSignupInvite(s, user, w, r, cfg)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func listSignupInvites(s store.Storer, adminID uuid.UUID, w http.ResponseWriter) {
	invites, err := s.ListSignupInvites(adminID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]SignupInviteResponse, 0, len(invites))
	for _, inv := range invites {
		resp := SignupInviteResponse{
			ID:        inv.ID.String(),
			CreatedAt: inv.CreatedAt,
			ExpiresAt: inv.ExpiresAt,
			UsedAt:    inv.UsedAt,
		}
		if inv.UsedByUserID != nil {
			if u, err := s.GetUser(*inv.UsedByUserID); err == nil {
				resp.UsedByEmail = u.Email
			}
		}
		out = append(out, resp)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"invites": out})
}

func createSignupInvite(s store.Storer, user *store.User, w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	var req CreateSignupInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	hours := req.ExpiresInHours
	if hours < 1 {
		hours = 24
	}
	if hours > 720 {
		hours = 720
	}
	expiresAt := time.Now().Add(time.Duration(hours) * time.Hour)
	orgID := user.OrganizationID
	role := store.RoleUser
	if auth.IsGlobalAdmin(user) {
		orgID = req.OrganizationID
		if req.Role == store.RoleAdmin {
			role = store.RoleAdmin
		}
	}
	if err := auth.RequireGlobalAdminForNilOrg(user, orgID); err != nil {
		auth.WriteJSONError(w, err.Error(), http.StatusForbidden)
		return
	}
	inv, rawToken, err := s.CreateSignupInvite(user.ID, expiresAt, orgID, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = inv
	var baseURL string
	if cfg != nil && strings.TrimSpace(cfg.AppOrigin) != "" {
		baseURL = strings.TrimSuffix(strings.TrimSpace(cfg.AppOrigin), "/")
	} else {
		baseURL = baseURLFromRequest(r)
		baseURL = baseURL + strings.TrimSuffix(appBasePathFromRequest(r.URL.Path), "/")
	}
	inviteURL := baseURL + "/#signup?token=" + rawToken
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateSignupInviteResponse{
		InviteURL: inviteURL,
		Token:     rawToken,
		ExpiresAt: expiresAt,
	})
}

// RevokeSignupInviteHandler handles DELETE /api/admin/signup-invites/:id. Admin only.
func RevokeSignupInviteHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user := auth.UserFromContext(r.Context())
		if user == nil || user.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/signup-invites/")
		idStr = strings.Trim(idStr, "/")
		if idStr == "" {
			auth.WriteJSONError(w, "invite id required", http.StatusBadRequest)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			auth.WriteJSONError(w, "invalid invite id", http.StatusBadRequest)
			return
		}
		if err := s.DeleteSignupInvite(id); err != nil {
			auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// baseURLFromRequest returns the origin (scheme + host) for building absolute URLs.
func baseURLFromRequest(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return strings.TrimSuffix(scheme+"://"+host, "/")
}

// appBasePathFromRequest returns the path prefix before /api..., preserving app subpaths
// such as /ipam when deployed behind a reverse proxy.
func appBasePathFromRequest(requestPath string) string {
	const marker = "/api/"
	idx := strings.Index(requestPath, marker)
	if idx == -1 {
		return "/"
	}
	prefix := strings.TrimSuffix(requestPath[:idx], "/")
	if prefix == "" {
		return "/"
	}
	return prefix + "/"
}

// ValidateSignupInviteHandler handles GET /api/signup/validate?token=xxx. No auth.
func ValidateSignupInviteHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			auth.WriteJSONError(w, "token required", http.StatusBadRequest)
			return
		}
		inv, err := s.GetSignupInviteByToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(ValidateSignupInviteResponse{Valid: false})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ValidateSignupInviteResponse{
			Valid:     true,
			ExpiresAt: inv.ExpiresAt,
		})
	}
}

// RegisterWithInviteHandler handles POST /api/signup/register. No auth. Creates user, consumes invite, sets session.
func RegisterWithInviteHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req RegisterWithInviteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token := strings.TrimSpace(req.Token)
		if token == "" {
			auth.WriteJSONError(w, "token required", http.StatusBadRequest)
			return
		}
		inv, err := s.GetSignupInviteByToken(token)
		if err != nil {
			auth.WriteJSONError(w, "invalid or expired invite link", http.StatusBadRequest)
			return
		}
		inviter, err := s.GetUser(inv.CreatedBy)
		if err != nil {
			auth.WriteJSONError(w, "invalid invite", http.StatusBadRequest)
			return
		}
		orgID := inv.OrganizationID
		if orgID == uuid.Nil {
			orgID = inviter.OrganizationID
		}
		role := inv.Role
		if role != store.RoleAdmin {
			role = store.RoleUser
		}
		if !validation.ValidateEmail(req.Email) {
			auth.WriteJSONError(w, "valid email required", http.StatusBadRequest)
			return
		}
		if !validation.ValidatePassword(req.Password) {
			auth.WriteJSONError(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		email := strings.TrimSpace(strings.ToLower(req.Email))
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		newUser := &store.User{
			Email:          email,
			PasswordHash:   string(hash),
			Role:           role,
			OrganizationID: orgID,
		}
		if err := s.CreateUser(newUser); err != nil {
			auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.MarkSignupInviteUsed(inv.ID, newUser.ID); err != nil {
			_ = err
		}
		sessionID := auth.NewSessionID()
		s.CreateSession(sessionID, newUser.ID, time.Now().Add(auth.SessionDuration))
		secure := r.TLS != nil
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			secure = true
		}
		auth.SetSessionCookie(w, sessionID, secure)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": userToResponse(newUser),
		})
	}
}
