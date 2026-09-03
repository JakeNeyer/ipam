<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import DataTable from '../lib/DataTable.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { formatBlockCount, compareBlockCount } from '../lib/blockCount.js'
  import { totalIPsForCidr, poolUsedIPs, poolUtilizationPercent } from '../lib/poolUsage.js'
  import { user, selectedOrgForGlobalAdmin, isGlobalAdmin } from '../lib/auth.js'
  import SearchableSelect from '../lib/SearchableSelect.svelte'
  import { listEnvironments, listAllocations, listPools, createEnvironment, createPool, updateEnvironment, updatePool, deleteEnvironment, deletePool, getEnvironment } from '../lib/api.js'
  import { idsMatch, getPoolDepth, sortPoolsByHierarchy, ancestorPools } from '../lib/poolHierarchy.js'

  export let openCreateFromQuery = false
  export let openEnvironmentId = null
  export let openPoolId = null
  const dispatch = createEventDispatcher()

  let loading = true
  let error = ''
  let environments = []
  let allocations = []
  let expandedEnvBlocks = []
  let showCreate = false
  let openedCreateFromQuery = false
  $: if (openCreateFromQuery) {
    showCreate = true
    if (!openedCreateFromQuery) {
      openedCreateFromQuery = true
      dispatch('clearCreateQuery')
    }
  } else {
    openedCreateFromQuery = false
  }
  let createName = ''
  let initialPoolName = ''
  let initialPoolCidr = ''
  let createSubmitting = false
  let createError = ''

  let editingId = null
  let editName = ''
  let editSubmitting = false
  let editError = ''

  let expandedEnvPools = []
  let showAddPool = false
  let newPoolName = ''
  let newPoolCidr = ''
  let newPoolParentId = ''
  let newPoolSubmitting = false
  let newPoolError = ''
  let editingPoolId = null
  let editPoolName = ''
  let editPoolCidr = ''
  let editPoolSubmitting = false
  let editPoolError = ''
  let deletePoolId = null
  let deletePoolName = ''
  let deletePoolSubmitting = false
  let deletePoolError = ''

  let deleteConfirmId = null
  let deleteConfirmName = ''
  let deleteBlockCount = 0
  let deleteSubmitting = false
  let deleteError = ''

  let openMenuId = null
  let openPoolMenuId = null
  let menuDropdownStyle = { left: 0, top: 0 }
  let poolDropdownStyle = { left: 0, top: 0 }
  let errorModalMessage = ''

  let envPage = 0
  let envPageSize = 10
  let envTotal = 0

  let envSortBy = 'name' // 'name' | 'id'
  let envSortDir = 'asc' // 'asc' | 'desc'

  function setEnvSort(column) {
    if (envSortBy === column) {
      envSortDir = envSortDir === 'asc' ? 'desc' : 'asc'
    } else {
      envSortBy = column
      envSortDir = 'asc'
    }
  }

  $: sortedEnvironments = (() => {
    const list = [...environments]
    const mult = envSortDir === 'asc' ? 1 : -1
    if (envSortBy === 'name') {
      list.sort((a, b) => mult * (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
    } else if (envSortBy === 'id') {
      list.sort((a, b) => mult * (String(a.id || '').localeCompare(String(b.id || ''), undefined, { sensitivity: 'base' })))
    }
    return list
  })()

  function listOpts(extra = {}) {
    const opts = { ...extra }
    if (isGlobalAdmin($user) && $selectedOrgForGlobalAdmin) opts.organization_id = $selectedOrgForGlobalAdmin
    return opts
  }

  async function load() {
    loading = true
    error = ''
    try {
      if (openEnvironmentId) {
        const [detail, allocsRes, poolsRes] = await Promise.all([
          getEnvironment(openEnvironmentId),
          listAllocations(listOpts()),
          listPools(openEnvironmentId),
        ])
        environments = [{ id: detail.id, name: detail.name }]
        envTotal = 1
        allocations = allocsRes.allocations
        expandedEnvBlocks = detail.blocks || []
        expandedEnvPools = poolsRes.pools || []
      } else {
        const envsRes = await listEnvironments(listOpts({ limit: envPageSize, offset: envPage * envPageSize }))
        environments = envsRes.environments
        envTotal = envsRes.total
        allocations = []
        expandedEnvBlocks = []
        expandedEnvPools = []
      }
    } catch (e) {
      error = e.message || 'Failed to load environments'
      errorModalMessage = error
    } finally {
      loading = false
    }
  }

  $: openEnvironmentId, $selectedOrgForGlobalAdmin, load()

  $: envStart = envTotal === 0 ? 0 : envPage * envPageSize + 1
  $: envEnd = Math.min(envPage * envPageSize + envPageSize, envTotal)
  $: envTotalPages = envPageSize > 0 ? Math.ceil(envTotal / envPageSize) : 0

  $: expandedEnvPoolsOrdered = sortPoolsByHierarchy(expandedEnvPools)
  $: scopeEnv = openEnvironmentId ? (environments[0] || null) : null
  $: scopePool = openPoolId ? expandedEnvPools.find((p) => idsMatch(p.id, openPoolId)) : null
  $: poolAncestors = scopePool ? ancestorPools(scopePool, expandedEnvPools) : []
  $: displayedPools = !openPoolId
    ? expandedEnvPoolsOrdered
    : expandedEnvPoolsOrdered.filter((p) => ancestorPools(p, expandedEnvPools).some((a) => idsMatch(a.id, openPoolId)))
  $: displayedBlocks = openPoolId
    ? expandedEnvBlocks.filter((b) => idsMatch(b.pool_id, openPoolId))
    : []
  $: blocksWithoutPool = openPoolId ? [] : expandedEnvBlocks.filter((b) => !b.pool_id)

  function envHash(envId, poolId) {
    const params = new URLSearchParams()
    if (envId) params.set('env', envId)
    if (poolId) params.set('pool', poolId)
    const q = params.toString()
    window.location.hash = 'environments' + (q ? '?' + q : '')
  }

  function drillToEnv(envId) {
    envHash(envId, null)
  }

  function drillToPool(envId, poolId) {
    envHash(envId, poolId)
  }

  function clearScope() {
    window.location.hash = 'environments'
  }

  function networksHref() {
    if (openPoolId) return `#networks?pool=${encodeURIComponent(openPoolId)}`
    if (openEnvironmentId) return `#networks?pool=env:${encodeURIComponent(openEnvironmentId)}`
    return '#networks'
  }

  function blockNetworksHref(block) {
    const params = new URLSearchParams()
    if (block?.name) params.set('block', block.name)
    if (block?.pool_id) params.set('pool', block.pool_id)
    else if (openEnvironmentId) params.set('pool', 'env:' + openEnvironmentId)
    const q = params.toString()
    return '#networks' + (q ? '?' + q : '')
  }

  onMount(() => {
    function handleClickOutside(e) {
      if (!e.target.closest('.actions-menu-wrap')) {
        openMenuId = null
        openPoolMenuId = null
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  })

  function allocationsForBlock(blockName) {
    if (!blockName) return []
    const name = String(blockName).trim().toLowerCase()
    return allocations.filter((a) => (a.block_name || '').trim().toLowerCase() === name)
  }

  function toggleEnvMenu(e, env) {
    if (openMenuId === env.id) {
      openMenuId = null
      return
    }
    const r = e.currentTarget.getBoundingClientRect()
    menuDropdownStyle = { left: r.right, top: r.bottom + 2 }
    openMenuId = env.id
    openPoolMenuId = null
  }

  function togglePoolMenu(e, pool) {
    if (openPoolMenuId === pool.id) {
      openPoolMenuId = null
      return
    }
    const r = e.currentTarget.getBoundingClientRect()
    poolDropdownStyle = { left: r.right, top: r.bottom + 2 }
    openPoolMenuId = pool.id
    openMenuId = null
  }

  async function handleCreate() {
    const name = createName.trim()
    const poolName = initialPoolName.trim()
    const poolCidr = initialPoolCidr.trim()
    if (!name) {
      createError = 'Name is required'
      errorModalMessage = createError
      return
    }
    if (!poolName || !poolCidr) {
      createError = 'Pool name and CIDR are required. Every environment must have a pool that blocks draw from.'
      errorModalMessage = createError
      return
    }
    createSubmitting = true
    createError = ''
    try {
      const pools = [{ name: poolName, cidr: poolCidr }]
      await createEnvironment(name, pools, isGlobalAdmin($user) ? $selectedOrgForGlobalAdmin : null)
      createName = ''
      initialPoolName = ''
      initialPoolCidr = ''
      showCreate = false
      await load()
    } catch (e) {
      createError = e.message || 'Failed to create environment'
      errorModalMessage = createError
    } finally {
      createSubmitting = false
    }
  }

  function openCreate() {
    showCreate = true
    createName = ''
    initialPoolName = ''
    initialPoolCidr = ''
    createError = ''
  }

  function startEdit(env) {
    editingId = env.id
    editName = env.name
    editError = ''
  }

  function cancelEdit() {
    editingId = null
    editName = ''
    editError = ''
  }

  async function handleUpdate() {
    const name = editName.trim()
    if (!name) {
      editError = 'Name is required'
      errorModalMessage = editError
      return
    }
    editSubmitting = true
    editError = ''
    try {
      await updateEnvironment(editingId, name)
      cancelEdit()
      await load()
    } catch (e) {
      editError = e.message || 'Failed to update environment'
      errorModalMessage = editError
    } finally {
      editSubmitting = false
    }
  }

  async function openDeleteConfirm(env) {
    deleteConfirmId = env.id
    deleteConfirmName = env.name
    deleteError = ''
    deleteSubmitting = false
    try {
      const detail = await getEnvironment(env.id)
      deleteBlockCount = detail.blocks ? detail.blocks.length : 0
    } catch {
      deleteBlockCount = 0
    }
  }

  function closeDeleteConfirm() {
    deleteConfirmId = null
    deleteConfirmName = ''
    deleteBlockCount = 0
    deleteError = ''
  }

  async function handleDelete() {
    if (!deleteConfirmId) return
    deleteSubmitting = true
    deleteError = ''
    try {
      const wasOpen = openEnvironmentId && idsMatch(deleteConfirmId, openEnvironmentId)
      await deleteEnvironment(deleteConfirmId)
      closeDeleteConfirm()
      if (wasOpen) {
        clearScope()
      } else {
        await load()
      }
    } catch (e) {
      deleteError = e.message || 'Failed to delete environment'
      errorModalMessage = deleteError
    } finally {
      deleteSubmitting = false
    }
  }

  async function loadPoolsForExpandedEnv() {
    if (!openEnvironmentId) return
    try {
      const res = await listPools(openEnvironmentId)
      expandedEnvPools = res.pools || []
    } catch {
      expandedEnvPools = []
    }
  }

  function openAddPool() {
    showAddPool = true
    newPoolName = ''
    newPoolCidr = ''
    newPoolParentId = openPoolId ? String(openPoolId) : ''
    newPoolError = ''
  }

  async function handleAddPool() {
    const name = newPoolName.trim()
    const cidr = newPoolCidr.trim()
    if (!name || !cidr) {
      newPoolError = 'Name and CIDR are required'
      return
    }
    if (!openEnvironmentId) return
    newPoolSubmitting = true
    newPoolError = ''
    try {
      await createPool(openEnvironmentId, name, cidr, newPoolParentId?.trim() || null)
      newPoolName = ''
      newPoolCidr = ''
      newPoolParentId = ''
      showAddPool = false
      await loadPoolsForExpandedEnv()
    } catch (e) {
      newPoolError = e.message || 'Failed to create pool'
    } finally {
      newPoolSubmitting = false
    }
  }

  function startEditPool(pool) {
    editingPoolId = pool.id
    editPoolName = pool.name
    editPoolCidr = pool.cidr
    editPoolError = ''
  }

  function cancelEditPool() {
    editingPoolId = null
    editPoolName = ''
    editPoolCidr = ''
    editPoolError = ''
  }

  async function handleUpdatePool() {
    const name = editPoolName.trim()
    const cidr = editPoolCidr.trim()
    if (!name || !cidr) {
      editPoolError = 'Name and CIDR are required'
      return
    }
    if (!editingPoolId) return
    editPoolSubmitting = true
    editPoolError = ''
    try {
      await updatePool(editingPoolId, name, cidr)
      cancelEditPool()
      await loadPoolsForExpandedEnv()
    } catch (e) {
      editPoolError = e.message || 'Failed to update pool'
    } finally {
      editPoolSubmitting = false
    }
  }

  function openDeletePoolConfirm(pool) {
    deletePoolId = pool.id
    deletePoolName = pool.name
    deletePoolError = ''
    deletePoolSubmitting = false
  }

  function closeDeletePoolConfirm() {
    deletePoolId = null
    deletePoolName = ''
    deletePoolError = ''
  }

  async function handleDeletePool() {
    if (!deletePoolId) return
    deletePoolSubmitting = true
    deletePoolError = ''
    try {
      const wasFocused = openPoolId && idsMatch(deletePoolId, openPoolId)
      await deletePool(deletePoolId)
      closeDeletePoolConfirm()
      if (wasFocused && openEnvironmentId) {
        drillToEnv(openEnvironmentId)
      } else {
        await loadPoolsForExpandedEnv()
      }
    } catch (e) {
      deletePoolError = e.message || 'Failed to delete pool'
    } finally {
      deletePoolSubmitting = false
    }
  }
</script>

<div class="environments">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Environments</h1>
      <p class="page-desc">Logical groupings for your network blocks (e.g. production, staging). Click an environment to see its pools.</p>
    </div>
    <div class="header-actions">
      {#if scopeEnv}
        <a class="btn" href={networksHref()}>Open in Networks</a>
        <button type="button" class="btn" on:click={openAddPool}>Add pool</button>
        <div class="actions-menu-wrap" role="group">
          <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openMenuId === scopeEnv.id} on:click|stopPropagation={(e) => toggleEnvMenu(e, scopeEnv)} title="Actions"><Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" /></button>
          {#if openMenuId === scopeEnv.id}
            <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{menuDropdownStyle.left}px;top:{menuDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
              <button type="button" role="menuitem" on:click|stopPropagation={() => { startEdit(scopeEnv); openMenuId = null }}>Edit</button>
              <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteConfirm(scopeEnv); openMenuId = null }}>Delete</button>
            </div>
          {/if}
        </div>
      {/if}
      <button type="button" class="btn btn-primary" on:click={openCreate}>Create environment</button>
    </div>
  </header>

  {#if showCreate}
    <div class="form-card">
      <h3>New environment</h3>
      <form on:submit|preventDefault={handleCreate}>
        <label>
          <span>Name</span>
          <input type="text" bind:value={createName} placeholder="e.g. production" disabled={createSubmitting} />
        </label>
        <div class="initial-pool">
          <span class="initial-pool-label">Pool (required)</span>
          <p class="initial-pool-desc">Every environment must have a pool: a CIDR range that network blocks in this environment can draw from.</p>
          <div class="form-row">
            <label>
              <span>Pool name</span>
              <input type="text" bind:value={initialPoolName} placeholder="e.g. default or prod-pool" disabled={createSubmitting} />
            </label>
            <label>
              <span>Pool CIDR</span>
              <input type="text" bind:value={initialPoolCidr} placeholder="e.g. 10.0.0.0/8 or fd00::/64" disabled={createSubmitting} />
            </label>
          </div>
        </div>
        <div class="form-actions">
          <button type="button" class="btn" on:click={() => (showCreate = false)} disabled={createSubmitting}>Cancel</button>
          <button type="submit" class="btn btn-primary" disabled={createSubmitting}>
            {createSubmitting ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </div>
  {/if}

  {#if editingId && scopeEnv && editingId === scopeEnv.id}
    <div class="form-card">
      <h3>Rename environment</h3>
      <form on:submit|preventDefault={handleUpdate}>
        <label>
          <span>Name</span>
          <input type="text" bind:value={editName} placeholder="Name" disabled={editSubmitting} />
        </label>
        {#if editError}
          <p class="form-error">{editError}</p>
        {/if}
        <div class="form-actions">
          <button type="button" class="btn" on:click={cancelEdit} disabled={editSubmitting}>Cancel</button>
          <button type="submit" class="btn btn-primary" disabled={editSubmitting}>
            {editSubmitting ? 'Saving…' : 'Save'}
          </button>
        </div>
      </form>
    </div>
  {/if}

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    {#if openEnvironmentId}
      <div class="list-toolbar">
        <div class="scope-row">
          <button type="button" class="link-back" on:click={clearScope}>← All environments</button>
          {#if scopeEnv}
            <nav class="scope-path" aria-label="Current scope">
              {#if scopePool}
                <button type="button" class="scope-crumb" on:click={() => drillToEnv(scopeEnv.id)}>{scopeEnv.name}</button>
                <span class="scope-sep" aria-hidden="true">/</span>
                {#each poolAncestors.slice(0, -1) as anc}
                  <button type="button" class="scope-crumb" on:click={() => drillToPool(scopeEnv.id, anc.id)}>{anc.name}</button>
                  <span class="scope-sep" aria-hidden="true">/</span>
                {/each}
                <span class="scope-current">{scopePool.name}</span>
              {:else}
                <span class="scope-current">{scopeEnv.name}</span>
              {/if}
            </nav>
          {/if}
        </div>
      </div>

      <section class="section">
        <div class="section-header">
          <h2>Pools {#if displayedPools.length > 0}<span class="section-count">({displayedPools.length})</span>{/if}</h2>
        </div>
        {#if showAddPool}
          <div class="form-card">
            <h3>New pool</h3>
            <form on:submit|preventDefault={handleAddPool}>
              <div class="form-row">
                <label>
                  <span>Parent pool (optional)</span>
                  <SearchableSelect
                    options={[
                      { value: '', label: '— None (top-level) —' },
                      ...expandedEnvPoolsOrdered.map((p) => ({ value: String(p.id), label: `${p.name} (${p.cidr || '—'})` }))
                    ]}
                    bind:value={newPoolParentId}
                    placeholder="Select parent for child pool"
                  />
                </label>
                <label>
                  <span>Name</span>
                  <input type="text" bind:value={newPoolName} placeholder="e.g. prod-pool" disabled={newPoolSubmitting} />
                </label>
                <label>
                  <span>CIDR</span>
                  <input type="text" bind:value={newPoolCidr} placeholder="e.g. 10.0.0.0/8" disabled={newPoolSubmitting} />
                </label>
              </div>
              {#if newPoolParentId && expandedEnvPoolsOrdered.find((p) => String(p.id) === newPoolParentId)}
                <p class="form-hint" role="status">Child pool CIDR must be contained in the parent pool's CIDR.</p>
              {/if}
              {#if newPoolError}
                <p class="form-error">{newPoolError}</p>
              {/if}
              <div class="form-actions">
                <button type="button" class="btn" on:click={() => (showAddPool = false)} disabled={newPoolSubmitting}>Cancel</button>
                <button type="submit" class="btn btn-primary" disabled={newPoolSubmitting}>
                  {newPoolSubmitting ? 'Adding…' : 'Add pool'}
                </button>
              </div>
            </form>
          </div>
        {/if}
        {#if displayedPools.length > 0}
          <div class="pools-table-wrap">
            <DataTable>
              <svelte:fragment slot="header">
                <tr>
                  <th>Name</th>
                  <th>CIDR</th>
                  <th class="num">Total IPs</th>
                  <th class="num">Used</th>
                  <th class="num">Available</th>
                  <th>Usage</th>
                  <th class="actions">Actions</th>
                </tr>
              </svelte:fragment>
              <svelte:fragment slot="body">
                {#each displayedPools as pool}
                  {@const poolTotal = totalIPsForCidr(pool.cidr)}
                  {@const used = poolUsedIPs(pool, expandedEnvPools, expandedEnvBlocks)}
                  {@const pct = poolUtilizationPercent(pool, expandedEnvPools, expandedEnvBlocks)}
                  {@const available = (() => { try { const t = BigInt(poolTotal || '0'); const u = BigInt(used || '0'); return t >= u ? (t - u).toString() : '0'; } catch { return '0'; } })()}
                  {@const poolDepthAbs = getPoolDepth(pool, expandedEnvPools)}
                  {@const poolDepth = Math.max(0, poolDepthAbs - (scopePool ? getPoolDepth(scopePool, expandedEnvPools) : 0))}
                  <tr class:pool-child-row={poolDepth > 0}>
                    {#if editingPoolId === pool.id}
                      <td colspan="6" class="edit-cell">
                        <form class="inline-edit" on:submit|preventDefault={handleUpdatePool}>
                          <input type="text" bind:value={editPoolName} placeholder="Name" disabled={editPoolSubmitting} />
                          <input type="text" bind:value={editPoolCidr} placeholder="CIDR" disabled={editPoolSubmitting} />
                          <div class="inline-actions">
                            <button type="button" class="btn btn-small" on:click={cancelEditPool} disabled={editPoolSubmitting}>Cancel</button>
                            <button type="submit" class="btn btn-primary btn-small" disabled={editPoolSubmitting}>
                              {editPoolSubmitting ? 'Saving…' : 'Save'}
                            </button>
                          </div>
                        </form>
                        {#if editPoolError}
                          <span class="form-error">{editPoolError}</span>
                        {/if}
                      </td>
                      <td class="actions"></td>
                    {:else}
                      <td class="name cell-pool-name" style="padding-left: {1 + poolDepth * 1.25}rem">
                        <div class="pool-name-cell-content">
                          {#if poolDepth > 0}
                            <span class="pool-name-indent" aria-hidden="true"><Icon icon="lucide:corner-down-right" width="1em" height="1em" /></span>
                          {/if}
                          <button type="button" class="link-name" on:click={() => drillToPool(openEnvironmentId, pool.id)}>{pool.name}</button>
                        </div>
                      </td>
                      <td class="cidr"><code>{pool.cidr}</code></td>
                      <td class="num">{formatBlockCount(poolTotal)}</td>
                      <td class="num">{formatBlockCount(used)}</td>
                      <td class="num">{formatBlockCount(available)}</td>
                      <td>
                        <div class="usage-cell" title="Used (child pools + blocks): {formatBlockCount(used)} / {formatBlockCount(poolTotal)}">
                          <div class="bar-wrap">
                            <div
                              class="bar"
                              class:high={pct >= 80}
                              class:mid={pct >= 50 && pct < 80}
                              style="width: {pct < 1 && compareBlockCount(used, '0') > 0 ? 1 : Math.min(100, Math.round(pct))}%"
                            ></div>
                          </div>
                          <span class="pct">{pct < 1 && compareBlockCount(used, '0') > 0 ? '<1' : Math.round(pct)}%</span>
                        </div>
                      </td>
                      <td class="actions">
                        <div class="actions-menu-wrap" role="group">
                          <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openPoolMenuId === pool.id} on:click|stopPropagation={(e) => togglePoolMenu(e, pool)} title="Actions"><Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" /></button>
                          {#if openPoolMenuId === pool.id}
                            <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{poolDropdownStyle.left}px;top:{poolDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                              <button type="button" role="menuitem" on:click|stopPropagation={() => { startEditPool(pool); openPoolMenuId = null }}>Edit</button>
                              <button type="button" role="menuitem" on:click|stopPropagation={() => { openAddPool(); newPoolParentId = String(pool.id); openPoolMenuId = null }}>Add child pool</button>
                              <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeletePoolConfirm(pool); openPoolMenuId = null }}>Delete</button>
                            </div>
                          {/if}
                        </div>
                      </td>
                    {/if}
                  </tr>
                {/each}
              </svelte:fragment>
            </DataTable>
          </div>
        {:else if !showAddPool}
          <p class="table-empty-cell">No pools yet. Add a pool to define a CIDR range for blocks.</p>
        {/if}
      </section>

      {#if openPoolId}
        <section class="section">
          <div class="section-header">
            <h2>Network blocks {#if displayedBlocks.length > 0}<span class="section-count">({displayedBlocks.length})</span>{/if}</h2>
          </div>
          <DataTable>
            <svelte:fragment slot="header">
              <tr>
                <th>Name</th>
                <th>CIDR</th>
                <th class="num">Total IPs</th>
                <th class="num">Allocations</th>
              </tr>
            </svelte:fragment>
            <svelte:fragment slot="body">
              {#if displayedBlocks.length === 0}
                <tr>
                  <td colspan="4" class="table-empty-cell">No network blocks in this pool.</td>
                </tr>
              {:else}
                {#each displayedBlocks as block}
                  {@const blockAllocs = allocationsForBlock(block.name)}
                  {@const blockRange = cidrRange(block.cidr)}
                  <tr>
                    <td class="name">
                      <a class="link-name" href={blockNetworksHref(block)}>{block.name}</a>
                    </td>
                    <td class="cidr">
                      <code>{block.cidr}</code>
                      {#if blockRange}
                        <span class="cidr-range">{blockRange.start} – {blockRange.end}</span>
                      {/if}
                    </td>
                    <td class="num">{formatBlockCount(block.total_ips)}</td>
                    <td class="num">{blockAllocs.length}</td>
                  </tr>
                {/each}
              {/if}
            </svelte:fragment>
          </DataTable>
        </section>
      {:else if blocksWithoutPool.length > 0}
        <section class="section">
          <div class="section-header">
            <h2>Blocks without pool <span class="section-count">({blocksWithoutPool.length})</span></h2>
          </div>
          <p class="section-desc">Network blocks in this environment that are not assigned to any pool.</p>
          <DataTable>
            <svelte:fragment slot="header">
              <tr>
                <th>Name</th>
                <th>CIDR</th>
                <th class="num">Total IPs</th>
                <th class="num">Allocations</th>
              </tr>
            </svelte:fragment>
            <svelte:fragment slot="body">
              {#each blocksWithoutPool as block}
                {@const blockAllocs = allocationsForBlock(block.name)}
                {@const blockRange = cidrRange(block.cidr)}
                <tr>
                  <td class="name">
                    <a class="link-name" href={blockNetworksHref(block)}>{block.name}</a>
                  </td>
                  <td class="cidr">
                    <code>{block.cidr}</code>
                    {#if blockRange}
                      <span class="cidr-range">{blockRange.start} – {blockRange.end}</span>
                    {/if}
                  </td>
                  <td class="num">{formatBlockCount(block.total_ips)}</td>
                  <td class="num">{blockAllocs.length}</td>
                </tr>
              {/each}
            </svelte:fragment>
          </DataTable>
        </section>
      {/if}
    {:else}
      <DataTable>
        <svelte:fragment slot="header">
          <tr>
            <th class="sortable" class:sorted={envSortBy === 'name'}>
              <button type="button" class="th-sort" on:click={() => setEnvSort('name')}>
                <span class="th-sort-label">Name</span>
                {#if envSortBy === 'name'}
                  <span class="sort-icon" aria-hidden="true"><Icon icon={envSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                {/if}
              </button>
            </th>
            <th class="sortable" class:sorted={envSortBy === 'id'}>
              <button type="button" class="th-sort" on:click={() => setEnvSort('id')}>
                <span class="th-sort-label">ID</span>
                {#if envSortBy === 'id'}
                  <span class="sort-icon" aria-hidden="true"><Icon icon={envSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                {/if}
              </button>
            </th>
            <th class="actions">Actions</th>
          </tr>
        </svelte:fragment>
        <svelte:fragment slot="body">
          {#if envTotal === 0 && !showCreate}
            <tr>
              <td colspan="3" class="table-empty-cell">No environments yet. Create one above.</td>
            </tr>
          {:else}
            {#each sortedEnvironments as env}
              <tr>
                {#if editingId === env.id}
                  <td colspan="2" class="edit-cell">
                    <form class="inline-edit" on:submit|preventDefault={handleUpdate}>
                      <input type="text" bind:value={editName} placeholder="Name" disabled={editSubmitting} />
                      <div class="inline-actions">
                        <button type="button" class="btn btn-small" on:click={cancelEdit} disabled={editSubmitting}>Cancel</button>
                        <button type="submit" class="btn btn-primary btn-small" disabled={editSubmitting}>
                          {editSubmitting ? 'Saving…' : 'Save'}
                        </button>
                      </div>
                    </form>
                    {#if editError}
                      <span class="form-error">{editError}</span>
                    {/if}
                  </td>
                  <td class="actions"></td>
                {:else}
                  <td class="name">
                    <button type="button" class="link-name" on:click={() => drillToEnv(env.id)}>{env.name}</button>
                  </td>
                  <td class="id"><code>{env.id}</code></td>
                  <td class="actions">
                    <div class="actions-menu-wrap" role="group">
                      <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openMenuId === env.id} on:click|stopPropagation={(e) => toggleEnvMenu(e, env)} title="Actions"><Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" /></button>
                      {#if openMenuId === env.id}
                        <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{menuDropdownStyle.left}px;top:{menuDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                          <button type="button" role="menuitem" on:click|stopPropagation={() => { startEdit(env); openMenuId = null }}>Edit</button>
                          <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteConfirm(env); openMenuId = null }}>Delete</button>
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
      <div class="pagination">
        <span class="pagination-info">Showing {envStart}–{envEnd} of {envTotal}</span>
        <div class="pagination-controls">
          <button type="button" class="btn btn-small" disabled={envPage <= 0} on:click={() => { envPage -= 1; load() }}>Previous</button>
          <span class="pagination-page">Page {envPage + 1} of {envTotalPages || 1}</span>
          <button type="button" class="btn btn-small" disabled={envPage >= envTotalPages - 1} on:click={() => { envPage += 1; load() }}>Next</button>
        </div>
        <label class="page-size">
          <span>Per page</span>
          <select bind:value={envPageSize} on:change={() => { envPage = 0; load() }}>
            <option value={10}>10</option>
            <option value={25}>25</option>
            <option value={50}>50</option>
          </select>
        </label>
      </div>
    {/if}
  {/if}

  {#if deleteConfirmId}
    <div class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="delete-dialog-title">
      <div class="modal">
        <h3 id="delete-dialog-title">Delete environment</h3>
        <p class="modal-warning">
          <strong>This will permanently delete the environment “{deleteConfirmName}” and all of its network blocks.</strong>
          {#if deleteBlockCount > 0}
            <br /><span class="block-count">{deleteBlockCount} network block(s) will be removed.</span>
          {/if}
          This action cannot be undone.
        </p>
        <div class="modal-actions">
          <button type="button" class="btn" on:click={closeDeleteConfirm} disabled={deleteSubmitting}>Cancel</button>
          <button type="button" class="btn btn-danger" on:click={handleDelete} disabled={deleteSubmitting}>
            {deleteSubmitting ? 'Deleting…' : 'Delete environment'}
          </button>
        </div>
      </div>
    </div>
  {/if}

  {#if deletePoolId}
    <div class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="delete-pool-dialog-title">
      <div class="modal">
        <h3 id="delete-pool-dialog-title">Delete pool</h3>
        <p class="modal-warning">
          Delete pool <strong>{deletePoolName}</strong>? Blocks in this pool will be unassigned from the pool (not deleted).
        </p>
        {#if deletePoolError}
          <p class="form-error">{deletePoolError}</p>
        {/if}
        <div class="modal-actions">
          <button type="button" class="btn" on:click={closeDeletePoolConfirm} disabled={deletePoolSubmitting}>Cancel</button>
          <button type="button" class="btn btn-danger" on:click={handleDeletePool} disabled={deletePoolSubmitting}>
            {deletePoolSubmitting ? 'Deleting…' : 'Delete pool'}
          </button>
        </div>
      </div>
    </div>
  {/if}

  {#if errorModalMessage}
    <ErrorModal message={errorModalMessage} on:close={() => (errorModalMessage = '')} />
  {/if}
</div>

<style>
  .environments {
    padding-top: 0.5rem;
  }
  .header-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }
  .list-toolbar {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  .scope-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
  }
  .link-back {
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    font-size: 0.9rem;
    color: var(--accent);
    cursor: pointer;
  }
  .link-back:hover {
    text-decoration: underline;
  }
  .scope-path {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem;
    font-size: 0.9rem;
    min-width: 0;
  }
  .scope-sep {
    color: var(--text-muted);
    user-select: none;
  }
  .scope-crumb {
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    color: var(--accent);
    cursor: pointer;
  }
  .scope-crumb:hover {
    text-decoration: underline;
  }
  .scope-current {
    color: var(--text);
    font-weight: 600;
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
  .section-count {
    font-weight: 400;
    color: var(--text-muted);
  }
  .section-desc {
    margin: -0.25rem 0 0.75rem 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .link-name {
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    font-weight: 500;
    color: var(--accent);
    cursor: pointer;
    text-align: left;
    text-decoration: none;
  }
  .link-name:hover {
    text-decoration: underline;
  }
  a.link-name {
    display: inline;
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
  .actions {
    text-align: right;
    white-space: nowrap;
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
    max-width: 200px;
    padding: 0.4rem 0.6rem;
    font-size: 0.9rem;
    font-family: var(--font-sans);
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text);
  }
  .inline-edit input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .inline-edit input::placeholder {
    color: var(--text-muted);
  }
  .inline-actions {
    display: flex;
    gap: 0.35rem;
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
    margin-bottom: 1.5rem;
  }
  .form-card h3 {
    margin: 0 0 1rem 0;
    font-size: 1rem;
    font-weight: 600;
  }
  .form-card label {
    display: block;
    margin-bottom: 1rem;
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
    max-width: 280px;
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
  .initial-pool {
    margin: 1rem 0;
    padding: 1rem;
    background: rgba(0, 0, 0, 0.15);
    border-radius: var(--radius);
  }
  .initial-pool-label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .initial-pool-desc {
    margin: 0.25rem 0 0.75rem 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .form-row {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
  }
  .form-row label {
    flex: 1;
    min-width: 140px;
  }
  .form-actions {
    display: flex;
    gap: 0.5rem;
  }
  .form-hint {
    margin: 0 0 1rem 0;
    font-size: 0.875rem;
    color: var(--text-muted);
  }
  .form-error {
    margin: 0.35rem 0 0 0;
    font-size: 0.85rem;
    color: var(--danger);
  }
  .loading {
    color: var(--text-muted);
    padding: 2rem;
  }
  .name {
    font-weight: 500;
  }
  .id code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .pools-table-wrap :global(th:first-child),
  .pools-table-wrap :global(td.name.cell-pool-name) {
    min-width: 10rem;
    max-width: 28rem;
  }
  .pool-name-cell-content {
    display: inline-flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.35rem;
    min-width: 0;
  }
  .pool-name-indent {
    display: inline-flex;
    margin-right: 0.35rem;
    color: var(--text-muted);
    vertical-align: middle;
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
  .table-empty-cell {
    color: var(--text-muted);
    padding: 1rem;
  }
</style>
