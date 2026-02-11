package validation

import (
	"regexp"
	"strings"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
	MaxEmailLength    = 254
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail returns true if email is non-empty, within length, and matches a simple format.
func ValidateEmail(email string) bool {
	s := strings.TrimSpace(strings.ToLower(email))
	if s == "" || len(s) > MaxEmailLength {
		return false
	}
	return emailRegex.MatchString(s)
}

// ValidatePassword returns true if password meets minimum length and is within bcrypt max.
func ValidatePassword(password string) bool {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return false
	}
	return true
}
