<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import { tick } from 'svelte'
  import Icon from '@iconify/svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import CidrWizard from '../lib/CidrWizard.svelte'
  import DataTable from '../lib/DataTable.svelte'
  import SearchableSelect from '../lib/SearchableSelect.svelte'
  import { cidrRange, parseCidrToInt } from '../lib/cidr.js'
  import { formatBlockCount, compareBlockCount, utilizationPercent as utilPct } from '../lib/blockCount.js'
  import { user, selectedOrgForGlobalAdmin, isGlobalAdmin } from '../lib/auth.js'
  import { listEnvironments, listBlocks, listAllocations, createBlock, createAllocation, updateBlock, updateAllocation, deleteBlock, deleteAllocation } from '../lib/api.js'

  export let environmentId = null
  export let orphanedOnly = false
  export let blockNameFilter = null
  export let allocationFilter = null
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
    openedCreateAllocFromQuery = true
    dispatch('clearCreateQuery')
    if (blocks.length > 0) showCreateAlloc = true
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

  let editingAllocId = null
  let editAllocName = ''
  let editAllocSubmitting = false
  let editAllocError = ''

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

  let blockFilterOptions = []
  let allocationFilterOptions = []
  let allocationBlockNamesFromFilter = null

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

  function listOpts(extra = {}) {
    const o = { ...extra }
    if (isGlobalAdmin($user) && $selectedOrgForGlobalAdmin) o.organization_id = $selectedOrgForGlobalAdmin
    return o
  }

  async function load() {
    loading = true
    error = ''
    try {
      const blockOpts = listOpts({ limit: blockPageSize, offset: blockPage * blockPageSize })
      if (blockNameFilter && String(blockNameFilter).trim() !== '') blockOpts.name = String(blockNameFilter).trim()
      if (effectiveFilter === 'orphaned') blockOpts.orphaned_only = true
      else if (effectiveFilter !== 'all') blockOpts.environment_id = effectiveFilter
      const blockFilterOpts = listOpts({ limit: 500, offset: 0 })
      if (effectiveFilter === 'orphaned') blockFilterOpts.orphaned_only = true
      else if (effectiveFilter !== 'all') blockFilterOpts.environment_id = effectiveFilter
      const allocOpts = listOpts({ limit: allocPageSize, offset: allocPage * allocPageSize })
      if (blockNameFilter && String(blockNameFilter).trim() !== '') allocOpts.block_name = String(blockNameFilter).trim()
      if (allocationFilter && String(allocationFilter).trim() !== '') allocOpts.name = String(allocationFilter).trim()
      if (effectiveFilter !== 'all' && effectiveFilter !== 'orphaned') allocOpts.environment_id = effectiveFilter
      const allocOptionsOpts = listOpts({ limit: 500, offset: 0 })
      if (blockNameFilter && String(blockNameFilter).trim() !== '') allocOptionsOpts.block_name = String(blockNameFilter).trim()
      if (effectiveFilter !== 'all' && effectiveFilter !== 'orphaned') allocOptionsOpts.environment_id = effectiveFilter
      const promises = [
        listEnvironments(listOpts()),
        listBlocks(blockOpts),
        listAllocations(allocOpts),
        listBlocks(blockFilterOpts),
        listAllocations(allocOptionsOpts),
      ]
      if (allocationFilter && String(allocationFilter).trim() !== '') {
        const allocByNameOpts = listOpts({ limit: 500, name: String(allocationFilter).trim() })
        if (effectiveFilter !== 'all' && effectiveFilter !== 'orphaned') allocByNameOpts.environment_id = effectiveFilter
        promises.push(listAllocations(allocByNameOpts))
      }
      const results = await Promise.all(promises)
      const envsRes = results[0]
      const blksRes = results[1]
      const allocsRes = results[2]
      const blockNamesRes = results[3]
      const allocOptionsRes = results[4]
      const allocBlockNamesRes = results.length > 5 ? results[5] : null
      environments = envsRes.environments
      blocks = blksRes.blocks
      blockTotal = blksRes.total
      allocations = allocsRes.allocations
      allocTotal = allocsRes.total
      blockFilterOptions = blockNamesRes.blocks.map((b) => ({ value: b.name, label: b.name }))
      const blockNamesSet = new Set(blockNamesRes.blocks.map((b) => (b.name || '').trim().toLowerCase()))
      allocationFilterOptions = (allocOptionsRes.allocations || [])
        .filter((a) => blockNamesSet.has((a.block_name || '').trim().toLowerCase()))
        .map((a) => ({ value: a.name, label: a.name }))
      if (allocationFilter && String(allocationFilter).trim() !== '') {
        if (allocBlockNamesRes && allocBlockNamesRes.allocations && allocBlockNamesRes.allocations.length > 0) {
          allocationBlockNamesFromFilter = new Set(allocBlockNamesRes.allocations.map((a) => (a.block_name || '').trim()))
        } else {
          allocationBlockNamesFromFilter = new Set()
        }
      } else {
        allocationBlockNamesFromFilter = null
      }
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

  $: effectiveFilter, blockNameFilter, allocationFilter, $selectedOrgForGlobalAdmin, (blockPage = 0, allocPage = 0, load())

  onMount(() => {
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

  let blockSortBy = 'name' // 'name' | 'environment' | 'cidr' | 'total_ips' | 'used_ips' | 'available_ips' | 'usage'
  let blockSortDir = 'asc' // 'asc' | 'desc'

  function setBlockSort(column) {
    if (blockSortBy === column) {
      blockSortDir = blockSortDir === 'asc' ? 'desc' : 'asc'
    } else {
      blockSortBy = column
      blockSortDir = 'asc'
    }
  }

  $: displayedBlocks = (() => {
    if (allocationFilter && String(allocationFilter).trim() !== '' && allocationBlockNamesFromFilter) {
      return blocks.filter((b) => allocationBlockNamesFromFilter.has((b.name || '').trim()))
    }
    if (blockNameFilter != null && String(blockNameFilter).trim() !== '') {
      return blocks.filter((b) => String(b.name || '').trim().toLowerCase() === String(blockNameFilter).trim().toLowerCase())
    }
    return blocks
  })()

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
    } else if (blockSortBy === 'cidr') {
      list.sort((a, b) => {
        const pa = parseCidrToInt(a.cidr)
        const pb = parseCidrToInt(b.cidr)
        if (!pa && !pb) return mult * (a.cidr || '').localeCompare(b.cidr || '', undefined, { sensitivity: 'base' })
        if (!pa) return 1
        if (!pb) return -1
        return mult * (pa.baseInt - pb.baseInt)
      })
    } else if (blockSortBy === 'total_ips') {
      list.sort((a, b) => mult * compareBlockCount(a.total_ips, b.total_ips))
    } else if (blockSortBy === 'used_ips') {
      list.sort((a, b) => mult * compareBlockCount(a.used_ips, b.used_ips))
    } else if (blockSortBy === 'available_ips') {
      list.sort((a, b) => mult * compareBlockCount(a.available_ips, b.available_ips))
    } else if (blockSortBy === 'usage') {
      list.sort((a, b) => mult * (utilizationPercent(a) - utilizationPercent(b)))
    }
    return list
  })()

  $: envBlockNames =
    effectiveFilter && effectiveFilter !== 'all' && blocks.length > 0
      ? new Set(blocks.map((b) => (b.name || '').trim().toLowerCase()))
      : null

  $: displayedAllocations = (() => {
    if (allocationFilter && String(allocationFilter).trim() !== '') {
      return allocations.filter((a) => (a.name || '').trim() === (allocationFilter || '').trim())
    }
    if (envBlockNames) {
      return allocations.filter((a) => envBlockNames.has((a.block_name || '').trim().toLowerCase()))
    }
    return allocations
  })()

  let allocSortBy = 'name' // 'name' | 'block' | 'cidr'
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
    } else if (allocSortBy === 'cidr') {
      list.sort((a, b) => {
        const pa = parseCidrToInt(a.cidr)
        const pb = parseCidrToInt(b.cidr)
        if (!pa && !pb) return mult * (a.cidr || '').localeCompare(b.cidr || '', undefined, { sensitivity: 'base' })
        if (!pa) return 1
        if (!pb) return -1
        return mult * (pa.baseInt - pb.baseInt)
      })
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
    if (!block) return 0
    const p = utilPct(block.total_ips, block.used_ips)
    if (p > 0 && p < 1) return 1
    return Math.round(p)
  }

  function utilizationPercentLabel(block) {
    if (!block) return '0%'
    const p = utilPct(block.total_ips, block.used_ips)
    if (compareBlockCount(block.used_ips, '0') > 0 && p < 1) return '<1%'
    return Math.round(p) + '%'
  }

  async function handleCreateBlock() {
    const name = blockName.trim()
    const cidr = blockCidr.trim()
    if (!name || !cidr) {
      blockError = 'Name and CIDR are required'
      errorModalMessage = blockError
      return
    }
    const envId = blockEnvironmentId && blockEnvironmentId !== '' ? blockEnvironmentId : null
    if (!envId && isGlobalAdmin($user) && (!$selectedOrgForGlobalAdmin || $selectedOrgForGlobalAdmin === '')) {
      blockError = 'Select an organization for orphan blocks (blocks without an environment)'
      errorModalMessage = blockError
      return
    }
    const orgId = !envId && isGlobalAdmin($user) && $selectedOrgForGlobalAdmin ? $selectedOrgForGlobalAdmin : null
    blockSubmitting = true
    blockError = ''
    try {
      await createBlock(name, cidr, envId, orgId)
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
    const newEnvId = editBlockEnvironmentId ? String(editBlockEnvironmentId) : ''
    if (newEnvId === '' && isGlobalAdmin($user) && (!$selectedOrgForGlobalAdmin || $selectedOrgForGlobalAdmin === '')) {
      editBlockError = 'Select an organization for orphan blocks'
      errorModalMessage = editBlockError
      return
    }
    const block = blocks.find((b) => String(b.id) === String(editingBlockId))
    if (block) {
      const origName = (block.name || '').trim()
      const origEnvId = isOrphanedBlock(block) ? '' : String(block.environment_id ?? '')
      if (origName === name && origEnvId === newEnvId) {
        cancelEditBlock()
        return
      }
    }
    const orgId = newEnvId === '' && isGlobalAdmin($user) && $selectedOrgForGlobalAdmin ? $selectedOrgForGlobalAdmin : null
    editBlockSubmitting = true
    editBlockError = ''
    try {
      await updateBlock(editingBlockId, name, editBlockEnvironmentId || '', orgId)
      cancelEditBlock()
      await load()
    } catch (e) {
      editBlockError = e.message || 'Failed to update block'
      errorModalMessage = editBlockError
    } finally {
      editBlockSubmitting = false
    }
  }

  async function clearEnvFilter() {
    blockFilter = 'all'
    blockPage = 0
    dispatch('clearEnv')
    window.location.hash = 'networks'
    await tick()
    load()
  }

  async function clearAllFilters() {
    blockFilter = 'all'
    blockPage = 0
    allocPage = 0
    dispatch('clearEnv')
    await tick()
    load()
  }

  async function setFilter(value) {
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
    await tick()
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

  function startEditAllocation(alloc) {
    editingAllocId = alloc.id
    editAllocName = alloc.name
    editAllocError = ''
  }

  function cancelEditAllocation() {
    editingAllocId = null
    editAllocName = ''
    editAllocError = ''
  }

  async function handleUpdateAllocation() {
    const name = editAllocName.trim()
    if (!name) {
      editAllocError = 'Name is required'
      errorModalMessage = editAllocError
      return
    }
    const alloc = allocations.find((a) => String(a.id) === String(editingAllocId))
    if (alloc && (alloc.name || '').trim() === name) {
      cancelEditAllocation()
      return
    }
    editAllocSubmitting = true
    editAllocError = ''
    try {
      await updateAllocation(editingAllocId, name)
      cancelEditAllocation()
      await load()
    } catch (e) {
      editAllocError = e.message || 'Failed to update allocation'
      errorModalMessage = editAllocError
    } finally {
      editAllocSubmitting = false
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
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Networks</h1>
      <p class="page-desc">Network blocks define your IP ranges; allocations are subnets within those blocks.</p>
    </div>
  </header>

  <div class="filter-bar">
    <div class="filter-label" role="group" aria-label="Environment">
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
    </div>
    <div class="filter-label" role="group" aria-label="Block">
      <span>Block</span>
      <SearchableSelect
        options={[{ value: '', label: 'All' }, ...blockFilterOptions]}
        value={blockNameFilter != null && String(blockNameFilter).trim() !== '' ? String(blockNameFilter).trim() : ''}
        on:change={(e) => dispatch('setBlockFilter', { block: e.detail === '' ? null : e.detail })}
        placeholder="All"
      />
    </div>
    <div class="filter-label" role="group" aria-label="Allocation">
      <span>Allocation</span>
      <SearchableSelect
        options={[{ value: '', label: 'All' }, ...allocationFilterOptions]}
        value={allocationFilter != null && String(allocationFilter).trim() !== '' ? String(allocationFilter).trim() : ''}
        on:change={(e) => dispatch('setAllocationFilter', { allocation: e.detail === '' ? null : e.detail })}
        placeholder="All"
      />
    </div>
    {#if effectiveFilter !== 'all' || blockNameFilter || (allocationFilter && String(allocationFilter).trim() !== '')}
      <button type="button" class="btn btn-small" on:click={clearAllFilters}>Show all</button>
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
              <label for="create-block-cidr">
                <span>CIDR (from wizard or type here)</span>
                <input id="create-block-cidr" type="text" bind:value={blockCidr} placeholder="e.g. 10.0.0.0/8" disabled={blockSubmitting} />
              </label>
            </div>
            <div class="form-row">
              <label for="create-block-name">
                <span>Name</span>
                <input id="create-block-name" type="text" bind:value={blockName} placeholder="e.g. prod-vpc" disabled={blockSubmitting} />
              </label>
              <div class="form-label-wrap" role="group" aria-label="Environment">
                <span>Environment</span>
                <SearchableSelect
                  options={[{ value: '', label: '— None —' }, ...environments.map((e) => ({ value: String(e.id), label: e.name }))]}
                  bind:value={blockEnvironmentId}
                  placeholder="— None —"
                  disabled={blockSubmitting}
                />
              </div>
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
      {#if !showCreateBlock}
        <DataTable>
          <svelte:fragment slot="header">
            <tr>
              <th class="sortable" class:sorted={blockSortBy === 'name'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('name')}>
                  <span class="th-sort-label">Name</span>
                  {#if blockSortBy === 'name'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'environment'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('environment')}>
                  <span class="th-sort-label">Environment</span>
                  {#if blockSortBy === 'environment'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'cidr'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('cidr')}>
                  <span class="th-sort-label">CIDR</span>
                  {#if blockSortBy === 'cidr'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'total_ips'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('total_ips')}>
                  <span class="th-sort-label">Total IPs</span>
                  {#if blockSortBy === 'total_ips'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'used_ips'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('used_ips')}>
                  <span class="th-sort-label">Used</span>
                  {#if blockSortBy === 'used_ips'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'available_ips'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('available_ips')}>
                  <span class="th-sort-label">Available</span>
                  {#if blockSortBy === 'available_ips'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={blockSortBy === 'usage'}>
                <button type="button" class="th-sort" on:click={() => setBlockSort('usage')}>
                  <span class="th-sort-label">Usage</span>
                  {#if blockSortBy === 'usage'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={blockSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="actions">Actions</th>
            </tr>
          </svelte:fragment>
          <svelte:fragment slot="body">
            {#if displayedBlocks.length === 0}
              <tr>
                <td colspan="7" class="table-empty-cell">
                  {#if effectiveFilter === 'orphaned'}
                    No orphaned blocks.
                  {:else if effectiveFilter !== 'all'}
                    No blocks in this environment yet. Create one above.
                  {:else}
                    No blocks yet. Create one above.
                  {/if}
                </td>
              </tr>
            {:else}
              {#each sortedBlocks as block}
                {@const blockRange = cidrRange(block.cidr)}
                <tr>
                  {#if editingBlockId === block.id}
                    <td colspan="7" class="edit-cell">
                      <form class="inline-edit" on:submit|preventDefault={handleUpdateBlock}>
                        <input type="text" bind:value={editBlockName} placeholder="Block name" disabled={editBlockSubmitting} />
                        <div class="inline-edit-env" role="group" aria-label="Environment">
                          <span class="sr-only">Environment</span>
                          <SearchableSelect
                            options={[{ value: '', label: '— None —' }, ...environments.map((e) => ({ value: String(e.id), label: e.name }))]}
                            bind:value={editBlockEnvironmentId}
                            placeholder="— None —"
                            disabled={editBlockSubmitting}
                          />
                        </div>
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
                    <td class="num">{formatBlockCount(block.total_ips)}</td>
                    <td class="num">{formatBlockCount(block.used_ips)}</td>
                    <td class="num">{formatBlockCount(block.available_ips)}</td>
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
                        }} title="Actions"><Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" /></button>
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
            {/if}
          </svelte:fragment>
        </DataTable>
        {#if displayedBlocks.length > 0}
          <div class="pagination">
            <span class="pagination-info">Showing {blockStart}–{blockEnd} of {blockTotal}</span>
            <div class="pagination-controls">
              <button type="button" class="btn btn-small" disabled={blockPage <= 0} on:click={() => { blockPage -= 1; load() }}>Previous</button>
              <span class="pagination-page">Page {blockPage + 1} of {blockTotalPages || 1}</span>
              <button type="button" class="btn btn-small" disabled={blockPage >= blockTotalPages - 1} on:click={() => { blockPage += 1; load() }}>Next</button>
            </div>
            <label class="page-size" for="block-page-size">
              <span>Per page</span>
              <select id="block-page-size" bind:value={blockPageSize} on:change={() => { blockPage = 0; load() }}>
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
              </select>
            </label>
          </div>
        {/if}
      {/if}
    </section>

    <section class="section">
      <div class="section-header">
        <h2>Allocations</h2>
        <button class="btn btn-primary" disabled={blocks.length === 0} on:click={() => { showCreateAlloc = true; allocError = ''; allocName = ''; allocBlockName = ''; allocCidr = '' }} title={blocks.length === 0 ? 'Create a network block first' : ''}>Create allocation</button>
      </div>
      {#if showCreateAlloc && blocks.length > 0}
        <div class="form-card">
          <h3>New allocation</h3>
          <form on:submit|preventDefault={handleCreateAllocation}>
            <div class="form-row">
              <div role="group" aria-label="Block name" id="searchable-select-label">
                <span>Block name</span>
                <SearchableSelect
                  options={[{ value: '', label: '— Select parent block —' }, ...displayedBlocks.map((b) => ({ value: b.name, label: b.name }))]}
                  bind:value={allocBlockName}
                  placeholder="— Select parent block —"
                  disabled={allocSubmitting}
                />
              </div>
            </div>
            <div class="wizard-display">
              <h4 class="wizard-heading">CIDR wizard</h4>
              <CidrWizard mode="allocation" parentCidr={allocParentCidr} blockId={allocBlockId} bind:value={allocCidr} disabled={allocSubmitting} />
            </div>
            <div class="form-row">
              <label for="create-alloc-cidr">
                <span>CIDR (from wizard or type here)</span>
                <input id="create-alloc-cidr" type="text" bind:value={allocCidr} placeholder="e.g. 10.0.1.0/24" disabled={allocSubmitting} />
              </label>
            </div>
            <div class="form-row">
              <label for="create-alloc-name">
                <span>Name</span>
                <input id="create-alloc-name" type="text" bind:value={allocName} placeholder="e.g. app-subnet" disabled={allocSubmitting} />
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
      {:else}
        <DataTable>
          <svelte:fragment slot="header">
            <tr>
                <th class="sortable" class:sorted={allocSortBy === 'name'}>
                <button type="button" class="th-sort" on:click={() => setAllocSort('name')}>
                  <span class="th-sort-label">Name</span>
                  {#if allocSortBy === 'name'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={allocSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={allocSortBy === 'block'}>
                <button type="button" class="th-sort" on:click={() => setAllocSort('block')}>
                  <span class="th-sort-label">Block</span>
                  {#if allocSortBy === 'block'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={allocSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={allocSortBy === 'cidr'}>
                <button type="button" class="th-sort" on:click={() => setAllocSort('cidr')}>
                  <span class="th-sort-label">CIDR</span>
                  {#if allocSortBy === 'cidr'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={allocSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="actions">Actions</th>
            </tr>
          </svelte:fragment>
          <svelte:fragment slot="body">
            {#if blocks.length === 0}
              <tr>
                <td colspan="4" class="table-empty-cell">Create a network block above before adding allocations.</td>
              </tr>
            {:else if displayedAllocations.length === 0}
              <tr>
                <td colspan="4" class="table-empty-cell">
                  {#if allocationFilter && String(allocationFilter).trim() !== ''}
                    No allocations match the allocation filter.
                  {:else if effectiveFilter !== 'all'}
                    No allocations in the filtered blocks yet.
                  {:else}
                    No allocations yet. Create one above.
                  {/if}
                </td>
              </tr>
            {:else}
              {#each sortedAllocations as alloc}
                {@const allocRange = cidrRange(alloc.cidr)}
                <tr>
                  {#if editingAllocId === alloc.id}
                    <td colspan="3" class="edit-cell">
                      <form class="inline-edit" on:submit|preventDefault={handleUpdateAllocation}>
                        <input type="text" bind:value={editAllocName} placeholder="Allocation name" disabled={editAllocSubmitting} />
                        <div class="inline-actions">
                          <button type="button" class="btn btn-small" on:click={cancelEditAllocation} disabled={editAllocSubmitting}>Cancel</button>
                          <button type="submit" class="btn btn-primary btn-small" disabled={editAllocSubmitting}>
                            {editAllocSubmitting ? 'Saving…' : 'Save'}
                          </button>
                        </div>
                      </form>
                    </td>
                    <td class="actions"></td>
                  {:else}
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
                        }} title="Actions"><Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" /></button>
                        {#if openAllocMenuId === alloc.id}
                          <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{allocDropdownStyle.left}px;top:{allocDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                            <button type="button" role="menuitem" on:click|stopPropagation={() => { startEditAllocation(alloc); openAllocMenuId = null }}>Edit</button>
                            <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteAllocConfirm(alloc); openAllocMenuId = null }}>Delete</button>
                          </div>
                        {/if}
                      </div>
                    </td>
                  {/if}
                </tr>
              {/each}
            {/if}
          </svelte:fragment>
        </DataTable>
        {#if displayedAllocations.length > 0}
          <div class="pagination">
            <span class="pagination-info">Showing {allocStart}–{allocEnd} of {allocTotal}</span>
            <div class="pagination-controls">
              <button type="button" class="btn btn-small" disabled={allocPage <= 0} on:click={() => { allocPage -= 1; load() }}>Previous</button>
              <span class="pagination-page">Page {allocPage + 1} of {allocTotalPages || 1}</span>
              <button type="button" class="btn btn-small" disabled={allocPage >= allocTotalPages - 1} on:click={() => { allocPage += 1; load() }}>Next</button>
            </div>
            <label class="page-size" for="alloc-page-size">
              <span>Per page</span>
              <select id="alloc-page-size" bind:value={allocPageSize} on:change={() => { allocPage = 0; load() }}>
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
              </select>
            </label>
          </div>
        {/if}
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
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .filter-label span {
    font-size: 0.85rem;
    white-space: nowrap;
  }
  .filter-label :global(.searchable-select) {
    min-width: 160px;
  }
  .form-card .form-label-wrap {
    display: block;
    flex: 1;
    min-width: 140px;
  }
  .form-card .form-label-wrap span {
    display: block;
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
    margin-bottom: 0.35rem;
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
  .name {
    font-weight: 500;
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
  .cidr code {
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
