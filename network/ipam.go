package network

import (
	"fmt"
	"net"
)

// NextAvailableCIDR finds the next available CIDR block within the given supernet. It attempts best-effort bin-packing
// to minimize fragmentation of the supernet.
func NextAvailableCIDR(supernet string, prefixLength int) (string, error) {
	_, supernet_net, err := net.ParseCIDR(supernet)
	if err != nil {
		return "", fmt.Errorf("invalid supernet CIDR: %w", err)
	}

	supernet_prefix, supernet_bits := supernet_net.Mask.Size()
	if prefixLength < supernet_prefix {
		return "", fmt.Errorf("prefix length %d must be greater than or equal to supernet prefix length %d", prefixLength, supernet_prefix)
	}

	if prefixLength > supernet_bits {
		return "", fmt.Errorf("prefix length %d exceeds maximum for IP version", prefixLength)
	}

	// Start from the supernet address and increment until we find an available CIDR
	current := make(net.IP, len(supernet_net.IP))
	copy(current, supernet_net.IP)

	for {
		candidate := &net.IPNet{IP: current, Mask: net.CIDRMask(prefixLength, supernet_bits)}

		// Check if candidate start is within supernet
		if !supernet_net.Contains(candidate.IP) {
			break
		}

		// Calculate the last address in the candidate CIDR block
		candidate_last := make(net.IP, len(candidate.IP))
		copy(candidate_last, candidate.IP)
		for i := 0; i < len(candidate_last); i++ {
			candidate_last[i] = candidate_last[i] | (candidate.Mask[i] ^ 0xff)
		}

		// If candidate's last address is within supernet, we found a valid one
		if supernet_net.Contains(candidate_last) {
			return candidate.String(), nil
		}

		// Increment to the next candidate CIDR
		increment := uint(supernet_bits - prefixLength)
		for i := len(current) - 1; i >= 0; i-- {
			current[i] += 1 << increment
			if current[i] != 0 {
				break
			}
		}
	}

	return "", fmt.Errorf("no available CIDR blocks found in supernet")
}

// IsCIDRAvailable checks if the specified CIDR block is available within the given supernet.
func IsCIDRAvailable(supernet string, cidr string) (bool, error) {
	if !ValidateCIDR(cidr) {
		return false, fmt.Errorf("invalid CIDR: %s", cidr)
	}

	contained, err := Contains(supernet, cidr)
	if err != nil {
		return false, err
	}

	return contained, nil
}

// ValidateCIDR checks if the provided CIDR block is valid.
func ValidateCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// Overlaps checks if two CIDR blocks overlap.
func Overlaps(cidr1 string, cidr2 string) (bool, error) {
	_, net1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR1: %w", err)
	}

	_, net2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR2: %w", err)
	}

	// Two networks overlap if either contains part of the other
	return net1.Contains(net2.IP) || net2.Contains(net1.IP), nil
}

// Contains checks if the supernet contains the specified CIDR block.
func Contains(supernet string, cidr string) (bool, error) {
	_, supernet_net, err := net.ParseCIDR(supernet)
	if err != nil {
		return false, fmt.Errorf("invalid supernet CIDR: %w", err)
	}

	_, cidr_net, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR: %w", err)
	}

	// Check if supernet contains the start of the CIDR block
	if !supernet_net.Contains(cidr_net.IP) {
		return false, nil
	}

	// Calculate the last address in the CIDR block
	// For CIDR X.X.X.X/N, the last address is the broadcast address
	cidr_last := make(net.IP, len(cidr_net.IP))
	copy(cidr_last, cidr_net.IP)

	// OR the inverted mask to get the last address
	for i := 0; i < len(cidr_last); i++ {
		cidr_last[i] = cidr_last[i] | (cidr_net.Mask[i] ^ 0xff)
	}

	// Check if supernet contains the last address of the CIDR block
	return supernet_net.Contains(cidr_last), nil
}
