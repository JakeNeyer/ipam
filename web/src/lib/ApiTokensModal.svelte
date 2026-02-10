<script>
  import { createEventDispatcher } from 'svelte'
  import { createToken } from './api.js'

  export let open = false

  const dispatch = createEventDispatcher()

  const expiresInOptions = [
    { value: '', label: 'Never' },
    { value: '7', label: '7 days' },
    { value: '30', label: '30 days' },
    { value: '90', label: '90 days' },
    { value: '365', label: '1 year' }
  ]

  let error = ''
  let createName = ''
  let createExpiresIn = ''
  let creating = false
  let newToken = null
  let copied = false

  function getExpiresAt() {
    const days = parseInt(createExpiresIn, 10)
    if (!createExpiresIn || isNaN(days) || days <= 0) return null
    const d = new Date()
    d.setDate(d.getDate() + days)
    d.setHours(0, 0, 0, 0)
    return d.toISOString()
  }

  async function handleCreate() {
    const name = (createName || '').trim()
    if (!name) return
    creating = true
    error = ''
    newToken = null
    try {
      const expiresAt = getExpiresAt()
      const res = await createToken(name, expiresAt ? { expires_at: expiresAt } : {})
      newToken = res?.token ?? null
      createName = ''
      createExpiresIn = ''
    } catch (e) {
      error = e?.message ?? 'Failed to create token'
    } finally {
      creating = false
    }
  }

  function copyToken() {
    if (!newToken?.token) return
    navigator.clipboard.writeText(newToken.token).then(() => {
      copied = true
      setTimeout(() => (copied = false), 2000)
    })
  }

  function dismissNewToken() {
    newToken = null
    copied = false
  }

  function close() {
    newToken = null
    createName = ''
    createExpiresIn = ''
    error = ''
    dispatch('close')
  }
</script>

<svelte:window on:keydown={(e) => open && e.key === 'Escape' && close()} />

{#if open}
  <div class="modal-backdrop" role="presentation" on:click={close}>
    <div class="modal" role="dialog" aria-labelledby="api-tokens-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="api-tokens-title">Add token</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={close}>×</button>
      </div>
      <p class="modal-desc">Use in <code>Authorization: Bearer &lt;token&gt;</code>. Copy the token after creating; it won’t be shown again.</p>

      {#if error}
        <div class="modal-error" role="alert">{error}</div>
      {/if}

      {#if newToken}
        <div class="new-token-box">
          <p class="new-token-label">New token (copy it now):</p>
          <div class="new-token-row">
            <code class="new-token-value">{newToken.token}</code>
            <button type="button" class="btn btn-primary" on:click={copyToken}>
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <button type="button" class="btn" on:click={dismissNewToken}>Done</button>
        </div>
      {:else}
        <div class="create-section">
          <label for="token-name">Name</label>
          <div class="create-row">
            <input id="token-name" type="text" bind:value={createName} placeholder="e.g. CI / CLI" disabled={creating} />
            <button type="button" class="btn btn-primary" on:click={handleCreate} disabled={creating || !(createName || '').trim()}>
              {creating ? 'Creating…' : 'Create token'}
            </button>
          </div>
          <label for="token-expires">Expires in</label>
          <div class="create-row">
            <select id="token-expires" bind:value={createExpiresIn} disabled={creating}>
              {#each expiresInOptions as opt}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          </div>
        </div>
      {/if}

      <div class="modal-footer">
        <button type="button" class="btn" on:click={close}>Close</button>
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
  .modal-desc code {
    font-size: 0.8em;
    background: var(--table-header-bg);
    padding: 0.1em 0.3em;
    border-radius: 3px;
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
    margin-bottom: 0.5rem;
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
  .create-row input,
  .create-row select {
    flex: 1;
    padding: 0.45rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.875rem;
  }
  .create-section .create-row + label {
    margin-top: 0.75rem;
  }
  .modal-footer {
    padding: 0.65rem 1rem;
    border-top: 1px solid var(--border);
  }
</style>
