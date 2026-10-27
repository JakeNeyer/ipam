package oauth

import (
	"fmt"
	"strings"

	"github.com/JakeNeyer/ipam/server/config"
	"github.com/JakeNeyer/ipam/store"
)

// OrganizationMapping is the input for JIT organization resolution.
type OrganizationMapping struct {
	Claims         map[string]any
	Email          string
	OrgClaim       string
	OrgMap         []config.ClaimMapEntry
	DefaultOrg     string
	AllowedDomains []string
	DomainOrgMap   []config.ClaimMapEntry
}

// MapRole returns the IPAM role for JIT provisioning.
// If any mapped claim value is admin, admin wins; else user if mapped; else the default role.
func MapRole(claims map[string]any, roleClaim string, roleMap []config.ClaimMapEntry, defaultRole string) (string, error) {
	def, err := normalizeRole(defaultRole)
	if err != nil {
		return "", err
	}
	values := ClaimValues(claims, roleClaim)
	if len(values) == 0 || len(roleMap) == 0 {
		return def, nil
	}
	have := make(map[string]bool, len(values))
	for _, v := range values {
		have[v] = true
	}
	sawAdmin, sawUser := false, false
	for _, entry := range roleMap {
		if !have[entry.Value] {
			continue
		}
		mapped, mapErr := normalizeRole(entry.Mapped)
		if mapErr != nil {
			continue
		}
		if mapped == store.RoleAdmin {
			sawAdmin = true
		}
		if mapped == store.RoleUser {
			sawUser = true
		}
	}
	if sawAdmin {
		return store.RoleAdmin, nil
	}
	if sawUser {
		return store.RoleUser, nil
	}
	return def, nil
}

// MapOrganization returns an organization name or UUID to resolve.
// Precedence: allowlisted email domain, then mapped org claim, then domain→org map, then default.
// A present org claim that matches nothing is denied (no domain/default fallback).
func MapOrganization(in OrganizationMapping) (string, error) {
	domain := EmailDomain(in.Email)
	if err := checkJITEmailDomain(domain, in.AllowedDomains, in.DomainOrgMap); err != nil {
		return "", err
	}

	values := ClaimValues(in.Claims, in.OrgClaim)
	if len(values) > 0 {
		if mapped := firstMappedValue(values, in.OrgMap); mapped != "" {
			return mapped, nil
		}
		return "", fmt.Errorf("no organization mapping for claim values")
	}

	if mapped := mappedDomainOrg(domain, in.DomainOrgMap); mapped != "" {
		return mapped, nil
	}
	defaultOrg := strings.TrimSpace(in.DefaultOrg)
	if defaultOrg == "" {
		return "", fmt.Errorf("JIT default organization is not configured")
	}
	return defaultOrg, nil
}

// EmailDomain returns the exact domain after the last "@", lowercased.
func EmailDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	i := strings.LastIndex(email, "@")
	if i < 0 || i == len(email)-1 {
		return ""
	}
	return config.NormalizeEmailDomain(email[i+1:])
}

func checkJITEmailDomain(domain string, allowed []string, domainOrgMap []config.ClaimMapEntry) error {
	if domain == "" {
		return fmt.Errorf("email domain is not allowed")
	}
	if len(allowed) > 0 {
		for _, d := range allowed {
			if domain == config.NormalizeEmailDomain(d) {
				return nil
			}
		}
		return fmt.Errorf("email domain is not allowed")
	}
	if len(domainOrgMap) > 0 {
		if mappedDomainOrg(domain, domainOrgMap) == "" {
			return fmt.Errorf("email domain is not allowed")
		}
	}
	return nil
}

func mappedDomainOrg(domain string, domainOrgMap []config.ClaimMapEntry) string {
	domain = config.NormalizeEmailDomain(domain)
	if domain == "" {
		return ""
	}
	for _, entry := range domainOrgMap {
		if domain == config.NormalizeEmailDomain(entry.Value) {
			if mapped := strings.TrimSpace(entry.Mapped); mapped != "" {
				return mapped
			}
		}
	}
	return ""
}

func firstMappedValue(values []string, orgMap []config.ClaimMapEntry) string {
	have := make(map[string]bool, len(values))
	for _, v := range values {
		have[v] = true
	}
	for _, entry := range orgMap {
		if have[entry.Value] && strings.TrimSpace(entry.Mapped) != "" {
			return strings.TrimSpace(entry.Mapped)
		}
	}
	return ""
}

func normalizeRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return store.RoleUser, nil
	}
	if role == store.RoleUser || role == store.RoleAdmin {
		return role, nil
	}
	return "", fmt.Errorf("invalid role %q", role)
}
