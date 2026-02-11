<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import '../lib/theme.js'
  import DataTable from '../lib/DataTable.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { listReservedBlocks, updateReservedBlock, deleteReservedBlock } from '../lib/api.js'
  import ReservedBlocksModal from '../lib/ReservedBlocksModal.svelte'

  let reservedBlocks = []
  let loading = true
  let error = ''
  let showAddModal = false
  let deletingId = null
  let editingId = null
  let editName = ''
  let updatingId = null
  let openMenuId = null
  let menuDropdownStyle = { left: 0, top: 0 }

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
      openMenuId = null
      await load()
    } catch (e) {
      error = e?.message || 'Failed to delete reserved block'
    } finally {
      deletingId = null
    }
  }

  function startEdit(row) {
    editingId = row.id
    editName = row.name || ''
    error = ''
  }

  function cancelEdit() {
    editingId = null
    editName = ''
  }

  async function handleSaveEdit(id) {
    if (!id) return
    updatingId = id
    error = ''
    try {
      await updateReservedBlock(id, editName)
      cancelEdit()
      openMenuId = null
      await load()
    } catch (e) {
      error = e?.message || 'Failed to update reserved block'
    } finally {
      updatingId = null
    }
  }

  onMount(() => {
    function handleClickOutside(e) {
      if (!e.target.closest('.actions-menu-wrap')) {
        openMenuId = null
      }
    }
    document.addEventListener('click', handleClickOutside)
    load()
    return () => document.removeEventListener('click', handleClickOutside)
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
            <th class="table-actions">Actions</th>
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
                <td class="name">
                  {#if editingId === r.id}
                    <input
                      type="text"
                      class="edit-name-input"
                      bind:value={editName}
                      disabled={updatingId === r.id}
                      placeholder="Reserved block name"
                    />
                  {:else}
                    {r.name || '—'}
                  {/if}
                </td>
                <td class="cidr">
                  <code>{r.cidr}</code>
                  {#if range}
                    <span class="cidr-range">{range.start} – {range.end}</span>
                  {/if}
                </td>
                <td>{r.reason || '—'}</td>
                <td class="table-actions">
                  <div class="actions-menu-wrap" role="group">
                    {#if editingId === r.id}
                      <div class="table-actions-wrap">
                        <button
                          type="button"
                          class="btn btn-small"
                          on:click={cancelEdit}
                          disabled={updatingId === r.id}
                          title="Cancel edit"
                        >
                          Cancel
                        </button>
                        <button
                          type="button"
                          class="btn btn-primary btn-small"
                          on:click={() => handleSaveEdit(r.id)}
                          disabled={updatingId === r.id}
                          title="Save name"
                        >
                          {updatingId === r.id ? 'Saving…' : 'Save'}
                        </button>
                      </div>
                    {:else}
                      <button
                        type="button"
                        class="menu-trigger"
                        aria-haspopup="true"
                        aria-expanded={openMenuId === r.id}
                        on:click|stopPropagation={(e) => {
                          if (openMenuId === r.id) {
                            openMenuId = null
                          } else {
                            const rect = e.currentTarget.getBoundingClientRect()
                            menuDropdownStyle = { left: rect.right, top: rect.bottom + 2 }
                            openMenuId = r.id
                          }
                        }}
                        title="Actions"
                      >
                        <Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" />
                      </button>
                      {#if openMenuId === r.id}
                        <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{menuDropdownStyle.left}px;top:{menuDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                          <button type="button" role="menuitem" on:click|stopPropagation={() => { startEdit(r); openMenuId = null }}>Edit</button>
                          <button type="button" role="menuitem" class="menu-item-danger" disabled={deletingId === r.id} on:click|stopPropagation={() => { handleDelete(r.id) }}>
                            {deletingId === r.id ? 'Deleting…' : 'Delete'}
                          </button>
                        </div>
                      {/if}
                    {/if}
                  </div>
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
  .edit-name-input {
    width: 100%;
    min-width: 140px;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.85rem;
  }
  .table-actions-wrap {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
  }
  .actions-menu-wrap {
    position: relative;
    display: inline-flex;
    justify-content: flex-end;
  }
  .menu-trigger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    padding: 0;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .menu-trigger:hover {
    color: var(--text);
    background: var(--table-row-hover);
  }
  .menu-dropdown {
    min-width: 7rem;
    padding: 0.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .menu-dropdown-fixed {
    /* position/left/top/transform/z-index set inline */
  }
  .menu-dropdown [role='menuitem'] {
    display: block;
    width: 100%;
    padding: 0.4rem 0.75rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.875rem;
    text-align: left;
    cursor: pointer;
    transition: background 0.15s;
  }
  .menu-dropdown [role='menuitem']:hover {
    background: var(--table-row-hover);
  }
  .menu-dropdown [role='menuitem']:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .menu-dropdown .menu-item-danger {
    color: var(--danger);
  }
  .menu-dropdown .menu-item-danger:hover {
    background: rgba(239, 68, 68, 0.1);
  }
</style>
