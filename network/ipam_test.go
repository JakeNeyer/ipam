package network

import (
	"testing"
)

// TestValidateCIDR tests the ValidateCIDR function
func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected bool
	}{
		{
			name:     "valid IPv4 CIDR",
			cidr:     "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "valid IPv4 CIDR with /32",
			cidr:     "192.168.1.1/32",
			expected: true,
		},
		{
			name:     "valid IPv4 CIDR with /16",
			cidr:     "10.0.0.0/16",
			expected: true,
		},
		{
			name:     "valid IPv6 CIDR",
			cidr:     "2001:db8::/32",
			expected: true,
		},
		{
			name:     "invalid CIDR missing prefix",
			cidr:     "192.168.1.0",
			expected: false,
		},
		{
			name:     "invalid CIDR malformed",
			cidr:     "invalid",
			expected: false,
		},
		{
			name:     "invalid CIDR empty string",
			cidr:     "",
			expected: false,
		},
		{
			name:     "invalid CIDR prefix too large",
			cidr:     "192.168.1.0/33",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCIDR(tt.cidr)
			if result != tt.expected {
				t.Errorf("ValidateCIDR(%s) = %v, want %v", tt.cidr, result, tt.expected)
			}
		})
	}
}

// TestContains tests the Contains function
func TestContains(t *testing.T) {
	tests := []struct {
		name      string
		supernet  string
		cidr      string
		expected  bool
		expectErr bool
	}{
		{
			name:      "supernet contains exact CIDR",
			supernet:  "10.0.0.0/16",
			cidr:      "10.0.0.0/16",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "supernet contains smaller subnet",
			supernet:  "10.0.0.0/16",
			cidr:      "10.0.1.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "supernet contains single IP",
			supernet:  "10.0.0.0/16",
			cidr:      "10.0.1.5/32",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "supernet does not contain CIDR",
			supernet:  "10.0.0.0/16",
			cidr:      "10.1.0.0/16",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "supernet does not contain partially overlapping CIDR",
			supernet:  "10.0.0.0/16",
			cidr:      "10.0.128.0/15",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "invalid supernet",
			supernet:  "invalid",
			cidr:      "10.0.0.0/24",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "invalid CIDR",
			supernet:  "10.0.0.0/16",
			cidr:      "invalid",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "IPv6 supernet contains IPv6 CIDR",
			supernet:  "2001:db8::/32",
			cidr:      "2001:db8:1::/48",
			expected:  true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Contains(tt.supernet, tt.cidr)
			if (err != nil) != tt.expectErr {
				t.Errorf("Contains(%s, %s) error = %v, expectErr %v", tt.supernet, tt.cidr, err, tt.expectErr)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("Contains(%s, %s) = %v, want %v", tt.supernet, tt.cidr, result, tt.expected)
			}
		})
	}
}

// TestOverlaps tests the Overlaps function
func TestOverlaps(t *testing.T) {
	tests := []struct {
		name      string
		cidr1     string
		cidr2     string
		expected  bool
		expectErr bool
	}{
		{
			name:      "identical CIDRs overlap",
			cidr1:     "10.0.0.0/24",
			cidr2:     "10.0.0.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "supernet and subnet overlap",
			cidr1:     "10.0.0.0/16",
			cidr2:     "10.0.1.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "adjacent subnets do not overlap",
			cidr1:     "10.0.0.0/24",
			cidr2:     "10.0.1.0/24",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "overlapping subnets",
			cidr1:     "10.0.0.0/23",
			cidr2:     "10.0.1.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "non-overlapping separate networks",
			cidr1:     "10.0.0.0/16",
			cidr2:     "192.168.0.0/16",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "invalid first CIDR",
			cidr1:     "invalid",
			cidr2:     "10.0.0.0/24",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "invalid second CIDR",
			cidr1:     "10.0.0.0/24",
			cidr2:     "invalid",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "IPv6 overlapping networks",
			cidr1:     "2001:db8::/32",
			cidr2:     "2001:db8:1::/48",
			expected:  true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Overlaps(tt.cidr1, tt.cidr2)
			if (err != nil) != tt.expectErr {
				t.Errorf("Overlaps(%s, %s) error = %v, expectErr %v", tt.cidr1, tt.cidr2, err, tt.expectErr)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("Overlaps(%s, %s) = %v, want %v", tt.cidr1, tt.cidr2, result, tt.expected)
			}
		})
	}
}

