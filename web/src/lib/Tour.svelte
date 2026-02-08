<script>
  import { createEventDispatcher } from 'svelte'
  import { tick } from 'svelte'

  export let steps = []
  export let open = false
  export let currentStep = 0

  const dispatch = createEventDispatcher()

  let spotlightRect = null
  let tooltipPosition = { top: 0, left: 0, placement: 'bottom' }

  $: step = steps[currentStep] || null
  $: total = steps.length
  $: isFirst = currentStep === 0
  $: isLast = currentStep === total - 1

  async function updateSpotlight() {
    await tick()
    if (!step) {
      spotlightRect = null
      return
    }
    const targetId = step.targetId
    if (!targetId) {
      spotlightRect = null
      const center = { top: window.innerHeight / 2, left: window.innerWidth / 2, width: 0, height: 0 }
      tooltipPosition = { ...center, placement: 'center' }
      return
    }
    const el = document.querySelector(`[data-tour="${targetId}"]`)
    if (!el) {
      spotlightRect = null
      tooltipPosition = { top: 100, left: 50, placement: 'center' }
      return
    }
    const rect = el.getBoundingClientRect()
    spotlightRect = { top: rect.top, left: rect.left, width: rect.width, height: rect.height }
    const padding = 16
    const tooltipWidth = 280
    const tooltipHeight = 220
    const spaceBelow = window.innerHeight - rect.bottom
    const spaceAbove = rect.top
    if (spaceBelow >= 120 || spaceBelow >= spaceAbove) {
      tooltipPosition = {
        top: rect.bottom + padding,
        left: rect.left + Math.max(0, (rect.width - tooltipWidth) / 2),
        placement: 'bottom',
      }
    } else {
      tooltipPosition = {
        top: rect.top - padding - tooltipHeight,
        left: rect.left + Math.max(0, (rect.width - tooltipWidth) / 2),
        placement: 'top',
      }
    }
    if (tooltipPosition.left + tooltipWidth > window.innerWidth - 16) tooltipPosition.left = window.innerWidth - tooltipWidth - 16
    if (tooltipPosition.left < 16) tooltipPosition.left = 16
    if (tooltipPosition.top + tooltipHeight > window.innerHeight - 16) tooltipPosition.top = window.innerHeight - tooltipHeight - 16
    if (tooltipPosition.top < 16) tooltipPosition.top = 16
  }

  $: if (open && step) {
    updateSpotlight()
  }

  function next() {
    if (isLast) {
      dispatch('done')
      return
    }
    dispatch('step', { index: currentStep + 1 })
  }

  function back() {
    if (!isFirst) dispatch('step', { index: currentStep - 1 })
  }

  function skip() {
    dispatch('skip')
  }

  function done() {
    dispatch('done')
  }
</script>

{#if open && steps.length > 0}
  <div class="tour-overlay" role="dialog" aria-modal="true" aria-labelledby="tour-title" aria-describedby="tour-body">
    <!-- Spotlight: transparent "hole" with box-shadow dimming the rest -->
    {#if spotlightRect}
      <div
        class="tour-spotlight"
        style="top: {spotlightRect.top}px; left: {spotlightRect.left}px; width: {spotlightRect.width}px; height: {spotlightRect.height}px;"
        aria-hidden="true"
      ></div>
    {:else}
      <div class="tour-spotlight tour-spotlight-center" aria-hidden="true"></div>
    {/if}

    <div
      class="tour-tooltip"
      class:tooltip-center={tooltipPosition.placement === 'center'}
      style="top: {tooltipPosition.top}px; left: {tooltipPosition.left}px;"
    >
      <h2 id="tour-title" class="tour-title">{step?.title}</h2>
      <p id="tour-body" class="tour-body">{step?.body}</p>
      <div class="tour-actions">
        <button type="button" class="tour-skip" on:click={skip}>Skip</button>
        <div class="tour-nav">
          {#if !isFirst}
            <button type="button" class="tour-back" on:click={back}>Back</button>
          {/if}
          {#if isLast}
            <button type="button" class="tour-next tour-done" on:click={done}>Done</button>
          {:else}
            <button type="button" class="tour-next" on:click={next}>Next</button>
          {/if}
        </div>
      </div>
      <div class="tour-progress" aria-hidden="true">
        {currentStep + 1} / {total}
      </div>
    </div>
  </div>
{/if}

<style>
  .tour-overlay {
    position: fixed;
    inset: 0;
    z-index: 10000;
    pointer-events: none;
  }
  .tour-overlay .tour-tooltip {
    pointer-events: auto;
  }
  .tour-spotlight {
    position: fixed;
    border-radius: var(--radius, 8px);
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.55);
    pointer-events: none;
  }
  .tour-spotlight-center {
    top: 50% !important;
    left: 50% !important;
    width: 0 !important;
    height: 0 !important;
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.55);
  }
  .tour-tooltip {
    position: fixed;
    width: 280px;
    padding: 1rem 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    z-index: 10001;
  }
  .tour-tooltip.tooltip-center {
    top: 50% !important;
    left: 50% !important;
    transform: translate(-50%, -50%);
  }
  .tour-title {
    margin: 0 0 0.5rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text);
  }
  .tour-body {
    margin: 0 0 1rem;
    font-size: 0.9rem;
    line-height: 1.45;
    color: var(--text-muted);
  }
  .tour-actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }
  .tour-skip {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.85rem;
    cursor: pointer;
    padding: 0.25rem 0;
  }
  .tour-skip:hover {
    color: var(--text);
  }
  .tour-nav {
    display: flex;
    gap: 0.5rem;
  }
  .tour-back,
  .tour-next {
    padding: 0.4rem 0.75rem;
    border-radius: var(--radius);
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid var(--border);
    background: var(--surface-elevated);
    color: var(--text);
  }
  .tour-back:hover,
  .tour-next:hover {
    background: var(--border);
  }
  .tour-next.tour-done,
  .tour-next:not(.tour-done):hover {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--btn-primary-text, #fff);
  }
  .tour-progress {
    margin-top: 0.75rem;
    font-size: 0.75rem;
    color: var(--text-muted);
  }
</style>
