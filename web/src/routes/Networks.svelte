<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import CidrWizard from '../lib/CidrWizard.svelte'
  import SearchableSelect from '../lib/SearchableSelect.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { listEnvironments, listBlocks, listAllocations, createBlock, createAllocation, updateBlock, deleteBlock, deleteAllocation } from '../lib/api.js'

  export let environmentId = null
  export let orphanedOnly = false
  export let blockNameFilter = null
  export let openCreateBlockFromQuery = false
  export let openCreateAllocationFromQuery = false
  const dispatch = createEventDispatcher()

  let loading = true
  let error = ''
  let blocks = []
  let allocations = []
  let environments = []
  let envFilterName = null

  let openedCreateBlockFromQuery = false
  let openedCreateAllocFromQuery = false
  $: if (openCreateBlockFromQuery && !openedCreateBlockFromQuery) {
    showCreateBlock = true
    openedCreateBlockFromQuery = true
    dispatch('clearCreateQuery')
  } else if (!openCreateBlockFromQuery) {
    openedCreateBlockFromQuery = false
  }
  $: if (openCreateAllocationFromQuery && !openedCreateAllocFromQuery) {
    showCreateAlloc = true
    openedCreateAllocFromQuery = true
    dispatch('clearCreateQuery')
  } else if (!openCreateAllocationFromQuery) {
    openedCreateAllocFromQuery = false
  }

  let showCreateBlock = false
  let blockName = ''
  let blockCidr = ''
  let blockEnvironmentId = ''
  let blockSubmitting = false
  let blockError = ''

  let editingBlockId = null
  let editBlockName = ''
  let editBlockEnvironmentId = ''
  let editBlockSubmitting = false
  let editBlockError = ''

  let blockFilter = 'all' // 'all' | 'orphaned' | envId (when no URL env)
  let showCreateAlloc = false
  let allocName = ''
  let allocBlockName = ''
  let allocCidr = ''
  let allocSubmitting = false
  let allocError = ''

  let deleteBlockId = null
  let deleteBlockName = ''
  let deleteBlockAllocCount = 0
  let deleteBlockSubmitting = false
  let deleteBlockError = ''

  let deleteAllocId = null
  let deleteAllocName = ''
  let deleteAllocSubmitting = false
  let deleteAllocError = ''

  let errorModalMessage = ''

  let blockPage = 0
  let blockPageSize = 25
  let blockTotal = 0
  let allocPage = 0
  let allocPageSize = 25
  let allocTotal = 0

  let openBlockMenuId = null
  let openAllocMenuId = null
  let blockMenuTriggerEl = null
  let blockDropdownStyle = { left: 0, top: 0 }
  let allocMenuTriggerEl = null
  let allocDropdownStyle = { left: 0, top: 0 }

  function allocationCountForBlock(blockName) {
    if (!blockName) return 0
    const name = String(blockName).trim().toLowerCase()
    return allocations.filter((a) => (a.block_name || '').trim().toLowerCase() === name).length
  }

  async function load() {
    loading = true
    error = ''
    try {
      const blockOpts = { limit: blockPageSize, offset: blockPage * blockPageSize }
      if (blockNameFilter && String(blockNameFilter).trim() !== '') blockOpts.name = String(blockNameFilter).trim()
      if (effectiveFilter === 'orphaned') blockOpts.orphaned_only = true
      else if (effectiveFilter !== 'all') blockOpts.environment_id = effectiveFilter
      const allocOpts = { limit: allocPageSize, offset: allocPage * allocPageSize }
      if (blockNameFilter && String(blockNameFilter).trim() !== '') allocOpts.block_name = String(blockNameFilter).trim()
      const [envsRes, blksRes, allocsRes] = await Promise.all([
        listEnvironments(),
        listBlocks(blockOpts),
        listAllocations(allocOpts),
      ])
      environments = envsRes.environments
      blocks = blksRes.blocks
      blockTotal = blksRes.total
      allocations = allocsRes.allocations
      allocTotal = allocsRes.total
      envFilterName = environmentId
        ? (envsRes.environments.find((e) => e.id === environmentId)?.name ?? null)
        : null
    } catch (e) {
      error = e.message || 'Failed to load networks'
      errorModalMessage = error
    } finally {
      loading = false
    }
  }

  $: blockStart = blockTotal === 0 ? 0 : blockPage * blockPageSize + 1
  $: blockEnd = Math.min(blockPage * blockPageSize + blockPageSize, blockTotal)
  $: blockTotalPages = blockPageSize > 0 ? Math.ceil(blockTotal / blockPageSize) : 0
  $: allocStart = allocTotal === 0 ? 0 : allocPage * allocPageSize + 1
  $: allocEnd = Math.min(allocPage * allocPageSize + allocPageSize, allocTotal)
  $: allocTotalPages = allocPageSize > 0 ? Math.ceil(allocTotal / allocPageSize) : 0

  onMount(() => {
    load()
    function handleClickOutside(e) {
      if (!e.target.closest('.actions-menu-wrap')) {
        openBlockMenuId = null
        openAllocMenuId = null
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  })

  function envIdsMatch(a, b) {
    if (a == null || b == null) return false
    return String(a).toLowerCase() === String(b).toLowerCase()
  }

  const NIL_UUID = '00000000-0000-0000-0000-000000000000'
  function isOrphanedBlock(block) {
    const id = block.environment_id
    return id == null || id === '' || String(id).toLowerCase() === NIL_UUID
  }

  function getEnvironmentName(envId) {
    if (envId == null || envId === '') return null
    const env = environments.find((e) => envIdsMatch(e.id, envId))
    return env?.name ?? null
  }

  $: effectiveFilter = orphanedOnly ? 'orphaned' : (environmentId ?? blockFilter)

  let blockSortBy = 'name' // 'name' | 'environment' | 'usage'
  let blockSortDir = 'asc' // 'asc' | 'desc'

  function setBlockSort(column) {
    if (blockSortBy === column) {
      blockSortDir = blockSortDir === 'asc' ? 'desc' : 'asc'
    } else {
      blockSortBy = column
      blockSortDir = 'asc'
    }
  }

  $: displayedBlocks = (blockNameFilter != null && String(blockNameFilter).trim() !== '')
    ? blocks.filter((b) => String(b.name || '').trim().toLowerCase() === String(blockNameFilter).trim().toLowerCase())
    : blocks

  $: sortedBlocks = (() => {
    const list = [...displayedBlocks]
    const mult = blockSortDir === 'asc' ? 1 : -1
    if (blockSortBy === 'name') {
      list.sort((a, b) => mult * (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
    } else if (blockSortBy === 'environment') {
      list.sort((a, b) => {
        const na = getEnvironmentName(a.environment_id) ?? (isOrphanedBlock(a) ? 'Orphaned' : '')
        const nb = getEnvironmentName(b.environment_id) ?? (isOrphanedBlock(b) ? 'Orphaned' : '')
        return mult * na.localeCompare(nb, undefined, { sensitivity: 'base' })
      })
    } else if (blockSortBy === 'usage') {
      list.sort((a, b) => mult * (utilizationPercent(a) - utilizationPercent(b)))
    }
    return list
  })()

  $: displayedAllocations = allocations

  let allocSortBy = 'name' // 'name' | 'block'
  let allocSortDir = 'asc' // 'asc' | 'desc'

  function setAllocSort(column) {
    if (allocSortBy === column) {
      allocSortDir = allocSortDir === 'asc' ? 'desc' : 'asc'
    } else {
      allocSortBy = column
      allocSortDir = 'asc'
    }
  }

  $: sortedAllocations = (() => {
    const list = [...displayedAllocations]
    const mult = allocSortDir === 'asc' ? 1 : -1
    if (allocSortBy === 'name') {
      list.sort((a, b) => mult * (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
    } else if (allocSortBy === 'block') {
      list.sort((a, b) => mult * (a.block_name || '').localeCompare(b.block_name || '', undefined, { sensitivity: 'base' }))
    }
    return list
  })()

  $: allocParentCidr = (() => {
    const name = allocBlockName?.trim()
    if (!name) return ''
    const block = displayedBlocks.find((b) => b.name === name)
    return block?.cidr ?? ''
  })()

  $: allocBlockId = (() => {
    const name = allocBlockName?.trim()
    if (!name) return null
    const block = displayedBlocks.find((b) => b.name === name)
    return block?.id ?? null
  })()

  function utilizationPercent(block) {
    if (!block || block.total_ips === 0) return 0
    const raw = (block.used_ips / block.total_ips) * 100
    if (raw > 0 && raw < 1) return 1
    return Math.round(raw)
  }

  function utilizationPercentLabel(block) {
    if (!block || block.total_ips === 0) return '0%'
    const raw = (block.used_ips / block.total_ips) * 100
    if (block.used_ips > 0 && raw < 1) return '<1%'
    return Math.round(raw) + '%'
  }

  async function handleCreateBlock() {
    const name = blockName.trim()
    const cidr = blockCidr.trim()
    if (!name || !cidr) {
      blockError = 'Name and CIDR are required'
      errorModalMessage = blockError
      return
    }
    blockSubmitting = true
    blockError = ''
    try {
      const envId = blockEnvironmentId && blockEnvironmentId !== '' ? blockEnvironmentId : null
      await createBlock(name, cidr, envId)
      blockName = ''
      blockCidr = ''
      blockEnvironmentId = ''
      showCreateBlock = false
      await load()
    } catch (e) {
      blockError = e.message || 'Failed to create block'
      errorModalMessage = blockError
    } finally {
      blockSubmitting = false
    }
  }

  function openCreateBlock() {
    showCreateBlock = true
    blockError = ''
    blockName = ''
    blockCidr = ''
    blockEnvironmentId = environmentId || ''
  }

  function startEditBlock(block) {
    editingBlockId = block.id
    editBlockName = block.name
    editBlockEnvironmentId = isOrphanedBlock(block) ? '' : (block.environment_id ?? '')
    editBlockError = ''
  }

  function cancelEditBlock() {
    editingBlockId = null
    editBlockName = ''
    editBlockEnvironmentId = ''
    editBlockError = ''
  }

  async function handleUpdateBlock() {
    const name = editBlockName.trim()
    if (!name) {
      editBlockError = 'Name is required'
      errorModalMessage = editBlockError
      return
    }
    const block = blocks.find((b) => String(b.id) === String(editingBlockId))
    if (block) {
      const origName = (block.name || '').trim()
      const origEnvId = isOrphanedBlock(block) ? '' : String(block.environment_id ?? '')
      const newEnvId = editBlockEnvironmentId ? String(editBlockEnvironmentId) : ''
      if (origName === name && origEnvId === newEnvId) {
        cancelEditBlock()
        return
      }
    }
    editBlockSubmitting = true
    editBlockError = ''
    try {
      await updateBlock(editingBlockId, name, editBlockEnvironmentId)
      cancelEditBlock()
      await load()
    } catch (e) {
      editBlockError = e.message || 'Failed to update block'
      errorModalMessage = editBlockError
    } finally {
      editBlockSubmitting = false
    }
  }

  function clearEnvFilter() {
    blockFilter = 'all'
    dispatch('clearEnv')
    window.location.hash = 'networks'
  }

  function setFilter(value) {
    if (value === 'all') {
      blockFilter = 'all'
      dispatch('clearEnv')
      window.location.hash = 'networks'
    } else if (value === 'orphaned') {
      blockFilter = 'orphaned'
      dispatch('clearEnv')
      window.location.hash = 'networks'
    } else {
      blockFilter = value
      dispatch('setEnv', value)
    }
    blockPage = 0
    load()
  }

  async function handleCreateAllocation() {
    const name = allocName.trim()
    const block_name = allocBlockName.trim()
    const cidr = allocCidr.trim()
    if (!name || !block_name || !cidr) {
      allocError = 'Name, block name, and CIDR are required'
      errorModalMessage = allocError
      return
    }
    allocSubmitting = true
    allocError = ''
    try {
      await createAllocation(name, block_name, cidr)
      allocName = ''
      allocBlockName = ''
      allocCidr = ''
      showCreateAlloc = false
      await load()
    } catch (e) {
      allocError = e.message || 'Failed to create allocation'
      errorModalMessage = allocError
    } finally {
      allocSubmitting = false
    }
  }

  function openDeleteBlockConfirm(block) {
    deleteBlockId = block.id
    deleteBlockName = block.name
    deleteBlockAllocCount = allocationCountForBlock(block.name)
    deleteBlockError = ''
    deleteBlockSubmitting = false
  }

  function closeDeleteBlockConfirm() {
    deleteBlockId = null
    deleteBlockName = ''
    deleteBlockAllocCount = 0
    deleteBlockError = ''
  }

  async function handleDeleteBlock() {
    if (!deleteBlockId) return
    deleteBlockSubmitting = true
    deleteBlockError = ''
    try {
      await deleteBlock(deleteBlockId)
      closeDeleteBlockConfirm()
      await load()
    } catch (e) {
      deleteBlockError = e.message || 'Failed to delete block'
      errorModalMessage = deleteBlockError
    } finally {
      deleteBlockSubmitting = false
    }
  }

  function openDeleteAllocConfirm(alloc) {
    deleteAllocId = alloc.id
    deleteAllocName = alloc.name
    deleteAllocError = ''
    deleteAllocSubmitting = false
  }

  function closeDeleteAllocConfirm() {
    deleteAllocId = null
    deleteAllocName = ''
    deleteAllocError = ''
  }

  async function handleDeleteAllocation() {
    if (!deleteAllocId) return
    deleteAllocSubmitting = true
    deleteAllocError = ''
    try {
      await deleteAllocation(deleteAllocId)
      closeDeleteAllocConfirm()
      await load()
    } catch (e) {
      deleteAllocError = e.message || 'Failed to delete allocation'
      errorModalMessage = deleteAllocError
    } finally {
      deleteAllocSubmitting = false
    }
  }
</script>

<div class="networks">
  <header class="header">
    <div class="header-text">
      <h1>Networks</h1>
    </div>
  </header>

  <div class="filter-bar">
    <label class="filter-label">
      <span>Environment</span>
      <SearchableSelect
        options={[
          { value: 'all', label: 'All' },
          { value: 'orphaned', label: 'Orphaned only' },
          ...environments.map((e) => ({ value: String(e.id), label: e.name })),
        ]}
        value={String(effectiveFilter)}
        on:change={(e) => setFilter(e.detail)}
        placeholder="All"
      />
    </label>
    {#if effectiveFilter !== 'all' || blockNameFilter}
      <button type="button" class="btn btn-small" on:click={clearEnvFilter}>Show all</button>
    {/if}
  </div>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    <section class="section">
      <div class="section-header">
        <h2>Network blocks</h2>
        <button class="btn btn-primary" on:click={openCreateBlock}>Create block</button>
      </div>
      {#if showCreateBlock}
        <div class="form-card">
          <h3>New network block</h3>
          <form on:submit|preventDefault={handleCreateBlock}>
            <div class="wizard-display">
              <h4 class="wizard-heading">CIDR wizard</h4>
              <CidrWizard mode="block" environmentId={blockEnvironmentId || environmentId || null} bind:value={blockCidr} disabled={blockSubmitting} />
            </div>
            <div class="form-row">
              <label>
                <span>CIDR (from wizard or type here)</span>
                <input type="text" bind:value={blockCidr} placeholder="e.g. 10.0.0.0/8" disabled={blockSubmitting} />
              </label>
            </div>
            <div class="form-row">
              <label>
                <span>Name</span>
                <input type="text" bind:value={blockName} placeholder="e.g. prod-vpc" disabled={blockSubmitting} />
              </label>
              <label>
                <span>Environment</span>
                <SearchableSelect
                  options={[{ value: '', label: '— None —' }, ...environments.map((e) => ({ value: String(e.id), label: e.name }))]}
                  bind:value={blockEnvironmentId}
                  placeholder="— None —"
                  disabled={blockSubmitting}
                />
              </label>
            </div>
            <div class="form-actions">
              <button type="button" class="btn" on:click={() => (showCreateBlock = false)} disabled={blockSubmitting}>Cancel</button>
              <button type="submit" class="btn btn-primary" disabled={blockSubmitting}>
                {blockSubmitting ? 'Creating…' : 'Create'}
              </button>
            </div>
          </form>
        </div>
      {/if}
      {#if displayedBlocks.length === 0 && !showCreateBlock}
        <div class="empty">
          {#if effectiveFilter === 'orphaned'}
            No orphaned blocks.
          {:else if effectiveFilter !== 'all'}
            No blocks in this environment yet. Create one above.
          {:else}
            No blocks yet. Create one above.
          {/if}
        </div>
      {:else if displayedBlocks.length > 0}
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th class="sortable" class:sorted={blockSortBy === 'name'}>
                  <button type="button" class="th-sort" on:click={() => setBlockSort('name')}>
                    Name
                    {#if blockSortBy === 'name'}
                      <span class="sort-icon" aria-hidden="true">{blockSortDir === 'asc' ? '▲' : '▼'}</span>
                    {/if}
                  </button>
                </th>
                <th class="sortable" class:sorted={blockSortBy === 'environment'}>
                  <button type="button" class="th-sort" on:click={() => setBlockSort('environment')}>
                    Environment
                    {#if blockSortBy === 'environment'}
                      <span class="sort-icon" aria-hidden="true">{blockSortDir === 'asc' ? '▲' : '▼'}</span>
                    {/if}
                  </button>
                </th>
                <th>CIDR</th>
                <th>Total IPs</th>
                <th>Used</th>
                <th>Available</th>
                <th class="sortable" class:sorted={blockSortBy === 'usage'}>
                  <button type="button" class="th-sort" on:click={() => setBlockSort('usage')}>
                    Usage
                    {#if blockSortBy === 'usage'}
                      <span class="sort-icon" aria-hidden="true">{blockSortDir === 'asc' ? '▲' : '▼'}</span>
                    {/if}
                  </button>
                </th>
                <th class="actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each sortedBlocks as block}
                {@const blockRange = cidrRange(block.cidr)}
                <tr>
                  {#if editingBlockId === block.id}
                    <td colspan="7" class="edit-cell">
                      <form class="inline-edit" on:submit|preventDefault={handleUpdateBlock}>
                        <input type="text" bind:value={editBlockName} placeholder="Block name" disabled={editBlockSubmitting} />
                        <label class="inline-edit-env">
                          <span class="sr-only">Environment</span>
                          <SearchableSelect
                            options={[{ value: '', label: '— None —' }, ...environments.map((e) => ({ value: String(e.id), label: e.name }))]}
                            bind:value={editBlockEnvironmentId}
                            placeholder="— None —"
                            disabled={editBlockSubmitting}
                          />
                        </label>
                        <div class="inline-actions">
                          <button type="button" class="btn btn-small" on:click={cancelEditBlock} disabled={editBlockSubmitting}>Cancel</button>
                          <button type="submit" class="btn btn-primary btn-small" disabled={editBlockSubmitting}>
                            {editBlockSubmitting ? 'Saving…' : 'Save'}
                          </button>
                        </div>
                      </form>
                    </td>
                    <td class="actions"></td>
                  {:else}
                    <td class="name">{block.name}</td>
                    <td class="environment">
                      {#if isOrphanedBlock(block)}
                        <span class="tag tag-orphaned">Orphaned</span>
                      {:else}
                        {@const envName = getEnvironmentName(block.environment_id)}
                        <span class="tag tag-env">{envName ?? '—'}</span>
                      {/if}
                    </td>
                    <td class="cidr">
                      <code>{block.cidr}</code>
                      {#if blockRange}
                        <span class="cidr-range">{blockRange.start} – {blockRange.end}</span>
                      {/if}
                    </td>
                    <td class="num">{block.total_ips.toLocaleString()}</td>
                    <td class="num">{block.used_ips.toLocaleString()}</td>
                    <td class="num">{block.available_ips.toLocaleString()}</td>
                    <td>
                      <div class="usage-cell">
                        <div class="bar-wrap">
                          <div
                            class="bar"
                            class:high={utilizationPercent(block) >= 80}
                            class:mid={utilizationPercent(block) >= 50 && utilizationPercent(block) < 80}
                            style="width: {utilizationPercent(block)}%"
                          ></div>
                        </div>
                        <span class="pct">{utilizationPercentLabel(block)}</span>
                      </div>
                    </td>
                    <td class="actions">
                      <div class="actions-menu-wrap" role="group">
                        <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openBlockMenuId === block.id} on:click|stopPropagation={(e) => {
                          if (openBlockMenuId === block.id) {
                            openBlockMenuId = null;
                            blockMenuTriggerEl = null;
                          } else {
                            blockMenuTriggerEl = e.currentTarget;
                            const r = e.currentTarget.getBoundingClientRect();
                            blockDropdownStyle = { left: r.right, top: r.bottom + 2 };
                            openBlockMenuId = block.id;
                          }
                        }} title="Actions">⋮</button>
                        {#if openBlockMenuId === block.id}
                          <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{blockDropdownStyle.left}px;top:{blockDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                            <button type="button" role="menuitem" on:click|stopPropagation={() => { startEditBlock(block); openBlockMenuId = null }}>Edit</button>
                            <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteBlockConfirm(block); openBlockMenuId = null }}>Delete</button>
                          </div>
                        {/if}
                      </div>
                    </td>
                  {/if}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <div class="pagination">
          <span class="pagination-info">Showing {blockStart}–{blockEnd} of {blockTotal}</span>
          <div class="pagination-controls">
            <button type="button" class="btn btn-small" disabled={blockPage <= 0} on:click={() => { blockPage -= 1; load() }}>Previous</button>
            <span class="pagination-page">Page {blockPage + 1} of {blockTotalPages || 1}</span>
            <button type="button" class="btn btn-small" disabled={blockPage >= blockTotalPages - 1} on:click={() => { blockPage += 1; load() }}>Next</button>
          </div>
          <label class="page-size">
            <span>Per page</span>
            <select bind:value={blockPageSize} on:change={() => { blockPage = 0; load() }}>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
            </select>
          </label>
        </div>
      {/if}
    </section>

    <section class="section">
      <div class="section-header">
        <h2>Allocations</h2>
        <button class="btn btn-primary" on:click={() => { showCreateAlloc = true; allocError = ''; allocName = ''; allocBlockName = ''; allocCidr = '' }}>Create allocation</button>
      </div>
      {#if showCreateAlloc}
        <div class="form-card">
          <h3>New allocation</h3>
          <form on:submit|preventDefault={handleCreateAllocation}>
            <div class="form-row">
              <label id="searchable-select-label">
                <span>Block name</span>
                <SearchableSelect
                  options={[{ value: '', label: '— Select parent block —' }, ...displayedBlocks.map((b) => ({ value: b.name, label: b.name }))]}
                  bind:value={allocBlockName}
                  placeholder="— Select parent block —"
                  disabled={allocSubmitting}
                />
              </label>
            </div>
            <div class="wizard-display">
              <h4 class="wizard-heading">CIDR wizard</h4>
              <CidrWizard mode="allocation" parentCidr={allocParentCidr} blockId={allocBlockId} bind:value={allocCidr} disabled={allocSubmitting} />
            </div>
            <div class="form-row">
              <label>
                <span>CIDR (from wizard or type here)</span>
                <input type="text" bind:value={allocCidr} placeholder="e.g. 10.0.1.0/24" disabled={allocSubmitting} />
              </label>
            </div>
            <div class="form-row">
              <label>
                <span>Name</span>
                <input type="text" bind:value={allocName} placeholder="e.g. app-subnet" disabled={allocSubmitting} />
              </label>
            </div>
            <div class="form-actions">
              <button type="button" class="btn" on:click={() => (showCreateAlloc = false)} disabled={allocSubmitting}>Cancel</button>
              <button type="submit" class="btn btn-primary" disabled={allocSubmitting}>
                {allocSubmitting ? 'Creating…' : 'Create'}
              </button>
            </div>
          </form>
        </div>
      {/if}
      {#if displayedAllocations.length === 0 && !showCreateAlloc}
        <div class="empty">
          {#if effectiveFilter !== 'all'}
            No allocations in the filtered blocks yet.
          {:else}
            No allocations yet. Create one above.
          {/if}
        </div>
      {:else if displayedAllocations.length > 0}
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th class="sortable" class:sorted={allocSortBy === 'name'}>
                  <button type="button" class="th-sort" on:click={() => setAllocSort('name')}>
                    Name
                    {#if allocSortBy === 'name'}
                      <span class="sort-icon" aria-hidden="true">{allocSortDir === 'asc' ? '▲' : '▼'}</span>
                    {/if}
                  </button>
                </th>
                <th class="sortable" class:sorted={allocSortBy === 'block'}>
                  <button type="button" class="th-sort" on:click={() => setAllocSort('block')}>
                    Block
                    {#if allocSortBy === 'block'}
                      <span class="sort-icon" aria-hidden="true">{allocSortDir === 'asc' ? '▲' : '▼'}</span>
                    {/if}
                  </button>
                </th>
                <th>CIDR</th>
                <th class="actions">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each sortedAllocations as alloc}
                {@const allocRange = cidrRange(alloc.cidr)}
                <tr>
                  <td class="name">{alloc.name}</td>
                  <td>{alloc.block_name}</td>
                  <td class="cidr">
                    <code>{alloc.cidr}</code>
                    {#if allocRange}
                      <span class="cidr-range">{allocRange.start} – {allocRange.end}</span>
                    {/if}
                  </td>
                  <td class="actions">
                    <div class="actions-menu-wrap" role="group">
                      <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openAllocMenuId === alloc.id} on:click|stopPropagation={(e) => {
                        if (openAllocMenuId === alloc.id) {
                          openAllocMenuId = null;
                          allocMenuTriggerEl = null;
                        } else {
                          allocMenuTriggerEl = e.currentTarget;
                          const r = e.currentTarget.getBoundingClientRect();
                          allocDropdownStyle = { left: r.right, top: r.bottom + 2 };
                          openAllocMenuId = alloc.id;
                        }
                      }} title="Actions">⋮</button>
                      {#if openAllocMenuId === alloc.id}
                        <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{allocDropdownStyle.left}px;top:{allocDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                          <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteAllocConfirm(alloc); openAllocMenuId = null }}>Delete</button>
                        </div>
                      {/if}
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <div class="pagination">
          <span class="pagination-info">Showing {allocStart}–{allocEnd} of {allocTotal}</span>
          <div class="pagination-controls">
            <button type="button" class="btn btn-small" disabled={allocPage <= 0} on:click={() => { allocPage -= 1; load() }}>Previous</button>
            <span class="pagination-page">Page {allocPage + 1} of {allocTotalPages || 1}</span>
            <button type="button" class="btn btn-small" disabled={allocPage >= allocTotalPages - 1} on:click={() => { allocPage += 1; load() }}>Next</button>
          </div>
          <label class="page-size">
            <span>Per page</span>
            <select bind:value={allocPageSize} on:change={() => { allocPage = 0; load() }}>
              <option value={10}>10</option>
              <option value={25}>25</option>
              <option value={50}>50</option>
            </select>
          </label>
        </div>
      {/if}
    </section>

    {#if deleteBlockId}
      <div class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="delete-block-dialog-title">
        <div class="modal">
          <h3 id="delete-block-dialog-title">Delete network block</h3>
          <p class="modal-warning">
            <strong>This will permanently delete the block “{deleteBlockName}”.</strong>
            {#if deleteBlockAllocCount > 0}
              <br /><span class="block-count">This block contains {deleteBlockAllocCount} allocation(s). Deleting the block will also delete all of them (cascading delete).</span>
            {/if}
            This action cannot be undone.
          </p>
          <div class="modal-actions">
            <button type="button" class="btn" on:click={closeDeleteBlockConfirm} disabled={deleteBlockSubmitting}>Cancel</button>
            <button type="button" class="btn btn-danger" on:click={handleDeleteBlock} disabled={deleteBlockSubmitting}>
              {deleteBlockSubmitting ? 'Deleting…' : 'Delete block'}
            </button>
          </div>
        </div>
      </div>
    {/if}

    {#if deleteAllocId}
      <div class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="delete-alloc-dialog-title">
        <div class="modal">
          <h3 id="delete-alloc-dialog-title">Delete allocation</h3>
          <p class="modal-warning">
            <strong>This will permanently delete the allocation “{deleteAllocName}”.</strong>
            This action cannot be undone.
          </p>
          <div class="modal-actions">
            <button type="button" class="btn" on:click={closeDeleteAllocConfirm} disabled={deleteAllocSubmitting}>Cancel</button>
            <button type="button" class="btn btn-danger" on:click={handleDeleteAllocation} disabled={deleteAllocSubmitting}>
              {deleteAllocSubmitting ? 'Deleting…' : 'Delete allocation'}
            </button>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  {#if errorModalMessage}
    <ErrorModal message={errorModalMessage} on:close={() => (errorModalMessage = '')} />
  {/if}
</div>

<style>
  .networks {
    padding-top: 0.5rem;
  }
  .header {
    margin-bottom: 1.5rem;
  }
  .header h1 {
    margin: 0;
    font-size: 1.35rem;
    font-weight: 600;
    letter-spacing: -0.02em;
  }
  .loading {
    color: var(--text-muted);
    padding: 2rem;
  }
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
  }
  .filter-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0;
  }
  .filter-label span {
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .filter-label :global(.searchable-select) {
    min-width: 160px;
  }
  .filter-select {
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    min-width: 160px;
  }
  .filter-select:focus {
    outline: none;
    border-color: var(--accent);
  }
  .btn-small {
    padding: 0.35rem 0.75rem;
    font-size: 0.85rem;
  }
  .form-card select {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    color-scheme: dark;
  }
  :global([data-theme='light']) .form-card select {
    color-scheme: light;
  }
  .form-card select:focus {
    outline: none;
    border-color: var(--accent);
  }
  .filter-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }
  .filter-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .filter-label span {
    white-space: nowrap;
  }
  .filter-input {
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-size: 0.9rem;
    min-width: 10rem;
  }
  .pagination {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 1rem;
    flex-wrap: wrap;
  }
  .pagination-info {
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .pagination-controls {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .pagination-page {
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .page-size {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .page-size select {
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-size: 0.9rem;
  }
  .section {
    margin-bottom: 2rem;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 0.75rem;
  }
  .section h2 {
    margin: 0;
    font-size: 0.95rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .btn {
    padding: 0.5rem 1rem;
    border-radius: var(--radius);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    transition: background 0.15s, border-color 0.15s;
  }
  .btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.06);
    border-color: var(--text-muted);
  }
  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .btn-primary {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--btn-primary-text);
  }
  .btn-primary:hover:not(:disabled) {
    background: var(--btn-primary-hover-bg);
    border-color: var(--btn-primary-hover-border);
  }
  .btn-danger {
    background: rgba(248, 81, 73, 0.2);
    border-color: var(--danger);
    color: #ff7b72;
  }
  .btn-danger:hover:not(:disabled) {
    background: rgba(248, 81, 73, 0.35);
    border-color: #ff7b72;
  }
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    padding: 1rem;
  }
  .modal {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    padding: 1.5rem;
    max-width: 420px;
    width: 100%;
  }
  .modal h3 {
    margin: 0 0 1rem 0;
    font-size: 1.1rem;
    font-weight: 600;
  }
  .modal-warning {
    margin: 0 0 1rem 0;
    font-size: 0.9rem;
    line-height: 1.5;
    color: var(--text-muted);
  }
  .modal-warning strong {
    color: var(--text);
  }
  .modal-warning .block-count {
    color: var(--warn);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  .form-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    padding: 1.25rem;
    margin-bottom: 1rem;
  }
  .form-card h3 {
    margin: 0 0 1rem 0;
    font-size: 1rem;
    font-weight: 600;
  }
  .wizard-display {
    margin: 0 0 1rem 0;
  }
  .wizard-heading {
    margin: 0 0 0.5rem 0;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-muted, #6c757d);
  }
  .form-row {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  .form-card label {
    display: block;
    flex: 1;
    min-width: 140px;
  }
  .form-card label span {
    display: block;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
    margin-bottom: 0.35rem;
  }
  .form-card input {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
  }
  .form-card input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .form-actions {
    display: flex;
    gap: 0.5rem;
  }
  .empty {
    color: var(--text-muted);
    padding: 1rem 0;
  }
  .table-wrap {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    overflow-x: auto;
  }
  .table {
    width: 100%;
    min-width: max-content;
    border-collapse: collapse;
  }
  .table th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
    background: var(--table-header-bg);
    border-bottom: 1px solid var(--border);
  }
  .table th.sortable {
    padding: 0;
  }
  .table th .th-sort {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    width: 100%;
    padding: 0.75rem 1rem;
    text-align: left;
    font-size: inherit;
    font-weight: inherit;
    text-transform: inherit;
    letter-spacing: inherit;
    color: inherit;
    background: none;
    border: none;
    cursor: pointer;
    font-family: inherit;
    transition: color 0.15s, background 0.15s;
  }
  .table th .th-sort:hover {
    color: var(--text);
    background: rgba(255, 255, 255, 0.04);
  }
  .table th.sortable.sorted .th-sort {
    color: var(--accent);
  }
  .table th .sort-icon {
    font-size: 0.65rem;
    opacity: 0.9;
  }
  .table td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .table tr:last-child td {
    border-bottom: none;
  }
  .table tr:hover td {
    background: var(--table-row-hover);
  }
  .table .actions {
    text-align: right;
    white-space: nowrap;
    min-width: 120px;
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
    font-size: 1.1rem;
    line-height: 1;
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
    /* position/left/top/transform/z-index set inline for fixed positioning above table */
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
  .menu-dropdown .menu-item-danger {
    color: var(--danger);
  }
  .menu-dropdown .menu-item-danger:hover {
    background: rgba(239, 68, 68, 0.1);
  }
  .edit-cell {
    padding: 0.5rem 1rem;
  }
  .inline-edit {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }
  .inline-edit input {
    max-width: 320px;
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
  }
  .inline-edit input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .inline-edit-env {
    margin: 0;
  }
  .inline-edit select {
    padding: 0.4rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    min-width: 140px;
  }
  .inline-edit select:focus {
    outline: none;
    border-color: var(--accent);
  }
  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .inline-actions {
    display: flex;
    gap: 0.35rem;
  }
  .table thead th:first-child {
    min-width: 160px;
  }
  .table td.name {
    min-width: 160px;
  }
  .name {
    font-weight: 500;
  }
  .name-cell {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .tag {
    display: inline-block;
    padding: 0.15rem 0.5rem;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }
  .tag-orphaned {
    background: rgba(210, 153, 34, 0.2);
    border: 1px solid var(--warn);
    color: var(--warn);
  }
  .tag-env {
    background: rgba(88, 166, 255, 0.15);
    border: 1px solid var(--accent);
    color: var(--accent);
  }
  .cidr {
    vertical-align: top;
  }
  .cidr code,
  .id code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .cidr-range {
    display: block;
    font-size: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    margin-top: 0.2rem;
  }
  .num {
    font-variant-numeric: tabular-nums;
  }
  .usage-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 100px;
  }
  .bar-wrap {
    flex: 1;
    height: 6px;
    background: var(--border);
    border-radius: 3px;
    overflow: hidden;
  }
  .bar {
    height: 100%;
    border-radius: 3px;
    background: var(--success);
    transition: width 0.2s;
  }
  .bar.mid {
    background: var(--warn);
  }
  .bar.high {
    background: var(--danger);
  }
  .pct {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    min-width: 2.5em;
  }
</style>
