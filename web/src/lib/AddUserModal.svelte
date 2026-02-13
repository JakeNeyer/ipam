<script>
  import { createEventDispatcher } from 'svelte'
  import { createUser, getAuthConfig } from './api.js'

  export let open = false
  /** When true, show organization dropdown (global admin). */
  export let isGlobalAdmin = false
  /** List of { id, name } for organization dropdown. Used when isGlobalAdmin. */
  export let organizations = []
  /** When true, password is optional (user will sign in via OAuth). Fallback from parent; modal also fetches when opened. */
  export let oauthEnabled = false

  const dispatch = createEventDispatcher()

  /** Resolved when modal opens so we always show correct password field for current server config. */
  let oauthEnabledResolved = false
  let fetchedForOpen = false
  $: if (open) {
    if (!fetchedForOpen) {
      fetchedForOpen = true
      oauthEnabledResolved = oauthEnabled
      getAuthConfig()
        .then((cfg) => {
          oauthEnabledResolved =
            (Array.isArray(cfg?.oauthProviders) && cfg.oauthProviders.length > 0) || cfg?.githubOAuthEnabled === true
        })
        .catch(() => {})
    }
  } else {
    fetchedForOpen = false
  }

  let error = ''
  let email = ''
  let password = ''
  let role = 'user'
  /** Organization ID when isGlobalAdmin; '' for no org (global admin). */
  let organizationId = ''
  let submitting = false

  async function handleSubmit(e) {
    e.preventDefault()
    error = ''
    if (!email.trim()) {
      error = 'Email is required.'
      return
    }
    if (!oauthEnabledResolved && !password) {
      error = 'Password is required.'
      return
    }
    submitting = true
    try {
      await createUser(email.trim(), password, role, organizationId || null)
      email = ''
      password = ''
      role = 'user'
      organizationId = ''
      dispatch('close')
      dispatch('created')
    } catch (e) {
      error = e?.message ?? 'Failed to create user'
    } finally {
      submitting = false
    }
  }

  function close() {
    email = ''
    password = ''
    role = 'user'
    organizationId = ''
    error = ''
    dispatch('close')
  }
</script>

<svelte:window on:keydown={(e) => open && e.key === 'Escape' && close()} />

{#if open}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    aria-label="Close modal"
    on:click={close}
    on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); close(); } }}
  >
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div class="modal" role="dialog" aria-labelledby="add-user-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="add-user-title">Add user</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={close}>×</button>
      </div>
      <form class="modal-form" on:submit={handleSubmit}>
        {#if error}
          <div class="modal-error" role="alert">{error}</div>
        {/if}
        <label class="modal-label">
          <span>Email</span>
          <input type="email" bind:value={email} placeholder="user@example.com" disabled={submitting} />
        </label>
        {#if !oauthEnabledResolved}
        <label class="modal-label">
          <span>Password</span>
          <input type="password" bind:value={password} placeholder="Password" disabled={submitting} />
        </label>
        {/if}
        <label class="modal-label">
          <span>Role</span>
          <select bind:value={role} disabled={submitting}>
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </label>
        {#if isGlobalAdmin && organizations.length > 0}
          <label class="modal-label">
            <span>Organization</span>
            <select bind:value={organizationId} disabled={submitting}>
              <option value="">— None (global admin)</option>
              {#each organizations as org (org.id)}
                <option value={org.id}>{org.name}</option>
              {/each}
            </select>
          </label>
        {/if}
        <div class="modal-footer">
          <button type="button" class="btn" on:click={close}>Cancel</button>
          <button type="submit" class="btn btn-primary" disabled={submitting}>
            {submitting ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.35);
    padding: 1rem;
  }
  .modal {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    max-width: 420px;
    width: 100%;
    max-height: 90vh;
    overflow: auto;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .modal-header h2 {
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 600;
    color: var(--text);
  }
  .modal-close {
    background: none;
    border: none;
    font-size: 1.25rem;
    line-height: 1;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.2rem;
  }
  .modal-close:hover {
    color: var(--text);
  }
  .modal-form {
    padding: 1rem;
  }
  .modal-error {
    margin: 0 0 0.75rem;
    padding: 0.4rem 0.6rem;
    font-size: 0.8125rem;
    color: var(--danger);
    background: rgba(220, 38, 38, 0.08);
    border-radius: var(--radius);
  }
  .modal-label {
    display: block;
    margin-bottom: 0.75rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .modal-label:last-of-type {
    margin-bottom: 0;
  }
  .modal-label span {
    display: block;
    margin-bottom: 0.25rem;
  }
  .modal-label input,
  .modal-label select {
    width: 100%;
    padding: 0.45rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.875rem;
    font-family: var(--font-sans);
  }
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border);
  }
  :global(.modal .btn) {
    font-family: var(--font-sans);
  }
</style>