// TestIsCIDRAvailable tests the IsCIDRAvailable function
func TestIsCIDRAvailable(t *testing.T) {
	tests := []struct {
		name      string
		supernet  string
		cidr      string
		expected  bool
		expectErr bool
	}{
		{
			name:      "CIDR is available in supernet",
			supernet:  "10.0.0.0/16",
			cidr:      "10.0.1.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "CIDR is exact match with supernet",
			supernet:  "10.0.0.0/24",
			cidr:      "10.0.0.0/24",
			expected:  true,
			expectErr: false,
		},
		{
			name:      "CIDR is not available (outside supernet)",
			supernet:  "10.0.0.0/16",
			cidr:      "10.1.0.0/16",
			expected:  false,
			expectErr: false,
		},
		{
			name:      "invalid CIDR",
			supernet:  "10.0.0.0/16",
			cidr:      "invalid",
			expected:  false,
			expectErr: true,
		},
		{
			name:      "invalid supernet",
			supernet:  "invalid",
			cidr:      "10.0.0.0/24",
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsCIDRAvailable(tt.supernet, tt.cidr)
			if (err != nil) != tt.expectErr {
				t.Errorf("IsCIDRAvailable(%s, %s) error = %v, expectErr %v", tt.supernet, tt.cidr, err, tt.expectErr)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("IsCIDRAvailable(%s, %s) = %v, want %v", tt.supernet, tt.cidr, result, tt.expected)
			}
		})
	}
}

// TestNextAvailableCIDR tests the NextAvailableCIDR function
func TestNextAvailableCIDR(t *testing.T) {
	tests := []struct {
		name             string
		supernet         string
		prefixLength     int
		expectedContains bool
		expectErr        bool
	}{
		{
			name:             "find next available /24 in /16",
			supernet:         "10.0.0.0/16",
			prefixLength:     24,
			expectedContains: true,
			expectErr:        false,
		},
		{
			name:             "find next available /26 in /24",
			supernet:         "10.0.0.0/24",
			prefixLength:     26,
			expectedContains: true,
			expectErr:        false,
		},
		{
			name:             "invalid supernet",
			supernet:         "invalid",
			prefixLength:     24,
			expectedContains: false,
			expectErr:        true,
		},
		{
			name:             "prefix length smaller than supernet",
			supernet:         "10.0.0.0/16",
			prefixLength:     8,
			expectedContains: false,
			expectErr:        true,
		},
		{
			name:             "prefix length too large",
			supernet:         "10.0.0.0/16",
			prefixLength:     33,
			expectedContains: false,
			expectErr:        true,
		},
		{
			name:             "find next available IPv6 /64 in /32",
			supernet:         "2001:db8::/32",
			prefixLength:     64,
			expectedContains: true,
			expectErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NextAvailableCIDR(tt.supernet, tt.prefixLength)
			if (err != nil) != tt.expectErr {
				t.Errorf("NextAvailableCIDR(%s, %d) error = %v, expectErr %v", tt.supernet, tt.prefixLength, err, tt.expectErr)
			}
			if !tt.expectErr {
				if !ValidateCIDR(result) {
					t.Errorf("NextAvailableCIDR(%s, %d) returned invalid CIDR: %s", tt.supernet, tt.prefixLength, result)
				}
				contained, _ := Contains(tt.supernet, result)
				if tt.expectedContains && !contained {
					t.Errorf("NextAvailableCIDR(%s, %d) = %s, but not contained in supernet", tt.supernet, tt.prefixLength, result)
				}
			}
		})
	}
}

func TestNextAvailableCIDRWithAllocations(t *testing.T) {
	tests := []struct {
		name          string
		supernet      string
		prefixLength  int
		allocated     []string
		expectedCIDR  string
		expectErr     bool
		expectInSuper bool
	}{
		{
			name:          "fill gap between two allocations",
			supernet:      "10.0.0.0/16",
			prefixLength:  24,
			allocated:     []string{"10.0.0.0/24", "10.0.2.0/24"},
			expectedCIDR:  "10.0.1.0/24",
			expectErr:     false,
			expectInSuper: true,
		},
		{
			name:          "next after last allocation when no gap fits",
			supernet:      "10.0.0.0/16",
			prefixLength:  24,
			allocated:     []string{"10.0.0.0/24", "10.0.1.0/24"},
			expectedCIDR:  "10.0.2.0/24",
			expectErr:     false,
			expectInSuper: true,
		},
		{
			name:          "first /24 when no allocations",
			supernet:      "10.0.0.0/16",
			prefixLength:  24,
			allocated:     nil,
			expectedCIDR:  "10.0.0.0/24",
			expectErr:     false,
			expectInSuper: true,
		},
		{
			name:          "fill gap at start",
			supernet:      "10.0.0.0/16",
			prefixLength:  24,
			allocated:     []string{"10.0.1.0/24", "10.0.2.0/24"},
			expectedCIDR:  "10.0.0.0/24",
			expectErr:     false,
			expectInSuper: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NextAvailableCIDRWithAllocations(tt.supernet, tt.prefixLength, tt.allocated)
			if (err != nil) != tt.expectErr {
				t.Errorf("NextAvailableCIDRWithAllocations error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.expectErr {
				return
			}
			if result != tt.expectedCIDR {
				t.Errorf("NextAvailableCIDRWithAllocations = %s, want %s", result, tt.expectedCIDR)
			}
			if tt.expectInSuper {
				contained, _ := Contains(tt.supernet, result)
				if !contained {
					t.Errorf("result %s not contained in supernet %s", result, tt.supernet)
				}
			}
		})
	}
}
