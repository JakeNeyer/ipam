package metrics

import (
	"math"
	"math/big"
	"testing"

	"github.com/JakeNeyer/ipam/network"
	"github.com/google/uuid"
)

func TestIPFamily(t *testing.T) {
	tests := []struct {
		cidr string
		want string
	}{
		{"10.0.0.0/24", "ipv4"},
		{"2001:db8::/64", "ipv6"},
		{"not-a-cidr", "unknown"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		if got := ipFamily(tt.cidr); got != tt.want {
			t.Errorf("ipFamily(%q) = %q, want %q", tt.cidr, got, tt.want)
		}
	}
}

func TestUtilizationRatio(t *testing.T) {
	tests := []struct {
		name string
		used int64
		tot  int64
		want float64
	}{
		{"empty", 0, 256, 0},
		{"half", 128, 256, 0.5},
		{"full", 256, 256, 1},
		{"overfull clamped", 300, 256, 1},
		{"zero total", 10, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utilizationRatio(big.NewInt(tt.used), big.NewInt(tt.tot))
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("utilizationRatio(%d,%d) = %v, want %v", tt.used, tt.tot, got, tt.want)
			}
		})
	}
}

func TestAvailableIPs(t *testing.T) {
	got := availableIPs(big.NewInt(256), big.NewInt(64))
	if got.Cmp(big.NewInt(192)) != 0 {
		t.Errorf("available = %s, want 192", got)
	}
	got = availableIPs(big.NewInt(10), big.NewInt(20))
	if got.Sign() != 0 {
		t.Errorf("over-used available = %s, want 0", got)
	}
}

func TestAttributeAllocations_nameAndContainment(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	blockA := &network.Block{ID: uuid.New(), Name: "app", CIDR: "10.0.0.0/16", OrganizationID: orgA}
	blockB := &network.Block{ID: uuid.New(), Name: "app", CIDR: "10.1.0.0/16", OrganizationID: orgB}
	allocA := &network.Allocation{Id: uuid.New(), Name: "a", Block: network.Block{Name: "app", CIDR: "10.0.1.0/24"}}
	allocB := &network.Allocation{Id: uuid.New(), Name: "b", Block: network.Block{Name: "app", CIDR: "10.1.1.0/24"}}

	used, counts, unmatched := attributeAllocations([]*network.Block{blockA, blockB}, []*network.Allocation{allocA, allocB})
	if counts[blockA.ID] != 1 || counts[blockB.ID] != 1 {
		t.Fatalf("counts A=%d B=%d, want 1 and 1", counts[blockA.ID], counts[blockB.ID])
	}
	if used[blockA.ID].Cmp(big.NewInt(256)) != 0 {
		t.Errorf("used A = %s, want 256", used[blockA.ID])
	}
	if used[blockB.ID].Cmp(big.NewInt(256)) != 0 {
		t.Errorf("used B = %s, want 256", used[blockB.ID])
	}
	if len(unmatched) != 0 {
		t.Errorf("unmatched = %d, want 0", len(unmatched))
	}
}

func TestAttributeAllocations_uniqueNameWithoutContainment(t *testing.T) {
	block := &network.Block{ID: uuid.New(), Name: "only", CIDR: "10.0.0.0/24"}
	alloc := &network.Allocation{Id: uuid.New(), Name: "a", Block: network.Block{Name: "only", CIDR: "10.0.0.0/28"}}
	_, counts, unmatched := attributeAllocations([]*network.Block{block}, []*network.Allocation{alloc})
	if counts[block.ID] != 1 {
		t.Fatalf("count = %d, want 1", counts[block.ID])
	}
	if len(unmatched) != 0 {
		t.Errorf("unmatched = %d, want 0", len(unmatched))
	}
}

func TestAttributeAllocations_unmatched(t *testing.T) {
	blockA := &network.Block{ID: uuid.New(), Name: "app", CIDR: "10.0.0.0/16"}
	blockB := &network.Block{ID: uuid.New(), Name: "app", CIDR: "10.1.0.0/16"}
	// No block named "ghost".
	orphan := &network.Allocation{Id: uuid.New(), Name: "o", Block: network.Block{Name: "ghost", CIDR: "10.9.0.0/24"}}
	// Same-named blocks, but the CIDR is contained in neither.
	stray := &network.Allocation{Id: uuid.New(), Name: "s", Block: network.Block{Name: "app", CIDR: "192.168.0.0/24"}}
	ok := &network.Allocation{Id: uuid.New(), Name: "k", Block: network.Block{Name: "APP ", CIDR: "10.1.2.0/24"}}

	used, counts, unmatched := attributeAllocations([]*network.Block{blockA, blockB}, []*network.Allocation{orphan, stray, ok})
	if len(unmatched) != 2 {
		t.Fatalf("unmatched = %d, want 2", len(unmatched))
	}
	if counts[blockB.ID] != 1 || counts[blockA.ID] != 0 {
		t.Errorf("counts A=%d B=%d, want 0 and 1", counts[blockA.ID], counts[blockB.ID])
	}
	if used[blockB.ID].Cmp(big.NewInt(256)) != 0 {
		t.Errorf("used B = %s, want 256", used[blockB.ID])
	}
}

func TestPoolUsedByID(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	leafID := uuid.New()
	parent := &network.Pool{ID: parentID, Name: "parent", CIDR: "10.0.0.0/16"}
	child := &network.Pool{ID: childID, Name: "child", CIDR: "10.0.0.0/20", ParentPoolID: &parentID}
	leaf := &network.Pool{ID: leafID, Name: "leaf", CIDR: "10.0.32.0/24"}
	block := &network.Block{ID: uuid.New(), Name: "b", CIDR: "10.0.16.0/24", PoolID: &parentID}
	childBlock := &network.Block{ID: uuid.New(), Name: "cb", CIDR: "10.0.0.0/24", PoolID: &childID}
	orphanBlock := &network.Block{ID: uuid.New(), Name: "ob", CIDR: "172.16.0.0/24"}

	used := poolUsedByID([]*network.Pool{parent, child, leaf}, []*network.Block{block, childBlock, orphanBlock})

	// parent: child pool /20 = 4096, plus block /24 = 256; childBlock is not direct.
	if want := big.NewInt(4096 + 256); used[parentID].Cmp(want) != 0 {
		t.Errorf("parent used = %s, want %s", used[parentID], want)
	}
	if want := big.NewInt(256); used[childID].Cmp(want) != 0 {
		t.Errorf("child used = %s, want %s", used[childID], want)
	}
	if _, ok := used[leafID]; ok {
		t.Errorf("leaf pool with nothing carved should be absent, got %s", used[leafID])
	}
	if len(used) != 2 {
		t.Errorf("len(used) = %d, want 2", len(used))
	}
}
