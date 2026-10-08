package metrics

import (
	"math/big"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// Per-resource series carry the resource UUID as "id". Names and CIDRs are not
// unique (blocks have no uniqueness constraint at all), and two series with an
// identical label set make the registry reject the whole scrape, so the id
// guarantees every row is a distinct series.
var (
	blockLabels = []string{"id", "organization", "environment", "pool", "block", "cidr", "family", "provider"}
	poolLabels  = []string{"id", "organization", "environment", "pool", "parent_pool", "cidr", "family", "provider"}
	resLabels   = []string{"id", "organization", "name", "cidr", "family"}
	orgLabel    = []string{"organization"}
	invLabels   = []string{"organization", "environment", "family"}
)

// Collector walks the IPAM store on each Prometheus scrape and emits gauges
// for current IP space: block fill, pool carve-out, reserved ranges, and counts.
type Collector struct {
	store store.Storer

	blockIPs          *prometheus.Desc
	blockUsedIPs      *prometheus.Desc
	blockAvailableIPs *prometheus.Desc
	blockUtil         *prometheus.Desc
	blockAllocations  *prometheus.Desc
	poolIPs           *prometheus.Desc
	poolUsedIPs       *prometheus.Desc
	poolAvailableIPs  *prometheus.Desc
	poolUtil          *prometheus.Desc
	reservedIPs       *prometheus.Desc
	organizations     *prometheus.Desc
	environments      *prometheus.Desc
	pools             *prometheus.Desc
	blocks            *prometheus.Desc
	allocations       *prometheus.Desc
	collectSuccess    *prometheus.Desc
	collectDuration   *prometheus.Desc
}

// NewCollector returns a Prometheus collector bound to s.
func NewCollector(s store.Storer) *Collector {
	return &Collector{
		store: s,
		blockIPs: prometheus.NewDesc("ipam_block_ips",
			"Addresses in the block CIDR", blockLabels, nil),
		blockUsedIPs: prometheus.NewDesc("ipam_block_used_ips",
			"Addresses allocated from the block", blockLabels, nil),
		blockAvailableIPs: prometheus.NewDesc("ipam_block_available_ips",
			"Addresses remaining in the block", blockLabels, nil),
		blockUtil: prometheus.NewDesc("ipam_block_utilization_ratio",
			"Fraction of the block that is allocated (0-1)", blockLabels, nil),
		blockAllocations: prometheus.NewDesc("ipam_block_allocations",
			"Number of allocations in the block", blockLabels, nil),
		poolIPs: prometheus.NewDesc("ipam_pool_ips",
			"Addresses in the pool CIDR", poolLabels, nil),
		poolUsedIPs: prometheus.NewDesc("ipam_pool_used_ips",
			"Addresses carved into child pools and blocks", poolLabels, nil),
		poolAvailableIPs: prometheus.NewDesc("ipam_pool_available_ips",
			"Addresses in the pool not yet carved into child pools or blocks", poolLabels, nil),
		poolUtil: prometheus.NewDesc("ipam_pool_utilization_ratio",
			"Fraction of the pool CIDR carved into child pools and blocks (0-1)", poolLabels, nil),
		reservedIPs: prometheus.NewDesc("ipam_reserved_ips",
			"Addresses in a reserved (blacklisted) CIDR", resLabels, nil),
		organizations: prometheus.NewDesc("ipam_organizations",
			"Number of organizations", nil, nil),
		environments: prometheus.NewDesc("ipam_environments",
			"Number of environments", orgLabel, nil),
		pools: prometheus.NewDesc("ipam_pools",
			"Number of pools", invLabels, nil),
		blocks: prometheus.NewDesc("ipam_blocks",
			"Number of network blocks", invLabels, nil),
		allocations: prometheus.NewDesc("ipam_allocations",
			"Number of allocations; allocations whose block cannot be resolved are reported with empty organization and environment", invLabels, nil),
		collectSuccess: prometheus.NewDesc("ipam_metrics_collect_success",
			"1 if the last metrics collection succeeded, 0 otherwise", nil, nil),
		collectDuration: prometheus.NewDesc("ipam_metrics_collect_duration_seconds",
			"Wall time of the last metrics collection", nil, nil),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.blockIPs
	ch <- c.blockUsedIPs
	ch <- c.blockAvailableIPs
	ch <- c.blockUtil
	ch <- c.blockAllocations
	ch <- c.poolIPs
	ch <- c.poolUsedIPs
	ch <- c.poolAvailableIPs
	ch <- c.poolUtil
	ch <- c.reservedIPs
	ch <- c.organizations
	ch <- c.environments
	ch <- c.pools
	ch <- c.blocks
	ch <- c.allocations
	ch <- c.collectSuccess
	ch <- c.collectDuration
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	err := c.collect(ch)
	success := 1.0
	if err != nil {
		success = 0
	}
	ch <- prometheus.MustNewConstMetric(c.collectSuccess, prometheus.GaugeValue, success)
	ch <- prometheus.MustNewConstMetric(c.collectDuration, prometheus.GaugeValue, time.Since(start).Seconds())
}

type countKey struct {
	org, env, family string
}

func (c *Collector) collect(ch chan<- prometheus.Metric) error {
	orgs, err := c.store.ListOrganizations()
	if err != nil {
		return err
	}
	envs, err := c.store.ListEnvironments()
	if err != nil {
		return err
	}
	blocks, err := c.store.ListBlocks()
	if err != nil {
		return err
	}
	allocs, err := c.store.ListAllocations()
	if err != nil {
		return err
	}
	reserved, err := c.store.ListReservedBlocks(nil)
	if err != nil {
		return err
	}

	var pools []*network.Pool
	for _, org := range orgs {
		orgPools, err := c.store.ListPoolsByOrganization(org.ID)
		if err != nil {
			return err
		}
		pools = append(pools, orgPools...)
	}

	orgName := make(map[uuid.UUID]string, len(orgs))
	for _, o := range orgs {
		orgName[o.ID] = o.Name
	}
	envByID := make(map[uuid.UUID]*network.Environment, len(envs))
	for _, e := range envs {
		envByID[e.Id] = e
	}
	poolByID := make(map[uuid.UUID]*network.Pool, len(pools))
	for _, p := range pools {
		poolByID[p.ID] = p
	}

	orgOf := func(envID, fallbackOrg uuid.UUID) string {
		if e, ok := envByID[envID]; ok {
			if n, ok := orgName[e.OrganizationID]; ok {
				return n
			}
		}
		if n, ok := orgName[fallbackOrg]; ok {
			return n
		}
		return ""
	}
	envName := func(envID uuid.UUID) string {
		if e, ok := envByID[envID]; ok {
			return e.Name
		}
		return ""
	}
	poolName := func(id *uuid.UUID) string {
		if id == nil || *id == uuid.Nil {
			return ""
		}
		if p, ok := poolByID[*id]; ok {
			return p.Name
		}
		return ""
	}

	usedByBlock, allocsByBlock, unmatchedAllocs := attributeAllocations(blocks, allocs)
	usedByPool := poolUsedByID(pools, blocks)

	envCount := make(map[string]int)
	poolCount := make(map[countKey]int)
	blockCount := make(map[countKey]int)
	allocCount := make(map[countKey]int)

	for _, e := range envs {
		envCount[orgName[e.OrganizationID]]++
	}

	for _, p := range pools {
		org := orgOf(p.EnvironmentID, p.OrganizationID)
		env := envName(p.EnvironmentID)
		family := ipFamily(p.CIDR)
		parent := poolName(p.ParentPoolID)
		labels := []string{p.ID.String(), org, env, p.Name, parent, p.CIDR, family, providerLabel(p.Provider)}
		total := cidrCount(p.CIDR)
		used := usedByPool[p.ID]
		if used == nil {
			used = big.NewInt(0)
		}
		emitSize(ch, c.poolIPs, c.poolUsedIPs, c.poolAvailableIPs, c.poolUtil, total, used, labels)
		poolCount[countKey{org, env, family}]++
	}

	for _, b := range blocks {
		org := orgOf(b.EnvironmentID, b.OrganizationID)
		env := envName(b.EnvironmentID)
		family := ipFamily(b.CIDR)
		labels := []string{b.ID.String(), org, env, poolName(b.PoolID), b.Name, b.CIDR, family, providerLabel(b.Provider)}
		total := cidrCount(b.CIDR)
		used := usedByBlock[b.ID]
		if used == nil {
			used = big.NewInt(0)
		}
		emitSize(ch, c.blockIPs, c.blockUsedIPs, c.blockAvailableIPs, c.blockUtil, total, used, labels)
		ch <- prometheus.MustNewConstMetric(c.blockAllocations, prometheus.GaugeValue, float64(allocsByBlock[b.ID]), labels...)
		blockCount[countKey{org, env, family}]++
		allocCount[countKey{org, env, family}] += allocsByBlock[b.ID]
	}
	// Keep the inventory total honest: allocations with no resolvable block are
	// still allocations, reported under empty organization/environment.
	for _, a := range unmatchedAllocs {
		allocCount[countKey{"", "", ipFamily(a.Block.CIDR)}]++
	}

	for _, r := range reserved {
		org := orgName[r.OrganizationID]
		ch <- prometheus.MustNewConstMetric(c.reservedIPs, prometheus.GaugeValue, bigToFloat(cidrCount(r.CIDR)),
			r.ID.String(), org, r.Name, r.CIDR, ipFamily(r.CIDR))
	}

	ch <- prometheus.MustNewConstMetric(c.organizations, prometheus.GaugeValue, float64(len(orgs)))
	for org, n := range envCount {
		ch <- prometheus.MustNewConstMetric(c.environments, prometheus.GaugeValue, float64(n), org)
	}
	for k, n := range poolCount {
		ch <- prometheus.MustNewConstMetric(c.pools, prometheus.GaugeValue, float64(n), k.org, k.env, k.family)
	}
	for k, n := range blockCount {
		ch <- prometheus.MustNewConstMetric(c.blocks, prometheus.GaugeValue, float64(n), k.org, k.env, k.family)
	}
	for k, n := range allocCount {
		ch <- prometheus.MustNewConstMetric(c.allocations, prometheus.GaugeValue, float64(n), k.org, k.env, k.family)
	}
	return nil
}

func emitSize(ch chan<- prometheus.Metric, totalDesc, usedDesc, availDesc, utilDesc *prometheus.Desc, total, used *big.Int, labels []string) {
	avail := availableIPs(total, used)
	ch <- prometheus.MustNewConstMetric(totalDesc, prometheus.GaugeValue, bigToFloat(total), labels...)
	ch <- prometheus.MustNewConstMetric(usedDesc, prometheus.GaugeValue, bigToFloat(used), labels...)
	ch <- prometheus.MustNewConstMetric(availDesc, prometheus.GaugeValue, bigToFloat(avail), labels...)
	ch <- prometheus.MustNewConstMetric(utilDesc, prometheus.GaugeValue, utilizationRatio(used, total), labels...)
}
