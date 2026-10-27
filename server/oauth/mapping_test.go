package oauth

import (
	"testing"

	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/store"
)

func TestMapRole(t *testing.T) {
	roleMap := []config.ClaimMapEntry{
		{Value: "ipam-admins", Mapped: "admin"},
		{Value: "ipam-users", Mapped: "user"},
	}
	claims := map[string]any{"groups": []any{"ipam-users", "ipam-admins"}}
	got, err := MapRole(claims, "groups", roleMap, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != store.RoleAdmin {
		t.Errorf("role = %q, want admin (highest privilege)", got)
	}

	got, err = MapRole(map[string]any{"groups": []any{"ipam-users"}}, "groups", roleMap, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got != store.RoleUser {
		t.Errorf("role = %q, want user", got)
	}

	got, err = MapRole(map[string]any{}, "groups", roleMap, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != store.RoleUser {
		t.Errorf("default role = %q, want user", got)
	}

	_, err = MapRole(map[string]any{}, "groups", roleMap, "superadmin")
	if err == nil {
		t.Fatal("expected error for invalid default role")
	}
}

func TestMapOrganization(t *testing.T) {
	orgMap := []config.ClaimMapEntry{
		{Value: "team-network", Mapped: "Network"},
		{Value: "team-platform", Mapped: "Platform"},
	}
	t.Run("first mapped value in ORG_MAP wins", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Claims:     map[string]any{"groups": []any{"team-platform", "team-network"}},
			Email:      "jane@acme.com",
			OrgClaim:   "groups",
			OrgMap:     orgMap,
			DefaultOrg: "Acme",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Network" {
			t.Errorf("org = %q, want Network (ORG_MAP order)", got)
		}
	})
	t.Run("missing claim uses default", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Email:      "jane@acme.com",
			OrgClaim:   "groups",
			OrgMap:     orgMap,
			DefaultOrg: "Acme",
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Acme" {
			t.Errorf("org = %q, want Acme", got)
		}
	})
	t.Run("unmapped claim values denied", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Claims:     map[string]any{"groups": []any{"unknown"}},
			Email:      "jane@acme.com",
			OrgClaim:   "groups",
			OrgMap:     orgMap,
			DefaultOrg: "Acme",
		})
		if err == nil {
			t.Fatal("expected error for unmapped claim")
		}
	})
	t.Run("missing default", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Email:    "jane@acme.com",
			OrgClaim: "groups",
			OrgMap:   orgMap,
		})
		if err == nil {
			t.Fatal("expected error when default org is empty")
		}
	})
}

func TestMapOrganization_EmailDomain(t *testing.T) {
	domainMap := []config.ClaimMapEntry{
		{Value: "acme.com", Mapped: "Acme"},
		{Value: "platform.io", Mapped: "Platform"},
	}
	orgMap := []config.ClaimMapEntry{{Value: "team-network", Mapped: "Network"}}

	t.Run("domain map selects org when claim missing", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Email:        "Jane@Acme.com",
			DomainOrgMap: domainMap,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Acme" {
			t.Errorf("org = %q, want Acme", got)
		}
	})
	t.Run("unlisted domain denied when domain map is allowlist", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Email:        "bob@gmail.com",
			DefaultOrg:   "Acme",
			DomainOrgMap: domainMap,
		})
		if err == nil {
			t.Fatal("expected gmail to be denied")
		}
	})
	t.Run("allowlist denies unlisted domain", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Email:          "bob@gmail.com",
			DefaultOrg:     "Acme",
			AllowedDomains: []string{"acme.com"},
		})
		if err == nil {
			t.Fatal("expected gmail to be denied")
		}
	})
	t.Run("allowlist permits domain and uses default", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Email:          "jane@acme.com",
			DefaultOrg:     "Acme",
			AllowedDomains: []string{"acme.com"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Acme" {
			t.Errorf("org = %q, want Acme", got)
		}
	})
	t.Run("claim mapping wins over domain", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Claims:       map[string]any{"groups": []any{"team-network"}},
			Email:        "jane@acme.com",
			OrgClaim:     "groups",
			OrgMap:       orgMap,
			DefaultOrg:   "Acme",
			DomainOrgMap: domainMap,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Network" {
			t.Errorf("org = %q, want Network from claim", got)
		}
	})
	t.Run("unmapped claim denied even if domain maps", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Claims:       map[string]any{"groups": []any{"unknown"}},
			Email:        "jane@acme.com",
			OrgClaim:     "groups",
			OrgMap:       orgMap,
			DefaultOrg:   "Acme",
			DomainOrgMap: domainMap,
		})
		if err == nil {
			t.Fatal("expected unmapped claim to deny")
		}
	})
	t.Run("evil suffix is not acme.com", func(t *testing.T) {
		_, err := MapOrganization(OrganizationMapping{
			Email:          "eve@evil-acme.com",
			DefaultOrg:     "Acme",
			AllowedDomains: []string{"acme.com"},
		})
		if err == nil {
			t.Fatal("expected suffix lookalike to be denied")
		}
	})
	t.Run("allowlist and domain map: listed domain uses default if not in map", func(t *testing.T) {
		got, err := MapOrganization(OrganizationMapping{
			Email:          "pat@contractor.com",
			DefaultOrg:     "Vendors",
			AllowedDomains: []string{"acme.com", "contractor.com"},
			DomainOrgMap:   domainMap,
		})
		if err != nil {
			t.Fatal(err)
		}
		if got != "Vendors" {
			t.Errorf("org = %q, want Vendors", got)
		}
	})
}

func TestEmailDomain(t *testing.T) {
	if got := EmailDomain("Jane@Acme.com"); got != "acme.com" {
		t.Errorf("EmailDomain = %q", got)
	}
	if got := EmailDomain("a@b@c.com"); got != "c.com" {
		t.Errorf("EmailDomain last @ = %q", got)
	}
	if EmailDomain("nodomain") != "" {
		t.Error("expected empty domain")
	}
}
