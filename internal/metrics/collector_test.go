package metrics

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JakeNeyer/ipam/network"
	"github.com/JakeNeyer/ipam/store"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// failingStore makes ListBlocks fail so collect() returns an error.
type failingStore struct {
	store.Storer
}

func (failingStore) ListBlocks() ([]*network.Block, error) {
	return nil, errors.New("boom")
}

// countingStore counts how many times the collector walks the store.
type countingStore struct {
	store.Storer
	walks atomic.Int32
}

func (c *countingStore) ListOrganizations() ([]*store.Organization, error) {
	c.walks.Add(1)
	return c.Storer.ListOrganizations()
}

func mustCreateBlock(t *testing.T, s *store.Store, b *network.Block) {
	t.Helper()
	if err := s.CreateBlock(b); err != nil {
		t.Fatal(err)
	}
}

func setupIPSpace(t *testing.T) *store.Store {
	t.Helper()
	s := store.NewStore()
	org := &store.Organization{ID: s.GenerateID(), Name: "acme"}
	if err := s.CreateOrganization(org); err != nil {
		t.Fatal(err)
	}
	env := &network.Environment{Id: s.GenerateID(), Name: "prod", OrganizationID: org.ID}
	if err := s.CreateEnvironment(env); err != nil {
		t.Fatal(err)
	}
	pool := &network.Pool{
		ID:             s.GenerateID(),
		OrganizationID: org.ID,
		EnvironmentID:  env.Id,
		Name:           "core",
		CIDR:           "10.0.0.0/16",
	}
	if err := s.CreatePool(pool); err != nil {
		t.Fatal(err)
	}
	block := &network.Block{
		ID:             s.GenerateID(),
		Name:           "app",
		CIDR:           "10.0.1.0/24",
		EnvironmentID:  env.Id,
		OrganizationID: org.ID,
		PoolID:         &pool.ID,
	}
	if err := s.CreateBlock(block); err != nil {
		t.Fatal(err)
	}
	alloc := &network.Allocation{
		Id:    s.GenerateID(),
		Name:  "web",
		Block: network.Block{Name: "app", CIDR: "10.0.1.0/26"},
	}
	if err := s.CreateAllocation(alloc.Id, alloc); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateReservedBlock(&store.ReservedBlock{
		Name:           "dmz",
		CIDR:           "10.255.0.0/16",
		OrganizationID: org.ID,
	}); err != nil {
		t.Fatal(err)
	}
	return s
}

func gather(t *testing.T, s store.Storer) map[string]*dto.MetricFamily {
	t.Helper()
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(NewCollector(s)); err != nil {
		t.Fatal(err)
	}
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]*dto.MetricFamily, len(mfs))
	for _, mf := range mfs {
		out[mf.GetName()] = mf
	}
	return out
}

func gaugeWithLabels(t *testing.T, mf *dto.MetricFamily, want map[string]string) float64 {
	t.Helper()
	if mf == nil {
		t.Fatal("missing metric family")
	}
	for _, m := range mf.Metric {
		got := make(map[string]string, len(m.Label))
		for _, l := range m.Label {
			got[l.GetName()] = l.GetValue()
		}
		match := true
		for k, v := range want {
			if got[k] != v {
				match = false
				break
			}
		}
		if match {
			return m.GetGauge().GetValue()
		}
	}
	t.Fatalf("no series matching labels %v in %s", want, mf.GetName())
	return 0
}

