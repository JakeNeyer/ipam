package oauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
)

// UserProfile is identity plus the claim map from userinfo (and, after MergeTokenClaims, the ID token).
type UserProfile struct {
	ProviderUserID string
	Email          string
	Claims         map[string]any
}

// MergeTokenClaims overlays ID-token claims on userinfo. The token is the one returned by the
// configured token endpoint after a confidential-client code exchange; the JWT signature is not re-verified.
func MergeTokenClaims(userinfo map[string]any, token *oauth2.Token) map[string]any {
	out := copyClaims(userinfo)
	raw := idTokenFromOAuthToken(token)
	if raw == "" {
		return out
	}
	idClaims, err := DecodeIDTokenClaims(raw)
	if err != nil || len(idClaims) == 0 {
		return out
	}
	for k, v := range idClaims {
		out[k] = v
	}
	return out
}

func idTokenFromOAuthToken(token *oauth2.Token) string {
	if token == nil {
		return ""
	}
	v, ok := token.Extra("id_token").(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// DecodeIDTokenClaims base64-decodes the JWT payload. Signature verification is intentionally skipped.
func DecodeIDTokenClaims(idToken string) (map[string]any, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, nil
	}
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 || parts[1] == "" {
		return nil, fmt.Errorf("invalid id_token")
	}
	payload, err := decodeJWTSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid id_token payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid id_token JSON: %w", err)
	}
	return claims, nil
}

func decodeJWTSegment(seg string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(seg); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(seg)
}

func copyClaims(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ClaimValues returns string values at a dotted JSON path. Missing paths yield nil.
// Strings, string arrays, and numeric scalars are collected; objects are ignored.
func ClaimValues(claims map[string]any, path string) []string {
	path = strings.TrimSpace(path)
	if path == "" || claims == nil {
		return nil
	}
	v, ok := claimAtPath(claims, path)
	if !ok || v == nil {
		return nil
	}
	return flattenClaimValues(v)
}

func claimAtPath(claims map[string]any, path string) (any, bool) {
	cur := any(claims)
	for _, key := range strings.Split(path, ".") {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := m[key]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func flattenClaimValues(v any) []string {
	switch x := v.(type) {
	case []any:
		var out []string
		for _, item := range x {
			if s := scalarClaimString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		var out []string
		for _, item := range x {
			if s := strings.TrimSpace(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		if s := scalarClaimString(x); s != "" {
			return []string{s}
		}
		return nil
	}
}

func scalarClaimString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case json.Number:
		return strings.TrimSpace(x.String())
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	case bool:
		if x {
			return "true"
		}
		return "false"
	case map[string]any:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", x))
	}
}
