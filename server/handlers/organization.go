package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

// CreateOrganizationRequest is the body for POST /api/admin/organizations.
type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

// OrganizationResponse is one organization in list or create response.
type OrganizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// AdminOrganizationsHandler handles GET (list) and POST (create) /api/admin/organizations. Global admin only.
func AdminOrganizationsHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !auth.IsGlobalAdmin(user) {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			orgs, err := s.ListOrganizations()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out := make([]OrganizationResponse, 0, len(orgs))
			for _, o := range orgs {
				out = append(out, OrganizationResponse{
					ID:        o.ID.String(),
					Name:      o.Name,
					CreatedAt: o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"organizations": out})
		case http.MethodPost:
			var req CreateOrganizationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			name := strings.TrimSpace(req.Name)
			if name == "" {
				auth.WriteJSONError(w, "name is required", http.StatusBadRequest)
				return
			}
			org := &store.Organization{Name: name}
			if err := s.CreateOrganization(org); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]OrganizationResponse{
				"organization": {
					ID:        org.ID.String(),
					Name:      org.Name,
					CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// UpdateOrganizationRequest is the body for PATCH /api/admin/organizations/:id.
type UpdateOrganizationRequest struct {
	Name string `json:"name"`
}

// AdminOrganizationByIDHandler handles PATCH (update name) and DELETE for /api/admin/organizations/:id. Global admin only.
func AdminOrganizationByIDHandler(s store.Storer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !auth.IsGlobalAdmin(user) {
			auth.WriteJSONError(w, "forbidden", http.StatusForbidden)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/api/admin/organizations/")
		idStr = strings.Trim(idStr, "/")
		if idStr == "" {
			auth.WriteJSONError(w, "organization id required", http.StatusBadRequest)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			auth.WriteJSONError(w, "invalid organization id", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPatch:
			var req UpdateOrganizationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			name := strings.TrimSpace(req.Name)
			if name == "" {
				auth.WriteJSONError(w, "name is required", http.StatusBadRequest)
				return
			}
			org, err := s.GetOrganization(id)
			if err != nil {
				auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			org.Name = name
			if err := s.UpdateOrganization(org); err != nil {
				auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]OrganizationResponse{
				"organization": {
					ID:        org.ID.String(),
					Name:      org.Name,
					CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				},
			})
		case http.MethodDelete:
			_, err := s.GetOrganization(id)
			if err != nil {
				auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
				return
			}
			if err := s.DeleteOrganization(id); err != nil {
				if err.Error() == "organization not found" {
					auth.WriteJSONError(w, err.Error(), http.StatusNotFound)
					return
				}
				auth.WriteJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
