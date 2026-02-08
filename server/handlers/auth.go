package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest is the body for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse is the user object returned by auth and admin endpoints.
type UserResponse struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	TourCompleted  bool   `json:"tour_completed"`
}

// LoginHandler returns a handler for POST /api/auth/login.
func LoginHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Email == "" || req.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}
		user, err := s.GetUserByEmail(req.Email)
		if err != nil {
			auth.WriteJSONError(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			auth.WriteJSONError(w, "invalid email or password", http.StatusUnauthorized)
			return
		}
		sessionID := auth.NewSessionID()
	s.CreateSession(sessionID, user.ID, time.Now().Add(auth.SessionDuration))
	auth.SetSessionCookie(w, sessionID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": {ID: user.ID.String(), Email: user.Email, Role: user.Role, TourCompleted: user.TourCompleted},
		})
	}
}

// LogoutHandler returns a handler for POST /api/auth/logout.
func LogoutHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if c, err := r.Cookie(auth.SessionCookieName); err == nil && c != nil && c.Value != "" {
			s.DeleteSession(c.Value)
		}
		auth.ClearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}

// MeHandler returns a handler for GET /api/auth/me. Requires auth (middleware sets user in context).
func MeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user := auth.UserFromContext(r.Context())
		if user == nil {
			auth.WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]UserResponse{
			"user": {ID: user.ID.String(), Email: user.Email, Role: user.Role, TourCompleted: user.TourCompleted},
		})
	}
}

// TourCompletedHandler returns a handler for POST /api/auth/me/tour-completed. Marks the tour as completed for the current user.
func TourCompletedHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		user := auth.UserFromContext(r.Context())
		if user == nil {
			auth.WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := s.SetUserTourCompleted(user.ID, true); err != nil {
			auth.WriteJSONError(w, "failed to update tour state", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
