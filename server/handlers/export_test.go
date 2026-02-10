package handlers

import (
	"testing"
)

// TestCidrStartEnd tests cidrStartEnd with table-driven cases.
func TestCidrStartEnd(t *testing.T) {
	tests := []struct {
		name      string
		cidr      string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "single IP /32",
			cidr:      "10.0.0.1/32",
			wantStart: "10.0.0.1",
			wantEnd:   "10.0.0.1",
		},
		{
			name:      "/24 block",
			cidr:      "10.0.0.0/24",
			wantStart: "10.0.0.0",
			wantEnd:   "10.0.0.255",
		},
		{
			name:      "/16 block",
			cidr:      "10.0.0.0/16",
			wantStart: "10.0.0.0",
			wantEnd:   "10.0.255.255",
		},
		{
			name:      "invalid CIDR",
			cidr:      "invalid",
			wantStart: "",
			wantEnd:   "",
		},
		{
			name:      "empty CIDR",
			cidr:      "",
			wantStart: "",
			wantEnd:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := cidrStartEnd(tt.cidr)
			if start != tt.wantStart {
				t.Errorf("cidrStartEnd(%q) start = %q, want %q", tt.cidr, start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("cidrStartEnd(%q) end = %q, want %q", tt.cidr, end, tt.wantEnd)
			}
		})
	}
}
