package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"golang.org/x/crypto/bcrypt"
)

// SetupStatusResponse is the response for GET /api/setup/status.
type SetupStatusResponse struct {
	SetupRequired bool `json:"setup_required"`
}

// SetupRequest is the body for POST /api/setup.
type SetupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GetSetupStatusHandler returns a handler for GET /api/setup/status.
// Returns setup_required: true when no users exist.
func GetSetupStatusHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		users, err := s.ListUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SetupStatusResponse{
			SetupRequired: len(users) == 0,
		})
	}
}

// PostSetupHandler returns a handler for POST /api/setup.
// Creates the first admin user only when no users exist.
func PostSetupHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		users, err := s.ListUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(users) > 0 {
			auth.WriteJSONError(w, "setup already completed", http.StatusForbidden)
			return
		}
		var req SetupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		admin := &store.User{
			Email:        req.Email,
			PasswordHash: string(hash),
			Role:         store.RoleAdmin,
		}
		if err := s.CreateUser(admin); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": {ID: admin.ID.String(), Email: admin.Email, Role: admin.Role, TourCompleted: admin.TourCompleted},
		})
	}
}
