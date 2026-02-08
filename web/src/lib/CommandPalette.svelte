<script>
  import { createEventDispatcher } from 'svelte'
  import { listEnvironments, listBlocks, listAllocations } from './api.js'

  export let open = false
  export let currentRoute = 'dashboard'
  export let currentUser = null

  const dispatch = createEventDispatcher()

  const STATIC_COMMANDS = [
    { type: 'nav', id: 'dashboard', label: 'Go to Dashboard', keywords: 'dashboard home' },
    { type: 'nav', id: 'environments', label: 'Go to Environments', keywords: 'environments envs' },
    { type: 'nav', id: 'networks', label: 'Go to Networks', keywords: 'networks blocks allocations' },
    { type: 'nav', id: 'docs', label: 'Go to Docs', keywords: 'docs documentation guide' },
    { type: 'create', id: 'create-env', label: 'Create environment', keywords: 'new environment' },
    { type: 'create', id: 'create-block', label: 'Create network block', keywords: 'new block' },
    { type: 'create', id: 'create-alloc', label: 'Create allocation', keywords: 'new allocation' },
  ]

  $: baseCommands = currentUser?.role === 'admin'
    ? [...STATIC_COMMANDS, { type: 'nav', id: 'admin', label: 'Go to Admin', keywords: 'admin users' }]
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
      const [envsRes, blocksRes, allocsRes] = await Promise.all([
        listEnvironments({ name: q, limit: 5, offset: 0 }),
        listBlocks({ name: q, limit: 5, offset: 0 }),
        listAllocations({ name: q, limit: 5, offset: 0 }),
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
      dispatch('navigate', { path: 'networks', block: item.payload.block_name })
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
  <div class="backdrop" role="dialog" aria-modal="true" aria-label="Command palette" on:click={handleBackdropClick}>
    <div class="palette" on:click|stopPropagation>
      <div class="palette-header">
        <span class="palette-icon" aria-hidden="true">⌘</span>
        <input
          bind:this={inputEl}
          type="text"
          class="palette-input"
          placeholder="Search environments, blocks, allocations or run a command…"
          bind:value={query}
          on:input={onQueryInput}
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
        <span>Search: environments, blocks, allocations</span>
        <span><kbd>↑</kbd><kbd>↓</kbd> <kbd>↵</kbd> <kbd>Esc</kbd></span>
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
    gap: 1rem;
    padding: 0.5rem 1rem;
    border-top: 1px solid var(--border);
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .palette-footer kbd {
    padding: 0.1rem 0.25rem;
    font-size: 0.7rem;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 3px;
  }
</style>
