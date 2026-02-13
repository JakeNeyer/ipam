<script>
  import { onMount } from 'svelte'
  import { cidrRange, ipVersion } from './cidr.js'
  import { suggestBlockCidr, suggestPoolBlockCidr } from './api.js'
  import { formatBlockCount } from './blockCount.js'

  /** @type {'block' | 'allocation'} */
  export let mode = 'block'
  /** Pool ID for block mode (used to suggest CIDR that does not overlap existing blocks in that pool). Bindable when poolOptions provided. */
  export let poolId = null
  /** Options for pool selector in block mode: [{ value, label }]. If non-empty, a Pool dropdown is shown. */
  export let poolOptions = []
  /** When in block mode and a pool is selected, the pool's CIDR (e.g. 10.0.0.0/8). IP version is locked to match. */
  export let poolCidr = ''
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

  const PREFIX_OPTIONS_BLOCK_IPV4 = [8, 12, 16, 20, 24, 25, 26, 27, 28, 29, 30]
  const PREFIX_OPTIONS_BLOCK_IPV6 = [16, 32, 40, 48, 52, 56, 60, 64, 80, 96, 112, 128]
  const MAX_PREFIX_IPV4 = 30
  const MAX_PREFIX_IPV6 = 128
  const MIN_PREFIX_ALLOC = 1

  let blockVersion = 4
  let octets = [10, 0, 0, 0]
  let baseAddressIPv6 = '2001:db8::'
  let selectedPrefix = 24

  $: blockCidrComputed =
    mode === 'block'
      ? blockVersion === 6
        ? `${baseAddressIPv6.trim()}/${selectedPrefix}`
        : `${octets[0]}.${octets[1]}.${octets[2]}.${octets[3]}/${selectedPrefix}`
      : ''

  // Sync external value → octets / baseAddressIPv6 (e.g. user typed in CIDR field). Depends only on value/mode to avoid cycle.
  $: if (mode === 'block' && value) {
    const c = parseCidr(value)
    if (c) {
      if (c.version === 6) {
        blockVersion = 6
        baseAddressIPv6 = c.network
        selectedPrefix = Math.min(MAX_PREFIX_IPV6, Math.max(16, c.prefix))
      } else {
        const parts = c.network.split('.')
        if (parts.length === 4) {
          blockVersion = 4
          octets = parts.map((p) => clampOctet(parseInt(p, 10)))
          selectedPrefix = Math.min(MAX_PREFIX_IPV4, Math.max(8, c.prefix))
        }
      }
    }
  }

  $: if (mode === 'allocation' && parentCidr) {
    const c = parseCidr(parentCidr)
    if (c && selectedPrefix <= c.prefix) {
      const maxP = c.version === 6 ? MAX_PREFIX_IPV6 : MAX_PREFIX_IPV4
      selectedPrefix = Math.min(maxP, c.prefix + 1)
    }
  }

  /** Parse CIDR; returns { network, prefix, version } for IPv4 and IPv6. */
  function parseCidr(cidr) {
    if (!cidr || typeof cidr !== 'string') return null
    const idx = cidr.indexOf('/')
    if (idx === -1) return null
    const network = cidr.slice(0, idx).trim()
    const prefix = parseInt(cidr.slice(idx + 1), 10)
    const v = ipVersion(cidr)
    if (v === 6) {
      if (isNaN(prefix) || prefix < 0 || prefix > 128) return null
      if (!network.includes(':')) return null
      return { network, prefix, version: 6 }
    }
    if (v === 4) {
      if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
      const parts = network.split('.')
      if (parts.length !== 4) return null
      return { network, prefix, version: 4 }
    }
    return null
  }

  function clampOctet(n) {
    if (isNaN(n)) return 0
    return Math.max(0, Math.min(255, n))
  }

  /** IP count for prefix; version 4 or 6. For IPv6 large counts return string. */
  function ipCountForPrefix(p, version = 4) {
    if (version === 6) {
      if (p < 0 || p > 128) return '0'
      const count = 2 ** (128 - p)
      return count > Number.MAX_SAFE_INTEGER ? String(count) : count
    }
    if (p < 0 || p > 32) return 0
    return Math.pow(2, 32 - p)
  }

  function formatIpCount(v) {
    if (v == null) return '0'
    return typeof v === 'string' ? formatBlockCount(v) : formatBlockCount(v)
  }

  $: blockIpCount = mode === 'block' ? ipCountForPrefix(selectedPrefix, blockVersion) : 0
  $: allocationIpCount =
    mode === 'allocation' && parentParsed
      ? ipCountForPrefix(selectedPrefix, parentParsed.version)
      : 0

  $: parentParsed = mode === 'allocation' && parentCidr ? parseCidr(parentCidr) : null
  $: allocationPrefixMax = parentParsed
    ? parentParsed.version === 6
      ? MAX_PREFIX_IPV6
      : MAX_PREFIX_IPV4
    : MAX_PREFIX_IPV4
  $: allocationPrefixMin = parentParsed ? Math.min(allocationPrefixMax, parentParsed.prefix + 1) : MIN_PREFIX_ALLOC
  $: allocationPrefixOptions = Array.from(
    { length: allocationPrefixMax - allocationPrefixMin + 1 },
    (_, i) => allocationPrefixMin + i
  )
  $: poolParsed = mode === 'block' && poolCidr ? parseCidr(poolCidr) : null
  $: poolVersion = poolParsed?.version ?? null
  $: if (mode === 'block' && poolVersion != null && blockVersion !== poolVersion) {
    blockVersion = poolVersion
  }
  $: blockPrefixOptions = blockVersion === 6 ? PREFIX_OPTIONS_BLOCK_IPV6 : PREFIX_OPTIONS_BLOCK_IPV4
  $: blockPrefixMin = blockVersion === 6 ? 16 : 8
  $: if (mode === 'block' && !blockPrefixOptions.includes(selectedPrefix)) {
    selectedPrefix = blockVersion === 6 ? 64 : 24
  }
  $: showBlockVersionSelect = mode === 'block' && poolVersion == null

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

  // Fetch suggested block CIDR for pool (no overlap with existing blocks in pool) when poolId and prefix set
  $: suggestedBlockTrigger =
    mode === 'block' && poolId && selectedPrefix >= blockPrefixMin
      ? [poolId, selectedPrefix]
      : []
  $: if (suggestedBlockTrigger.length) {
    const [pid, prefix] = suggestedBlockTrigger
    suggestedBlockLoading = true
    suggestPoolBlockCidr(pid, prefix)
      .then((cidr) => {
        if (pid === poolId && prefix === selectedPrefix) suggestedBlockCidrFromApi = cidr
      })
      .catch(() => {
        if (pid === poolId && prefix === selectedPrefix) suggestedBlockCidrFromApi = ''
      })
      .finally(() => {
        if (pid === poolId && prefix === selectedPrefix) suggestedBlockLoading = false
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
  function syncBlockValueIPv6() {
    if (mode === 'block' && blockVersion === 6) value = blockCidrComputed
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
        {#if poolId}
          <p class="wizard-hint">A suggested CIDR (optimal fit in the selected pool, no overlap with existing blocks) is shown below.</p>
        {:else if poolOptions.length > 0}
          <p class="wizard-hint">Select a pool and prefix to get a suggested CIDR that fits optimally in the pool.</p>
        {:else}
          <p class="wizard-hint">Choose the base address and prefix length for the block.</p>
        {/if}
      </header>
      <div class="wizard-fields">
        {#if poolOptions.length > 0}
          <div class="wizard-field">
            <span class="wizard-field-label">Pool</span>
            <select bind:value={poolId} disabled={disabled} class="prefix-select">
              <option value="">— None —</option>
              {#each poolOptions as opt}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          </div>
        {/if}
        {#if showBlockVersionSelect}
          <div class="wizard-field">
            <span class="wizard-field-label">IP version</span>
            <select bind:value={blockVersion} on:change={syncBlockValue} disabled={disabled} class="prefix-select">
              <option value={4}>IPv4</option>
              <option value={6}>IPv6</option>
            </select>
          </div>
        {:else if poolVersion != null}
          <div class="wizard-field">
            <span class="wizard-field-label">IP version</span>
            <span class="wizard-field-readonly" title="Determined by selected pool">{poolVersion === 6 ? 'IPv6' : 'IPv4'}</span>
          </div>
        {/if}
        {#if blockVersion === 6}
          <div class="wizard-field wizard-field-ipv6">
            <span class="wizard-field-label">Base address (IPv6)</span>
            <input
              type="text"
              class="input-ipv6"
              placeholder="e.g. 2001:db8::"
              bind:value={baseAddressIPv6}
              on:input={syncBlockValueIPv6}
              disabled={disabled}
            />
          </div>
        {:else}
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
        {/if}
        <div class="wizard-field">
          <span class="wizard-field-label">Prefix length</span>
          <div class="prefix-with-count">
            <select bind:value={selectedPrefix} on:change={syncBlockValue} disabled={disabled} class="prefix-select">
              {#each blockPrefixOptions as p}
                <option value={p}>/{p}</option>
              {/each}
            </select>
            <span class="ip-count">{formatIpCount(blockIpCount)} IPs</span>
          </div>
        </div>
      </div>
      {#if !poolId}
        <div class="wizard-result">
          <span class="result-label">Resulting CIDR</span>
          <code class="result-cidr">{blockCidrComputed}</code>
          {#if cidrRange(blockCidrComputed)}
            <span class="result-range">{cidrRange(blockCidrComputed).start} – {cidrRange(blockCidrComputed).end}</span>
          {/if}
          <span class="ip-count">{formatIpCount(blockIpCount)} IPs</span>
          <button type="button" class="btn btn-primary btn-small" on:click={applyBlockCidr} disabled={disabled}>
            Use manual CIDR
          </button>
        </div>
      {/if}
      {#if poolId}
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
            <span class="ip-count">{formatIpCount(blockIpCount)} IPs</span>
            <button type="button" class="btn btn-primary btn-small" on:click={applySuggestedBlockCidr} disabled={disabled || suggestedBlockLoading} title="Use suggested CIDR that does not overlap existing blocks in this pool">
              Use this CIDR
            </button>
          {:else if selectedPrefix < blockPrefixMin}
            <span class="wizard-hint wizard-hint-empty">Use prefix /{blockPrefixMin} or higher for a suggestion.</span>
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
              <span class="ip-count">{formatIpCount(allocationIpCount)} IPs</span>
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
          <span class="ip-count">{formatIpCount(allocationIpCount)} IPs</span>
          <button type="button" class="btn btn-primary btn-small" on:click={applyAllocationCidr} disabled={disabled || suggestedLoading} title={suggestedCidrFromApi ? 'Uses bin-packed suggestion from existing allocations' : ''}>
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
  .wizard-field-readonly {
    font-size: 0.9375rem;
    color: var(--text);
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
  :global(.dark) .prefix-select {
    background: var(--bg);
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
  .wizard-field-ipv6 {
    min-width: 200px;
  }
  .input-ipv6 {
    width: 100%;
    min-width: 180px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9rem;
    font-family: var(--font-mono);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
  }
  .input-ipv6:focus {
    outline: none;
    border-color: var(--accent);
  }
  .input-ipv6:disabled {
    opacity: 0.6;
    cursor: not-allowed;
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
  .wizard-result .btn {
    margin-left: auto;
  }
</style>
