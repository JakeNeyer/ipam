package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest is the body for POST /api/admin/users.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // "user" or "admin"
}

// AdminUsersHandler handles GET (list) and POST (create) /api/admin/users. Admin only.
func AdminUsersHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || user.Role != store.RoleAdmin {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			listUsers(s, w)
		case http.MethodPost:
			createUser(s, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func listUsers(s *store.Store, w http.ResponseWriter) {
	users, err := s.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, UserResponse{ID: u.ID.String(), Email: u.Email, Role: u.Role, TourCompleted: u.TourCompleted})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"users": out})
}

func createUser(s *store.Store, w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	role := store.RoleUser
	if req.Role == store.RoleAdmin {
		role = store.RoleAdmin
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}
	newUser := &store.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.CreateUser(newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]UserResponse{
		"user": {ID: newUser.ID.String(), Email: newUser.Email, Role: newUser.Role, TourCompleted: newUser.TourCompleted},
	})
}
