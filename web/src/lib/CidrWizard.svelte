<script>
  import { onMount } from 'svelte'
  import { cidrRange } from './cidr.js'
  import { suggestBlockCidr, suggestEnvironmentBlockCidr } from './api.js'

  /** @type {'block' | 'allocation'} */
  export let mode = 'block'
  /** Environment ID for block mode (used to suggest CIDR that does not overlap existing blocks in that environment) */
  export let environmentId = null
  /** Parent block CIDR when mode is 'allocation' */
  export let parentCidr = ''
  /** Block ID for allocation mode (used to fetch suggested CIDR from API) */
  export let blockId = null
  /** Current CIDR value (bindable) */
  export let value = ''
  export let disabled = false

  let suggestedCidrFromApi = ''
  let suggestedLoading = false
  let suggestedBlockCidrFromApi = ''
  let suggestedBlockLoading = false

  const PREFIX_OPTIONS_BLOCK = [8, 12, 16, 20, 24, 25, 26, 27, 28, 29, 30]
  const MAX_PREFIX = 30
  const MIN_PREFIX_ALLOC = 1

  let octets = [10, 0, 0, 0]
  let selectedPrefix = 24

  $: blockCidrComputed =
    mode === 'block'
      ? `${octets[0]}.${octets[1]}.${octets[2]}.${octets[3]}/${selectedPrefix}`
      : ''

  // Sync external value → octets (e.g. user typed in CIDR field). Depends only on value/mode to avoid cycle.
  $: if (mode === 'block' && value) {
    const c = parseCidr(value)
    if (c) {
      const parts = c.network.split('.')
      if (parts.length === 4) {
        octets = parts.map((p) => clampOctet(parseInt(p, 10)))
        selectedPrefix = Math.min(MAX_PREFIX, Math.max(8, c.prefix))
      }
    }
  }

  $: if (mode === 'allocation' && parentCidr) {
    const c = parseCidr(parentCidr)
    if (c && selectedPrefix <= c.prefix) selectedPrefix = Math.min(MAX_PREFIX, c.prefix + 1)
  }

  function parseCidr(cidr) {
    if (!cidr || typeof cidr !== 'string') return null
    const idx = cidr.indexOf('/')
    if (idx === -1) return null
    const network = cidr.slice(0, idx).trim()
    const prefix = parseInt(cidr.slice(idx + 1), 10)
    if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
    return { network, prefix }
  }

  function clampOctet(n) {
    if (isNaN(n)) return 0
    return Math.max(0, Math.min(255, n))
  }

  /** IPv4: /p has 2^(32-p) addresses */
  function ipCountForPrefix(p) {
    if (p < 0 || p > 32) return 0
    return Math.pow(2, 32 - p)
  }

  $: blockIpCount = mode === 'block' ? ipCountForPrefix(selectedPrefix) : 0
  $: allocationIpCount = mode === 'allocation' ? ipCountForPrefix(selectedPrefix) : 0

  $: parentParsed = mode === 'allocation' && parentCidr ? parseCidr(parentCidr) : null
  $: allocationPrefixMin = parentParsed ? Math.min(MAX_PREFIX, parentParsed.prefix + 1) : MIN_PREFIX_ALLOC
  $: allocationPrefixOptions = Array.from(
    { length: MAX_PREFIX - allocationPrefixMin + 1 },
    (_, i) => allocationPrefixMin + i
  )

  $: allocationCidrComputed =
    mode === 'allocation' && parentParsed
      ? `${parentParsed.network}/${selectedPrefix}`
      : ''

  // Fetch suggested CIDR from API (bin-packed) when block and prefix are set
  $: suggestedTrigger =
    mode === 'allocation' && blockId && parentCidr && selectedPrefix >= allocationPrefixMin
      ? [blockId, selectedPrefix]
      : []
  $: if (suggestedTrigger.length) {
    const [bid, prefix] = suggestedTrigger
    suggestedLoading = true
    suggestBlockCidr(bid, prefix)
      .then((cidr) => {
        if (bid === blockId && prefix === selectedPrefix) suggestedCidrFromApi = cidr
      })
      .catch(() => {
        if (bid === blockId && prefix === selectedPrefix) suggestedCidrFromApi = ''
      })
      .finally(() => {
        if (bid === blockId && prefix === selectedPrefix) suggestedLoading = false
      })
  } else {
    suggestedCidrFromApi = ''
    suggestedLoading = false
  }

  $: allocationSuggestedCidr = suggestedCidrFromApi || allocationCidrComputed

  // Fetch suggested block CIDR for environment (no overlap with existing blocks) when environmentId and prefix set
  $: suggestedBlockTrigger =
    mode === 'block' && environmentId && selectedPrefix >= 9
      ? [environmentId, selectedPrefix]
      : []
  $: if (suggestedBlockTrigger.length) {
    const [eid, prefix] = suggestedBlockTrigger
    suggestedBlockLoading = true
    suggestEnvironmentBlockCidr(eid, prefix)
      .then((cidr) => {
        if (eid === environmentId && prefix === selectedPrefix) suggestedBlockCidrFromApi = cidr
      })
      .catch(() => {
        if (eid === environmentId && prefix === selectedPrefix) suggestedBlockCidrFromApi = ''
      })
      .finally(() => {
        if (eid === environmentId && prefix === selectedPrefix) suggestedBlockLoading = false
      })
  } else {
    suggestedBlockCidrFromApi = ''
    suggestedBlockLoading = false
  }

  function applyBlockCidr() {
    value = blockCidrComputed
  }

  function applySuggestedBlockCidr() {
    if (suggestedBlockCidrFromApi) value = suggestedBlockCidrFromApi
  }

  function applyAllocationCidr() {
    value = allocationSuggestedCidr
  }

  // Update bound value when user changes wizard (no reactive to avoid cycle)
  function syncBlockValue() {
    if (mode === 'block') value = blockCidrComputed
  }

  onMount(() => {
    if (mode === 'block' && !value) value = blockCidrComputed
  })
