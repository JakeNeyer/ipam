package validation

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"valid simple", "user@example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"valid with dots", "user.name@example.co.uk", true},
		{"valid lowercase", "USER@EXAMPLE.COM", true},
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"trimmed empty", "  \t  ", false},
		{"no at", "userexample.com", false},
		{"no domain", "user@", false},
		{"no local", "@example.com", false},
		{"double at", "user@@example.com", false},
		{"too long", "a@" + strings.Repeat("x", MaxEmailLength) + ".com", false},
		{"at max length", "a@" + strings.Repeat("x", MaxEmailLength-6) + ".com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			if got != tt.expected {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.expected)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"valid min length", strings.Repeat("a", MinPasswordLength), true},
		{"valid max length", strings.Repeat("a", MaxPasswordLength), true},
		{"too short", strings.Repeat("a", MinPasswordLength-1), false},
		{"empty", "", false},
		{"too long", strings.Repeat("a", MaxPasswordLength+1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePassword(tt.password)
			if got != tt.expected {
				t.Errorf("ValidatePassword(len=%d) = %v, want %v", len(tt.password), got, tt.expected)
			}
		})
	}
}
