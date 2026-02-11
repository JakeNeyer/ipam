<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import '../lib/theme.js'
  import { listUsers, listTokens, deleteToken, listSignupInvites, revokeSignupInvite, updateUserRole, updateUserOrganization, deleteUser, listOrganizations, createOrganization, updateOrganization, deleteOrganization } from '../lib/api.js'
  import { user } from '../lib/auth.js'
  import ApiTokensModal from '../lib/ApiTokensModal.svelte'
  import AddUserModal from '../lib/AddUserModal.svelte'
  import SignupInviteModal from '../lib/SignupInviteModal.svelte'

  /** Global admin has no organization_id (only they can create/list organizations). */
  $: isGlobalAdmin = $user && ($user.organization_id == null || $user.organization_id === '')

  let users = []
  let loading = true
  let error = ''
  let showAddUserModal = false
  let showApiTokensModal = false
  let showSignupInviteModal = false
  let invites = []
  let invitesLoading = true
  let invitesError = ''
  let revokingInviteId = null
  let tokens = []
  let tokensLoading = true
  let tokensError = ''
  let deletingTokenId = null
  let updatingUserRoleId = null
  let updatingUserOrgId = null
  let deletingUserId = null

  let organizations = []
  let organizationsLoading = true
  let organizationsError = ''
  let showCreateOrgForm = false
  let newOrgName = ''
  let creatingOrg = false
  let createOrgError = ''
  let editingOrgId = null
  let editingOrgName = ''
  let updatingOrgId = null
  let deletingOrgId = null

  let userSortBy = 'email' // 'email' | 'role'
  let userSortDir = 'asc'
  function setUserSort(column) {
    if (userSortBy === column) userSortDir = userSortDir === 'asc' ? 'desc' : 'asc'
    else { userSortBy = column; userSortDir = 'asc' }
  }
  $: sortedUsers = (() => {
    const list = [...users]
    const mult = userSortDir === 'asc' ? 1 : -1
    if (userSortBy === 'email') list.sort((a, b) => mult * (a.email || '').localeCompare(b.email || '', undefined, { sensitivity: 'base' }))
    else if (userSortBy === 'role') list.sort((a, b) => mult * (a.role || '').localeCompare(b.role || '', undefined, { sensitivity: 'base' }))
    return list
  })()

  let tokenSortBy = 'name' // 'name' | 'created' | 'expires'
  let tokenSortDir = 'asc'
  function setTokenSort(column) {
    if (tokenSortBy === column) tokenSortDir = tokenSortDir === 'asc' ? 'desc' : 'asc'
    else { tokenSortBy = column; tokenSortDir = 'asc' }
  }
  $: sortedTokens = (() => {
    const list = [...tokens]
    const mult = tokenSortDir === 'asc' ? 1 : -1
    if (tokenSortBy === 'name') list.sort((a, b) => mult * (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
    else if (tokenSortBy === 'created') list.sort((a, b) => mult * (new Date(a.created_at || 0).getTime() - new Date(b.created_at || 0).getTime()))
    else if (tokenSortBy === 'expires') list.sort((a, b) => {
      const da = a.expires_at ? new Date(a.expires_at).getTime() : Number.MAX_SAFE_INTEGER
      const db = b.expires_at ? new Date(b.expires_at).getTime() : Number.MAX_SAFE_INTEGER
      return mult * (da - db)
    })
    return list
  })()

  async function load() {
    loading = true
    error = ''
    try {
      const res = await listUsers()
      users = res.users || []
    } catch (e) {
      error = e.message || 'Failed to load users'
    } finally {
      loading = false
    }
  }

  async function handleUpdateUserRole(id, role) {
    if (!id || !role) return
    updatingUserRoleId = id
    try {
      const updated = await updateUserRole(id, role)
      if (updated) {
        users = users.map((u) => (u.id === id ? { ...u, role: updated.role } : u))
      } else {
        await load()
      }
    } catch (e) {
      error = e?.message || 'Failed to update user role'
      await load()
    } finally {
      updatingUserRoleId = null
    }
  }

  async function handleUpdateUserOrganization(userId, organizationId) {
    if (!userId) return
    updatingUserOrgId = userId
    try {
      const updated = await updateUserOrganization(userId, organizationId || '')
      if (updated) {
        users = users.map((u) => (u.id === userId ? { ...u, organization_id: updated.organization_id ?? '' } : u))
      } else {
        await load()
      }
    } catch (e) {
      error = e?.message || 'Failed to update user organization'
      await load()
    } finally {
      updatingUserOrgId = null
    }
  }

  async function handleDeleteUser(id) {
    if (!id) return
    deletingUserId = id
    try {
      await deleteUser(id)
      users = users.filter((u) => u.id !== id)
    } catch (e) {
      error = e?.message || 'Failed to delete user'
      await load()
    } finally {
      deletingUserId = null
    }
  }

  async function loadInvites() {
    invitesLoading = true
    invitesError = ''
    try {
      const res = await listSignupInvites()
      invites = res.invites || []
    } catch (e) {
      invitesError = e?.message || 'Failed to load signup links'
      invites = []
    } finally {
      invitesLoading = false
    }
  }

  async function handleRevokeInvite(id) {
    if (!id) return
    revokingInviteId = id
    try {
      await revokeSignupInvite(id)
      await loadInvites()
    } catch (e) {
      invitesError = e?.message || 'Failed to revoke link'
    } finally {
      revokingInviteId = null
    }
  }

  function inviteStatus(inv) {
    if (inv.used_at) return 'Used'
    if (new Date(inv.expires_at) < new Date()) return 'Expired'
    return 'Active'
  }

  function formatInviteDate(iso) {
    if (!iso) return ''
    try {
      return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'short', timeStyle: 'short' })
    } catch {
      return iso
    }
  }

  async function loadTokens() {
    tokensLoading = true
    tokensError = ''
    try {
      const res = await listTokens()
      tokens = res.tokens || []
    } catch (e) {
      tokensError = e?.message || 'Failed to load tokens'
      tokens = []
    } finally {
      tokensLoading = false
    }
  }

  async function handleDeleteToken(id) {
    if (!id) return
    deletingTokenId = id
    try {
      await deleteToken(id)
      await loadTokens()
    } catch (e) {
      tokensError = e?.message || 'Failed to delete token'
    } finally {
      deletingTokenId = null
    }
  }

  function formatTokenDate(iso) {
    if (!iso) return ''
    try {
      return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'short' })
    } catch {
      return iso
    }
  }

  function formatExpiry(expiresAt) {
    if (expiresAt == null || expiresAt === '') return 'Never'
    try {
      const d = new Date(expiresAt)
      if (isNaN(d.getTime())) return expiresAt
      if (d < new Date()) return 'Expired'
      return d.toLocaleDateString(undefined, { dateStyle: 'short' })
    } catch {
      return expiresAt
    }
  }

  function isExpired(expiresAt) {
    if (expiresAt == null || expiresAt === '') return false
    try {
      return new Date(expiresAt) < new Date()
    } catch {
      return false
    }
  }

  async function loadOrganizations() {
    organizationsLoading = true
    organizationsError = ''
    try {
      const res = await listOrganizations()
      organizations = res.organizations || []
    } catch (e) {
      organizationsError = e?.message || 'Failed to load organizations'
      organizations = []
    } finally {
      organizationsLoading = false
    }
  }

  function openCreateOrgForm() {
    showCreateOrgForm = true
    newOrgName = ''
    createOrgError = ''
  }

  function closeCreateOrgForm() {
    showCreateOrgForm = false
    newOrgName = ''
    createOrgError = ''
  }

  async function handleCreateOrganization() {
    const name = newOrgName?.trim()
    if (!name) {
      createOrgError = 'Name is required'
      return
    }
    creatingOrg = true
    createOrgError = ''
    try {
      await createOrganization(name)
      await loadOrganizations()
      closeCreateOrgForm()
    } catch (e) {
      createOrgError = e?.message || 'Failed to create organization'
    } finally {
      creatingOrg = false
    }
  }

  function startEditOrg(org) {
    editingOrgId = org.id
    editingOrgName = org.name
    organizationsError = ''
  }

  function cancelEditOrg() {
    editingOrgId = null
    editingOrgName = ''
  }

  async function handleUpdateOrganization(orgId, name) {
    const trimmed = name?.trim()
    if (!trimmed) return
    updatingOrgId = orgId
    organizationsError = ''
    try {
      const data = await updateOrganization(orgId, trimmed)
      organizations = organizations.map((o) => (o.id === orgId ? { ...o, name: data.organization?.name ?? trimmed, created_at: o.created_at } : o))
      cancelEditOrg()
    } catch (e) {
      organizationsError = e?.message || 'Failed to update organization'
    } finally {
      updatingOrgId = null
    }
  }

  async function handleDeleteOrganization(orgId) {
    if (!orgId) return
    if (!confirm('Delete this organization? It must have no users and no environments.')) return
    deletingOrgId = orgId
    organizationsError = ''
    try {
      await deleteOrganization(orgId)
      organizations = organizations.filter((o) => o.id !== orgId)
    } catch (e) {
      organizationsError = e?.message || 'Failed to delete organization'
    } finally {
      deletingOrgId = null
    }
  }

  function formatOrgDate(iso) {
    if (!iso) return ''
    try {
      return new Date(iso).toLocaleDateString(undefined, { dateStyle: 'short' })
    } catch {
      return iso
    }
  }

  onMount(() => {
    load()
    loadInvites()
    loadTokens()
    loadOrganizations()
  })