</script>

<div class="cidr-wizard">
  {#if mode === 'block'}
    <div class="wizard-section">
      <header class="wizard-header">
        <h4 class="wizard-title">CIDR range</h4>
        {#if environmentId}
          <p class="wizard-hint">A suggested CIDR (no overlap with existing blocks in this environment) is shown below.</p>
        {:else}
          <p class="wizard-hint">Choose the base address and prefix length for the block.</p>
        {/if}
      </header>
      <div class="wizard-fields">
        <div class="wizard-field">
          <span class="wizard-field-label">Base address (IPv4)</span>
          <div class="octets">
            {#each [0, 1, 2, 3] as i}
              <input
                type="number"
                min="0"
                max="255"
                bind:value={octets[i]}
                on:input={syncBlockValue}
                disabled={disabled}
                class="octet"
              />
            {/each}
          </div>
        </div>
        <div class="wizard-field">
          <span class="wizard-field-label">Prefix length</span>
          <div class="prefix-with-count">
            <select bind:value={selectedPrefix} on:change={syncBlockValue} disabled={disabled} class="prefix-select">
              {#each PREFIX_OPTIONS_BLOCK as p}
                <option value={p}>/{p}</option>
              {/each}
            </select>
            <span class="ip-count">{blockIpCount.toLocaleString()} IPs</span>
          </div>
        </div>
      </div>
      {#if !environmentId}
        <div class="wizard-result">
          <span class="result-label">Resulting CIDR</span>
          <code class="result-cidr">{blockCidrComputed}</code>
          {#if cidrRange(blockCidrComputed)}
            <span class="result-range">{cidrRange(blockCidrComputed).start} – {cidrRange(blockCidrComputed).end}</span>
          {/if}
          <span class="ip-count">{blockIpCount.toLocaleString()} IPs</span>
          <button type="button" class="wizard-btn" on:click={applyBlockCidr} disabled={disabled}>
            Use manual CIDR
          </button>
        </div>
      {/if}
      {#if environmentId}
        <div class="wizard-result wizard-result-suggested">
          <span class="result-label">Suggested CIDR</span>
          {#if suggestedBlockLoading}
            <span class="suggested-loading">…</span>
          {:else if suggestedBlockCidrFromApi}
            {@const suggestedRange = cidrRange(suggestedBlockCidrFromApi)}
            <code class="result-cidr">{suggestedBlockCidrFromApi}</code>
            {#if suggestedRange}
              <span class="result-range">{suggestedRange.start} – {suggestedRange.end}</span>
            {/if}
            <span class="ip-count">{blockIpCount.toLocaleString()} IPs</span>
            <button type="button" class="wizard-btn" on:click={applySuggestedBlockCidr} disabled={disabled || suggestedBlockLoading} title="Use suggested CIDR that does not overlap existing blocks in this environment">
              Use this CIDR
            </button>
          {:else if selectedPrefix < 9}
            <span class="wizard-hint wizard-hint-empty">Use prefix /9 or higher for a suggestion.</span>
          {:else}
            <span class="wizard-hint wizard-hint-empty">No suggestion available.</span>
          {/if}
        </div>
      {/if}
    </div>
  {:else if mode === 'allocation'}
    <div class="wizard-section">
      <header class="wizard-header">
        <h4 class="wizard-title">Subnet within block</h4>
        {#if parentCidr}
          <p class="wizard-hint">Choose a prefix length for the allocation. The first possible subnet in the block is suggested.</p>
        {:else}
          <p class="wizard-hint wizard-hint-empty">Select a block above to see prefix options and suggested CIDR.</p>
        {/if}
      </header>
      {#if parentCidr}
        <div class="wizard-fields">
          <div class="wizard-field parent-cidr-box">
            <span class="wizard-field-label">Parent block</span>
            <code class="parent-cidr-code">{parentCidr}</code>
          </div>
          <div class="wizard-field">
            <span class="wizard-field-label">Prefix length (subnet size)</span>
            <div class="prefix-with-count">
              <select bind:value={selectedPrefix} disabled={disabled} class="prefix-select">
                {#each allocationPrefixOptions as p}
                  <option value={p}>/{p}</option>
                {/each}
              </select>
              <span class="ip-count">{allocationIpCount.toLocaleString()} IPs</span>
            </div>
          </div>
        </div>
        <div class="wizard-result wizard-result-suggested">
          <span class="result-label">Suggested CIDR</span>
          {#if suggestedLoading}
            <span class="suggested-loading">…</span>
          {:else}
            {@const allocRange = allocationSuggestedCidr ? cidrRange(allocationSuggestedCidr) : null}
            <code class="result-cidr">{allocationSuggestedCidr}</code>
            {#if allocRange}
              <span class="result-range">{allocRange.start} – {allocRange.end}</span>
            {/if}
          {/if}
          <span class="ip-count">{allocationIpCount.toLocaleString()} IPs</span>
          <button type="button" class="wizard-btn" on:click={applyAllocationCidr} disabled={disabled || suggestedLoading} title={suggestedCidrFromApi ? 'Uses bin-packed suggestion from existing allocations' : ''}>
            Use this CIDR
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .cidr-wizard {
    margin: 0.75rem 0;
    padding: 1.25rem;
    background: var(--table-header-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  :global([data-theme='light']) .cidr-wizard {
    background: rgba(0, 0, 0, 0.03);
  }
  .wizard-section {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }
  .wizard-header {
    margin: 0;
    padding-bottom: 0.25rem;
  }
  .wizard-title {
    margin: 0 0 0.35rem 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text);
  }
  .wizard-hint {
    margin: 0;
    font-size: 0.8rem;
    line-height: 1.4;
    color: var(--text-muted);
  }
  .wizard-hint-empty {
    font-style: italic;
  }
  .wizard-fields {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 1rem 1.5rem;
    align-items: start;
  }
  .wizard-field {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    min-width: 0;
  }
  .wizard-field-label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .octets {
    display: flex;
    gap: 0.35rem;
    align-items: center;
  }
  .octet {
    width: 3.5rem;
    padding: 0.5rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    font-variant-numeric: tabular-nums;
  }
  .octet:focus {
    outline: none;
    border-color: var(--accent);
  }
  .octet:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .prefix-select {
    width: 5.5rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    color-scheme: dark;
  }
  :global([data-theme='light']) .prefix-select {
    color-scheme: light;
  }
  .prefix-select:focus {
    outline: none;
    border-color: var(--accent);
  }
  .prefix-select:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .prefix-with-count {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .ip-count {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .parent-cidr-box {
    padding: 0.6rem 0.75rem;
    background: rgba(0, 0, 0, 0.25);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .parent-cidr-box .wizard-field-label {
    margin-bottom: 0.1rem;
  }
  .parent-cidr-code {
    font-size: 0.9rem;
    font-family: var(--font-mono);
    color: var(--text);
    word-break: break-all;
  }
  .wizard-result {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.6rem 1rem;
    padding: 0.85rem 0.9rem;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .wizard-result .result-label {
    flex-shrink: 0;
  }
  .wizard-result-suggested {
    border: 1px solid var(--accent);
    background: var(--accent-dim);
  }
  :global([data-theme='light']) .wizard-result-suggested {
    background: var(--accent-dim);
  }
  .result-label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .suggested-loading {
    color: var(--text-muted);
    font-size: 0.9rem;
  }
  .result-cidr {
    padding: 0.3rem 0.6rem;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-size: 0.9rem;
    font-family: var(--font-mono);
    color: var(--text);
  }
  .result-range {
    display: block;
    font-size: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    margin-top: 0.15rem;
  }
  .wizard-btn {
    margin-left: auto;
    padding: 0.4rem 0.9rem;
    border-radius: var(--radius);
    font-family: var(--font-sans);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    transition: background 0.15s, border-color 0.15s;
  }
  .wizard-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.06);
    border-color: var(--text-muted);
  }
  .wizard-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
