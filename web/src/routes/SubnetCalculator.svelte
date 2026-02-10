<script>
  import { getSubnetInfo, divideSubnets, getParentSubnet, getSiblingCidr } from '../lib/cidr.js'

  const PREFIX_OPTIONS = [8, 12, 16, 20, 24, 25, 26, 27, 28, 29, 30, 31, 32]

  let networkInput = '10.0.0.0'
  let selectedPrefix = 24
  let rows = []
  let lastBase = { network: '', prefix: -1 }

  $: prefixNum = typeof selectedPrefix === 'number' ? selectedPrefix : parseInt(selectedPrefix, 10)

  $: currentInfo = (() => {
    if (!networkInput.trim() || isNaN(prefixNum) || prefixNum < 0 || prefixNum > 32) return null
    return getSubnetInfo(networkInput.trim(), prefixNum)
  })()

  $: if (currentInfo) {
    if (
      rows.length === 0 ||
      networkInput.trim() !== lastBase.network ||
      prefixNum !== lastBase.prefix
    ) {
      lastBase = { network: networkInput.trim(), prefix: prefixNum }
      rows = [currentInfo]
    }
  } else {
    rows = []
  }

  $: error = (() => {
    if (!networkInput.trim()) return ''
    const info = getSubnetInfo(networkInput.trim(), prefixNum)
    if (!info) return 'Invalid network address (e.g. 10.0.0.0)'
    return ''
  })()

  function prefixFromCidr(cidr) {
    const idx = cidr.indexOf('/')
    return idx === -1 ? null : parseInt(cidr.slice(idx + 1), 10)
  }

  function divideRow(index) {
    const row = rows[index]
    if (!row || row.cidr == null) return
    const p = prefixFromCidr(row.cidr)
    if (p == null || p >= 32) return
    const halves = divideSubnets(row.cidr, p + 1)
    if (!halves || halves.length !== 2) return
    rows = [...rows.slice(0, index), halves[0], halves[1], ...rows.slice(index + 1)]
  }

  function joinRow(index) {
    const row = rows[index]
    if (!row) return
    const sibling = getSiblingCidr(row.cidr)
    if (!sibling || rows[index + 1]?.cidr !== sibling) return
    const parent = getParentSubnet(row.cidr)
    if (!parent) return
    rows = [...rows.slice(0, index), parent, ...rows.slice(index + 2)]
  }

  function canJoin(index) {
    if (index < 0 || index >= rows.length) return false
    const sibling = getSiblingCidr(rows[index].cidr)
    return sibling != null && rows[index + 1]?.cidr === sibling
  }

  function resetTable() {
    lastBase = { network: '', prefix: -1 }
  }
</script>

<div class="page">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Subnet calculator</h1>
      <p class="page-desc">Enter a network and mask to see the resulting subnet.</p>
    </div>
  </header>

  <div class="subnet-form-card card">
    <form class="subnet-form" on:submit|preventDefault>
      <div class="form-row">
        <label for="subnet-network">Network address</label>
        <input
          id="subnet-network"
          type="text"
          class="input"
          placeholder="e.g. 10.0.0.0"
          bind:value={networkInput}
          aria-invalid={networkInput.trim() && !currentInfo}
        />
      </div>
      <div class="form-row">
        <label for="subnet-mask">Mask bits /</label>
        <select
          id="subnet-mask"
          class="input select"
          bind:value={selectedPrefix}
          aria-label="Network mask prefix length"
        >
          {#each PREFIX_OPTIONS as p}
            <option value={p}>/{p}</option>
          {/each}
        </select>
      </div>
      {#if error}
        <p class="form-error">{error}</p>
      {/if}
      {#if rows.length > 0}
        <div class="subnet-form-actions">
          <button type="button" class="btn btn-secondary btn-small" on:click={resetTable}>
            Reset
          </button>
        </div>
      {/if}
    </form>
  </div>

  {#if rows.length > 0}
    <div class="subnet-table-wrap card">
      <table class="subnet-table">
        <thead>
          <tr>
            <th>Subnet address</th>
            <th>Range of addresses</th>
            <th>Useable IPs</th>
            <th>Hosts</th>
            <th>Divide</th>
            <th>Join</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as row, i (row.cidr + '-' + i)}
            {@const p = prefixFromCidr(row.cidr)}
            <tr>
              <td><code class="subnet-code">{row.cidr}</code></td>
              <td class="subnet-range">{row.first} – {row.last}</td>
              <td>{row.usable.toLocaleString()}</td>
              <td>{row.usable.toLocaleString()}</td>
              <td>
                {#if p != null && p < 32}
                  <button
                    type="button"
                    class="subnet-link"
                    on:click={() => divideRow(i)}
                  >
                    Divide
                  </button>
                {:else}
                  <span class="subnet-muted">—</span>
                {/if}
              </td>
              <td>
                {#if canJoin(i)}
                  <button
                    type="button"
                    class="subnet-link"
                    on:click={() => joinRow(i)}
                  >
                    Join
                  </button>
                {:else}
                  <span class="subnet-muted">—</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 0;
  }
  .subnet-form-card {
    margin-bottom: 1.5rem;
  }
  .subnet-form {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem 1.5rem;
    align-items: flex-end;
  }
  .form-row {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }
  .form-row label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text);
  }
  .form-row .input {
    width: 100%;
    min-width: 140px;
    max-width: 220px;
    padding: 0.5rem 0.75rem;
    font-size: 0.9375rem;
    font-family: var(--font-mono);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
  }
  .form-row .input:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent-dim);
  }
  .form-row .input[aria-invalid="true"] {
    border-color: var(--danger);
  }
  .form-row .select {
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12' fill='none' stroke='%236b7280' stroke-width='2'%3E%3Cpath d='M2 4 L6 8 L10 4'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 0.6rem center;
    padding-right: 2rem;
  }
  .form-error {
    margin: 0;
    font-size: 0.875rem;
    color: var(--danger);
    width: 100%;
  }
  .subnet-form-actions {
    margin-top: 0.25rem;
    width: 100%;
  }
  .card {
    padding: 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .subnet-table-wrap {
    overflow-x: auto;
    padding: 0;
  }
  .subnet-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }
  .subnet-table th,
  .subnet-table td {
    padding: 0.6rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }
  .subnet-table th {
    font-weight: 600;
    color: var(--text-muted);
    white-space: nowrap;
  }
  .subnet-table tbody tr:hover {
    background: var(--accent-dim);
  }
  .subnet-table .subnet-range {
    white-space: nowrap;
  }
  .subnet-code {
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    background: var(--bg);
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
  }
  .subnet-link {
    background: none;
    border: none;
    padding: 0;
    font-family: var(--font-sans);
    font-size: inherit;
    font-weight: 500;
    color: var(--accent);
    cursor: pointer;
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .subnet-link:hover {
    color: var(--accent);
    text-decoration: none;
  }
  .subnet-muted {
    color: var(--text-muted);
  }
</style>
