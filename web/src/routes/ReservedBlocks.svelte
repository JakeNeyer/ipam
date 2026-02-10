<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import '../lib/theme.js'
  import DataTable from '../lib/DataTable.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { listReservedBlocks, deleteReservedBlock } from '../lib/api.js'
  import ReservedBlocksModal from '../lib/ReservedBlocksModal.svelte'

  let reservedBlocks = []
  let loading = true
  let error = ''
  let showAddModal = false
  let deletingId = null

  let sortBy = 'name' // 'name' | 'cidr' | 'reason'
  let sortDir = 'asc' // 'asc' | 'desc'

  function setSort(column) {
    if (sortBy === column) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc'
    } else {
      sortBy = column
      sortDir = 'asc'
    }
  }

  $: sortedReservedBlocks = (() => {
    const list = [...reservedBlocks]
    const mult = sortDir === 'asc' ? 1 : -1
    if (sortBy === 'name') {
      list.sort((a, b) => mult * ((a.name || '').localeCompare(b.name || '—', undefined, { sensitivity: 'base' })))
    } else if (sortBy === 'cidr') {
      list.sort((a, b) => mult * ((a.cidr || '').localeCompare(b.cidr || '', undefined, { sensitivity: 'base' })))
    } else if (sortBy === 'reason') {
      list.sort((a, b) => mult * ((a.reason || '').localeCompare(b.reason || '—', undefined, { sensitivity: 'base' })))
    }
    return list
  })()

  async function load() {
    loading = true
    error = ''
    try {
      const res = await listReservedBlocks()
      reservedBlocks = res.reserved_blocks || []
    } catch (e) {
      error = e?.message || 'Failed to load reserved blocks'
      reservedBlocks = []
    } finally {
      loading = false
    }
  }

  async function handleDelete(id) {
    if (!id) return
    deletingId = id
    try {
      await deleteReservedBlock(id)
      await load()
    } catch (e) {
      error = e?.message || 'Failed to delete reserved block'
    } finally {
      deletingId = null
    }
  }

  onMount(() => {
    load()
  })
</script>

<div class="page">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Reserved blocks</h1>
      <p class="page-desc">CIDR ranges that cannot be used as network blocks or allocations. Use them to preserve ranges for future use or other systems.</p>
    </div>
    <button type="button" class="btn btn-primary" on:click={() => (showAddModal = true)}>
      Add reserved block
    </button>
  </header>

  <ReservedBlocksModal
    open={showAddModal}
    on:close={() => { showAddModal = false; load() }}
    on:created={load}
  />

  <div class="card">
    {#if loading}
      <p class="muted">Loading…</p>
    {:else if error}
      <p class="error">{error}</p>
    {:else}
      <DataTable>
        <svelte:fragment slot="header">
          <tr>
            <th class="sortable" class:sorted={sortBy === 'name'}>
              <button type="button" class="th-sort" on:click={() => setSort('name')}>
                <span class="th-sort-label">Name</span>
                {#if sortBy === 'name'}
                  <span class="sort-icon" aria-hidden="true"><Icon icon={sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                {/if}
              </button>
            </th>
            <th class="sortable" class:sorted={sortBy === 'cidr'}>
              <button type="button" class="th-sort" on:click={() => setSort('cidr')}>
                <span class="th-sort-label">CIDR</span>
                {#if sortBy === 'cidr'}
                  <span class="sort-icon" aria-hidden="true"><Icon icon={sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                {/if}
              </button>
            </th>
            <th class="sortable" class:sorted={sortBy === 'reason'}>
              <button type="button" class="th-sort" on:click={() => setSort('reason')}>
                <span class="th-sort-label">Reason</span>
                {#if sortBy === 'reason'}
                  <span class="sort-icon" aria-hidden="true"><Icon icon={sortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                {/if}
              </button>
            </th>
            <th class="table-actions"></th>
          </tr>
        </svelte:fragment>
        <svelte:fragment slot="body">
          {#if reservedBlocks.length === 0}
            <tr>
              <td colspan="4" class="table-empty-cell">No reserved blocks yet.</td>
            </tr>
          {:else}
            {#each sortedReservedBlocks as r (r.id)}
              {@const range = cidrRange(r.cidr)}
              <tr>
                <td class="name">{r.name || '—'}</td>
                <td class="cidr">
                  <code>{r.cidr}</code>
                  {#if range}
                    <span class="cidr-range">{range.start} – {range.end}</span>
                  {/if}
                </td>
                <td>{r.reason || '—'}</td>
                <td class="table-actions">
                  <button
                    type="button"
                    class="btn btn-danger btn-small"
                    disabled={deletingId === r.id}
                    on:click={() => handleDelete(r.id)}
                    title="Delete reserved block"
                  >
                    {deletingId === r.id ? 'Deleting…' : 'Delete'}
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </svelte:fragment>
      </DataTable>
    {/if}
  </div>
</div>

<style>
  .page {
    padding: 0;
  }
  .card {
    padding: 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .muted {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .error {
    margin: 0;
    font-size: 0.9rem;
    color: var(--danger);
  }
  .cidr {
    vertical-align: top;
  }
  .cidr code {
    font-family: var(--font-mono);
  }
  .cidr-range {
    display: block;
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-top: 0.2rem;
  }
</style>
