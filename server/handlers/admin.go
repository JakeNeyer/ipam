package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/server/validation"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest is the body for POST /api/admin/users.
type CreateUserRequest struct {
	Email          string    `json:"email"`
	Password       string    `json:"password"` // #nosec G117 -- request DTO for admin create user, not a log/secret leak
	Role           string    `json:"role"`
	OrganizationID uuid.UUID `json:"organization_id,omitempty"`
}

// UpdateUserRoleRequest is the body for PATCH /api/admin/users/:id/role.
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserOrganizationRequest is the body for PATCH /api/admin/users/:id/organization. Global admin only.
type UpdateUserOrganizationRequest struct {
	OrganizationID uuid.UUID `json:"organization_id"`
}

// AdminUsersHandler handles GET (list) and POST (create) /api/admin/users. Admin only.
func AdminUsersHandler(s store.Storer, cfg *config.Config) http.HandlerFunc {
	oauthEnabled := cfg != nil && len(cfg.EnabledOAuthProviders()) > 0
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || user.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			listUsers(s, w, r)
		case http.MethodPost:
			createUser(s, oauthEnabled, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func listUsers(s store.Storer, w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	var orgID *uuid.UUID
	if user != nil && !auth.IsGlobalAdmin(user) {
		orgID = &user.OrganizationID
	}
	users, err := s.ListUsers(orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userToResponse(u))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"users": out})
}

func createUser(s store.Storer, oauthEnabled bool, w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !validation.ValidateEmail(req.Email) {
		http.Error(w, "valid email required", http.StatusBadRequest)
		return
	}
	// Password is optional when OAuth is enabled; the user will sign in via OAuth.
	var passwordHash string
	if req.Password != "" {
		if !validation.ValidatePassword(req.Password) {
			http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = string(hash)
	} else if !oauthEnabled {
		http.Error(w, "password required when OAuth is not configured", http.StatusBadRequest)
		return
	}
	role := store.RoleUser
	if req.Role == store.RoleAdmin {
		role = store.RoleAdmin
	}
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orgID := user.OrganizationID
	if auth.IsGlobalAdmin(user) {
		orgID = req.OrganizationID
	}
	if err := auth.RequireGlobalAdminForNilOrg(user, orgID); err != nil {
		auth.WriteJSONError(w, err.Error(), http.StatusForbidden)
		return
	}
	newUser := &store.User{
		Email:          strings.TrimSpace(strings.ToLower(req.Email)),
		PasswordHash:   passwordHash,
		Role:           role,
		OrganizationID: orgID,
	}
	if err := s.CreateUser(newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]UserResponse{
		"user": userToResponse(newUser),
	})
}

// UpdateUserRoleHandler handles PATCH /api/admin/users/:id/role. Admin only.
func UpdateUserRoleHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user := auth.UserFromContext(r.Context())
		if user == nil || user.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
		idStr = strings.TrimSuffix(idStr, "/role")
		idStr = strings.Trim(idStr, "/")
		if idStr == "" {
			auth.WriteJSONError(w, "user id required", http.StatusBadRequest)
			return
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			auth.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		if userID == user.ID {
			auth.WriteJSONError(w, "cannot change your own role", http.StatusBadRequest)
			return
		}

		var req UpdateUserRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			auth.WriteJSONError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		role := strings.TrimSpace(strings.ToLower(req.Role))
		if role != store.RoleUser && role != store.RoleAdmin {
			auth.WriteJSONError(w, "invalid role", http.StatusBadRequest)
			return
		}

		if err := s.SetUserRole(userID, role); err != nil {
			if err.Error() == "user not found" {
				auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := s.GetUser(userID)
		if err != nil {
			auth.WriteJSONError(w, "failed to fetch updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": userToResponse(updated),
		})
	}
}

// UpdateUserOrganizationHandler handles PATCH /api/admin/users/:id/organization. Global admin only.
func UpdateUserOrganizationHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user := auth.UserFromContext(r.Context())
		if user == nil || !auth.IsGlobalAdmin(user) {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
		idStr = strings.TrimSuffix(idStr, "/organization")
		idStr = strings.Trim(idStr, "/")
		if idStr == "" {
			auth.WriteJSONError(w, "user id required", http.StatusBadRequest)
			return
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			auth.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		if userID == user.ID {
			auth.WriteJSONError(w, "cannot change your own organization", http.StatusForbidden)
			return
		}

		var req UpdateUserOrganizationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			auth.WriteJSONError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := auth.RequireGlobalAdminForNilOrg(user, req.OrganizationID); err != nil {
			auth.WriteJSONError(w, err.Error(), http.StatusForbidden)
			return
		}

		if err := s.SetUserOrganization(userID, req.OrganizationID); err != nil {
			if err.Error() == "user not found" {
				auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := s.GetUser(userID)
		if err != nil {
			auth.WriteJSONError(w, "failed to fetch updated user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": userToResponse(updated),
		})
	}
}

// DeleteUserHandler handles DELETE /api/admin/users/:id. Admin only.
func DeleteUserHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requester := auth.UserFromContext(r.Context())
		if requester == nil || requester.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
		idStr = strings.Trim(idStr, "/")
		if idStr == "" {
			auth.WriteJSONError(w, "user id required", http.StatusBadRequest)
			return
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			auth.WriteJSONError(w, "invalid user id", http.StatusBadRequest)
			return
		}
		if userID == requester.ID {
			auth.WriteJSONError(w, "cannot delete your own account", http.StatusBadRequest)
			return
		}

		target, err := s.GetUser(userID)
		if err != nil {
			auth.WriteJSONError(w, "user not found", http.StatusNotFound)
			return
		}

		if target.Role == store.RoleAdmin {
			users, err := s.ListUsers(nil)
			if err != nil {
				auth.WriteJSONError(w, "failed to list users", http.StatusInternalServerError)
				return
			}
			adminCount := 0
			for _, u := range users {
				if u != nil && u.Role == store.RoleAdmin {
					adminCount++
				}
			}
			if adminCount <= 1 {
				auth.WriteJSONError(w, "cannot delete the last admin", http.StatusBadRequest)
				return
			}
		}

		if err := s.DeleteUser(userID); err != nil {
			if err.Error() == "user not found" {
				auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
