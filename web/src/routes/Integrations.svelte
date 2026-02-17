<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import '../lib/theme.js'
  import DataTable from '../lib/DataTable.svelte'
  import { user, selectedOrgForGlobalAdmin, isGlobalAdmin } from '../lib/auth.js'
  import {
    listIntegrations,
    createIntegration,
    createEnvironment,
    updateIntegration,
    deleteIntegration,
    syncIntegration,
    listEnvironments,
    listOrganizations,
  } from '../lib/api.js'

  let integrations = []
  let environments = []
  let organizations = []
  let loading = true
  let error = ''
  let showCreateModal = false
  /** Step 1: null = provider picker; else provider id (e.g. 'aws') = provider-specific form. */
  let selectedProvider = null
  let syncingId = null
  let deletingId = null
  let openMenuId = null
  let menuDropdownStyle = { left: 0, top: 0 }

  // Create form (provider-specific; currently AWS)
  let createName = ''
  let createRegion = 'us-east-1'
  let createIpamScopeId = ''
  let createEnvChoice = 'existing' // 'existing' | 'new'
  let createEnvironmentId = ''
  let createNewEnvName = ''
  /** Sync interval in minutes; 0 = off. Default 5. */
  let createSyncIntervalMinutes = 5
  /** Sync mode: read_only | read_write. Default read_only. */
  let createSyncMode = 'read_only'
  /** Conflict resolution: cloud | ipam. Only used when sync mode is read-write. */
  let createConflictResolution = 'cloud'
  /** Which resources to sync (AWS). Default true. */
  let createSyncPools = true
  let createSyncBlocks = true
  let createSyncAllocations = true
  let creating = false
  let createError = ''

  // Edit form (AWS; same fields as create except no "new environment" option)
  let editingIntegrationId = null
  let editName = ''
  let editRegion = 'us-east-1'
  let editIpamScopeId = ''
  let editEnvironmentId = ''
  let editSyncIntervalMinutes = 5
  let editSyncMode = 'read_only'
  let editConflictResolution = 'cloud'
  let editSyncPools = true
  let editSyncBlocks = true
  let editSyncAllocations = true
  let editSubmitting = false
  let editError = ''

  const PROVIDERS = [
    { id: 'aws', label: 'AWS', icon: 'simple-icons:amazonaws', available: true },
    { id: 'azure', label: 'Azure', icon: 'simple-icons:microsoftazure', available: false },
    { id: 'gcp', label: 'GCP', icon: 'simple-icons:googlecloud', available: false },
  ]

  function getProviderInfo(providerId) {
    if (!providerId) return null
    const id = String(providerId).toLowerCase()
    return PROVIDERS.find((p) => p.id === id) || { id, label: providerId, icon: null }
  }

  function orgId() {
    if (!$user) return null
    if (isGlobalAdmin($user) && $selectedOrgForGlobalAdmin) return $selectedOrgForGlobalAdmin
    return $user.organization_id ?? null
  }

  function listOpts() {
    const opts = {}
    const oid = orgId()
    if (oid) opts.organization_id = oid
    return opts
  }

  async function load() {
    const oid = orgId()
    if (!oid) {
      integrations = []
      loading = false
      return
    }
    loading = true
    error = ''
    try {
      const res = await listIntegrations(listOpts())
      integrations = res.integrations || []
    } catch (e) {
      error = e?.message || 'Failed to load integrations'
      // Keep previous integrations so the table does not disappear
    } finally {
      loading = false
    }
  }

  async function loadEnvironments() {
    const oid = orgId()
    if (!oid) return
    try {
      const res = await listEnvironments({ limit: 500, organization_id: oid })
      environments = res.environments || []
    } catch {
      environments = []
    }
  }

  async function loadOrganizations() {
    if (!isGlobalAdmin($user)) return
    try {
      const res = await listOrganizations()
      organizations = res.organizations || []
    } catch {
      organizations = []
    }
  }

  $: orgId(), load()

  async function handleSync(id) {
    if (!id) return
    syncingId = id
    error = ''
    try {
      await syncIntegration(id)
      openMenuId = null
      await load()
    } catch (e) {
      error = e?.message || 'Sync failed'
    } finally {
      syncingId = null
    }
  }

  async function handleDelete(id) {
    if (!id) return
    deletingId = id
    try {
      await deleteIntegration(id)
      openMenuId = null
      await load()
    } catch (e) {
      error = e?.message || 'Failed to delete integration'
    } finally {
      deletingId = null
    }
  }

  function openCreate() {
    showCreateModal = true
    selectedProvider = null
    createName = ''
    createRegion = 'us-east-1'
    createIpamScopeId = ''
    createEnvChoice = 'existing'
    createEnvironmentId = ''
    createNewEnvName = ''
    createError = ''
  }

  function closeCreate() {
    showCreateModal = false
    selectedProvider = null
    createSyncIntervalMinutes = 5
    createSyncMode = 'read_only'
    createConflictResolution = 'cloud'
    createSyncPools = true
    createSyncBlocks = true
    createSyncAllocations = true
    createError = ''
  }

  function formatSyncMode(mode) {
    if (mode === 'read_write') return 'Read-write'
    return 'Read-only'
  }

  function formatConflictResolution(res) {
    if (res === 'ipam') return 'IPAM'
    return 'Cloud' // cloud, or legacy last-write-wins/app/integration/aws
  }

  function pickProvider(providerId) {
    if (providerId === 'aws') {
      selectedProvider = 'aws'
      loadEnvironments()
    }
  }

  function backToProviderPicker() {
    selectedProvider = null
    createError = ''
  }

  function openEdit(integration) {
    if (!integration || !integration.id) return
    const provider = String(integration.provider || '').toLowerCase()
    if (provider !== 'aws') return
    editingIntegrationId = integration.id
    editName = integration.name || ''
    const config = integration.config && typeof integration.config === 'object' ? integration.config : {}
    editRegion = config.region || config.Region || 'us-east-1'
    editIpamScopeId = config.ipam_scope_id ?? config.ipamScopeId ?? ''
    editEnvironmentId = config.environment_id ?? config.environmentId ?? ''
    editSyncIntervalMinutes = integration.sync_interval_minutes ?? integration.syncIntervalMinutes ?? 5
    editSyncMode = integration.sync_mode === 'read_write' ? 'read_write' : 'read_only'
    editConflictResolution = ['cloud', 'ipam'].includes(integration.conflict_resolution ?? integration.conflictResolution)
      ? (integration.conflict_resolution ?? integration.conflictResolution)
      : 'cloud'
    editSyncPools = config.sync_pools !== false
    editSyncBlocks = config.sync_blocks !== false
    editSyncAllocations = config.sync_allocations !== false
    editError = ''
    loadEnvironments()
  }

  function closeEdit() {
    editingIntegrationId = null
    editError = ''
  }

  async function handleUpdate() {
    if (!editingIntegrationId) return
    const name = (editName || '').trim()
    if (!name) {
      editError = 'Name is required'
      return
    }
    const envId = (editEnvironmentId || '').trim()
    if (!envId) {
      editError = 'Select an environment.'
      return
    }
    if (editSyncMode === 'read_write' && !(editIpamScopeId || '').trim()) {
      editError = 'IPAM scope ID is required for read-write sync.'
      return
    }
    editSubmitting = true
    editError = ''
    try {
      const config = { region: (editRegion || '').trim() || 'us-east-1', environment_id: envId }
      if ((editIpamScopeId || '').trim()) config.ipam_scope_id = editIpamScopeId.trim()
      config.sync_pools = editSyncPools
      config.sync_blocks = editSyncBlocks
      config.sync_allocations = editSyncAllocations
      const syncMins = Math.max(0, Math.min(1440, Math.floor(Number(editSyncIntervalMinutes) || 5)))
      const syncMode = editSyncMode === 'read_write' ? 'read_write' : 'read_only'
      const conflictRes = editSyncMode === 'read_write' && ['cloud', 'ipam'].includes(editConflictResolution) ? editConflictResolution : 'cloud'
      await updateIntegration(editingIntegrationId, name, config, syncMins, syncMode, conflictRes)
      closeEdit()
      openMenuId = null
      await load()
    } catch (e) {
      editError = e?.message ?? 'Failed to update integration'
    } finally {
      editSubmitting = false
    }
  }

  async function handleCreate() {
    const oid = orgId()
    if (!oid) return
    const name = (createName || '').trim()
    if (!name) {
      createError = 'Name is required'
      return
    }
    let envId = ''
    if (createEnvChoice === 'new') {
      const newEnvName = (createNewEnvName || '').trim()
      if (!newEnvName) {
        createError = 'Enter a name for the new environment.'
        return
      }
      envId = newEnvName // placeholder; we'll get real id after create
    } else {
      envId = (createEnvironmentId || '').trim()
      if (!envId) {
        createError = 'Select an environment, or choose "Create new environment" and enter a name.'
        return
      }
    }
    if (createSyncMode === 'read_write' && !(createIpamScopeId || '').trim()) {
      createError = 'IPAM scope ID is required for read-write sync so that pools created in IPAM can be created in AWS.'
      return
    }
    creating = true
    createError = ''
    try {
      if (createEnvChoice === 'new') {
        const newEnvName = (createNewEnvName || '').trim()
        const envRes = await createEnvironment(newEnvName, [], oid)
        envId = envRes.id ?? envRes.Id ?? ''
        if (!envId) {
          createError = 'Created environment but could not get its ID.'
          creating = false
          return
        }
      }
      const config = { region: (createRegion || '').trim() || 'us-east-1', environment_id: envId }
      if ((createIpamScopeId || '').trim()) config.ipam_scope_id = createIpamScopeId.trim()
      config.sync_pools = createSyncPools
      config.sync_blocks = createSyncBlocks
      config.sync_allocations = createSyncAllocations
      const syncMins = Math.max(0, Math.min(1440, Math.floor(Number(createSyncIntervalMinutes) || 5)))
      const syncMode = createSyncMode === 'read_write' ? 'read_write' : 'read_only'
      const conflictRes = createSyncMode === 'read_write' && ['cloud', 'ipam'].includes(createConflictResolution) ? createConflictResolution : 'cloud'
      await createIntegration(oid, 'aws', name, config, syncMins, syncMode, conflictRes)
      closeCreate()
      await load()
    } catch (e) {
      createError = e?.message ?? 'Failed to create integration'
    } finally {
      creating = false
    }
  }

  function formatSyncStatus(integration) {
    const status = integration.last_sync_status
    if (!status) return '—'
    if (status === 'success') return 'Success'
    if (status === 'failed') return 'Failed'
    if (status === 'syncing') return 'Syncing…'
    return status
  }

  function formatSyncAt(integration) {
    const at = integration.last_sync_at
    if (!at) return '—'
    try {
      const d = new Date(at)
      return d.toLocaleString()
    } catch {
      return at
    }
  }

  onMount(() => {
    function handleClickOutside(e) {
      if (!e.target.closest('.actions-menu-wrap')) openMenuId = null
    }
    document.addEventListener('click', handleClickOutside)
    if (isGlobalAdmin($user)) loadOrganizations()
    return () => document.removeEventListener('click', handleClickOutside)
  })
