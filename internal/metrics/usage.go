package metrics

import (
	"math/big"
	"net"
	"strings"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

func ipFamily(cidr string) string {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "unknown"
	}
	if ip.To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}

func providerLabel(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "native"
	}
	return p
}

func bigToFloat(n *big.Int) float64 {
	if n == nil {
		return 0
	}
	f, _ := new(big.Float).SetInt(n).Float64()
	return f
}

func cidrCount(cidr string) *big.Int {
	n, err := network.CIDRAddressCount(cidr)
	if err != nil || n == nil {
		return big.NewInt(0)
	}
	return n
}

func availableIPs(total, used *big.Int) *big.Int {
	if total == nil {
		total = big.NewInt(0)
	}
	if used == nil {
		used = big.NewInt(0)
	}
	avail := new(big.Int).Sub(total, used)
	if avail.Sign() < 0 {
		return big.NewInt(0)
	}
	return avail
}

func utilizationRatio(used, total *big.Int) float64 {
	if total == nil || total.Sign() <= 0 {
		return 0
	}
	if used == nil || used.Sign() <= 0 {
		return 0
	}
	tf, _ := new(big.Float).SetInt(total).Float64()
	uf, _ := new(big.Float).SetInt(used).Float64()
	if tf <= 0 {
		return 0
	}
	r := uf / tf
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// attributeAllocations maps each allocation to its parent block and sums the
// allocated address space per block.
//
// Allocations reference their block by name only (network.Allocation.Block has
// no ID), so matching is by case-insensitive name, the same rule the REST API's
// derivedBlockUsage applies. The API usually scopes that lookup to one
// organization; here the walk is global, so when several blocks share a name
// (for example across organizations) the allocation is attributed to the
// smallest same-named block whose CIDR contains it. Allocations that cannot be
// attributed are returned in unmatched so callers can still count them.
func attributeAllocations(blocks []*network.Block, allocs []*network.Allocation) (used map[uuid.UUID]*big.Int, counts map[uuid.UUID]int, unmatched []*network.Allocation) {
	used = make(map[uuid.UUID]*big.Int)
	counts = make(map[uuid.UUID]int)
	byName := make(map[string][]*network.Block)
	for _, b := range blocks {
		if b == nil {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(b.Name))
		byName[k] = append(byName[k], b)
	}
	for _, a := range allocs {
		if a == nil {
			continue
		}
		parent := matchAllocationBlock(byName[strings.ToLower(strings.TrimSpace(a.Block.Name))], a.Block.CIDR)
		if parent == nil {
			unmatched = append(unmatched, a)
			continue
		}
		c := cidrCount(a.Block.CIDR)
		if used[parent.ID] == nil {
			used[parent.ID] = new(big.Int)
		}
		used[parent.ID].Add(used[parent.ID], c)
		counts[parent.ID]++
	}
	return used, counts, unmatched
}

func matchAllocationBlock(candidates []*network.Block, allocCIDR string) *network.Block {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	var best *network.Block
	var bestSize *big.Int
	for _, b := range candidates {
		ok, err := network.Contains(b.CIDR, allocCIDR)
		if err != nil || !ok {
			continue
		}
		size := cidrCount(b.CIDR)
		if best == nil || (size != nil && bestSize != nil && size.Cmp(bestSize) < 0) {
			best = b
			bestSize = size
		}
	}
	return best
}

// poolUsedByID returns, for every pool, the address space carved into its
// direct child pools and blocks. One pass over pools and blocks (O(P+B)).
// Pools with nothing carved out are absent from the map.
func poolUsedByID(pools []*network.Pool, blocks []*network.Block) map[uuid.UUID]*big.Int {
	used := make(map[uuid.UUID]*big.Int, len(pools))
	add := func(parent uuid.UUID, cidr string) {
		if used[parent] == nil {
			used[parent] = new(big.Int)
		}
		used[parent].Add(used[parent], cidrCount(cidr))
	}
	for _, p := range pools {
		if p == nil || p.ParentPoolID == nil || *p.ParentPoolID == uuid.Nil {
			continue
		}
		add(*p.ParentPoolID, p.CIDR)
	}
	for _, b := range blocks {
		if b == nil || b.PoolID == nil || *b.PoolID == uuid.Nil {
			continue
		}
		add(*b.PoolID, b.CIDR)
	}
	return used
}
