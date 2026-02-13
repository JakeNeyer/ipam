<script>
  import { onMount } from 'svelte'
  import { createEventDispatcher } from 'svelte'
  import Icon from '@iconify/svelte'
  import { listOrganizations, listUsers, listEnvironments, listBlocks, listAllocations, listPools } from '../lib/api.js'

  const dispatch = createEventDispatcher()

  let loading = true
  let error = ''
  /** @type {Array<{ id: string, name: string, users: number, environments: number, blocks: number, allocations: number, pools: number }>} */
  let orgStats = []

  onMount(() => {
    load()
    const onVisible = () => {
      if (document.visibilityState === 'visible') load()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  })

  async function load() {
    loading = true
    error = ''
    orgStats = []
    try {
      const [orgsRes, usersRes] = await Promise.all([
        listOrganizations(),
        listUsers(),
      ])
      const organizations = (orgsRes.organizations ?? []).slice().sort((a, b) => (a.name || '').localeCompare(b.name || ''))
      const users = usersRes.users ?? []
      const usersByOrg = new Map()
      for (const u of users) {
        const oid = u.organization_id ?? ''
        usersByOrg.set(oid, (usersByOrg.get(oid) ?? 0) + 1)
      }
      const stats = await Promise.all(
        organizations.map(async (org) => {
          const [envsRes, blocksRes, allocsRes] = await Promise.all([
            listEnvironments({ organization_id: org.id, limit: 200, offset: 0 }),
            listBlocks({ organization_id: org.id, limit: 1, offset: 0 }),
            listAllocations({ organization_id: org.id, limit: 1, offset: 0 }),
          ])
          const envs = envsRes.environments ?? []
          let pools = 0
          if (envs.length > 0) {
            const poolResults = await Promise.all(envs.map((e) => listPools(e.id)))
            pools = poolResults.reduce((s, r) => s + (r.pools?.length ?? 0), 0)
          }
          return {
            id: org.id,
            name: org.name,
            users: usersByOrg.get(org.id) ?? 0,
            environments: envsRes.total ?? 0,
            blocks: blocksRes.total ?? 0,
            allocations: allocsRes.total ?? 0,
            pools,
          }
        })
      )
      orgStats = stats
    } catch (e) {
      error = e?.message ?? 'Failed to load dashboard'
    } finally {
      loading = false
    }
  }

  function selectOrg(org) {
    dispatch('selectOrg', { id: org.id, name: org.name })
  }
</script>

<div class="global-admin-dashboard">
  <header class="page-header">
    <h1 class="page-title">Dashboard</h1>
  </header>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else if error}
    <div class="dashboard-error" role="alert">{error}</div>
  {:else if orgStats.length === 0}
    <div class="empty-state">
      <span class="empty-icon" aria-hidden="true"><Icon icon="lucide:building-2" width="2.5rem" height="2.5rem" /></span>
      <p>No organizations yet. Create one from <a href="#admin">Admin</a> to get started.</p>
    </div>
  {:else}
    <section class="stats-section">
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.length}</span>
        <span class="stats-summary-label">Organizations</span>
      </div>
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.reduce((s, o) => s + o.users, 0)}</span>
        <span class="stats-summary-label">Total users</span>
      </div>
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.reduce((s, o) => s + o.environments, 0)}</span>
        <span class="stats-summary-label">Environments</span>
      </div>
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.reduce((s, o) => s + o.blocks, 0)}</span>
        <span class="stats-summary-label">Blocks</span>
      </div>
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.reduce((s, o) => s + o.allocations, 0)}</span>
        <span class="stats-summary-label">Allocations</span>
      </div>
      <div class="stats-summary">
        <span class="stats-summary-value">{orgStats.reduce((s, o) => s + o.pools, 0)}</span>
        <span class="stats-summary-label">Pools</span>
      </div>
    </section>

    <section class="org-table-section">
      <h2 class="section-title">By organization</h2>
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Organization</th>
              <th class="num">Users</th>
              <th class="num">Environments</th>
              <th class="num">Blocks</th>
              <th class="num">Allocations</th>
              <th class="num">Pools</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each orgStats as org (org.id)}
              <tr>
                <td class="name">{org.name}</td>
                <td class="num">{org.users}</td>
                <td class="num">{org.environments}</td>
                <td class="num">{org.blocks}</td>
                <td class="num">{org.allocations}</td>
                <td class="num">{org.pools}</td>
                <td class="action">
                  <button
                    type="button"
                    class="btn btn-primary btn-small"
                    on:click={() => selectOrg(org)}
                  >
                    View
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </section>
  {/if}
</div>

<style>
  .global-admin-dashboard {
    padding: 1.25rem 1.5rem;
    max-width: 100%;
  }
  .page-header {
    margin-bottom: 1.5rem;
  }
  .page-title {
    margin: 0 0 0.25rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text);
  }
  .loading {
    padding: 2rem;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.9375rem;
  }
  .dashboard-error {
    padding: 1rem 1.25rem;
    background: rgba(220, 38, 38, 0.08);
    color: var(--danger);
    border-radius: var(--radius);
    font-size: 0.875rem;
  }
  .empty-state {
    text-align: center;
    padding: 3rem 2rem;
    color: var(--text-muted);
    font-size: 0.9375rem;
  }
  .empty-state a {
    color: var(--accent);
    text-decoration: none;
  }
  .empty-state a:hover {
    text-decoration: underline;
  }
  .empty-icon {
    display: block;
    margin: 0 auto 1rem;
    color: var(--text-muted);
    opacity: 0.7;
  }
  .empty-icon :global(svg) {
    display: block;
    margin: 0 auto;
  }
  .stats-section {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin-bottom: 1.75rem;
  }
  .stats-summary {
    display: flex;
    flex-direction: column;
    min-width: 6rem;
    padding: 0.75rem 1rem;
    background: var(--table-header-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .stats-summary-value {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text);
  }
  .stats-summary-label {
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-top: 0.15rem;
  }
  .org-table-section {
    margin-top: 0.5rem;
  }
  .section-title {
    margin: 0 0 0.75rem;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
  }
  .table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }
  .table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }
  .table th {
    text-align: left;
    padding: 0.6rem 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    background: var(--table-header-bg);
    border-bottom: 1px solid var(--border);
  }
  .table th.num {
    text-align: right;
  }
  .table td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border);
    color: var(--text);
  }
  .table td.num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .table td.name {
    font-weight: 500;
  }
  .table td.action {
    white-space: nowrap;
  }
  .table tbody tr:last-child td {
    border-bottom: none;
  }
  .table tbody tr:hover {
    background: var(--table-header-bg);
  }
</style>
