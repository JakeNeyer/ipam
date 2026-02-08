<script>
  import { onMount } from 'svelte'
  import '../lib/theme.js'
  import { listUsers, createUser } from '../lib/api.js'

  let users = []
  let loading = true
  let error = ''
  let showCreate = false
  let createEmail = ''
  let createPassword = ''
  let createRole = 'user'
  let createSubmitting = false
  let createError = ''

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

  async function handleCreate(e) {
    e.preventDefault()
    createError = ''
    if (!createEmail.trim() || !createPassword) {
      createError = 'Email and password are required.'
      return
    }
    createSubmitting = true
    try {
      await createUser(createEmail.trim(), createPassword, createRole)
      createEmail = ''
      createPassword = ''
      createRole = 'user'
      showCreate = false
      await load()
    } catch (e) {
      createError = e.message || 'Failed to create user'
    } finally {
      createSubmitting = false
    }
  }

  onMount(() => {
    load()
  })
</script>

<div class="admin-page">
  <header class="admin-header">
    <h1 class="admin-title">Admin</h1>
    <button type="button" class="btn btn-primary" on:click={() => (showCreate = !showCreate)}>
      {showCreate ? 'Cancel' : 'Add user'}
    </button>
  </header>

  {#if showCreate}
    <div class="admin-card">
      <h2 class="admin-card-title">New user</h2>
      <form class="admin-form" on:submit={handleCreate}>
        {#if createError}
          <div class="admin-error" role="alert">{createError}</div>
        {/if}
        <label class="admin-label">
          <span>Email</span>
          <input type="email" bind:value={createEmail} placeholder="user@example.com" disabled={createSubmitting} />
        </label>
        <label class="admin-label">
          <span>Password</span>
          <input type="password" bind:value={createPassword} placeholder="Password" disabled={createSubmitting} />
        </label>
        <label class="admin-label">
          <span>Role</span>
          <select bind:value={createRole} disabled={createSubmitting}>
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </label>
        <div class="admin-form-actions">
          <button type="submit" class="btn btn-primary" disabled={createSubmitting}>
            {createSubmitting ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </div>
  {/if}

  <div class="admin-card">
    <h2 class="admin-card-title">Users</h2>
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
              <th>Email</th>
              <th>Role</th>
            </tr>
          </thead>
          <tbody>
            {#each users as u}
              <tr>
                <td class="name">{u.email}</td>
                <td><span class="role-badge" class:admin={u.role === 'admin'}>{u.role}</span></td>
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
  .admin-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1.5rem;
  }
  .admin-title {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
  }
  .admin-card {
    margin-bottom: 1.5rem;
    padding: 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .admin-card-title {
    margin: 0 0 1rem 0;
    font-size: 1.1rem;
    font-weight: 600;
  }
  .admin-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 20rem;
  }
  .admin-label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }
  .admin-label input,
  .admin-label select {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.95rem;
  }
  .admin-form-actions {
    margin-top: 0.25rem;
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
  .role-badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    font-size: 0.8rem;
    font-weight: 500;
    text-transform: capitalize;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 4px;
  }
  .role-badge.admin {
    background: var(--accent-dim);
    border-color: var(--accent);
    color: var(--accent);
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
  .table td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
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
  :global(.btn) {
    font-family: var(--font-sans);
  }
</style>
