package network

import (
	"bytes"
	"fmt"
	"net"
	"sort"
)

// ipRange holds the first and last IP of a CIDR (inclusive).
type ipRange struct {
	first, last net.IP
}

func cidrToRange(cidr string) (ipRange, error) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return ipRange{}, err
	}
	first := make(net.IP, len(n.IP))
	copy(first, n.IP)
	last := make(net.IP, len(n.IP))
	copy(last, n.IP)
	for i := range last {
		last[i] = last[i] | (n.Mask[i] ^ 0xff)
	}
	return ipRange{first: first, last: last}, nil
}

// ipLess returns true if a < b (lexicographic order, IPv4).
func ipLess(a, b net.IP) bool {
	return bytes.Compare(a.To4(), b.To4()) < 0
}

// nextAligned returns the smallest network address for prefixLen that is >= ip.
// For IPv4 /p, the host part is the lower (32-p) bits; we round up ip to the next boundary.
func nextAligned(ip net.IP, prefixLen int) net.IP {
	ip = ip.To4()
	if ip == nil || prefixLen <= 0 || prefixLen > 32 {
		return ip
	}
	hostBits := uint(32 - prefixLen)
	blockSize := uint32(1 << hostBits)
	var v uint32
	for i := 0; i < 4; i++ {
		v = v<<8 | uint32(ip[i])
	}
	// round up to next multiple of blockSize
	rem := v % blockSize
	if rem != 0 {
		v += blockSize - rem
	}
	out := make(net.IP, 4)
	for i := 3; i >= 0; i-- {
		out[i] = byte(v)
		v >>= 8
	}
	return out
}

// NextAvailableCIDRWithAllocations returns a suggested CIDR of the given prefix length
// within the supernet, considering existing allocations (IPv4 only). It bin-packs: first
// tries to fill a gap between allocations; if no gap fits, returns the next available CIDR
// after the last allocation. If allocatedCIDRs is empty, delegates to NextAvailableCIDR.
func NextAvailableCIDRWithAllocations(supernet string, prefixLength int, allocatedCIDRs []string) (string, error) {
	if len(allocatedCIDRs) == 0 {
		return NextAvailableCIDR(supernet, prefixLength)
	}
	_, supernetNet, err := net.ParseCIDR(supernet)
	if err != nil {
		return "", fmt.Errorf("invalid supernet CIDR: %w", err)
	}
	if supernetNet.IP.To4() == nil {
		return "", fmt.Errorf("suggest with allocations is IPv4 only")
	}
	supernetRange, err := cidrToRange(supernet)
	if err != nil {
		return "", err
	}
	supernetPrefix, bits := supernetNet.Mask.Size()
	if prefixLength < supernetPrefix {
		return "", fmt.Errorf("prefix length %d must be greater than supernet prefix %d", prefixLength, supernetPrefix)
	}
	if prefixLength > bits {
		return "", fmt.Errorf("prefix length %d exceeds maximum", prefixLength)
	}

	// Collect allocated ranges that lie inside the supernet
	var ranges []ipRange
	for _, c := range allocatedCIDRs {
		if c == "" {
			continue
		}
		contained, err := Contains(supernet, c)
		if err != nil || !contained {
			continue
		}
		r, err := cidrToRange(c)
		if err != nil {
			continue
		}
		ranges = append(ranges, r)
	}
	// Sort by first IP
	sort.Slice(ranges, func(i, j int) bool { return ipLess(ranges[i].first, ranges[j].first) })

	// Build gaps: [supernetFirst, r0.first-1], [r0.last+1, r1.first-1], ...
	type gap struct{ first, last net.IP }
	var gaps []gap
	cur := make(net.IP, len(supernetRange.first))
	copy(cur, supernetRange.first)
	for _, r := range ranges {
		if ipLess(cur, r.first) {
			gaps = append(gaps, gap{first: dupIP(cur), last: prevIP(r.first)})
		}
		// move cur past this range
		cur = nextIP(r.last)
		if ipLess(supernetRange.last, cur) {
			cur = nil
			break
		}
	}
	if cur != nil && !ipLess(supernetRange.last, cur) {
		gaps = append(gaps, gap{first: dupIP(cur), last: dupIP(supernetRange.last)})
	}

	// Subnet size in addresses
	subnetSize := uint32(1 << (bits - prefixLength))

	// Try each gap: first prefix-aligned address in gap, check if full subnet fits
	for _, g := range gaps {
		start := nextAligned(g.first, prefixLength)
		if ipLess(g.last, start) {
			continue
		}
		// end of subnet starting at start
		endU32 := ipToU32(start) + subnetSize - 1
		endIP := u32ToIP(endU32)
		if ipLess(g.last, endIP) {
			continue // subnet doesn't fit in this gap
		}
		if !supernetNet.Contains(endIP) {
			continue
		}
		return fmt.Sprintf("%s/%d", start.String(), prefixLength), nil
	}

	// No gap fits: use next available after last allocation (original behavior from block start or after last alloc)
	var start net.IP
	if len(ranges) == 0 {
		start = supernetRange.first
	} else {
		start = nextIP(ranges[len(ranges)-1].last)
	}
	if ipLess(supernetRange.last, start) {
		return "", fmt.Errorf("no space left in block")
	}
	aligned := nextAligned(start, prefixLength)
	if !supernetNet.Contains(aligned) {
		return "", fmt.Errorf("no available CIDR in block")
	}
	endU32 := ipToU32(aligned) + subnetSize - 1
	if !supernetNet.Contains(u32ToIP(endU32)) {
		return "", fmt.Errorf("no available CIDR in block")
	}
	return fmt.Sprintf("%s/%d", aligned.String(), prefixLength), nil
}

func dupIP(ip net.IP) net.IP {
	d := make(net.IP, len(ip))
	copy(d, ip)
	return d
}

func prevIP(ip net.IP) net.IP {
	ip = ip.To4()
	if ip == nil {
		return ip
	}
	v := ipToU32(ip)
	if v == 0 {
		return ip
	}
	return u32ToIP(v - 1)
}

func nextIP(ip net.IP) net.IP {
	ip = ip.To4()
	if ip == nil {
		return ip
	}
	return u32ToIP(ipToU32(ip) + 1)
}

func ipToU32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func u32ToIP(v uint32) net.IP {
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

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