</script>

<div class="admin-page">
  <header class="page-header">
    <h1 class="page-title">Admin</h1>
  </header>

  <AddUserModal open={showAddUserModal} isGlobalAdmin={isGlobalAdmin} organizations={organizations} on:close={() => (showAddUserModal = false)} on:created={load} />
  <ApiTokensModal open={showApiTokensModal} on:close={() => { showApiTokensModal = false; loadTokens() }} />
  <SignupInviteModal open={showSignupInviteModal} isGlobalAdmin={isGlobalAdmin} organizations={organizations} on:close={() => { showSignupInviteModal = false; loadInvites() }} />

  {#if isGlobalAdmin}
  <div class="admin-card">
    <div class="admin-card-header">
      <h2 class="admin-card-title">Organizations</h2>
      {#if !showCreateOrgForm}
        <button type="button" class="btn btn-primary btn-small" on:click={openCreateOrgForm}>
          Create organization
        </button>
      {/if}
    </div>
    <p class="admin-muted">Organizations are tenants. Only the global admin can create organizations. Users and environments belong to an organization.</p>
    {#if showCreateOrgForm}
      <div class="create-org-form">
        <input
          type="text"
          class="create-org-input"
          placeholder="Organization name"
          bind:value={newOrgName}
          on:keydown={(e) => e.key === 'Enter' && handleCreateOrganization()}
        />
        <div class="create-org-actions">
          <button type="button" class="btn btn-primary btn-small" disabled={creatingOrg || !newOrgName?.trim()} on:click={handleCreateOrganization}>
            {creatingOrg ? 'Creating…' : 'Create'}
          </button>
          <button type="button" class="btn btn-secondary btn-small" disabled={creatingOrg} on:click={closeCreateOrgForm}>
            Cancel
          </button>
        </div>
        {#if createOrgError}
          <p class="admin-error">{createOrgError}</p>
        {/if}
      </div>
    {:else if organizationsLoading}
      <p class="admin-muted">Loading…</p>
    {:else if organizationsError}
      <p class="admin-error">{organizationsError}</p>
    {:else if organizations.length === 0}
      <p class="admin-muted">No organizations yet. Create one to assign users and environments to it.</p>
    {:else}
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Created</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each organizations as org (org.id)}
              <tr>
                <td class="name">
                  {#if editingOrgId === org.id}
                    <input
                      type="text"
                      class="org-name-input"
                      bind:value={editingOrgName}
                      on:keydown={(e) => {
                        if (e.key === 'Enter') handleUpdateOrganization(org.id, editingOrgName)
                        else if (e.key === 'Escape') cancelEditOrg()
                      }}
                    />
                  {:else}
                    {org.name}
                  {/if}
                </td>
                <td>{formatOrgDate(org.created_at)}</td>
                <td class="table-actions">
                  {#if editingOrgId === org.id}
                    <button
                      type="button"
                      class="btn btn-primary btn-small"
                      disabled={updatingOrgId === org.id || !editingOrgName?.trim()}
                      on:click={() => handleUpdateOrganization(org.id, editingOrgName)}
                    >
                      {updatingOrgId === org.id ? 'Saving…' : 'Save'}
                    </button>
                    <button type="button" class="btn btn-secondary btn-small" disabled={updatingOrgId === org.id} on:click={cancelEditOrg}>
                      Cancel
                    </button>
                  {:else}
                    <button
                      type="button"
                      class="btn btn-secondary btn-small"
                      disabled={deletingOrgId === org.id}
                      on:click={() => startEditOrg(org)}
                      title="Edit name"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      class="btn btn-danger btn-small"
                      disabled={deletingOrgId === org.id}
                      on:click={() => handleDeleteOrganization(org.id)}
                      title="Delete organization"
                    >
                      {deletingOrgId === org.id ? 'Deleting…' : 'Delete'}
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      {#if organizationsError}
        <p class="admin-error">{organizationsError}</p>
      {/if}
    {/if}
  </div>
  {/if}

  <div class="admin-card">
    <div class="admin-card-header">
      <h2 class="admin-card-title">Signup links</h2>
      <button type="button" class="btn btn-primary btn-small" on:click={() => (showSignupInviteModal = true)}>
        Create signup link
      </button>
    </div>
    <p class="admin-muted">Create a time-bound link for new users to sign up. The link expires after the chosen duration and can only be used once.</p>
    {#if invitesLoading}
      <p class="admin-muted">Loading…</p>
    {:else if invitesError}
      <p class="admin-error">{invitesError}</p>
    {:else if invites.length === 0}
      <p class="admin-muted">No signup links yet.</p>
    {:else}
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>Created</th>
              <th>Expires</th>
              <th>Status</th>
              <th>Used by</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each invites as inv (inv.id)}
              <tr class:invite-expired={inviteStatus(inv) === 'Expired'}>
                <td>{formatInviteDate(inv.created_at)}</td>
                <td>{formatInviteDate(inv.expires_at)}</td>
                <td>
                  <span class="invite-status-badge" class:used={inviteStatus(inv) === 'Used'} class:expired={inviteStatus(inv) === 'Expired'}>
                    {inviteStatus(inv)}
                  </span>
                </td>
                <td>{inv.used_by_email || '—'}</td>
                <td class="table-actions">
                  {#if inviteStatus(inv) === 'Active'}
                    <button
                      type="button"
                      class="btn btn-danger btn-small"
                      disabled={revokingInviteId === inv.id}
                      on:click={() => handleRevokeInvite(inv.id)}
                      title="Revoke link"
                    >
                      {revokingInviteId === inv.id ? 'Revoking…' : 'Revoke'}
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <div class="admin-card">
    <div class="admin-card-header">
      <h2 class="admin-card-title">Users</h2>
      <button type="button" class="btn btn-primary btn-small" on:click={() => (showAddUserModal = true)}>
        Add user
      </button>
    </div>
    {#if loading}
      <p class="admin-muted">Loading…</p>
    {:else if error}
      <p class="admin-error">{error}</p>
    {:else if users.length === 0}
      <p class="admin-muted">No users yet.</p>
    {:else}
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th class="sortable" class:sorted={userSortBy === 'email'}>
                <button type="button" class="th-sort" on:click={() => setUserSort('email')}>
                  <span class="th-sort-label">Email</span>
                  {#if userSortBy === 'email'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={userSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={userSortBy === 'role'}>
                <button type="button" class="th-sort" on:click={() => setUserSort('role')}>
                  <span class="th-sort-label">Role</span>
                  {#if userSortBy === 'role'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={userSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              {#if isGlobalAdmin}
                <th>Organization</th>
              {/if}
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each sortedUsers as u}
              <tr>
                <td class="name">{u.email}</td>
                <td>
                  <select
                    class="role-select"
                    value={u.role}
                    disabled={updatingUserRoleId === u.id || deletingUserId === u.id || $user?.id === u.id}
                    on:change={(e) => handleUpdateUserRole(u.id, e.currentTarget.value)}
                  >
                    <option value="user">user</option>
                    <option value="admin">admin</option>
                  </select>
                </td>
                {#if isGlobalAdmin}
                  <td>
                    <select
                      class="org-select"
                      value={u.organization_id ?? ''}
                      disabled={updatingUserOrgId === u.id || deletingUserId === u.id}
                      on:change={(e) => handleUpdateUserOrganization(u.id, e.currentTarget.value)}
                    >
                      <option value="">— None (global admin)</option>
                      {#each organizations as org (org.id)}
                        <option value={org.id}>{org.name}</option>
                      {/each}
                    </select>
                  </td>
                {/if}
                <td class="table-actions">
                  <button
                    type="button"
                    class="btn btn-danger btn-small"
                    disabled={deletingUserId === u.id}
                    on:click={() => handleDeleteUser(u.id)}
                    title="Delete user"
                  >
                    {deletingUserId === u.id ? 'Deleting…' : 'Delete'}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <div class="admin-card">
    <div class="admin-card-header">
      <h2 class="admin-card-title">API tokens</h2>
      <button type="button" class="btn btn-primary btn-small" on:click={() => (showApiTokensModal = true)}>
        Add token
      </button>
    </div>
    {#if tokensLoading}
      <p class="admin-muted">Loading…</p>
    {:else if tokensError}
      <p class="admin-error">{tokensError}</p>
    {:else if tokens.length === 0}
      <p class="admin-muted">No tokens yet.</p>
    {:else}
      <div class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th class="sortable" class:sorted={tokenSortBy === 'name'}>
                <button type="button" class="th-sort" on:click={() => setTokenSort('name')}>
                  <span class="th-sort-label">Name</span>
                  {#if tokenSortBy === 'name'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={tokenSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={tokenSortBy === 'created'}>
                <button type="button" class="th-sort" on:click={() => setTokenSort('created')}>
                  <span class="th-sort-label">Created</span>
                  {#if tokenSortBy === 'created'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={tokenSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th class="sortable" class:sorted={tokenSortBy === 'expires'}>
                <button type="button" class="th-sort" on:click={() => setTokenSort('expires')}>
                  <span class="th-sort-label">Expires</span>
                  {#if tokenSortBy === 'expires'}
                    <span class="sort-icon" aria-hidden="true"><Icon icon={tokenSortDir === 'asc' ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="0.875em" height="0.875em" /></span>
                  {/if}
                </button>
              </th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each sortedTokens as t (t.id)}
              <tr class:expired={isExpired(t.expires_at)}>
                <td class="name">{t.name}</td>
                <td>{formatTokenDate(t.created_at)}</td>
                <td class="expires">{formatExpiry(t.expires_at)}</td>
                <td class="table-actions">
                  <button
                    type="button"
                    class="btn btn-danger btn-small"
                    disabled={deletingTokenId === t.id}
                    on:click={() => handleDeleteToken(t.id)}
                    title="Delete token"
                  >
                    {deletingTokenId === t.id ? 'Deleting…' : 'Delete'}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<style>
  .admin-page {
    padding: 0;
  }
  .admin-card {
    margin-bottom: 1.5rem;
    padding: 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .admin-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }
  .admin-card-title {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 600;
  }
  .table-actions {
    text-align: right;
    white-space: nowrap;
  }
  tr.expired .name,
  tr.expired .expires {
    color: var(--text-muted);
  }
  .admin-error {
    padding: 0.5rem 0;
    font-size: 0.9rem;
    color: var(--danger);
  }
  .admin-muted {
    margin: 0;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .role-select,
  .org-select {
    min-width: 6.5rem;
    padding: 0.35rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.85rem;
  }
  .org-select {
    min-width: 10rem;
  }
  .invite-status-badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    font-size: 0.8rem;
    font-weight: 500;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 4px;
  }
  .invite-status-badge.used {
    background: rgba(34, 197, 94, 0.12);
    border-color: rgba(34, 197, 94, 0.4);
    color: #16a34a;
  }
  .invite-status-badge.expired {
    background: var(--surface-elevated);
    color: var(--text-muted);
  }
  tr.invite-expired td {
    color: var(--text-muted);
  }
  .create-org-form {
    margin-top: 0.5rem;
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    gap: 0.75rem;
  }
  .create-org-input,
  .org-name-input {
    min-width: 12rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.9rem;
  }
  .create-org-input::placeholder,
  .org-name-input::placeholder {
    color: var(--text-muted);
  }
  .org-name-input {
    min-width: 10rem;
    width: 100%;
  }
  .create-org-actions {
    display: flex;
    gap: 0.5rem;
  }
  .btn-secondary {
    background: var(--surface-elevated);
    color: var(--text);
    border: 1px solid var(--border);
  }
  .btn-secondary:hover:not(:disabled) {
    background: var(--table-row-hover);
  }
  .table-wrap {
    overflow-x: auto;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }
  .table th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    background: var(--table-header-bg);
    border-bottom: 1px solid var(--border);
  }
  .table th.sortable {
    padding: 0;
  }
  .table th .th-sort {
    display: inline-flex;
    flex-direction: row;
    flex-wrap: nowrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.35rem;
    width: 100%;
    min-width: 0;
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
  .table th .th-sort-label {
    flex-shrink: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .table th .sort-icon {
    flex-shrink: 0;
    flex-grow: 0;
    font-size: 0.65rem;
  }
  .table td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--table-row-border);
    color: var(--text);
  }
  .table tbody tr:last-child td {
    border-bottom: none;
  }
  .table tbody tr:hover td {
    background: var(--table-row-hover);
  }
  .table td.name {
    font-weight: 500;
  }
</style>
