<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { listEnvironments, listBlocks, listAllocations, exportCSV } from '../lib/api.js'

  const dispatch = createEventDispatcher()
  const NIL_UUID = '00000000-0000-0000-0000-000000000000'

  let loading = true
  let error = ''
  let errorModalMessage = ''
  let exporting = false
  let environments = []
  let blocks = []
  let allocations = []

  $: totalIPs = blocks.reduce((s, b) => s + (b.total_ips ?? 0), 0)
  $: usedIPs = blocks.reduce((s, b) => s + (b.used_ips ?? 0), 0)
  $: utilizationPercent = totalIPs > 0 ? Math.round((usedIPs / totalIPs) * 100) : 0
  $: overallUtilizationDisplay =
    totalIPs <= 0 ? '0' : (usedIPs / totalIPs) * 100 < 1 && usedIPs > 0 ? '<1' : String(utilizationPercent)
  $: orphanedCount = blocks.filter(
    (b) =>
      b.environment_id == null ||
      b.environment_id === '' ||
      String(b.environment_id).toLowerCase() === NIL_UUID
  ).length

  function blockUtilization(block) {
    if (!block || block.total_ips === 0) return 0
    return Math.min(100, Math.round((block.used_ips / block.total_ips) * 100))
  }

  function utilizationLabel(block) {
    if (!block || block.total_ips === 0) return '0%'
    const p = (block.used_ips / block.total_ips) * 100
    if (block.used_ips > 0 && p < 1) return '<1%'
    return Math.round(p) + '%'
  }

  $: blocksForChart = [...blocks].sort((a, b) => (b.used_ips ?? 0) - (a.used_ips ?? 0)).slice(0, 12)

  function envIdsMatch(a, b) {
    if (a == null || b == null) return false
    return String(a).toLowerCase() === String(b).toLowerCase()
  }
  function isOrphanedBlock(block) {
    const id = block.environment_id
    return id == null || id === '' || String(id).toLowerCase() === NIL_UUID
  }

  const GRAPH = {
    padding: 24,
    colWidth: 180,
    colGap1: 40,
    colGap2: 88,
    rowHeight: 40,
    nodeWidth: 160,
    nodeHeight: 34,
  }
  let graphHovered = null // { type: 'env'|'block'|'alloc', id: string } | null
  $: graphData = (() => {
    const envList = [...environments].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    const hasOrphaned = blocks.some(isOrphanedBlock)
    const envRows = envList.map((e, i) => ({ id: e.id, name: e.name || e.id, y: GRAPH.padding + i * GRAPH.rowHeight }))
    if (hasOrphaned) envRows.push({ id: 'orphaned', name: 'Orphaned', y: GRAPH.padding + envList.length * GRAPH.rowHeight })

    const blocksByEnv = new Map()
    envList.forEach((e) => blocksByEnv.set(String(e.id).toLowerCase(), []))
    if (hasOrphaned) blocksByEnv.set('orphaned', [])
    blocks.forEach((b) => {
      const key = isOrphanedBlock(b) ? 'orphaned' : String(b.environment_id).toLowerCase()
      const list = blocksByEnv.get(key) || []
      list.push(b)
    })
    blocksByEnv.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '')))
    const blockOrder = []
    envRows.forEach((env) => {
      const key = env.id === 'orphaned' ? 'orphaned' : String(env.id).toLowerCase()
      ;(blocksByEnv.get(key) || []).forEach((b) => blockOrder.push({ block: b, envId: env.id, envY: env.y }))
    })
    const blockRows = blockOrder.map((item, i) => ({
      id: item.block.id,
      name: item.block.name,
      cidr: item.block.cidr || '',
      environmentId: item.envId,
      y: GRAPH.padding + i * GRAPH.rowHeight,
      envY: item.envY,
    }))

    const allocsByBlock = new Map()
    blockOrder.forEach(({ block }) => allocsByBlock.set((block.name || '').toLowerCase(), []))
    allocations.forEach((a) => {
      const key = (a.block_name || '').toLowerCase()
      const list = allocsByBlock.get(key) || []
      list.push(a)
    })
    allocsByBlock.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '')))
    const allocOrder = []
    blockRows.forEach((br) => {
      const key = (br.name || '').toLowerCase()
      ;(allocsByBlock.get(key) || []).forEach((a) => allocOrder.push({ alloc: a, blockY: br.y }))
    })
    const allocRows = allocOrder.map((item, i) => ({
      id: item.alloc.id,
      name: item.alloc.name,
      cidr: item.alloc.cidr || '',
      blockName: item.alloc.block_name,
      y: GRAPH.padding + i * GRAPH.rowHeight,
      blockY: item.blockY,
    }))

    const col1X = GRAPH.padding
    const col2X = GRAPH.padding + GRAPH.nodeWidth + GRAPH.colGap1
    const col3X = col2X + GRAPH.nodeWidth + GRAPH.colGap2
    envRows.forEach((r) => { r.x = col1X })
    blockRows.forEach((r) => { r.x = col2X })
    allocRows.forEach((r) => { r.x = col3X })

    const edges = []
    blockRows.forEach((br) => {
      const env = envRows.find((e) => e.id === br.environmentId)
      if (env) edges.push({ from: { x: col1X + GRAPH.nodeWidth, y: env.y + GRAPH.nodeHeight / 2 }, to: { x: col2X, y: br.y + GRAPH.nodeHeight / 2 }, envId: env.id, blockId: br.id, allocId: null })
    })
    allocRows.forEach((ar) => {
      const br = blockRows.find((b) => (b.name || '').toLowerCase() === (ar.blockName || '').toLowerCase())
      if (br) edges.push({ from: { x: col2X + GRAPH.nodeWidth, y: br.y + GRAPH.nodeHeight / 2 }, to: { x: col3X, y: ar.y + GRAPH.nodeHeight / 2 }, envId: null, blockId: br.id, allocId: ar.id })
    })

    const width = col3X + GRAPH.nodeWidth + GRAPH.padding
    const height = Math.max(
      envRows.length ? envRows[envRows.length - 1].y + GRAPH.nodeHeight + GRAPH.padding : GRAPH.padding * 2,
      blockRows.length ? blockRows[blockRows.length - 1].y + GRAPH.nodeHeight + GRAPH.padding : GRAPH.padding * 2,
      allocRows.length ? allocRows[allocRows.length - 1].y + GRAPH.nodeHeight + GRAPH.padding : GRAPH.padding * 2
    )
    return { envRows, blockRows, allocRows, edges, width, height }
  })()

  function isEdgeHighlighted(edge) {
    if (!graphHovered) return false
    const id = (x) => (x != null ? String(x).toLowerCase() : '')
    if (graphHovered.type === 'env') return id(edge.envId) === id(graphHovered.id)
    if (graphHovered.type === 'block') return id(edge.blockId) === id(graphHovered.id)
    if (graphHovered.type === 'alloc') return id(edge.allocId) === id(graphHovered.id)
    return false
  }

  $: graphHighlightedEdgeSet = (() => {
    if (!graphHovered || !graphData.edges) return new Set()
    const set = new Set()
    graphData.edges.forEach((edge, i) => {
      if (isEdgeHighlighted(edge)) set.add(i)
    })
    return set
  })()

  async function doExportCSV() {
    exporting = true
    errorModalMessage = ''
    try {
      await exportCSV()
    } catch (e) {
      errorModalMessage = e.message || 'Export failed'
    } finally {
      exporting = false
    }
  }

  onMount(async () => {
    loading = true
    error = ''
    try {
      const [envsRes, blksRes, allocsRes] = await Promise.all([
        listEnvironments(),
        listBlocks(),
        listAllocations(),
      ])
      environments = envsRes.environments
      blocks = blksRes.blocks
      allocations = allocsRes.allocations
    } catch (e) {
      error = e.message || 'Failed to load dashboard'
      errorModalMessage = error + (error.includes('8011') ? '' : ' Ensure the API is running at localhost:8011.')
    } finally {
      loading = false
    }
  })
