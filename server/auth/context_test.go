package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
)

func TestWithUser_UserFromContext_UserIDFromContext(t *testing.T) {
	ctx := context.Background()
	u := &store.User{ID: uuid.New(), Email: "u@example.com", Role: store.RoleUser}
	ctx = WithUser(ctx, u)
	got := UserFromContext(ctx)
	if got != u {
		t.Errorf("UserFromContext() = %p, want %p", got, u)
	}
	if id := UserIDFromContext(ctx); id != u.ID {
		t.Errorf("UserIDFromContext() = %v, want %v", id, u.ID)
	}
	// nil context
	if UserFromContext(context.Background()) != nil {
		t.Error("UserFromContext(empty) expected nil")
	}
	if UserIDFromContext(context.Background()) != uuid.Nil {
		t.Error("UserIDFromContext(empty) expected uuid.Nil")
	}
}

func TestWithEffectiveOrganization_EffectiveOrganizationID(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	ctx = WithEffectiveOrganization(ctx, orgID)
	got := EffectiveOrganizationID(ctx)
	if got != orgID {
		t.Errorf("EffectiveOrganizationID() = %v, want %v", got, orgID)
	}
	if EffectiveOrganizationID(context.Background()) != uuid.Nil {
		t.Error("EffectiveOrganizationID(empty) expected uuid.Nil")
	}
}

func TestResolveOrgID(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	tests := []struct {
		name       string
		ctx        context.Context
		user       *store.User
		inputOrgID uuid.UUID
		wantNil    bool
		wantOrg    uuid.UUID
	}{
		{"effective org set", WithEffectiveOrganization(context.Background(), orgA), &store.User{OrganizationID: orgB}, uuid.Nil, false, orgA},
		{"nil user", context.Background(), nil, uuid.Nil, true, uuid.Nil},
		{"user not global admin", context.Background(), &store.User{OrganizationID: orgA}, uuid.Nil, false, orgA},
		{"global admin with input org", context.Background(), &store.User{OrganizationID: uuid.Nil}, orgB, false, orgB},
		{"global admin no input org", context.Background(), &store.User{OrganizationID: uuid.Nil}, uuid.Nil, true, uuid.Nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOrgID(tt.ctx, tt.user, tt.inputOrgID)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ResolveOrgID() = %v, want nil", got)
				}
				return
			}
			if got == nil || *got != tt.wantOrg {
				t.Errorf("ResolveOrgID() = %v, want %v", got, &tt.wantOrg)
			}
		})
	}
}

func TestUserOrgForAccess(t *testing.T) {
	orgA := uuid.New()
	ctxEff := WithEffectiveOrganization(context.Background(), orgA)
	orgB := uuid.New()
	tests := []struct {
		name string
		ctx  context.Context
		user *store.User
		want uuid.UUID
	}{
		{"effective org set", ctxEff, &store.User{OrganizationID: orgB}, orgA},
		{"nil user", context.Background(), nil, uuid.Nil},
		{"user org", context.Background(), &store.User{OrganizationID: orgA}, orgA},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserOrgForAccess(tt.ctx, tt.user)
			if got != tt.want {
				t.Errorf("UserOrgForAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsGlobalAdmin(t *testing.T) {
	if IsGlobalAdmin(nil) {
		t.Error("IsGlobalAdmin(nil) expected false")
	}
	if !IsGlobalAdmin(&store.User{OrganizationID: uuid.Nil}) {
		t.Error("IsGlobalAdmin(org=Nil) expected true")
	}
	if IsGlobalAdmin(&store.User{OrganizationID: uuid.New()}) {
		t.Error("IsGlobalAdmin(org=set) expected false")
	}
}

func TestRequireGlobalAdminForNilOrg(t *testing.T) {
	admin := &store.User{OrganizationID: uuid.Nil}
	user := &store.User{OrganizationID: uuid.New()}
	if err := RequireGlobalAdminForNilOrg(admin, uuid.Nil); err != nil {
		t.Errorf("RequireGlobalAdminForNilOrg(admin, Nil) = %v", err)
	}
	if err := RequireGlobalAdminForNilOrg(admin, uuid.New()); err != nil {
		t.Errorf("RequireGlobalAdminForNilOrg(admin, set) = %v", err)
	}
	if err := RequireGlobalAdminForNilOrg(user, uuid.Nil); err == nil {
		t.Error("RequireGlobalAdminForNilOrg(user, Nil) expected error")
	}
	if err := RequireGlobalAdminForNilOrg(user, uuid.New()); err != nil {
		t.Errorf("RequireGlobalAdminForNilOrg(user, set) = %v", err)
	}
	if err := RequireGlobalAdminForNilOrg(nil, uuid.Nil); err == nil {
		t.Error("RequireGlobalAdminForNilOrg(nil, Nil) expected error")
	}
}

func TestWithRequest_RequestFromContext(t *testing.T) {
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/api/foo", nil)
	ctx = WithRequest(ctx, req)
	got := RequestFromContext(ctx)
	if got != req {
		t.Errorf("RequestFromContext() = %p, want %p", got, req)
	}
	if RequestFromContext(context.Background()) != nil {
		t.Error("RequestFromContext(empty) expected nil")
	}
}
