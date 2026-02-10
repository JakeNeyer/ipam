package handlers

import (
	"testing"
)

// TestCalculateTotalIPs tests calculateTotalIPs with table-driven cases.
func TestCalculateTotalIPs(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected int
	}{
		{
			name:     "/32 single IP",
			cidr:     "10.0.0.1/32",
			expected: 1,
		},
		{
			name:     "/24 block",
			cidr:     "10.0.0.0/24",
			expected: 256,
		},
		{
			name:     "/16 block",
			cidr:     "10.0.0.0/16",
			expected: 65536,
		},
		{
			name:     "/8 block",
			cidr:     "10.0.0.0/8",
			expected: 16777216,
		},
		{
			name:     "invalid CIDR returns zero",
			cidr:     "invalid",
			expected: 0,
		},
		{
			name:     "empty CIDR returns zero",
			cidr:     "",
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTotalIPs(tt.cidr)
			if got != tt.expected {
				t.Errorf("calculateTotalIPs(%q) = %v, want %v", tt.cidr, got, tt.expected)
			}
		})
	}
}

// TestBlockNamesMatch tests blockNamesMatch with table-driven cases.
func TestBlockNamesMatch(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "identical",
			a:        "my-block",
			b:        "my-block",
			expected: true,
		},
		{
			name:     "case insensitive",
			a:        "My-Block",
			b:        "my-block",
			expected: true,
		},
		{
			name:     "trimmed",
			a:        "  my-block  ",
			b:        "my-block",
			expected: true,
		},
		{
			name:     "different",
			a:        "block-a",
			b:        "block-b",
			expected: false,
		},
		{
			name:     "empty both",
			a:        "",
			b:        "",
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blockNamesMatch(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("blockNamesMatch(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
