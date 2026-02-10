<script>
  import { onMount } from 'svelte'
  import Icon from '@iconify/svelte'
  import '../lib/theme.js'
  import { listUsers, listTokens, deleteToken, listSignupInvites, revokeSignupInvite, updateUserRole, deleteUser } from '../lib/api.js'
  import { user } from '../lib/auth.js'
  import ApiTokensModal from '../lib/ApiTokensModal.svelte'
  import AddUserModal from '../lib/AddUserModal.svelte'
  import SignupInviteModal from '../lib/SignupInviteModal.svelte'

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
  let deletingUserId = null

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

  onMount(() => {
    load()
    loadInvites()
    loadTokens()
  })
</script>

<div class="admin-page">
  <header class="page-header">
    <h1 class="page-title">Admin</h1>
  </header>

  <AddUserModal open={showAddUserModal} on:close={() => (showAddUserModal = false)} on:created={load} />
  <ApiTokensModal open={showApiTokensModal} on:close={() => { showApiTokensModal = false; loadTokens() }} />
  <SignupInviteModal open={showSignupInviteModal} on:close={() => { showSignupInviteModal = false; loadInvites() }} />

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
  .role-select {
    min-width: 6.5rem;
    padding: 0.35rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.85rem;
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
