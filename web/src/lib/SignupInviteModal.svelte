<script>
  import { createEventDispatcher } from 'svelte'
  import { createSignupInvite } from './api.js'

  export let open = false
  /** When true, show organization and role dropdowns (global admin). */
  export let isGlobalAdmin = false
  /** List of { id, name } for organization dropdown. Used when isGlobalAdmin. */
  export let organizations = []

  const dispatch = createEventDispatcher()

  const expiresInOptions = [
    { value: 24, label: '24 hours' },
    { value: 48, label: '48 hours' },
    { value: 72, label: '3 days' },
    { value: 168, label: '7 days' },
    { value: 720, label: '30 days' }
  ]

  let error = ''
  let createExpiresIn = 24
  let createOrganizationId = ''
  let createRole = 'user'
  let creating = false
  let inviteUrl = ''
  let copied = false

  async function handleCreate() {
    creating = true
    error = ''
    inviteUrl = ''
    try {
      const res = await createSignupInvite(createExpiresIn, createOrganizationId || null, createRole)
      inviteUrl = res?.invite_url ?? ''
      if (!inviteUrl && res?.token) {
        inviteUrl = window.location.origin + window.location.pathname.replace(/\/$/, '') + '#signup?token=' + encodeURIComponent(res.token)
      }
    } catch (e) {
      error = e?.message ?? 'Failed to create signup link'
    } finally {
      creating = false
    }
  }

  function copyLink() {
    if (!inviteUrl) return
    navigator.clipboard.writeText(inviteUrl).then(() => {
      copied = true
      setTimeout(() => (copied = false), 2000)
    })
  }

  function close() {
    inviteUrl = ''
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
    <div class="modal" role="dialog" aria-labelledby="signup-invite-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="signup-invite-title">Create signup link</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={close}>×</button>
      </div>
      <p class="modal-desc">Generate a time-bound link for new users to create an account. Share the link; it can only be used once and will expire.</p>

      {#if error}
        <div class="modal-error" role="alert">{error}</div>
      {/if}

      {#if inviteUrl}
        <div class="new-token-box">
          <p class="new-token-label">Signup link (share this):</p>
          <div class="new-token-row">
            <code class="new-token-value">{inviteUrl}</code>
            <button type="button" class="btn btn-primary" on:click={copyLink}>
              {copied ? 'Copied' : 'Copy link'}
            </button>
          </div>
        </div>
      {:else}
        <div class="create-section">
          <label for="invite-expires">Link expires in</label>
          <div class="create-row">
            <select id="invite-expires" bind:value={createExpiresIn} disabled={creating}>
              {#each expiresInOptions as opt}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
            <button type="button" class="btn btn-primary" on:click={handleCreate} disabled={creating}>
              {creating ? 'Creating…' : 'Create link'}
            </button>
          </div>
          {#if isGlobalAdmin && organizations.length > 0}
            <label for="invite-org">Organization (new user will join)</label>
            <div class="create-row">
              <select id="invite-org" bind:value={createOrganizationId} disabled={creating}>
                <option value="">— Select organization</option>
                {#each organizations as org (org.id)}
                  <option value={org.id}>{org.name}</option>
                {/each}
              </select>
            </div>
            <label for="invite-role">Role</label>
            <div class="create-row">
              <select id="invite-role" bind:value={createRole} disabled={creating}>
                <option value="user">user</option>
                <option value="admin">admin</option>
              </select>
            </div>
          {/if}
        </div>
      {/if}

      <div class="modal-footer">
        <button type="button" class="btn" on:click={close}>{inviteUrl ? 'Done' : 'Close'}</button>
      </div>
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
  .modal-desc {
    margin: 0 1rem 0.75rem;
    font-size: 0.8125rem;
    color: var(--text-muted);
  }
  .modal-error {
    margin: 0 1rem 0.75rem;
    padding: 0.4rem 0.6rem;
    font-size: 0.8125rem;
    color: var(--danger);
    background: rgba(220, 38, 38, 0.08);
    border-radius: var(--radius);
  }
  .new-token-box {
    margin: 0 1rem 0.75rem;
    padding: 0.75rem 1rem;
    background: var(--table-header-bg);
    border-radius: var(--radius);
    border: 1px solid var(--border);
  }
  .new-token-label {
    margin: 0 0 0.4rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text);
  }
  .new-token-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .new-token-value {
    flex: 1;
    font-size: 0.75rem;
    word-break: break-all;
    padding: 0.4rem 0.5rem;
    background: var(--bg);
    border-radius: 3px;
    color: var(--text);
    border: 1px solid var(--border);
  }
  .create-section {
    margin: 0 1rem 0.75rem;
  }
  .create-section .create-row + label {
    margin-top: 0.5rem;
  }
  .create-section label {
    display: block;
    margin-bottom: 0.2rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .create-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .create-row select {
    flex: 1;
    padding: 0.45rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.875rem;
  }
  .modal-footer {
    padding: 0.65rem 1rem;
    border-top: 1px solid var(--border);
  }
</style>
