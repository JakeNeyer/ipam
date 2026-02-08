<script>
  export let message = ''
  export let variant = 'page' // 'page' | 'inline'
</script>

{#if message}
  <div
    class="error-alert"
    class:error-alert-inline={variant === 'inline'}
    role="alert"
    aria-live="polite"
  >
    <span class="error-alert-icon" aria-hidden="true">!</span>
    <span class="error-alert-text">{message}</span>
  </div>
{/if}

<style>
  .error-alert {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    margin-bottom: 0.75rem;
    background: rgba(248, 81, 73, 0.08);
    border: 1px solid rgba(248, 81, 73, 0.35);
    border-radius: var(--radius);
    color: #ff7b72;
    font-size: 0.9rem;
    line-height: 1.45;
  }
  .error-alert-inline {
    padding: 0.6rem 0.75rem;
    font-size: 0.85rem;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .error-alert:last-child {
    margin-bottom: 0;
  }
  .error-alert-icon {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 50%;
    background: rgba(248, 81, 73, 0.25);
    font-weight: 700;
    font-size: 0.75rem;
  }
  .error-alert-inline .error-alert-icon {
    width: 1rem;
    height: 1rem;
    font-size: 0.65rem;
  }
  .error-alert-text {
    flex: 1;
    min-width: 0;
  }
</style>
