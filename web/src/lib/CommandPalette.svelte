<script>
  import { createEventDispatcher } from 'svelte'
  import Icon from '@iconify/svelte'
  import { listEnvironments, listBlocks, listAllocations } from './api.js'
  import { selectedOrgForGlobalAdmin, isGlobalAdmin } from './auth.js'

  export let open = false
  export let currentRoute = 'dashboard'
  export let currentUser = null

  const dispatch = createEventDispatcher()

  const STATIC_COMMANDS = [
    { type: 'nav', id: 'dashboard', label: 'Go to Dashboard', keywords: 'dashboard home' },
    { type: 'nav', id: 'environments', label: 'Go to Environments', keywords: 'environments envs' },
    { type: 'nav', id: 'networks', label: 'Go to Networks', keywords: 'networks blocks allocations' },
    { type: 'nav', id: 'network-advisor', label: 'Go to Network Advisor', keywords: 'network advisor planning architecture cidr' },
    { type: 'nav', id: 'subnet-calculator', label: 'Go to Subnet calculator', keywords: 'subnet calculator cidr divide' },
    { type: 'nav', id: 'docs', label: 'Go to Docs', keywords: 'docs documentation guide' },
    { type: 'create', id: 'create-env', label: 'Create environment', keywords: 'new environment' },
    { type: 'create', id: 'create-block', label: 'Create network block', keywords: 'new block' },
    { type: 'create', id: 'create-alloc', label: 'Create allocation', keywords: 'new allocation' },
  ]

  $: baseCommands = currentUser?.role === 'admin'
    ? [...STATIC_COMMANDS, { type: 'nav', id: 'reserved-blocks', label: 'Go to Reserved blocks', keywords: 'reserved blocks blacklist' }, { type: 'nav', id: 'admin', label: 'Go to Admin', keywords: 'admin users' }]
    : STATIC_COMMANDS

  function matchCommand(c, q) {
    if (!q) return true
    return c.label.toLowerCase().includes(q) || (c.keywords && c.keywords.includes(q))
  }

  let query = ''
  let searchResults = [] // { type: 'env'|'block'|'alloc', id, label, meta, payload }
  let searchLoading = false
  let selectedIndex = 0
  let inputEl

  $: queryLower = (query || '').trim().toLowerCase()
  $: filteredCommands = queryLower ? baseCommands.filter((c) => matchCommand(c, queryLower)) : baseCommands

  $: allItems = [...searchResults, ...filteredCommands]
  $: if (open && allItems.length > 0 && selectedIndex >= allItems.length) {
    selectedIndex = allItems.length - 1
  }

  async function runSearch() {
    const q = queryLower
    if (q.length < 1) {
      searchResults = []
      return
    }
    searchLoading = true
    searchResults = []
    try {
      const opts = { name: q, limit: 5, offset: 0 }
      if (isGlobalAdmin(currentUser) && $selectedOrgForGlobalAdmin) opts.organization_id = $selectedOrgForGlobalAdmin
      const [envsRes, blocksRes, allocsRes] = await Promise.all([
        listEnvironments(opts),
        listBlocks(opts),
        listAllocations(opts),
      ])
      const results = []
      ;(envsRes.environments || []).forEach((e) => {
        results.push({ type: 'search-env', id: e.id, label: e.name, meta: 'Environment', payload: e })
      })
      ;(blocksRes.blocks || []).forEach((b) => {
        results.push({ type: 'search-block', id: b.id, label: b.name, meta: 'Block', payload: b })
      })
      ;(allocsRes.allocations || []).forEach((a) => {
        results.push({
          type: 'search-alloc',
          id: a.id,
          label: a.name,
          meta: a.block_name ? `Allocation · ${a.block_name}` : 'Allocation',
          payload: a,
        })
      })
      searchResults = results
    } catch {
      searchResults = []
    } finally {
      searchLoading = false
    }
  }

  let listEl
  $: if (open && allItems.length > 0 && selectedIndex >= 0) {
    setTimeout(() => {
      const el = listEl?.querySelector('[aria-selected="true"]')
      el?.scrollIntoView({ block: 'nearest' })
    }, 0)
  }

  let searchDebounce
  function onQueryInput() {
    clearTimeout(searchDebounce)
    searchDebounce = setTimeout(runSearch, 200)
  }

  function handleSelect(item) {
    if (item.type === 'nav') {
      dispatch('navigate', { path: item.id })
    } else if (item.type === 'create') {
      dispatch('create', { action: item.id })
    } else if (item.type === 'search-env') {
      dispatch('navigate', { path: 'environments', environmentId: item.payload.id })
    } else if (item.type === 'search-block') {
      dispatch('navigate', { path: 'networks', block: item.payload.name })
    } else if (item.type === 'search-alloc') {
      dispatch('navigate', { path: 'networks', block: item.payload.block_name, allocation: item.payload.name })
    }
    close()
  }

  function close() {
    query = ''
    searchResults = []
    selectedIndex = 0
    dispatch('close')
  }

  function handleKeydown(e) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex = Math.min(selectedIndex + 1, allItems.length - 1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex = Math.max(selectedIndex - 1, 0)
      return
    }
    if (e.key === 'Enter' && allItems.length > 0) {
      e.preventDefault()
      handleSelect(allItems[selectedIndex])
      return
    }
  }

  function handleBackdropClick(e) {
    if (e.target === e.currentTarget) close()
  }

  $: if (open && inputEl) {
    setTimeout(() => inputEl?.focus(), 0)
  }
  $: if (!open) selectedIndex = 0
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
  <div
    class="backdrop"
    role="dialog"
    aria-modal="true"
    aria-label="Command palette"
    tabindex="-1"
    on:click={handleBackdropClick}
    on:keydown={handleKeydown}
  >
    <div class="palette" role="presentation" on:click|stopPropagation on:keydown|stopPropagation>
      <div class="palette-header">
        <span class="palette-icon" aria-hidden="true"><Icon icon="lucide:command" width="1rem" height="1rem" /></span>
        <input
          bind:this={inputEl}
          type="text"
          class="palette-input"
          placeholder="Search environments, blocks, allocations or run a command…"
          bind:value={query}
          on:input={onQueryInput}
          on:keydown={handleKeydown}
        />
      </div>
      <div class="palette-list" role="listbox" bind:this={listEl}>
        {#if searchLoading}
          <div class="palette-item palette-item-muted">Searching…</div>
        {:else if allItems.length === 0}
          <div class="palette-item palette-item-muted">No results</div>
        {:else}
          {#each allItems as item, i}
            <button
              type="button"
              class="palette-item"
              class:selected={i === selectedIndex}
              role="option"
              aria-selected={i === selectedIndex}
              aria-current={item.type === 'nav' && item.id === currentRoute ? 'true' : undefined}
              on:click={() => handleSelect(item)}
              on:mouseenter={() => (selectedIndex = i)}
            >
              <span class="palette-item-label">{item.label}</span>
              {#if item.meta}
                <span class="palette-item-meta">{item.meta}</span>
              {/if}
            </button>
          {/each}
        {/if}
      </div>
      <div class="palette-footer">
        <span class="palette-footer-hint">Search: environments, blocks, allocations</span>
        <span class="palette-footer-keys">
          <kbd><Icon icon="lucide:chevron-up" width="0.75em" height="0.75em" inline /></kbd>
          <kbd><Icon icon="lucide:chevron-down" width="0.75em" height="0.75em" inline /></kbd>
          <kbd>↵</kbd>
          <kbd>Esc</kbd>
        </span>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 15vh;
    z-index: 1000;
  }
  .palette {
    width: 100%;
    max-width: 28rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    overflow: hidden;
  }
  .palette-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .palette-icon {
    font-size: 1rem;
    color: var(--text-muted);
  }
  .palette-input {
    flex: 1;
    padding: 0.35rem 0;
    border: none;
    background: transparent;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.95rem;
    outline: none;
  }
  .palette-input::placeholder {
    color: var(--text-muted);
  }
  .palette-list {
    max-height: 18rem;
    overflow-y: auto;
    padding: 0.25rem 0;
  }
  .palette-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    width: 100%;
    padding: 0.5rem 1rem;
    border: none;
    background: transparent;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    text-align: left;
    cursor: pointer;
    transition: background 0.1s;
  }
  .palette-item:hover,
  .palette-item.selected {
    background: var(--accent-dim);
  }
  .palette-item-label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .palette-item-meta {
    font-size: 0.8rem;
    color: var(--text-muted);
    flex-shrink: 0;
  }
  .palette-item-muted {
    padding: 0.75rem 1rem;
    color: var(--text-muted);
    font-size: 0.875rem;
    cursor: default;
  }
  .palette-item-muted:hover {
    background: transparent;
  }
  .palette-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.5rem 1rem;
    border-top: 1px solid var(--border);
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .palette-footer-hint {
    flex-shrink: 0;
  }
  .palette-footer-keys {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    flex-shrink: 0;
  }
  .palette-footer-keys kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.5em;
    padding: 0.2rem 0.35rem;
    font-size: 0.7rem;
    font-family: inherit;
    line-height: 1;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 3px;
  }
  .palette-footer-keys kbd :global(svg) {
    display: block;
  }
</style>
