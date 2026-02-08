<script>
  import { createEventDispatcher } from 'svelte'

  export let message = ''
  const dispatch = createEventDispatcher()
</script>

{#if message}
  <div
    class="error-modal-backdrop"
    role="alertdialog"
    aria-modal="true"
    aria-labelledby="error-modal-title"
    aria-describedby="error-modal-desc"
    on:click={() => dispatch('close')}
  >
    <div class="error-modal" on:click|stopPropagation>
      <h3 id="error-modal-title">Error</h3>
      <p id="error-modal-desc" class="error-modal-message">{message}</p>
      <div class="error-modal-actions">
        <button type="button" on:click={() => dispatch('close')}>OK</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .error-modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1.5rem;
  }
  .error-modal {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    padding: 1.5rem;
    max-width: 420px;
    width: 100%;
  }
  .error-modal h3 {
    margin: 0 0 1rem 0;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--danger);
  }
  .error-modal-message {
    margin: 0 0 1.25rem 0;
    font-size: 0.9rem;
    line-height: 1.5;
    color: var(--text);
  }
  .error-modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  .error-modal-actions button {
    padding: 0.5rem 1rem;
    border-radius: var(--radius);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--btn-primary-text);
    transition: background 0.15s, border-color 0.15s;
  }
  .error-modal-actions button:hover {
    background: var(--btn-primary-hover-bg);
    border-color: var(--btn-primary-hover-border);
  }
</style>
