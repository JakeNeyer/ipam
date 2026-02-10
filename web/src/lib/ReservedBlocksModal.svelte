<script>
  import { createEventDispatcher } from 'svelte'
  import { createReservedBlock } from './api.js'

  export let open = false

  const dispatch = createEventDispatcher()

  let error = ''
  let createName = ''
  let createCidr = ''
  let createReason = ''
  let creating = false

  async function handleCreate() {
    const cidr = (createCidr || '').trim()
    if (!cidr) return
    creating = true
    error = ''
    try {
      await createReservedBlock({
        name: (createName || '').trim() || undefined,
        cidr,
        reason: (createReason || '').trim() || undefined,
      })
      createName = ''
      createCidr = ''
      createReason = ''
      dispatch('created')
      dispatch('close')
    } catch (e) {
      error = e?.message ?? 'Failed to add reserved block'
    } finally {
      creating = false
    }
  }

  function close() {
    createName = ''
    createCidr = ''
    createReason = ''
    error = ''
    dispatch('close')
  }
</script>

<svelte:window on:keydown={(e) => open && e.key === 'Escape' && close()} />

{#if open}
  <div class="modal-backdrop" role="presentation" on:click={close}>
    <div class="modal" role="dialog" aria-labelledby="reserved-blocks-title" aria-modal="true" on:click={(e) => e.stopPropagation()} on:keydown={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2 id="reserved-blocks-title">Add reserved block</h2>
        <button type="button" class="modal-close" aria-label="Close" on:click={close}>×</button>
      </div>
      <p class="modal-desc">Reserved blocks are CIDR ranges that cannot be used as network blocks or allocations. Use them to preserve ranges for future use or other systems.</p>

      {#if error}
        <div class="modal-error" role="alert">{error}</div>
      {/if}

      <div class="create-section">
        <label for="reserved-name">Name (optional)</label>
        <div class="create-row">
          <input id="reserved-name" type="text" bind:value={createName} placeholder="e.g. DMZ" disabled={creating} />
        </div>
        <label for="reserved-cidr">CIDR <span class="required">*</span></label>
        <div class="create-row">
          <input id="reserved-cidr" type="text" bind:value={createCidr} placeholder="e.g. 10.0.0.0/24" disabled={creating} />
        </div>
        <label for="reserved-reason">Reason (optional)</label>
        <div class="create-row">
          <input id="reserved-reason" type="text" bind:value={createReason} placeholder="e.g. Reserved for DMZ" disabled={creating} />
        </div>
        <div class="create-row create-actions">
          <button type="button" class="btn btn-primary" on:click={handleCreate} disabled={creating || !(createCidr || '').trim()}>
            {creating ? 'Adding…' : 'Add reserved block'}
          </button>
        </div>
      </div>

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
  .modal-error {
    margin: 0 1rem 0.75rem;
    padding: 0.4rem 0.6rem;
    font-size: 0.8125rem;
    color: var(--danger);
    background: rgba(220, 38, 38, 0.08);
    border-radius: var(--radius);
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
  .create-section .create-row + label {
    margin-top: 0.75rem;
  }
  .required {
    color: var(--danger);
  }
  .create-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .create-row input {
    flex: 1;
    padding: 0.45rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-size: 0.875rem;
  }
  .create-actions {
    margin-top: 1rem;
  }
  .modal-footer {
    padding: 0.65rem 1rem;
    border-top: 1px solid var(--border);
  }
</style>