func TestCollector_ipSpace(t *testing.T) {
	s := setupIPSpace(t)
	mfs := gather(t, s)

	if mfs["ipam_metrics_collect_success"].Metric[0].GetGauge().GetValue() != 1 {
		t.Fatal("expected successful collect")
	}
	if mfs["ipam_organizations"].Metric[0].GetGauge().GetValue() != 1 {
		t.Errorf("organizations = %v, want 1", mfs["ipam_organizations"].Metric[0].GetGauge().GetValue())
	}

	blockLabels := map[string]string{
		"organization": "acme",
		"environment":  "prod",
		"pool":         "core",
		"block":        "app",
		"cidr":         "10.0.1.0/24",
		"family":       "ipv4",
	}
	for _, l := range mfs["ipam_block_ips"].Metric[0].Label {
		if l.GetName() == "id" {
			if _, err := uuid.Parse(l.GetValue()); err != nil {
				t.Errorf("block id label %q is not a UUID", l.GetValue())
			}
		}
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_ips"], blockLabels); v != 256 {
		t.Errorf("block_ips = %v, want 256", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_used_ips"], blockLabels); v != 64 {
		t.Errorf("block_used_ips = %v, want 64 (/26)", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_available_ips"], blockLabels); v != 192 {
		t.Errorf("block_available_ips = %v, want 192", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_utilization_ratio"], blockLabels); v != 0.25 {
		t.Errorf("block_utilization_ratio = %v, want 0.25", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_allocations"], blockLabels); v != 1 {
		t.Errorf("block_allocations = %v, want 1", v)
	}

	poolLabels := map[string]string{
		"organization": "acme",
		"environment":  "prod",
		"pool":         "core",
		"cidr":         "10.0.0.0/16",
		"family":       "ipv4",
	}
	if v := gaugeWithLabels(t, mfs["ipam_pool_ips"], poolLabels); v != 65536 {
		t.Errorf("pool_ips = %v, want 65536", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_pool_used_ips"], poolLabels); v != 256 {
		t.Errorf("pool_used_ips = %v, want 256 (block CIDR carved from pool)", v)
	}

	resLabels := map[string]string{"organization": "acme", "name": "dmz", "cidr": "10.255.0.0/16"}
	if v := gaugeWithLabels(t, mfs["ipam_reserved_ips"], resLabels); v != 65536 {
		t.Errorf("reserved_ips = %v, want 65536", v)
	}
}

func TestCollector_emptyStore(t *testing.T) {
	mfs := gather(t, store.NewStore())
	if mfs["ipam_organizations"].Metric[0].GetGauge().GetValue() != 0 {
		t.Errorf("organizations = %v, want 0", mfs["ipam_organizations"].Metric[0].GetGauge().GetValue())
	}
	if _, ok := mfs["ipam_block_ips"]; ok {
		t.Error("empty store should not emit block series")
	}
}

// Blocks have no uniqueness constraint. Two blocks with the same name, CIDR,
// environment, pool and provider must still yield distinct series (via the id
// label) instead of a registry duplicate error that fails the whole scrape.
func TestCollector_duplicateBlocksAreDistinctSeries(t *testing.T) {
	s := setupIPSpace(t)
	envs, err := s.ListEnvironments()
	if err != nil || len(envs) != 1 {
		t.Fatalf("envs = %v, %v", envs, err)
	}
	env := envs[0]
	for i := 0; i < 2; i++ {
		mustCreateBlock(t, s, &network.Block{
			ID:             s.GenerateID(),
			Name:           "dup",
			CIDR:           "10.0.9.0/24",
			EnvironmentID:  env.Id,
			OrganizationID: env.OrganizationID,
		})
	}

	mfs := gather(t, s) // pedantic registry: fails the test on duplicate label sets
	ids := map[string]bool{}
	for _, m := range mfs["ipam_block_ips"].Metric {
		var name, id string
		for _, l := range m.Label {
			switch l.GetName() {
			case "block":
				name = l.GetValue()
			case "id":
				id = l.GetValue()
			}
		}
		if name == "dup" {
			ids[id] = true
		}
	}
	if len(ids) != 2 {
		t.Fatalf("want 2 distinct dup series, got ids %v", ids)
	}

	h := Handler(s, "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if n := strings.Count(rec.Body.String(), `block="dup"`); n < 2 {
		t.Errorf("expected both dup blocks in output, found %d occurrences", n)
	}
}

func TestCollector_storeError(t *testing.T) {
	mfs := gather(t, failingStore{Storer: setupIPSpace(t)})
	if v := mfs["ipam_metrics_collect_success"].Metric[0].GetGauge().GetValue(); v != 0 {
		t.Errorf("collect_success = %v, want 0", v)
	}
	if _, ok := mfs["ipam_block_ips"]; ok {
		t.Error("failed collect should not emit partial block series")
	}
	if _, ok := mfs["ipam_metrics_collect_duration_seconds"]; !ok {
		t.Error("collect duration should be emitted even on failure")
	}

	// The HTTP handler must still answer 200 so Prometheus records up=1 and
	// ipam_metrics_collect_success=0, rather than a scrape failure.
	h := Handler(failingStore{Storer: setupIPSpace(t)}, "")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ipam_metrics_collect_success 0") {
		t.Errorf("body missing collect_success 0:\n%s", rec.Body.String())
	}
}

func TestCollector_ipv6(t *testing.T) {
	s := setupIPSpace(t)
	envs, _ := s.ListEnvironments()
	env := envs[0]
	mustCreateBlock(t, s, &network.Block{
		ID:             s.GenerateID(),
		Name:           "v6",
		CIDR:           "2001:db8::/48",
		EnvironmentID:  env.Id,
		OrganizationID: env.OrganizationID,
	})
	alloc := &network.Allocation{Id: s.GenerateID(), Name: "v6-a", Block: network.Block{Name: "v6", CIDR: "2001:db8::/64"}}
	if err := s.CreateAllocation(alloc.Id, alloc); err != nil {
		t.Fatal(err)
	}

	mfs := gather(t, s)
	l := map[string]string{"block": "v6", "family": "ipv6"}
	const (
		total = 1208925819614629174706176 // 2^80
		used  = 18446744073709551616      // 2^64
	)
	if v := gaugeWithLabels(t, mfs["ipam_block_ips"], l); v != total {
		t.Errorf("block_ips = %v, want %v", v, float64(total))
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_used_ips"], l); v != used {
		t.Errorf("block_used_ips = %v, want %v", v, float64(used))
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_available_ips"], l); v != total-used {
		t.Errorf("block_available_ips = %v, want %v", v, float64(total-used))
	}
	if v := gaugeWithLabels(t, mfs["ipam_block_utilization_ratio"], l); v != 1.0/65536 {
		t.Errorf("block_utilization_ratio = %v, want %v", v, 1.0/65536)
	}
	if v := gaugeWithLabels(t, mfs["ipam_blocks"], map[string]string{"organization": "acme", "environment": "prod", "family": "ipv6"}); v != 1 {
		t.Errorf("ipam_blocks{ipv6} = %v, want 1", v)
	}
}

// Allocations whose block cannot be resolved still count toward
// ipam_allocations, under empty organization/environment.
func TestCollector_unattributedAllocationsCounted(t *testing.T) {
	s := setupIPSpace(t)
	ghost := &network.Allocation{Id: s.GenerateID(), Name: "ghost", Block: network.Block{Name: "no-such-block", CIDR: "10.200.0.0/24"}}
	if err := s.CreateAllocation(ghost.Id, ghost); err != nil {
		t.Fatal(err)
	}

	mfs := gather(t, s)
	if v := gaugeWithLabels(t, mfs["ipam_allocations"], map[string]string{"organization": "acme", "environment": "prod"}); v != 1 {
		t.Errorf("attributed allocations = %v, want 1", v)
	}
	if v := gaugeWithLabels(t, mfs["ipam_allocations"], map[string]string{"organization": "", "environment": "", "family": "ipv4"}); v != 1 {
		t.Errorf("unattributed allocations = %v, want 1", v)
	}
	var total float64
	for _, m := range mfs["ipam_allocations"].Metric {
		total += m.GetGauge().GetValue()
	}
	if total != 2 {
		t.Errorf("sum(ipam_allocations) = %v, want 2", total)
	}
}

func TestHandler_cachesWithinTTL(t *testing.T) {
	cs := &countingStore{Storer: setupIPSpace(t)}
	now := time.Unix(1_000_000, 0)
	clock := func() time.Time { return now }
	h := newHandler(cs, "", 5*time.Second, clock)

	scrape := func() {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	}

	scrape()
	scrape()
	now = now.Add(2 * time.Second)
	scrape()
	if got := cs.walks.Load(); got != 1 {
		t.Fatalf("store walks within TTL = %d, want 1", got)
	}

	now = now.Add(4 * time.Second) // 6s since first collection
	scrape()
	if got := cs.walks.Load(); got != 2 {
		t.Fatalf("store walks after TTL = %d, want 2", got)
	}
}

func TestHandler_noCacheWhenTTLZero(t *testing.T) {
	cs := &countingStore{Storer: setupIPSpace(t)}
	h := newHandler(cs, "", 0, time.Now)
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path, nil))
	}
	if got := cs.walks.Load(); got != 3 {
		t.Fatalf("store walks with ttl=0 = %d, want 3", got)
	}
}

func TestHandler_noToken(t *testing.T) {
	h := Handler(setupIPSpace(t), "")
	req := httptest.NewRequest(http.MethodGet, Path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ipam_block_utilization_ratio") {
		t.Errorf("body missing ipam_block_utilization_ratio:\n%s", body)
	}
	if !strings.Contains(body, `block="app"`) {
		t.Errorf("body missing block label:\n%s", body)
	}
}

func TestHandler_tokenRequired(t *testing.T) {
	h := Handler(store.NewStore(), "s3cret")
	req := httptest.NewRequest(http.MethodGet, Path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, Path, nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status = %d, want 401", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, Path, nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("good token status = %d, want 200", rec.Code)
	}
}

func TestWrap(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "next")
	})
	metrics := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "metrics")
	})
	h := Wrap(next, metrics)

	req := httptest.NewRequest(http.MethodGet, Path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "metrics" {
		t.Errorf("/metrics -> %d %q", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/blocks", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot || rec.Body.String() != "next" {
		t.Errorf("/api/blocks -> %d %q", rec.Code, rec.Body.String())
	}
}
