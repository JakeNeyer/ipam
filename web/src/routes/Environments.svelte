<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { listEnvironments, listAllocations, createEnvironment, updateEnvironment, deleteEnvironment, getEnvironment } from '../lib/api.js'

  export let openCreateFromQuery = false
  export let openEnvironmentId = null
  const dispatch = createEventDispatcher()

  let loading = true
  let error = ''
  let environments = []
  let allocations = []
  let expandedEnvId = null
  let expandedEnvBlocks = []
  let expandedBlockName = null
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
  let initialBlockName = ''
  let initialBlockCidr = ''
  let createSubmitting = false
  let createError = ''

  let editingId = null
  let editName = ''
  let editSubmitting = false
  let editError = ''

  let deleteConfirmId = null
  let deleteConfirmName = ''
  let deleteBlockCount = 0
  let deleteSubmitting = false
  let deleteError = ''

  let openMenuId = null
  let menuTriggerEl = null
  let menuDropdownStyle = { left: 0, top: 0 }
  let errorModalMessage = ''

  let envPage = 0
  let envPageSize = 10
  let envTotal = 0

  async function load() {
    loading = true
    error = ''
    try {
      if (openEnvironmentId) {
        const [detail, allocsRes] = await Promise.all([
          getEnvironment(openEnvironmentId),
          listAllocations(),
        ])
        environments = [{ id: detail.id, name: detail.name }]
        envTotal = 1
        allocations = allocsRes.allocations
        expandedEnvId = detail.id
        expandedEnvBlocks = detail.blocks || []
      } else {
        const [envsRes, allocsRes] = await Promise.all([
          listEnvironments({ limit: envPageSize, offset: envPage * envPageSize }),
          listAllocations(),
        ])
        environments = envsRes.environments
        envTotal = envsRes.total
        allocations = allocsRes.allocations
        expandedEnvId = null
        expandedEnvBlocks = []
      }
    } catch (e) {
      error = e.message || 'Failed to load environments'
      errorModalMessage = error
    } finally {
      loading = false
    }
  }

  $: openEnvironmentId, load()

  $: envStart = envTotal === 0 ? 0 : envPage * envPageSize + 1
  $: envEnd = Math.min(envPage * envPageSize + envPageSize, envTotal)
  $: envTotalPages = envPageSize > 0 ? Math.ceil(envTotal / envPageSize) : 0

  onMount(() => {
    function handleClickOutside(e) {
      if (!e.target.closest('.actions-menu-wrap')) {
        openMenuId = null
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  })

  async function toggleEnvRow(env) {
    if (expandedEnvId === env.id) {
      expandedEnvId = null
      expandedEnvBlocks = []
      expandedBlockName = null
      return
    }
    expandedEnvId = env.id
    expandedBlockName = null
    try {
      const detail = await getEnvironment(env.id)
      expandedEnvBlocks = detail.blocks || []
    } catch {
      expandedEnvBlocks = []
    }
  }

  function toggleBlockRow(blockName) {
    expandedBlockName = expandedBlockName === blockName ? null : blockName
  }

  function allocationsForBlock(blockName) {
    if (!blockName) return []
    const name = String(blockName).trim().toLowerCase()
    return allocations.filter((a) => (a.block_name || '').trim().toLowerCase() === name)
  }

  async function handleCreate() {
    const name = createName.trim()
    if (!name) {
      createError = 'Name is required'
      errorModalMessage = createError
      return
    }
    createSubmitting = true
    createError = ''
    try {
      const initialBlock =
        initialBlockName.trim() && initialBlockCidr.trim()
          ? { name: initialBlockName.trim(), cidr: initialBlockCidr.trim() }
          : null
      await createEnvironment(name, initialBlock)
      createName = ''
      initialBlockName = ''
      initialBlockCidr = ''
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
    initialBlockName = ''
    initialBlockCidr = ''
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
      await deleteEnvironment(deleteConfirmId)
      closeDeleteConfirm()
      await load()
    } catch (e) {
      deleteError = e.message || 'Failed to delete environment'
      errorModalMessage = deleteError
    } finally {
      deleteSubmitting = false
    }
  }
</script>

<div class="environments">
  <header class="header">
    <div class="header-text">
      <h1>Environments</h1>
    </div>
    <button class="btn btn-primary" on:click={openCreate}>Create environment</button>
  </header>

  {#if showCreate}
        <div class="form-card">
          <h3>New environment</h3>
          <form on:submit|preventDefault={handleCreate}>
        <label>
          <span>Name</span>
          <input type="text" bind:value={createName} placeholder="e.g. production" disabled={createSubmitting} />
        </label>
        <div class="initial-block">
          <span class="initial-block-label">Initial network block (optional)</span>
          <p class="initial-block-desc">Create the environment’s range as the first block.</p>
          <div class="form-row">
            <label>
              <span>Block name</span>
              <input type="text" bind:value={initialBlockName} placeholder="e.g. main-range" disabled={createSubmitting} />
            </label>
            <label>
              <span>CIDR</span>
              <input type="text" bind:value={initialBlockCidr} placeholder="e.g. 10.0.0.0/8" disabled={createSubmitting} />
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

  {#if loading}
    <div class="loading">Loading…</div>
  {:else if envTotal === 0 && !showCreate}
    <div class="empty">No environments yet. Create one above.</div>
  {:else}
    <div class="list-toolbar">
      {#if openEnvironmentId}
        <a href="#environments" class="link-back">← All environments</a>
      {/if}
    </div>
    <div class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>ID</th>
            <th class="actions">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each environments as env}
            <tr class="env-row" class:expanded={expandedEnvId === env.id} role="button" tabindex="0" on:click={() => editingId !== env.id && toggleEnvRow(env)} on:keydown={(e) => e.key === 'Enter' && editingId !== env.id && toggleEnvRow(env)}>
              {#if editingId === env.id}
                <td colspan="2" class="edit-cell">
                  <form class="inline-edit" on:submit|preventDefault={handleUpdate} on:click|stopPropagation>
                    <input type="text" bind:value={editName} placeholder="Name" disabled={editSubmitting} />
                    <div class="inline-actions">
                      <button type="button" class="btn btn-small" on:click={cancelEdit} disabled={editSubmitting}>Cancel</button>
                      <button type="submit" class="btn btn-primary btn-small" disabled={editSubmitting}>
                        {editSubmitting ? 'Saving…' : 'Save'}
                      </button>
                    </div>
                  </form>
                </td>
                <td class="actions" on:click|stopPropagation>
                  <div class="actions-menu-wrap" role="group">
                    <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openMenuId === env.id} on:click|stopPropagation={(e) => {
                      if (openMenuId === env.id) {
                        openMenuId = null;
                        menuTriggerEl = null;
                      } else {
                        menuTriggerEl = e.currentTarget;
                        const r = e.currentTarget.getBoundingClientRect();
                        menuDropdownStyle = { left: r.right, top: r.bottom + 2 };
                        openMenuId = env.id;
                      }
                    }} title="Actions">⋮</button>
                    {#if openMenuId === env.id}
                      <div class="menu-dropdown menu-dropdown-fixed" role="menu" style="position:fixed;left:{menuDropdownStyle.left}px;top:{menuDropdownStyle.top}px;transform:translateX(-100%);z-index:1000">
                        <button type="button" role="menuitem" on:click|stopPropagation={() => { startEdit(env); openMenuId = null }}>Edit</button>
                        <button type="button" role="menuitem" class="menu-item-danger" on:click|stopPropagation={() => { openDeleteConfirm(env); openMenuId = null }}>Delete</button>
                      </div>
                    {/if}
                  </div>
                </td>
              {:else}
                <td class="name">{env.name}</td>
                <td class="id"><code>{env.id}</code></td>
                <td class="actions" on:click|stopPropagation>
                  <div class="actions-menu-wrap" role="group">
                    <button type="button" class="menu-trigger" aria-haspopup="true" aria-expanded={openMenuId === env.id} on:click|stopPropagation={(e) => {
                      if (openMenuId === env.id) {
                        openMenuId = null;
                        menuTriggerEl = null;
                      } else {
                        menuTriggerEl = e.currentTarget;
                        const r = e.currentTarget.getBoundingClientRect();
                        menuDropdownStyle = { left: r.right, top: r.bottom + 2 };
                        openMenuId = env.id;
                      }
                    }} title="Actions">⋮</button>
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
            {#if expandedEnvId === env.id}
              <tr class="detail-row">
                <td colspan="3" class="detail-cell">
                  <div class="blocks-summary">
                    <h4 class="summary-title">Network blocks</h4>
                    {#if expandedEnvBlocks.length === 0}
                      <p class="summary-empty">No blocks in this environment.</p>
                    {:else}
                      <div class="block-list">
                        {#each expandedEnvBlocks as block}
                          {@const blockAllocs = allocationsForBlock(block.name)}
                          <div
                            class="block-item"
                            class:expanded={expandedBlockName === block.name}
                            role="button"
                            tabindex="0"
                            on:click|stopPropagation={() => toggleBlockRow(block.name)}
                            on:keydown={(e) => e.key === 'Enter' && toggleBlockRow(block.name)}
                          >
                            <div class="block-item-header">
                              <span class="block-name">{block.name}</span>
                              <code class="block-cidr">{block.cidr}</code>
                              {#if cidrRange(block.cidr)}
                                <span class="block-range">{cidrRange(block.cidr).start} – {cidrRange(block.cidr).end}</span>
                              {/if}
                              <span class="block-ips">{block.total_ips?.toLocaleString() ?? 0} IPs</span>
                              <span class="block-expand">{expandedBlockName === block.name ? '▼' : '▶'}</span>
                            </div>
                            {#if expandedBlockName === block.name}
                              <div class="allocations-summary">
                                <h5 class="allocations-title">Allocations</h5>
                                {#if blockAllocs.length === 0}
                                  <p class="summary-empty">No allocations in this block.</p>
                                {:else}
                                  <ul class="alloc-list">
                                    {#each blockAllocs as alloc}
                                      {@const allocRange = cidrRange(alloc.cidr)}
                                      <li class="alloc-item">
                                        <span class="alloc-name">{alloc.name}</span>
                                        <code class="alloc-cidr">{alloc.cidr}</code>
                                        {#if allocRange}
                                          <span class="alloc-range">{allocRange.start} – {allocRange.end}</span>
                                        {/if}
                                      </li>
                                    {/each}
                                  </ul>
                                {/if}
                              </div>
                            {/if}
                          </div>
                        {/each}
                      </div>
                    {/if}
                  </div>
                </td>
              </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>
    {#if !openEnvironmentId}
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

  {#if errorModalMessage}
    <ErrorModal message={errorModalMessage} on:close={() => (errorModalMessage = '')} />
  {/if}
</div>

<style>
  .environments {
    padding-top: 0.5rem;
  }
  .list-toolbar {
    margin-bottom: 1rem;
  }
  .link-back {
    font-size: 0.9rem;
    color: var(--accent);
    text-decoration: none;
  }
  .link-back:hover {
    text-decoration: underline;
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
  .header {
    margin-bottom: 1.5rem;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
  }
  .header-text {
    flex: 1;
  }
  .header h1 {
    margin: 0;
    font-size: 1.35rem;
    font-weight: 600;
    letter-spacing: -0.02em;
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
  .btn-small {
    padding: 0.35rem 0.75rem;
    font-size: 0.85rem;
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
  .initial-block {
    margin: 1rem 0;
    padding: 1rem;
    background: rgba(0, 0, 0, 0.15);
    border-radius: var(--radius);
  }
  .initial-block-label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .initial-block-desc {
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
  .loading,
  .empty {
    color: var(--text-muted);
    padding: 2rem;
  }
  .table-wrap {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }
  .table {
    width: 100%;
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
  .table td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .table tr:last-child td {
    border-bottom: none;
  }
  .table tr.env-row {
    cursor: pointer;
  }
  .table tr.env-row:hover td {
    background: var(--table-row-hover);
  }
  .table tr.env-row.expanded td {
    background: rgba(88, 166, 255, 0.06);
    border-bottom-color: var(--accent);
  }
  .table thead th:first-child {
    min-width: 220px;
  }
  .table td.name {
    min-width: 220px;
  }
  .name {
    font-weight: 500;
  }
  .id code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .detail-row {
    background: var(--bg);
  }
  .detail-row td {
    vertical-align: top;
    padding: 0;
    border-bottom: 1px solid var(--border);
  }
  .detail-cell {
    padding: 1rem 1.5rem 1rem 3rem;
  }
  .blocks-summary {
    font-size: 0.9rem;
  }
  .summary-title {
    margin: 0 0 0.75rem 0;
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
  }
  .summary-empty {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .block-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .block-item {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.6rem 0.75rem;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }
  .block-item:hover {
    background: rgba(255, 255, 255, 0.03);
    border-color: var(--text-muted);
  }
  .block-item.expanded {
    border-color: var(--accent);
    background: rgba(88, 166, 255, 0.04);
  }
  .block-item-header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1rem;
  }
  .block-name {
    font-weight: 500;
    color: var(--text);
  }
  .block-cidr {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .block-range {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .block-ips {
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .block-expand {
    margin-left: auto;
    font-size: 0.7rem;
    color: var(--text-muted);
  }
  .allocations-summary {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border);
  }
  .allocations-title {
    margin: 0 0 0.5rem 0;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
  }
  .alloc-list {
    margin: 0;
    padding-left: 1.25rem;
    list-style: disc;
  }
  .alloc-item {
    margin-bottom: 0.35rem;
    font-size: 0.85rem;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.35rem 0.75rem;
  }
  .alloc-name {
    font-weight: 500;
  }
  .alloc-cidr {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .alloc-range {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
</style>
