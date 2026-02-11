package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JakeNeyer/ipam/server/auth"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// setupGlobalAdminTest creates an in-memory store with a global admin user, an org, and an org admin user.
// Returns (store, globalAdmin, org, orgAdmin).
func setupGlobalAdminTest(t *testing.T) (*store.Store, *store.User, *store.Organization, *store.User) {
	t.Helper()
	s := store.NewStore()

	// Global admin (no organization)
	globalAdmin := &store.User{
		Email:          "global@example.com",
		PasswordHash:   mustHashPassword("password123"),
		Role:           store.RoleAdmin,
		OrganizationID: uuid.Nil,
	}
	if err := s.CreateUser(globalAdmin); err != nil {
		t.Fatalf("create global admin: %v", err)
	}

	// Organization
	org := &store.Organization{Name: "Test Org"}
	if err := s.CreateOrganization(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Org admin (belongs to org)
	orgAdmin := &store.User{
		Email:          "orgadmin@example.com",
		PasswordHash:   mustHashPassword("password123"),
		Role:           store.RoleAdmin,
		OrganizationID: org.ID,
	}
	if err := s.CreateUser(orgAdmin); err != nil {
		t.Fatalf("create org admin: %v", err)
	}

	return s, globalAdmin, org, orgAdmin
}

func mustHashPassword(pw string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

// requestWithOrgAdmin sets the request context to have the org admin as the authenticated user.
func requestWithOrgAdmin(r *http.Request, orgAdmin *store.User) *http.Request {
	return r.WithContext(auth.WithUser(r.Context(), orgAdmin))
}

// TestGlobalAdmin_OrgAdminCannotAssignUserToGlobalAdmin proves an org admin cannot set any user's organization to Nil (global admin).
func TestGlobalAdmin_OrgAdminCannotAssignUserToGlobalAdmin(t *testing.T) {
	s, globalAdmin, _, orgAdmin := setupGlobalAdminTest(t)
	handler := UpdateUserOrganizationHandler(s)

	body := json.RawMessage(`{"organization_id":"00000000-0000-0000-0000-000000000000"}`)
	path := "/api/admin/users/" + globalAdmin.ID.String() + "/organization"
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("PATCH user org to Nil as org admin: status = %d, want 403", rr.Code)
	}
	var errBody map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &errBody)
	msg := errBody["error"]
	// Handler may return "forbidden" (endpoint is global-admin-only) or "only global admin can assign global admin"; both prove no abuse.
	if msg != "forbidden" && msg != "only global admin can assign global admin" {
		t.Errorf("PATCH user org to Nil: error = %q, want 403 with 'forbidden' or 'only global admin can assign global admin'", msg)
	}
}

// TestGlobalAdmin_OrgAdminCannotCreateUserAsGlobalAdmin proves an org admin sending organization_id Nil still creates the user in their org (body is ignored; no global admin created).
func TestGlobalAdmin_OrgAdminCannotCreateUserAsGlobalAdmin(t *testing.T) {
	s, _, org, orgAdmin := setupGlobalAdminTest(t)
	handler := AdminUsersHandler(s)

	body := map[string]interface{}{
		"email":           "newuser@example.com",
		"password":        "password123",
		"role":            "user",
		"organization_id": "00000000-0000-0000-0000-000000000000",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST create user as org admin: status = %d, want 201", rr.Code)
	}
	var out struct {
		User struct {
			ID             string `json:"id"`
			Email          string `json:"email"`
			OrganizationID string `json:"organization_id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// Created user must be in org admin's org, not global admin (Nil).
	if out.User.OrganizationID == "" || out.User.OrganizationID == "00000000-0000-0000-0000-000000000000" {
		t.Errorf("created user organization_id = %q, must be org (not global admin)", out.User.OrganizationID)
	}
	if out.User.OrganizationID != org.ID.String() {
		t.Errorf("created user organization_id = %q, want %q (caller's org)", out.User.OrganizationID, org.ID.String())
	}
}

// TestGlobalAdmin_OrgAdminCannotListOrganizations proves an org admin cannot list organizations (global admin only).
func TestGlobalAdmin_OrgAdminCannotListOrganizations(t *testing.T) {
	s, _, _, orgAdmin := setupGlobalAdminTest(t)
	handler := AdminOrganizationsHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/organizations", nil)
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("GET /api/admin/organizations as org admin: status = %d, want 403", rr.Code)
	}
}

// TestGlobalAdmin_OrgAdminCannotCreateOrganization proves an org admin cannot create an organization (global admin only).
func TestGlobalAdmin_OrgAdminCannotCreateOrganization(t *testing.T) {
	s, _, _, orgAdmin := setupGlobalAdminTest(t)
	handler := AdminOrganizationsHandler(s)

	body := []byte(`{"name":"Evil Org"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/organizations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("POST /api/admin/organizations as org admin: status = %d, want 403", rr.Code)
	}
}

// TestGlobalAdmin_OrgAdminCannotUpdateOrganization proves an org admin cannot update an organization (global admin only).
func TestGlobalAdmin_OrgAdminCannotUpdateOrganization(t *testing.T) {
	s, _, org, orgAdmin := setupGlobalAdminTest(t)
	handler := AdminOrganizationByIDHandler(s)

	body := []byte(`{"name":"Hacked Name"}`)
	path := "/api/admin/organizations/" + org.ID.String()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("PATCH /api/admin/organizations/:id as org admin: status = %d, want 403", rr.Code)
	}
}

// TestGlobalAdmin_OrgAdminCannotDeleteOrganization proves an org admin cannot delete an organization (global admin only).
func TestGlobalAdmin_OrgAdminCannotDeleteOrganization(t *testing.T) {
	s, _, org, orgAdmin := setupGlobalAdminTest(t)
	handler := AdminOrganizationByIDHandler(s)

	path := "/api/admin/organizations/" + org.ID.String()
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("DELETE /api/admin/organizations/:id as org admin: status = %d, want 403", rr.Code)
	}
}

// TestGlobalAdmin_OrgAdminCannotCallUpdateUserOrganization proves an org admin cannot call PATCH user organization (handler is global admin only).
func TestGlobalAdmin_OrgAdminCannotCallUpdateUserOrganization(t *testing.T) {
	s, globalAdmin, _, orgAdmin := setupGlobalAdminTest(t)
	handler := UpdateUserOrganizationHandler(s)

	body := []byte(`{"organization_id":"` + globalAdmin.ID.String() + `"}`)
	path := "/api/admin/users/" + globalAdmin.ID.String() + "/organization"
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = requestWithOrgAdmin(req, orgAdmin)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("PATCH /api/admin/users/:id/organization as org admin: status = %d, want 403", rr.Code)
	}
}