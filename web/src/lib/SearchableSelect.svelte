<script>
  import { onMount } from 'svelte'
  import { createEventDispatcher } from 'svelte'
  import Icon from '@iconify/svelte'

  const dispatch = createEventDispatcher()

  /** @type {{ value: string, label: string }[]} */
  export let options = []
  export let value = ''
  export let placeholder = 'Select…'
  export let disabled = false

  let open = false
  let filter = ''
  let highlightedIndex = 0
  let triggerEl = null
  let listEl = null
  let inputEl = null
  let dropdownStyle = { top: 0, left: 0, width: 200 }
  const listboxId = 'searchable-select-listbox-' + Math.random().toString(36).slice(2, 11)

  $: selectedOption = options.find((o) => o.value === value)
  $: selectedLabel = selectedOption ? selectedOption.label : ''
  $: filteredOptions = filter.trim()
    ? options.filter((o) => o.label.toLowerCase().includes(filter.toLowerCase().trim()))
    : options
  $: highlightedIndexClamped = Math.min(Math.max(0, highlightedIndex), Math.max(0, filteredOptions.length - 1))

  function openDropdown() {
    if (disabled) return
    open = true
    filter = ''
    highlightedIndex = 0
  }

  function closeDropdown() {
    open = false
    filter = ''
  }

  function select(opt) {
    value = opt.value
    closeDropdown()
    dispatch('change', opt.value)
  }

  function onInputKeydown(e) {
    if (e.key === 'Escape') {
      closeDropdown()
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      highlightedIndex = Math.min(highlightedIndex + 1, filteredOptions.length - 1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      highlightedIndex = Math.max(highlightedIndex - 1, 0)
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      const opt = filteredOptions[highlightedIndexClamped]
      if (opt) select(opt)
      return
    }
  }

  function onTriggerKeydown(e) {
    if (disabled) return
    if (open) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault()
        inputEl?.focus()
        if (e.key === 'ArrowDown') highlightedIndex = Math.min(highlightedIndex + 1, filteredOptions.length - 1)
        else highlightedIndex = Math.max(highlightedIndex - 1, 0)
        return
      }
      if (e.key === 'Enter' && filteredOptions[highlightedIndexClamped]) {
        e.preventDefault()
        select(filteredOptions[highlightedIndexClamped])
        return
      }
      return
    }
    if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
      e.preventDefault()
      openDropdown()
    }
  }

  $: if (open && filteredOptions.length > 0) {
    highlightedIndex = Math.min(highlightedIndex, filteredOptions.length - 1)
  }

  $: if (open && triggerEl) {
    const r = triggerEl.getBoundingClientRect()
    dropdownStyle = {
      top: r.bottom + 2,
      left: r.left,
      width: Math.max(r.width, 200),
    }
  }

  onMount(() => {
    function handleClickOutside(e) {
      if (open && !e.target.closest('.searchable-select')) {
        closeDropdown()
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  })
</script>

<div class="searchable-select">
  <button
    bind:this={triggerEl}
    type="button"
    class="searchable-select-trigger"
    class:open
    disabled={disabled}
    on:click={() => (open ? closeDropdown() : openDropdown())}
    on:keydown={onTriggerKeydown}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-labelledby="searchable-select-label"
  >
    <span class="searchable-select-value">{open ? '' : (selectedLabel || placeholder)}</span>
    <span class="searchable-select-chevron" aria-hidden="true"><Icon icon={open ? 'lucide:chevron-up' : 'lucide:chevron-down'} width="1em" height="1em" /></span>
  </button>
  {#if open}
    <div
      class="searchable-select-dropdown searchable-select-dropdown-fixed"
      role="presentation"
      style="position: fixed; top: {dropdownStyle.top}px; left: {dropdownStyle.left}px; width: {dropdownStyle.width}px; z-index: 1000;"
    >
      <input
        bind:this={inputEl}
        type="text"
        class="searchable-select-input"
        bind:value={filter}
        placeholder="Filter (optional)"
        on:keydown={onInputKeydown}
        on:click|stopPropagation
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={open}
        aria-controls={listboxId}
      />
      <ul id={listboxId} class="searchable-select-list" bind:this={listEl} role="listbox">
        {#each filteredOptions as opt, i}
          <li
            class="searchable-select-option"
            class:highlighted={i === highlightedIndexClamped}
            role="option"
            aria-selected={opt.value === value}
            on:mousedown|preventDefault|stopPropagation={() => select(opt)}
            on:mouseenter={() => (highlightedIndex = i)}
          >
            {opt.label}
          </li>
        {/each}
        {#if filteredOptions.length === 0}
          <li class="searchable-select-empty">No matches</li>
        {/if}
      </ul>
    </div>
  {/if}
</div>

<style>
  .searchable-select {
    position: relative;
    width: 100%;
  }
  .searchable-select-trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    text-align: left;
    transition: border-color 0.15s;
  }
  :global(.dark) .searchable-select-trigger {
    background: var(--bg);
  }
  .searchable-select-trigger:hover:not(:disabled) {
    border-color: var(--text-muted);
  }
  .searchable-select-trigger.open {
    border-color: var(--accent);
  }
  .searchable-select-trigger:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .searchable-select-value {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .searchable-select-value:empty {
    color: var(--text-muted);
  }
  .searchable-select-chevron {
    flex-shrink: 0;
    margin-left: 0.5rem;
    font-size: 0.7rem;
    color: var(--text-muted);
  }
  .searchable-select-dropdown {
    padding: 0.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-md);
    max-height: 280px;
    display: flex;
    flex-direction: column;
  }
  .searchable-select-dropdown-fixed {
    /* position/top/left/width/z-index set inline for fixed positioning above table */
  }
  .searchable-select-input {
    width: 100%;
    padding: 0.4rem 0.6rem;
    margin-bottom: 0.25rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 2px);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
  }
  .searchable-select-input:focus {
    outline: none;
    border-color: var(--accent);
  }
  .searchable-select-list {
    list-style: none;
    margin: 0;
    padding: 0;
    overflow-y: auto;
    max-height: 220px;
  }
  .searchable-select-option {
    padding: 0.4rem 0.6rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
    transition: background 0.1s;
  }
  .searchable-select-option:hover,
  .searchable-select-option.highlighted {
    background: var(--table-row-hover);
  }
  .searchable-select-empty {
    padding: 0.5rem 0.6rem;
    font-size: 0.85rem;
    color: var(--text-muted);
  }
</style>