</script>

<div class="page">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Integrations</h1>
      <p class="page-desc">Connect cloud providers to sync pools and blocks into this organization.</p>
    </div>
    <button type="button" class="btn btn-primary" on:click={openCreate} disabled={!orgId()}>
      Add integration
    </button>
  </header>

  {#if !orgId() && isGlobalAdmin($user)}
    <div class="card card-muted">
      <p class="muted">Select an organization above to view or add integrations.</p>
    </div>
  {:else}
    <div class="card">
      {#if loading}
        <p class="muted">Loading…</p>
      {:else}
        <DataTable>
          <svelte:fragment slot="header">
            <tr>
              <th>Name</th>
              <th>Provider</th>
              <th>Mode</th>
              <th>Conflict</th>
              <th>Last sync</th>
              <th>Status</th>
              <th>Error</th>
              <th class="table-actions">Actions</th>
            </tr>
          </svelte:fragment>
          <svelte:fragment slot="body">
            {#if integrations.length === 0}
              <tr>
                <td colspan="8" class="table-empty-cell">No integrations yet. Add a connection to sync pools and blocks from your cloud provider.</td>
              </tr>
            {:else}
              {#each integrations as i (i.id)}
                <tr>
                  <td class="name">{i.name || '—'}</td>
                  <td class="provider-cell">
                    {#if getProviderInfo(i.provider)}
                      {@const info = getProviderInfo(i.provider)}
                      <span class="provider-cell-content" title={info.label}>
                        {#if info.icon}
                          <Icon icon={info.icon} width="1.5rem" height="1.5rem" class="provider-logo" aria-hidden="true" />
                        {/if}
                        <span class="provider-label">{info.label}</span>
                      </span>
                    {:else}
                      <span class="provider-badge">{i.provider || '—'}</span>
                    {/if}
                  </td>
                  <td class="mode-cell">{formatSyncMode(i.sync_mode)}</td>
                  <td class="conflict-cell">{i.sync_mode === 'read_write' ? formatConflictResolution(i.conflict_resolution) : '—'}</td>
                  <td class="sync-at">{formatSyncAt(i)}</td>
                  <td class="sync-status">
                    <span class="status status-{i.last_sync_status || 'none'}">{formatSyncStatus(i)}</span>
                  </td>
                  <td class="sync-error">
                    {#if i.last_sync_error}
                      <span class="error-text" title={i.last_sync_error}>{i.last_sync_error}</span>
                    {:else}
                      —
                    {/if}
                  </td>
                  <td class="table-actions">
                    <div class="actions-menu-wrap" role="group">
                      <button
                        type="button"
                        class="btn btn-small btn-primary"
                        disabled={syncingId !== null}
                        title="Sync pools and blocks now"
                        on:click={() => handleSync(i.id)}
                      >
                        {syncingId === i.id ? 'Syncing…' : 'Sync'}
                      </button>
                      <button
                        type="button"
                        class="menu-trigger"
                        aria-haspopup="true"
                        aria-expanded={openMenuId === i.id}
                        on:click|stopPropagation={(e) => {
                          if (openMenuId === i.id) openMenuId = null
                          else {
                            const rect = e.currentTarget.getBoundingClientRect()
                            menuDropdownStyle = { left: rect.right, top: rect.bottom + 2 }
                            openMenuId = i.id
                          }
                        }}
                        title="Actions"
                      >
                        <Icon icon="lucide:ellipsis-vertical" width="1.25em" height="1.25em" />
                      </button>
                      {#if openMenuId === i.id}
                        <div
                          class="menu-dropdown menu-dropdown-fixed"
                          role="menu"
                          style="position:fixed;left:{menuDropdownStyle.left}px;top:{menuDropdownStyle.top}px;transform:translateX(-100%);z-index:1000"
                        >
                          {#if String(i.provider || '').toLowerCase() === 'aws'}
                          <button
                            type="button"
                            role="menuitem"
                            disabled={editSubmitting}
                            on:click|stopPropagation={() => { openEdit(i); openMenuId = null }}
                          >
                            Edit
                          </button>
                          {/if}
                          <button
                            type="button"
                            role="menuitem"
                            class="menu-item-danger"
                            disabled={deletingId !== null}
                            on:click|stopPropagation={() => handleDelete(i.id)}
                          >
                            {deletingId === i.id ? 'Deleting…' : 'Delete'}
                          </button>
                        </div>
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
  {/if}
</div>

<!-- Error modal: show when error is set (load/sync/delete failed) -->
{#if error}
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="modal-backdrop"
    role="alertdialog"
    aria-labelledby="integrations-error-title"
    aria-describedby="integrations-error-desc"
    on:click={() => (error = '')}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); error = ''; } }}
  >
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal error-dialog" role="document" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="integrations-error-title">Error</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={() => (error = '')}>×</button>
      </div>
      <div class="modal-body">
        <p id="integrations-error-desc" class="error-modal-message">{error}</p>
      </div>
      <div class="modal-actions">
        <button type="button" class="btn btn-primary" on:click={() => (error = '')}>Dismiss</button>
      </div>
    </div>
  </div>
{/if}

<!-- Create modal: step 1 = provider picker, step 2 = provider-specific form -->
<svelte:window on:keydown={(e) => {
  if (e.key !== 'Escape') return
  if (error) error = ''
  else if (editingIntegrationId) closeEdit()
  else if (showCreateModal) selectedProvider ? backToProviderPicker() : closeCreate()
}} />
{#if editingIntegrationId}
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Close modal"
    on:click={closeEdit}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); closeEdit(); } }}
  >
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal" role="dialog" aria-labelledby="integrations-edit-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="integrations-edit-title">Edit integration — AWS</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={closeEdit}>×</button>
      </div>
      <div class="modal-body">
        {#if editError}
          <div class="modal-error" role="alert">{editError}</div>
        {/if}
        <div class="create-section">
          <label for="edit-sync-mode">Sync mode</label>
          <div class="create-row">
            <select id="edit-sync-mode" bind:value={editSyncMode} disabled={editSubmitting} aria-label="Sync mode">
              <option value="read_only">Read-only (sync from cloud only)</option>
              <option value="read_write">Read-write (bi-directional)</option>
            </select>
          </div>
          {#if editSyncMode === 'read_write'}
            <label for="edit-conflict-resolution">Conflict resolution</label>
            <div class="create-row">
              <select id="edit-conflict-resolution" bind:value={editConflictResolution} disabled={editSubmitting} aria-label="Conflict resolution">
                <option value="cloud">Cloud</option>
                <option value="ipam">IPAM</option>
              </select>
            </div>
            {#if editConflictResolution === 'ipam'}
              <div class="create-warning" role="alert">
                <strong>Warning:</strong> With IPAM as source of truth, deleting pools, blocks, or allocations in IPAM will cause those resources to be deleted in AWS on the next sync.
              </div>
            {/if}
          {/if}
          <span class="create-label-block">Resources to sync</span>
          <div class="sync-resources-row" role="group" aria-label="Resources to sync">
            <label class="sync-resource-cb">
              <input type="checkbox" bind:checked={editSyncPools} disabled={editSubmitting} />
              <span>Pools (IPAM pools)</span>
            </label>
            <label class="sync-resource-cb">
              <input type="checkbox" bind:checked={editSyncBlocks} disabled={editSubmitting} />
              <span>Blocks (VPCs)</span>
            </label>
            <label class="sync-resource-cb">
              <input type="checkbox" bind:checked={editSyncAllocations} disabled={editSubmitting} />
              <span>Allocations (subnets)</span>
            </label>
          </div>
          <label for="edit-name">Name <span class="required">*</span></label>
          <div class="create-row">
            <input id="edit-name" type="text" bind:value={editName} placeholder="e.g. Production" disabled={editSubmitting} />
          </div>
          <label for="edit-region">Region</label>
          <div class="create-row">
            <input id="edit-region" type="text" bind:value={editRegion} placeholder="e.g. us-east-1" disabled={editSubmitting} />
          </div>
          <label for="edit-ipam-scope">
            Scope ID
            {#if editSyncMode === 'read_write'}
              <span class="required">*</span>
            {:else}
              <span class="optional">(optional)</span>
            {/if}
          </label>
          <div class="create-row">
            <input
              id="edit-ipam-scope"
              type="text"
              bind:value={editIpamScopeId}
              placeholder={editSyncMode === 'read_write' ? 'e.g. ipam-scope-xxxxxxxx' : 'optional scope filter'}
              disabled={editSubmitting}
            />
          </div>
          <label for="edit-environment">Environment <span class="required">*</span></label>
          <div class="create-row">
            <select id="edit-environment" bind:value={editEnvironmentId} disabled={editSubmitting} aria-label="Select environment">
              <option value="">Select environment…</option>
              {#each environments as env (env.id)}
                <option value={env.id}>{env.name}</option>
              {/each}
            </select>
          </div>
          <label for="edit-sync-interval">Background sync</label>
          <div class="create-row">
            <select id="edit-sync-interval" bind:value={editSyncIntervalMinutes} disabled={editSubmitting} aria-label="Sync interval">
              <option value={0}>Off</option>
              <option value={5}>Every 5 minutes</option>
              <option value={10}>Every 10 minutes</option>
              <option value={15}>Every 15 minutes</option>
              <option value={30}>Every 30 minutes</option>
              <option value={60}>Every 60 minutes</option>
            </select>
          </div>
        </div>
      </div>
      <div class="modal-actions">
        <button type="button" class="btn" on:click={closeEdit} disabled={editSubmitting}>Cancel</button>
        <button type="button" class="btn btn-primary" on:click={handleUpdate} disabled={editSubmitting}>
          {editSubmitting ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  </div>
{/if}

{#if showCreateModal}
  {@const selectedProviderLabel = selectedProvider ? (PROVIDERS.find(p => p.id === selectedProvider)?.label ?? '') : ''}
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Close modal"
    on:click={() => { if (selectedProvider) backToProviderPicker(); else closeCreate(); }}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); if (selectedProvider) backToProviderPicker(); else closeCreate(); } }}
  >
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal" class:modal-wide={selectedProvider === null} role="dialog" aria-labelledby="integrations-create-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="integrations-create-title">Add integration{#if selectedProviderLabel} — {selectedProviderLabel}{/if}</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={closeCreate}>×</button>
      </div>

      <div class="modal-body">
        {#if selectedProvider === null}
          <p class="modal-desc">Choose a cloud provider to connect. Synced pools and blocks will appear in this organization.</p>
          <div class="provider-picker">
            {#each PROVIDERS as provider (provider.id)}
              <button
                type="button"
                class="provider-option"
                class:disabled={!provider.available}
                class:coming-soon={!provider.available}
                disabled={!provider.available}
                on:click={() => pickProvider(provider.id)}
              >
                <span class="provider-option-icon" aria-hidden="true">
                  <Icon icon={provider.icon} width="2.5rem" height="2.5rem" />
                </span>
                <span class="provider-option-label">{provider.label}</span>
                {#if !provider.available}
                  <span class="provider-option-badge">Coming soon</span>
                {/if}
              </button>
            {/each}
          </div>
        {:else if selectedProvider === 'aws'}
          <p class="modal-desc">Sync pools and allocations (as blocks) from this provider into your organization. Configure region and scope as needed; map to an environment so synced resources appear there.</p>

          <details class="setup-details">
            <summary>Setup &amp; authentication</summary>
            <div class="setup-content">
              <p><strong>How authentication works</strong></p>
              <p>Credentials are <em>not</em> stored in the app. The server uses the <strong>AWS default credential chain</strong> for both <strong>pull</strong> (sync from AWS) and <strong>push</strong> (when read-write: creating pools, blocks, or allocations in AWS from the app). The chain is used in this order:</p>
              <ol>
                <li><strong>Environment variables</strong> — <code>AWS_ACCESS_KEY_ID</code>, <code>AWS_SECRET_ACCESS_KEY</code>, and optionally <code>AWS_SESSION_TOKEN</code></li>
                <li><strong>Shared config</strong> — <code>~/.aws/credentials</code> (e.g. after <code>aws configure</code>)</li>
                <li><strong>IAM role</strong> — If the app runs on EC2, ECS, Lambda, or similar, the attached instance/task/execution role is used (recommended for production)</li>
              </ol>
              <p>Run this app in an environment that already has AWS credentials (e.g. EC2 with an instance profile, or set the env vars, or use local <code>~/.aws/credentials</code> for dev). No credentials are sent from the browser or stored in the database.</p>
              <p><strong>Read-only vs read-write</strong></p>
              <ul>
                <li><strong>Read-only</strong> — Sync (pull) only. The server uses the credential chain when you run sync to read IPAM pools and allocations from AWS.</li>
                <li><strong>Read-write</strong> — Sync (pull) plus push. The <em>same</em> credentials are used when the app creates or updates pools, blocks, or allocations in AWS (e.g. creating a pool in the app that is pushed to AWS). Ensure the identity has both read and write IAM permissions if you use read-write.</li>
              </ul>
              <p><strong>Required IAM permissions</strong></p>
              <p><strong>Pull (sync)</strong> — the identity must be allowed to call:</p>
              <ul>
                <li><code>ec2:DescribeIpamPools</code></li>
                <li><code>ec2:GetIpamPoolAllocations</code></li>
                <li><code>ec2:DescribeSubnets</code></li> (for syncing subnets as allocations)
              </ul>
              <p><strong>Push (read-write only)</strong> — when using read-write sync, the identity also needs permissions to create resources in AWS, for example:</p>
              <ul>
                <li><code>ec2:CreateIpamPool</code></li>
                <li><code>ec2:AllocateIpamPoolCidr</code></li>
                <li><code>ec2:CreateSubnet</code></li>
              </ul>
              <p>Scoped to the region and IPAM resources you use.</p>
            </div>
          </details>

          {#if createError}
            <div class="modal-error" role="alert">{createError}</div>
          {/if}

          <div class="create-section">
            <label for="create-sync-mode">Sync mode</label>
            <div class="create-row">
              <select id="create-sync-mode" bind:value={createSyncMode} disabled={creating} aria-label="Sync mode">
                <option value="read_only">Read-only (sync from cloud only)</option>
                <option value="read_write">Read-write (bi-directional)</option>
              </select>
            </div>
            {#if createSyncMode === 'read_write'}
              <label for="create-conflict-resolution">Conflict resolution</label>
              <div class="create-row">
                <select id="create-conflict-resolution" bind:value={createConflictResolution} disabled={creating} aria-label="Conflict resolution">
                  <option value="cloud">Cloud</option>
                  <option value="ipam">IPAM</option>
                </select>
              </div>
              <p class="create-hint">
                {#if createConflictResolution === 'cloud'}
                  When the same resource exists in both IPAM and cloud, the cloud version overwrites IPAM. Cloud is source of truth after sync.
                {:else}
                  When the same resource exists in both IPAM and cloud, the IPAM version is kept. New resources from the cloud are still created.
                {/if}
              </p>
              {#if createConflictResolution === 'ipam'}
                <div class="create-warning" role="alert">
                  <strong>Warning:</strong> With IPAM as source of truth, deleting pools, blocks, or allocations in IPAM will cause those resources to be deleted in AWS on the next sync.
                </div>
              {/if}
            {/if}
            <span class="create-label-block">Resources to sync</span>
            <div class="sync-resources-row" role="group" aria-label="Resources to sync">
              <label class="sync-resource-cb">
                <input type="checkbox" bind:checked={createSyncPools} disabled={creating} />
                <span>Pools (IPAM pools)</span>
              </label>
              <label class="sync-resource-cb">
                <input type="checkbox" bind:checked={createSyncBlocks} disabled={creating} />
                <span>Blocks (VPCs)</span>
              </label>
              <label class="sync-resource-cb">
                <input type="checkbox" bind:checked={createSyncAllocations} disabled={creating} />
                <span>Allocations (subnets)</span>
              </label>
            </div>
            <label for="create-name">Name <span class="required">*</span></label>
            <div class="create-row">
              <input id="create-name" type="text" bind:value={createName} placeholder="e.g. Production" disabled={creating} />
            </div>
            <label for="create-region">Region</label>
            <div class="create-row">
              <input id="create-region" type="text" bind:value={createRegion} placeholder="e.g. us-east-1" disabled={creating} />
            </div>
            <label for="create-ipam-scope">
              Scope ID
              {#if createSyncMode === 'read_write'}
                <span class="required">*</span>
              {:else}
                <span class="optional">(optional)</span>
              {/if}
            </label>
            <div class="create-row">
              <input
                id="create-ipam-scope"
                type="text"
                bind:value={createIpamScopeId}
                placeholder={createSyncMode === 'read_write' ? 'e.g. ipam-scope-xxxxxxxx' : 'optional scope filter'}
                disabled={creating}
                aria-required={createSyncMode === 'read_write'}
              />
            </div>
            {#if createSyncMode === 'read_write'}
              <p class="create-hint">Required for read-write: creating pools in IPAM (in this environment) will create them in AWS; the scope ID is needed for that.</p>
            {:else}
              <p class="create-hint">Optionally limit sync to pools in this IPAM scope. Leave blank to sync all scopes (read-only).</p>
            {/if}
            <span class="create-label-block">Environment <span class="required">*</span></span>
            <div class="create-env-choice" role="radiogroup" aria-label="Environment source">
              <label class="env-option" class:selected={createEnvChoice === 'existing'}>
                <input type="radio" name="env-choice" value="existing" bind:group={createEnvChoice} disabled={creating} class="env-option-input" />
                <span class="env-option-content">
                  <span class="env-option-title">Use existing environment</span>
                  <span class="env-option-desc">Attach synced pools and blocks to an environment you already have.</span>
                </span>
              </label>
              <label class="env-option" class:selected={createEnvChoice === 'new'}>
                <input type="radio" name="env-choice" value="new" bind:group={createEnvChoice} disabled={creating} class="env-option-input" />
                <span class="env-option-content">
                  <span class="env-option-title">Create new environment</span>
                  <span class="env-option-desc">Pools will come from sync. A new environment is created with no pools; run sync to pull them in.</span>
                </span>
              </label>
            </div>
            {#if createEnvChoice === 'existing'}
              <div class="create-row">
                <select id="create-environment" bind:value={createEnvironmentId} disabled={creating} aria-label="Select environment">
                  <option value="">Select environment…</option>
                  {#each environments as env (env.id)}
                    <option value={env.id}>{env.name}</option>
                  {/each}
                </select>
              </div>
              {#if environments.length === 0}
                <p class="create-hint">No environments yet. Choose "Create new environment" above, or create one on the Environments page first.</p>
              {/if}
            {:else}
              <div class="create-row">
                <input
                  id="create-new-env-name"
                  type="text"
                  bind:value={createNewEnvName}
                  placeholder="e.g. AWS Production"
                  disabled={creating}
                  aria-label="New environment name"
                />
              </div>
              <p class="create-hint">A new environment will be created with no pools. When you run sync, pools and blocks from the provider will appear here.</p>
            {/if}
            <label for="create-sync-interval">Background sync</label>
            <div class="create-row">
              <select id="create-sync-interval" bind:value={createSyncIntervalMinutes} disabled={creating} aria-label="Sync interval">
                <option value={0}>Off</option>
                <option value={5}>Every 5 minutes (default)</option>
                <option value={10}>Every 10 minutes</option>
                <option value={15}>Every 15 minutes</option>
                <option value={30}>Every 30 minutes</option>
                <option value={60}>Every 60 minutes</option>
              </select>
            </div>
            <p class="create-hint">Cloud data syncs in the background at this interval. You can also sync manually from the table.</p>
          </div>
        {/if}
      </div>

      {#if selectedProvider === 'aws'}
        <div class="modal-actions">
          <button type="button" class="btn" on:click={backToProviderPicker} disabled={creating}>Back</button>
          <button type="button" class="btn" on:click={closeCreate} disabled={creating}>Cancel</button>
          <button type="button" class="btn btn-primary" on:click={handleCreate} disabled={creating}>
            {creating ? 'Creating…' : 'Create'}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

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
  .card-muted {
    padding: 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-muted);
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
  .provider-cell-content {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
  }
  .provider-logo {
    flex-shrink: 0;
    color: var(--text);
  }
  .provider-label {
    font-size: 0.875rem;
    font-weight: 500;
    text-transform: capitalize;
  }
  .provider-badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: uppercase;
    background: var(--accent-dim);
    color: var(--accent);
    border-radius: var(--radius);
  }
  .sync-at {
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .sync-status .status {
    font-size: 0.85rem;
  }
  .status-success {
    color: var(--success, #16a34a);
  }
  .status-failed {
    color: var(--danger);
  }
  .status-syncing {
    color: var(--accent);
  }
  .sync-error .error-text {
    max-width: 220px;
    display: inline-block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.8rem;
    color: var(--danger);
  }
  .actions-menu-wrap {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }
  .menu-trigger {
    display: inline-flex;
    padding: 0.25rem;
    background: transparent;
    border: none;
    border-radius: var(--radius);
    color: var(--text-muted);
    cursor: pointer;
  }
  .menu-trigger:hover {
    color: var(--text);
    background: var(--accent-dim);
  }
  .menu-dropdown {
    min-width: 8rem;
    padding: 0.2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
  }
  .menu-dropdown button {
    display: block;
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: none;
    border-radius: calc(var(--radius) - 2px);
    background: transparent;
    color: var(--text);
    font-size: 0.9rem;
    text-align: left;
    cursor: pointer;
  }
  .menu-dropdown button:hover {
    background: var(--accent-dim);
  }
  .menu-item-danger {
    color: var(--danger) !important;
  }
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1rem;
  }
  .modal {
    display: flex;
    flex-direction: column;
    max-height: calc(100vh - 2rem);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-lg);
    max-width: 420px;
    width: 100%;
    padding: 1.5rem;
  }
  .modal-wide {
    max-width: 480px;
  }
  .modal-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }
  .provider-picker {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
    margin-top: 0.5rem;
  }
  .provider-option {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    padding: 1.25rem 1rem;
    background: var(--surface-elevated, var(--bg));
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.95rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
  }
  .provider-option:hover:not(.coming-soon) {
    background: var(--accent-dim);
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .provider-option.coming-soon {
    opacity: 0.6;
    cursor: not-allowed;
    position: relative;
  }
  .provider-option.coming-soon .provider-option-icon {
    filter: grayscale(1);
  }
  .provider-option-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--text);
  }
  .provider-option-icon :global(svg) {
    width: 2.5rem;
    height: 2.5rem;
  }
  .provider-option-label {
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .provider-option-badge {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
    padding: 0.2rem 0.5rem;
    background: var(--border);
    border-radius: var(--radius);
  }
  .modal-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }
  .modal-header h2 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
  }
  .modal-close {
    background: none;
    border: none;
    font-size: 1.5rem;
    line-height: 1;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.25rem;
  }
  .modal-close:hover {
    color: var(--text);
  }
  .modal-desc {
    margin: 0 0 1rem;
    font-size: 0.9rem;
    color: var(--text-muted);
    line-height: 1.4;
  }
  .setup-details {
    margin-bottom: 1rem;
    padding: 0.75rem 1rem;
    background: var(--surface-elevated, rgba(255, 255, 255, 0.04));
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-size: 0.85rem;
  }
  .setup-details summary {
    cursor: pointer;
    font-weight: 600;
    color: var(--text);
    user-select: none;
  }
  .setup-details summary:hover {
    color: var(--accent);
  }
  .setup-content {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    line-height: 1.5;
  }
  .setup-content p {
    margin: 0 0 0.5rem;
  }
  .setup-content p:last-child {
    margin-bottom: 0;
  }
  .setup-content strong {
    color: var(--text);
  }
  .setup-content code {
    font-family: var(--font-mono);
    font-size: 0.8em;
    padding: 0.1rem 0.3rem;
    background: var(--bg);
    border-radius: calc(var(--radius) - 2px);
  }
  .setup-content ol,
  .setup-content ul {
    margin: 0.25rem 0 0.5rem 1.25rem;
    padding: 0;
  }
  .setup-content li {
    margin-bottom: 0.25rem;
  }
  .error-dialog .error-modal-message {
    margin: 0;
    color: var(--text);
    word-break: break-word;
  }
  .modal-error {
    margin-bottom: 1rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.9rem;
    color: var(--danger);
    background: rgba(220, 38, 38, 0.1);
    border-radius: var(--radius);
  }
  .create-section label {
    display: block;
    font-size: 0.85rem;
    font-weight: 500;
    margin-bottom: 0.25rem;
    color: var(--text);
  }
  .create-label-block {
    margin-top: 0.5rem;
  }
  .create-section .required {
    color: var(--danger);
  }
  .create-section .optional {
    color: var(--text-muted);
    font-weight: normal;
  }
  .create-row {
    margin-bottom: 1rem;
  }
  .sync-resources-row {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem 1.5rem;
    margin-bottom: 1rem;
  }
  .sync-resource-cb {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-weight: normal;
    cursor: pointer;
    font-size: 0.9rem;
    color: var(--text);
  }
  .sync-resource-cb input {
    margin: 0;
  }
  .create-env-choice {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  .env-option {
    position: relative;
    display: block;
    cursor: pointer;
    padding: 0.875rem 1rem;
    background: var(--surface-elevated, rgba(255, 255, 255, 0.04));
    border: 1px solid var(--border);
    border-radius: var(--radius);
    transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
  }
  .env-option:hover {
    background: var(--accent-dim);
    border-color: var(--border);
  }
  .env-option.selected {
    border-color: var(--accent);
    background: var(--accent-dim);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .env-option:focus-within {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  .env-option-input {
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
  .env-option-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .env-option-title {
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--text);
  }
  .env-option-desc {
    font-size: 0.8rem;
    color: var(--text-muted);
    line-height: 1.35;
  }
  .create-hint {
    margin: -0.5rem 0 1rem;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .create-warning {
    margin: 0 0 1rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.85rem;
    color: var(--text);
    background: rgba(245, 158, 11, 0.15);
    border: 1px solid rgba(245, 158, 11, 0.4);
    border-radius: var(--radius);
  }
  .create-warning strong {
    color: #d97706;
  }
  .create-row input,
  .create-row select {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.9rem;
    box-sizing: border-box;
  }
  .modal-actions {
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
  }
</style>
