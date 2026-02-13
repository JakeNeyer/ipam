package setup

import (
	"os"
	"strings"
	"time"

	"github.com/JakeNeyer/ipam/internal/logger"
	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"log/slog"
)

const demoOrgName = "Demo"

// EnsureDemoFixtures creates a "Demo" organization with seed-like data when ENABLE_DEMO_FIXTURES is set.
// Idempotent: if an org named "Demo" already exists, nothing is done. Run on startup after EnsureInitialAdmin.
func EnsureDemoFixtures(st store.Storer) {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_DEMO_FIXTURES")))
	if v != "true" && v != "1" {
		return
	}
	orgs, err := st.ListOrganizations()
	if err != nil {
		logger.Error("demo fixtures: list organizations failed", logger.ErrAttr(err))
		return
	}
	for _, o := range orgs {
		if strings.TrimSpace(o.Name) == demoOrgName {
			logger.Info("demo fixtures: Demo organization already exists, skipping")
			return
		}
	}

	// Create Demo organization
	org := &store.Organization{
		ID:        st.GenerateID(),
		Name:      demoOrgName,
		CreatedAt: time.Now(),
	}
	if err := st.CreateOrganization(org); err != nil {
		logger.Error("demo fixtures: create organization failed", logger.ErrAttr(err))
		return
	}
	orgID := org.ID
	logger.Info("demo fixtures: created Demo organization", slog.String("org_id", orgID.String()))

	// Environments: Production, Staging
	prodEnv := &network.Environment{Id: st.GenerateID(), Name: "Production", OrganizationID: orgID}
	if err := st.CreateEnvironment(prodEnv); err != nil {
		logger.Error("demo fixtures: create Production environment failed", logger.ErrAttr(err))
		return
	}
	stagingEnv := &network.Environment{Id: st.GenerateID(), Name: "Staging", OrganizationID: orgID}
	if err := st.CreateEnvironment(stagingEnv); err != nil {
		logger.Error("demo fixtures: create Staging environment failed", logger.ErrAttr(err))
		return
	}
	prodID, stagingID := prodEnv.Id, stagingEnv.Id

	// Required pools per environment (blocks draw from these). Non-overlapping CIDRs within the org.
	prodPool := &network.Pool{ID: st.GenerateID(), OrganizationID: orgID, EnvironmentID: prodID, Name: "Production pool", CIDR: "10.0.0.0/9"}
	if err := st.CreatePool(prodPool); err != nil {
		logger.Error("demo fixtures: create Production pool failed", logger.ErrAttr(err))
		return
	}
	stagingPool := &network.Pool{ID: st.GenerateID(), OrganizationID: orgID, EnvironmentID: stagingID, Name: "Staging pool", CIDR: "10.128.0.0/9"}
	if err := st.CreatePool(stagingPool); err != nil {
		logger.Error("demo fixtures: create Staging pool failed", logger.ErrAttr(err))
		return
	}

	// Prod blocks in 10.0.0.0/9; staging blocks in 10.128.0.0/9 (no overlap)
	blocks := []struct {
		name        string
		cidr        string
		environment uuid.UUID
		poolID      *uuid.UUID
	}{
		{"prod-vpc", "10.0.0.0/16", prodID, &prodPool.ID},
		{"prod-dmz", "10.2.0.0/16", prodID, &prodPool.ID},
		{"prod-data", "10.4.0.0/16", prodID, &prodPool.ID},
		{"staging-vpc", "10.128.0.0/16", stagingID, &stagingPool.ID},
		{"staging-test", "10.130.0.0/16", stagingID, &stagingPool.ID},
		{"staging-dev", "10.132.0.0/16", stagingID, &stagingPool.ID},
		{"orphan-block", "192.168.0.0/24", uuid.Nil, nil},
		{"full-block", "10.7.0.0/26", prodID, &prodPool.ID},
		{"nearly-full-block", "10.8.0.0/24", prodID, &prodPool.ID},
	}
	for _, b := range blocks {
		block := &network.Block{
			Name:          b.name,
			CIDR:          b.cidr,
			EnvironmentID: b.environment,
			PoolID:        b.poolID,
			Usage:         network.Usage{TotalIPs: 1 << 20, UsedIPs: 0, AvailableIPs: 1 << 20},
		}
		if b.environment == uuid.Nil {
			block.OrganizationID = orgID
		}
		if err := st.CreateBlock(block); err != nil {
			logger.Error("demo fixtures: create block failed", slog.String("name", b.name), logger.ErrAttr(err))
			return
		}
	}

	allocations := []struct {
		name      string
		blockName string
		cidr      string
	}{
		{"prod-web", "prod-vpc", "10.0.0.0/24"},
		{"prod-db", "prod-vpc", "10.0.2.0/24"},
		{"staging-app", "staging-vpc", "10.128.0.0/24"},
		{"orphan-subnet", "orphan-block", "192.168.0.0/26"},
		{"full-alloc", "full-block", "10.7.0.0/26"},
		{"nearly-a", "nearly-full-block", "10.8.0.0/25"},
		{"nearly-b", "nearly-full-block", "10.8.0.128/26"},
		{"nearly-c", "nearly-full-block", "10.8.0.192/27"},
		{"nearly-d", "nearly-full-block", "10.8.0.224/28"},
	}
	for _, a := range allocations {
		allocation := &network.Allocation{
			Id:   st.GenerateID(),
			Name: a.name,
			Block: network.Block{Name: a.blockName, CIDR: a.cidr},
		}
		if err := st.CreateAllocation(allocation.Id, allocation); err != nil {
			logger.Error("demo fixtures: create allocation failed", slog.String("name", a.name), logger.ErrAttr(err))
			return
		}
	}

	reserved := []struct {
		name   string
		cidr   string
		reason string
	}{
		{"Future use", "10.6.0.0/16", "Reserved for future use"},
		{"Prod gap", "10.0.1.0/24", "Reserved gap in prod-vpc"},
		{"DMZ", "172.16.0.0/24", "DMZ reserve"},
	}
	for _, r := range reserved {
		rb := &store.ReservedBlock{
			Name:           r.name,
			CIDR:           r.cidr,
			Reason:         r.reason,
			CreatedAt:      time.Now(),
			OrganizationID: orgID,
		}
		if err := st.CreateReservedBlock(rb); err != nil {
			logger.Error("demo fixtures: create reserved block failed", slog.String("name", r.name), logger.ErrAttr(err))
			return
		}
	}

	logger.Info("demo fixtures: Demo organization and resources created", slog.String("org_id", orgID.String()))
}