</script>

<div class="dashboard">
  <header class="header">
    <h1>Dashboard</h1>
    <button class="export-btn" type="button" disabled={exporting} on:click={doExportCSV}>
      {exporting ? 'Exporting…' : 'Export CSV'}
    </button>
  </header>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    <section class="stats-section">
      <div class="stats-grid">
        <div class="stat-card">
          <span class="stat-value">{environments.length}</span>
          <span class="stat-label">Environments</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{blocks.length}</span>
          <span class="stat-label">Network blocks</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{allocations.length}</span>
          <span class="stat-label">Allocations</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{totalIPs.toLocaleString()}</span>
          <span class="stat-label">Total IPs</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{usedIPs.toLocaleString()}</span>
          <span class="stat-label">Used IPs</span>
        </div>
        <div class="stat-card stat-card-accent">
          <span class="stat-value">{overallUtilizationDisplay}%</span>
          <span class="stat-label">Overall utilization</span>
        </div>
      </div>
      {#if orphanedCount > 0}
        <div class="orphaned-notice">
          <span class="orphaned-badge">{orphanedCount} orphaned block{orphanedCount === 1 ? '' : 's'}</span>
          <a href="#networks?orphaned=1" class="orphaned-link">View on Networks →</a>
        </div>
      {/if}
    </section>

    {#if blocks.length > 0}
      <section class="chart-section">
        <h2 class="section-title">Block utilization</h2>
        <div class="chart">
          {#each blocksForChart as block}
            <div class="chart-row">
              <span class="chart-label" title={block.name}>
                {block.name.length > 20 ? block.name.slice(0, 17) + '…' : block.name}
              </span>
              <div class="chart-bar-wrap">
                <div
                  class="chart-bar"
                  class:high={blockUtilization(block) >= 80}
                  class:mid={blockUtilization(block) >= 50 && blockUtilization(block) < 80}
                  style="width: {blockUtilization(block)}%"
                  role="presentation"
                ></div>
              </div>
              <span class="chart-pct">{utilizationLabel(block)}</span>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    {#if environments.length > 0 || blocks.length > 0 || allocations.length > 0}
      <section class="graph-section">
        <h2 class="section-title">Resource graph</h2>
        <div class="graph-wrap">
          <svg
            class="resource-graph"
            viewBox="0 0 {graphData.width} {graphData.height}"
            width="100%"
            preserveAspectRatio="xMidYMid meet"
          >
            <!-- edges -->
            {#each graphData.edges as edge, i}
              <line
                x1={edge.from.x}
                y1={edge.from.y}
                x2={edge.to.x}
                y2={edge.to.y}
                class="graph-edge"
                class:graph-edge-highlight={graphHighlightedEdgeSet.has(i)}
              />
            {/each}
            <!-- env nodes -->
            {#each graphData.envRows as node}
              <g
                class="graph-node-wrap"
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'env', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => (node.id === 'orphaned' ? dispatch('viewOrphaned') : dispatch('envBlocks', node.id))}
                on:keydown={(e) => e.key === 'Enter' && (node.id === 'orphaned' ? dispatch('viewOrphaned') : dispatch('envBlocks', node.id))}
              >
                <rect
                  class="graph-node graph-node-env"
                  class:graph-node-orphaned={node.id === 'orphaned'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <text x={node.x + GRAPH.nodeWidth / 2} y={node.y + GRAPH.nodeHeight / 2 + 1} class="graph-label" text-anchor="middle">{node.name}</text>
              </g>
            {/each}
            <!-- block nodes (name + cidr) -->
            {#each graphData.blockRows as node}
              <g
                class="graph-node-wrap"
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'block', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => dispatch('viewBlock', node.name)}
                on:keydown={(e) => e.key === 'Enter' && dispatch('viewBlock', node.name)}
              >
                <rect
                  class="graph-node graph-node-block"
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <text x={node.x + GRAPH.nodeWidth / 2} y={node.y + 11} class="graph-label" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH.nodeWidth / 2} y={node.y + 24} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
                {#if node.cidr && cidrRange(node.cidr)}
                  <title>{node.cidr} → {cidrRange(node.cidr).start} – {cidrRange(node.cidr).end}</title>
                {/if}
              </g>
            {/each}
            <!-- allocation nodes -->
            {#each graphData.allocRows as node}
              <g
                class="graph-node-wrap"
                class:graph-node-alloc-hovered={graphHovered?.type === 'alloc' && graphHovered?.id === node.id}
                on:mouseenter={() => (graphHovered = { type: 'alloc', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
              >
                <rect
                  class="graph-node graph-node-alloc"
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <text x={node.x + GRAPH.nodeWidth / 2} y={node.y + 11} class="graph-label graph-label-alloc" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH.nodeWidth / 2} y={node.y + 24} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
                {#if node.cidr && cidrRange(node.cidr)}
                  <title>{node.cidr} → {cidrRange(node.cidr).start} – {cidrRange(node.cidr).end}</title>
                {/if}
              </g>
            {/each}
          </svg>
        </div>
      </section>
    {/if}

    {#if environments.length > 0}
      <section class="env-section">
        <h2 class="section-title">Environments</h2>
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>ID</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {#each environments as env}
                <tr>
                  <td class="name">{env.name}</td>
                  <td class="id"><code>{env.id}</code></td>
                  <td class="action">
                    <button type="button" class="btn-link" on:click={() => dispatch('envBlocks', env.id)}>
                      View blocks →
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {:else}
      <section class="empty-hint">
        <p>No environments yet. Create one from the <a href="#environments">Environments</a> page to get started.</p>
      </section>
    {/if}
  {/if}

  {#if errorModalMessage}
    <ErrorModal message={errorModalMessage} on:close={() => (errorModalMessage = '')} />
  {/if}
</div>

<style>
  .dashboard {
    padding-top: 0.5rem;
  }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }
  .header h1 {
    margin: 0;
    font-size: 1.35rem;
    font-weight: 600;
    letter-spacing: -0.02em;
  }
  .export-btn {
    padding: 0.5rem 0.875rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--accent);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    cursor: pointer;
  }
  .export-btn:hover:not(:disabled) {
    background: var(--surface-elevated);
  }
  .export-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .loading {
    color: var(--text-muted);
    padding: 2rem;
  }
  .section-title {
    margin: 0 0 0.75rem 0;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-muted);
  }
  .stats-section {
    margin-bottom: 1.75rem;
  }
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 0.75rem;
  }
  .stat-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    padding: 1rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  .stat-card-accent .stat-value {
    color: var(--accent);
  }
  .stat-value {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .stat-label {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .orphaned-notice {
    margin-top: 0.75rem;
    padding: 0.6rem 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .orphaned-badge {
    font-size: 0.85rem;
    color: var(--warn);
    font-weight: 500;
  }
  .orphaned-link {
    font-size: 0.85rem;
    color: var(--accent);
    text-decoration: none;
  }
  .orphaned-link:hover {
    text-decoration: underline;
  }
  .graph-section {
    margin-bottom: 2rem;
  }
  .graph-wrap {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
    min-height: 120px;
  }
  .resource-graph {
    display: block;
  }
  .graph-edge {
    stroke: var(--border);
    stroke-width: 1;
    transition: stroke 0.15s, stroke-width 0.15s;
    pointer-events: none;
  }
  .graph-edge.graph-edge-highlight {
    stroke: var(--accent);
    stroke-width: 2.5;
  }
  .graph-node-wrap {
    cursor: default;
  }
  .graph-node-wrap[role='button'] {
    cursor: pointer;
  }
  .graph-node-wrap[role='button']:hover .graph-node-block,
  .graph-node-wrap[role='button']:hover .graph-node-env:not(.graph-node-orphaned) {
    fill: var(--accent-dim);
    stroke: var(--accent);
  }
  .graph-node-wrap[role='button']:hover .graph-node-orphaned {
    fill: rgba(210, 153, 34, 0.25);
    stroke: var(--warn);
  }
  .graph-node {
    fill: var(--bg);
    stroke: var(--border);
    stroke-width: 1;
    transition: fill 0.15s, stroke 0.15s;
  }
  .graph-node-env {
    fill: var(--surface);
  }
  .graph-node-orphaned {
    fill: rgba(210, 153, 34, 0.12);
    stroke: var(--warn);
  }
  .graph-node-block {
    fill: var(--bg);
  }
  .graph-node-alloc {
    fill: rgba(88, 166, 255, 0.08);
    stroke: var(--accent);
  }
  .graph-node-wrap.graph-node-alloc-hovered .graph-node-alloc {
    fill: rgba(88, 166, 255, 0.2);
    stroke-width: 2;
  }
  .graph-label {
    font-size: 11px;
    fill: var(--text);
    pointer-events: none;
  }
  .graph-label-alloc {
    font-size: 10px;
    fill: var(--text-muted);
  }
  .graph-label.graph-label-cidr {
    font-size: 9px;
    fill: var(--text-muted);
  }
  .chart-section {
    margin-bottom: 2rem;
  }
  .chart {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    padding: 1rem 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }
  .chart-row {
    display: grid;
    grid-template-columns: minmax(0, 140px) 1fr minmax(2.5rem, auto);
    align-items: center;
    gap: 0.75rem;
  }
  .chart-label {
    font-size: 0.85rem;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chart-bar-wrap {
    height: 1.25rem;
    background: var(--bg);
    border-radius: 4px;
    overflow: hidden;
    min-width: 0;
  }
  .chart-bar {
    height: 100%;
    border-radius: 4px;
    background: var(--accent);
    transition: width 0.2s ease;
  }
  .chart-bar.mid {
    background: var(--warn);
  }
  .chart-bar.high {
    background: var(--danger);
  }
  .chart-pct {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    text-align: right;
    min-width: 2.5rem;
  }
  .env-section {
    margin-bottom: 1rem;
  }
  .table-wrap {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }
  .table {
    width: 100%;
    border-collapse: collapse;
  }
  .table th {
    text-align: left;
    padding: 0.75rem 1rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
    background: var(--table-header-bg);
    border-bottom: 1px solid var(--border);
  }
  .table td {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .table tr:hover td {
    background: var(--table-row-hover);
  }
  .table tr:last-child td {
    border-bottom: none;
  }
  .table .name {
    font-weight: 500;
  }
  .table .id code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .table .action {
    text-align: right;
  }
  .btn-link {
    background: none;
    border: none;
    color: var(--accent);
    font-family: var(--font-sans);
    font-size: 0.9rem;
    cursor: pointer;
    padding: 0.25rem 0;
  }
  .btn-link:hover {
    text-decoration: underline;
  }
  .empty-hint {
    margin-top: 1.5rem;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
    color: var(--text-muted);
    font-size: 0.9rem;
  }
  .empty-hint a {
    color: var(--accent);
    text-decoration: none;
  }
  .empty-hint a:hover {
    text-decoration: underline;
  }
</style>
