<script>
  import { createEventDispatcher } from 'svelte'
  import { onMount } from 'svelte'
  import { get } from 'svelte/store'
  import Icon from '@iconify/svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import { cidrRange } from '../lib/cidr.js'
  import { user } from '../lib/auth.js'
  import { listEnvironments, listBlocks, listAllocations, listReservedBlocks, exportCSV } from '../lib/api.js'

  const GRAPH_ICON_SIZE = 12
  const GRAPH_ICON_LEFT = 6

  const dispatch = createEventDispatcher()
  const NIL_UUID = '00000000-0000-0000-0000-000000000000'

  let loading = true
  let error = ''
  let errorModalMessage = ''
  let exporting = false
  let environments = []
  let blocks = []
  let allocations = []
  let reservedBlocks = []

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
    paddingRight: 48,
    colWidth: 180,
    colGap: 40,
    rowGap: 12,
    nodeWidth: 130,
    nodeHeight: 28,
  }
  const GRAPH_TEXT_CENTER_X_OFFSET = GRAPH_ICON_LEFT + GRAPH_ICON_SIZE + (GRAPH.nodeWidth - GRAPH_ICON_LEFT - GRAPH_ICON_SIZE) / 2
  const rowPitch = GRAPH.nodeHeight + GRAPH.rowGap
  let graphHovered = null // { type: 'env'|'block'|'alloc', id: string } | null
  $: graphData = (() => {
    const envList = [...environments].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    const hasOrphaned = blocks.some(isOrphanedBlock)
    const envRows = envList.map((e, i) => ({ id: e.id, name: e.name || e.id, y: GRAPH.padding + i * rowPitch }))
    if (hasOrphaned) envRows.push({ id: 'orphaned', name: 'Orphaned', y: GRAPH.padding + envList.length * rowPitch })
    if (reservedBlocks.length > 0) envRows.push({ id: 'reserved', name: 'Reserved', y: GRAPH.padding + envRows.length * rowPitch })

    const blocksByEnv = new Map()
    envList.forEach((e) => blocksByEnv.set(String(e.id).toLowerCase(), []))
    if (hasOrphaned) blocksByEnv.set('orphaned', [])
    if (reservedBlocks.length > 0) blocksByEnv.set('reserved', reservedBlocks.map((r) => ({ id: r.id, name: (r.name && String(r.name).trim()) || r.cidr || '—', cidr: r.cidr })))
    blocks.forEach((b) => {
      const key = isOrphanedBlock(b) ? 'orphaned' : String(b.environment_id).toLowerCase()
      const list = blocksByEnv.get(key) || []
      list.push(b)
    })
    blocksByEnv.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '')))
    let blockOrder = []
    envRows.forEach((env) => {
      const key = env.id === 'orphaned' ? 'orphaned' : String(env.id).toLowerCase()
      ;(blocksByEnv.get(key) || []).forEach((b) => blockOrder.push({ block: b, envId: env.id }))
    })

    const allocsByBlock = new Map()
    blockOrder.forEach(({ block }) => allocsByBlock.set((block.name || '').toLowerCase(), []))
    allocations.forEach((a) => {
      const key = (a.block_name || '').toLowerCase()
      const list = allocsByBlock.get(key) || []
      list.push(a)
    })
    allocsByBlock.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '')))

    let blockRows = blockOrder.map((item, i) => ({
      id: item.block.id,
      name: item.block.name,
      cidr: item.block.cidr || '',
      environmentId: item.envId,
      y: GRAPH.padding + i * rowPitch,
    }))
    let allocOrder = []
    blockRows.forEach((br) => {
      const key = (br.name || '').toLowerCase()
      ;(allocsByBlock.get(key) || []).forEach((a) => allocOrder.push({ alloc: a, blockId: br.id, blockName: br.name, blockEnvironmentId: br.environmentId }))
    })
    let allocRows = allocOrder.map((item, i) => ({
      id: item.alloc.id,
      name: item.alloc.name,
      cidr: item.alloc.cidr || '',
      blockName: item.alloc.block_name,
      y: GRAPH.padding + i * rowPitch,
      blockId: item.blockId,
      blockEnvironmentId: item.blockEnvironmentId,
    }))

    const blockAllocCenter = new Map()
    blockRows.forEach((br) => {
      const allocs = allocRows.filter((ar) => (ar.blockName || '').toLowerCase() === (br.name || '').toLowerCase())
      const center = allocs.length > 0 ? (Math.min(...allocs.map((a) => a.y)) + Math.max(...allocs.map((a) => a.y))) / 2 : br.y
      blockAllocCenter.set((br.name || '').toLowerCase(), center)
    })
    const envKey = (id) => (id === 'orphaned' ? 'orphaned' : String(id).toLowerCase())
    const newBlockOrder = []
    envRows.forEach((env) => {
      const key = envKey(env.id)
      const items = blockOrder.filter((item) => envKey(item.envId) === key)
      items.sort((a, b) => (blockAllocCenter.get((a.block.name || '').toLowerCase()) ?? 0) - (blockAllocCenter.get((b.block.name || '').toLowerCase()) ?? 0))
      items.forEach((item) => newBlockOrder.push(item))
    })
    blockOrder = newBlockOrder
    blockRows = blockOrder.map((item, i) => ({
      id: item.block.id,
      name: item.block.name,
      cidr: item.block.cidr || '',
      environmentId: item.envId,
      y: GRAPH.padding + i * rowPitch,
    }))
    allocOrder = []
    blockRows.forEach((br) => {
      const key = (br.name || '').toLowerCase()
      ;(allocsByBlock.get(key) || []).forEach((a) => allocOrder.push({ alloc: a, blockId: br.id, blockName: br.name, blockEnvironmentId: br.environmentId }))
    })
    allocRows = allocOrder.map((item, i) => ({
      id: item.alloc.id,
      name: item.alloc.name,
      cidr: item.alloc.cidr || '',
      blockName: item.alloc.block_name,
      y: GRAPH.padding + i * rowPitch,
      blockId: item.blockId,
      blockEnvironmentId: item.blockEnvironmentId,
    }))

    envRows.forEach((env) => {
      const envBlocks = blockRows.filter((br) => String(br.environmentId).toLowerCase() === String(env.id).toLowerCase())
      if (envBlocks.length > 0) {
        const minY = Math.min(...envBlocks.map((b) => b.y))
        const maxY = Math.max(...envBlocks.map((b) => b.y))
        env.y = (minY + maxY) / 2
      }
    })
    envRows
      .slice()
      .sort((a, b) => a.y - b.y)
      .forEach((env, i) => {
        env.y = GRAPH.padding + i * rowPitch
      })

    allocRows.forEach((ar) => {
      const br = blockRows.find((b) => (b.name || '').toLowerCase() === (ar.blockName || '').toLowerCase())
      if (br) ar.blockY = br.y
      else ar.blockY = ar.y
    })

    const col1X = GRAPH.padding
    const col2X = GRAPH.padding + GRAPH.nodeWidth + GRAPH.colGap
    const col3X = col2X + GRAPH.nodeWidth + GRAPH.colGap
    envRows.forEach((r) => { r.x = col1X })
    blockRows.forEach((r) => { r.x = col2X })
    allocRows.forEach((r) => { r.x = col3X })

    const edges = []
    blockRows.forEach((br) => {
      const env = envRows.find((e) => e.id === br.environmentId)
      if (env) {
        const edgeType = env.id === 'orphaned' ? 'orphaned' : env.id === 'reserved' ? 'reserved' : null
        edges.push({ from: { x: col1X + GRAPH.nodeWidth, y: env.y + GRAPH.nodeHeight / 2 }, to: { x: col2X, y: br.y + GRAPH.nodeHeight / 2 }, envId: env.id, blockId: br.id, allocId: null, edgeType })
      }
    })
    allocRows.forEach((ar) => {
      const br = blockRows.find((b) => (b.name || '').toLowerCase() === (ar.blockName || '').toLowerCase())
      if (br) {
        const edgeType = br.environmentId === 'orphaned' ? 'orphaned' : br.environmentId === 'reserved' ? 'reserved' : null
        edges.push({ from: { x: col2X + GRAPH.nodeWidth, y: br.y + GRAPH.nodeHeight / 2 }, to: { x: col3X, y: ar.y + GRAPH.nodeHeight / 2 }, envId: null, blockId: br.id, allocId: ar.id, edgeType })
      }
    })

    const width = col3X + GRAPH.nodeWidth + GRAPH.paddingRight
    const maxY = (rows) => (rows.length ? Math.max(...rows.map((r) => r.y)) + GRAPH.nodeHeight + GRAPH.padding : GRAPH.padding * 2)
    const height = Math.max(maxY(envRows), maxY(blockRows), maxY(allocRows))
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
    const u = get(user)
    try {
      const [envsRes, blksRes, allocsRes] = await Promise.all([
        listEnvironments(),
        listBlocks(),
        listAllocations(),
      ])
      environments = envsRes.environments
      blocks = blksRes.blocks
      allocations = allocsRes.allocations
      if (u?.role === 'admin') {
        try {
          const r = await listReservedBlocks()
          reservedBlocks = r.reserved_blocks || []
        } catch (_) {}
      }
    } catch (e) {
      error = e.message || 'Failed to load dashboard'
      errorModalMessage = error + (error.includes('8011') ? '' : ' Ensure the API is running at localhost:8011.')
    } finally {
      loading = false
    }
  })
</script>

<div class="dashboard">
  <header class="page-header">
    <h1 class="page-title">Dashboard</h1>
    <button class="btn btn-primary" type="button" disabled={exporting} on:click={doExportCSV}>
      {exporting ? 'Exporting…' : 'Export CSV'}
    </button>
  </header>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    <section class="stats-section">
      <div class="stats-grid">
        <a href="#environments" class="stat-card stat-card-link">
          <span class="stat-value">{environments.length}</span>
          <span class="stat-label">Environments</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link">
          <span class="stat-value">{blocks.length}</span>
          <span class="stat-label">Blocks</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link">
          <span class="stat-value">{allocations.length}</span>
          <span class="stat-label">Allocations</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link">
          <span class="stat-value">{totalIPs.toLocaleString()}</span>
          <span class="stat-label">Total IPs</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link">
          <span class="stat-value">{usedIPs.toLocaleString()}</span>
          <span class="stat-label">Allocated IPs</span>
        </a>
        <a href="#networks" class="stat-card stat-card-accent stat-card-link">
          <span class="stat-value">{overallUtilizationDisplay}%</span>
          <span class="stat-label">Utilization</span>
        </a>
        {#if $user?.role === 'admin'}
          <a href="#reserved-blocks" class="stat-card stat-card-link">
            <span class="stat-value">{reservedBlocks.length}</span>
            <span class="stat-label">Reserved blocks</span>
          </a>
        {/if}
      </div>
    </section>

    {#if orphanedCount > 0}
      <div class="orphaned-card">
        <span class="orphaned-card-message">
          <Icon icon="lucide:alert-triangle" class="orphaned-card-icon" width="1.125rem" height="1.125rem" />
          {orphanedCount} orphaned block{orphanedCount === 1 ? '' : 's'} — not assigned to any environment.
        </span>
        <a href="#networks?orphaned=1" class="orphaned-card-link">View on Networks →</a>
      </div>
    {/if}

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

    {#if environments.length > 0 || blocks.length > 0 || allocations.length > 0 || reservedBlocks.length > 0}
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
                class:graph-edge-orphaned={edge.edgeType === 'orphaned'}
                class:graph-edge-reserved={edge.edgeType === 'reserved'}
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
                on:click={() => { if (node.id === 'orphaned') dispatch('viewOrphaned'); else if (node.id === 'reserved') window.location.hash = 'reserved-blocks'; else dispatch('envBlocks', node.id) }}
                on:keydown={(e) => e.key === 'Enter' && (node.id === 'orphaned' ? dispatch('viewOrphaned') : node.id === 'reserved' ? (window.location.hash = 'reserved-blocks') : dispatch('envBlocks', node.id))}
              >
                <rect
                  class="graph-node graph-node-env"
                  class:graph-node-orphaned={node.id === 'orphaned'}
                  class:graph-node-reserved={node.id === 'reserved'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <foreignObject x={node.x + GRAPH_ICON_LEFT} y={node.y + (GRAPH.nodeHeight - GRAPH_ICON_SIZE) / 2} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE}>
                  <div xmlns="http://www.w3.org/1999/xhtml" class="graph-node-icon" class:graph-node-icon-orphaned={node.id === 'orphaned'} class:graph-node-icon-reserved={node.id === 'reserved'}>
                    <Icon icon={node.id === 'orphaned' ? 'lucide:alert-triangle' : node.id === 'reserved' ? 'lucide:shield-off' : 'lucide:layers'} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE} />
                  </div>
                </foreignObject>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + GRAPH.nodeHeight / 2 + 1} class="graph-label" text-anchor="middle">{node.name}</text>
              </g>
            {/each}
            <!-- block nodes (name + cidr) -->
            {#each graphData.blockRows as node}
              <g
                class="graph-node-wrap"
                class:graph-node-orphaned-block={node.environmentId === 'orphaned'}
                class:graph-node-reserved-block={node.environmentId === 'reserved'}
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'block', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => node.environmentId === 'reserved' ? (window.location.hash = 'reserved-blocks') : dispatch('viewBlock', node.name)}
                on:keydown={(e) => e.key === 'Enter' && (node.environmentId === 'reserved' ? (window.location.hash = 'reserved-blocks') : dispatch('viewBlock', node.name))}
              >
                <rect
                  class="graph-node graph-node-block"
                  class:graph-node-orphaned-block-rect={node.environmentId === 'orphaned'}
                  class:graph-node-reserved-block-rect={node.environmentId === 'reserved'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <foreignObject x={node.x + GRAPH_ICON_LEFT} y={node.y + (GRAPH.nodeHeight - GRAPH_ICON_SIZE) / 2} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE}>
                  <div xmlns="http://www.w3.org/1999/xhtml" class="graph-node-icon">
                    <Icon icon="lucide:network" width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE} />
                  </div>
                </foreignObject>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (11 * GRAPH.nodeHeight) / 34} class="graph-label" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (24 * GRAPH.nodeHeight) / 34} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
                {#if node.cidr && cidrRange(node.cidr)}
                  <title>{node.cidr} → {cidrRange(node.cidr).start} – {cidrRange(node.cidr).end}</title>
                {/if}
              </g>
            {/each}
            <!-- allocation nodes -->
            {#each graphData.allocRows as node}
              <g
                class="graph-node-wrap"
                class:graph-node-orphaned-alloc={node.blockEnvironmentId === 'orphaned'}
                class:graph-node-alloc-hovered={graphHovered?.type === 'alloc' && graphHovered?.id === node.id}
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'alloc', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => dispatch('viewAllocation', node.name)}
                on:keydown={(e) => e.key === 'Enter' && dispatch('viewAllocation', node.name)}
              >
                <rect
                  class="graph-node graph-node-alloc"
                  class:graph-node-orphaned-alloc-rect={node.blockEnvironmentId === 'orphaned'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <foreignObject x={node.x + GRAPH_ICON_LEFT} y={node.y + (GRAPH.nodeHeight - GRAPH_ICON_SIZE) / 2} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE}>
                  <div xmlns="http://www.w3.org/1999/xhtml" class="graph-node-icon" class:graph-node-icon-orphaned-alloc={node.blockEnvironmentId === 'orphaned'}>
                    <Icon icon="lucide:git-branch" width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE} />
                  </div>
                </foreignObject>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (11 * GRAPH.nodeHeight) / 34} class="graph-label graph-label-alloc" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (24 * GRAPH.nodeHeight) / 34} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
                {#if node.cidr && cidrRange(node.cidr)}
                  <title>{node.cidr} → {cidrRange(node.cidr).start} – {cidrRange(node.cidr).end}</title>
                {/if}
              </g>
            {/each}
          </svg>
        </div>
      </section>
    {/if}
  {/if}

  {#if errorModalMessage}
    <ErrorModal message={errorModalMessage} on:close={() => (errorModalMessage = '')} />
  {/if}
</div>

<style>
  .dashboard {
    padding: 1.5rem 0 2rem;
    max-width: 72rem;
  }
  .loading {
    color: var(--text-muted);
    font-size: 0.9rem;
    padding: 2.5rem 0;
  }
  .stats-section {
    margin-bottom: 2rem;
  }
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 0.875rem;
  }
  .stat-card {
    background: var(--surface);
    border-radius: var(--radius);
    padding: 1.125rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    box-shadow: var(--shadow-sm);
  }
  .stat-card-accent .stat-value {
    color: var(--accent);
  }
  .stat-card-link {
    text-decoration: none;
    color: inherit;
    border: 1px solid transparent;
    transition: border-color 0.15s, box-shadow 0.15s;
  }
  .stat-card-link:hover {
    border-color: var(--border);
    box-shadow: var(--shadow-sm);
  }
  .stat-value {
    font-size: 1.75rem;
    font-weight: 600;
    color: var(--text);
    font-variant-numeric: tabular-nums;
    line-height: 1.2;
  }
  .stat-label {
    font-size: 0.75rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    font-weight: 500;
  }
  .section-title {
    margin: 0 0 0.6rem 0;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
  }
  .orphaned-card {
    margin-bottom: 2rem;
    padding: 1rem 1.25rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
    background: rgba(210, 153, 34, 0.08);
    border: 1px solid rgba(210, 153, 34, 0.35);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
  }
  .orphaned-card-message {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    color: var(--warn);
    font-weight: 500;
  }
  .orphaned-card-icon {
    flex-shrink: 0;
    color: var(--warn);
  }
  .orphaned-card-link {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--accent);
    text-decoration: none;
  }
  .orphaned-card-link:hover {
    text-decoration: underline;
  }
  .graph-section {
    margin-top: 2.5rem;
    margin-bottom: 2.5rem;
  }
  .graph-wrap {
    background: var(--surface);
    border-radius: var(--radius);
    overflow: hidden;
    min-height: 140px;
    box-shadow: var(--shadow-sm);
  }
  .resource-graph {
    display: block;
  }
  .graph-edge {
    stroke: var(--graph-edge-stroke);
    stroke-width: 1.5;
    transition: stroke 0.15s, stroke-width 0.15s;
    pointer-events: none;
  }
  .graph-edge.graph-edge-highlight {
    stroke: var(--accent);
    stroke-width: 2.5;
  }
  .graph-edge.graph-edge-orphaned.graph-edge-highlight {
    stroke: var(--warn);
    stroke-width: 2.5;
  }
  .graph-edge.graph-edge-reserved.graph-edge-highlight {
    stroke: var(--danger);
    stroke-width: 2.5;
  }
  .graph-node-wrap {
    cursor: default;
  }
  .graph-node-wrap[role='button'] {
    cursor: pointer;
  }
  .graph-node-wrap[role='button']:hover .graph-node-block:not(.graph-node-orphaned-block-rect):not(.graph-node-reserved-block-rect),
  .graph-node-wrap[role='button']:hover .graph-node-env:not(.graph-node-orphaned):not(.graph-node-reserved) {
    fill: var(--accent-dim);
    stroke: var(--accent);
  }
  .graph-node-wrap[role='button']:hover .graph-node-orphaned {
    fill: rgba(210, 153, 34, 0.25);
    stroke: var(--warn);
  }
  .graph-node {
    fill: var(--bg);
    stroke: var(--graph-node-stroke);
    stroke-width: 1.5;
    transition: fill 0.15s, stroke 0.15s;
  }
  .graph-node-env:not(.graph-node-orphaned):not(.graph-node-reserved) {
    fill: rgba(88, 166, 255, 0.12);
    stroke: var(--accent);
  }
  .graph-node-orphaned {
    fill: rgba(210, 153, 34, 0.12);
    stroke: var(--warn);
  }
  .graph-node-orphaned-block-rect {
    fill: rgba(210, 153, 34, 0.08);
    stroke: var(--warn);
  }
  .graph-node-wrap[role='button']:hover .graph-node-orphaned-block-rect {
    fill: rgba(210, 153, 34, 0.2);
    stroke: var(--warn);
  }
  .graph-node-reserved {
    fill: rgba(239, 68, 68, 0.12);
    stroke: var(--danger);
  }
  .graph-node-wrap[role='button']:hover .graph-node-reserved {
    fill: rgba(239, 68, 68, 0.25);
    stroke: var(--danger);
  }
  .graph-node-icon-reserved {
    color: var(--danger);
  }
  .graph-node-reserved-block-rect {
    fill: rgba(239, 68, 68, 0.08);
    stroke: var(--danger);
  }
  .graph-node-wrap[role='button']:hover .graph-node-reserved-block-rect {
    fill: rgba(239, 68, 68, 0.2);
    stroke: var(--danger);
  }
  .graph-node-block:not(.graph-node-orphaned-block-rect):not(.graph-node-reserved-block-rect) {
    fill: rgba(88, 166, 255, 0.08);
    stroke: var(--accent);
  }
  .graph-node-alloc {
    fill: rgba(88, 166, 255, 0.08);
    stroke: var(--accent);
  }
  .graph-node-wrap.graph-node-alloc-hovered .graph-node-alloc:not(.graph-node-orphaned-alloc-rect) {
    fill: rgba(88, 166, 255, 0.2);
    stroke-width: 2;
  }
  .graph-node-orphaned-alloc-rect {
    fill: rgba(210, 153, 34, 0.08);
    stroke: var(--warn);
  }
  .graph-node-wrap.graph-node-orphaned-alloc:hover .graph-node-orphaned-alloc-rect {
    fill: rgba(210, 153, 34, 0.2);
    stroke: var(--warn);
  }
  .graph-node-icon-orphaned-alloc {
    color: var(--warn);
  }
  .graph-node-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    color: var(--text-muted);
    opacity: 0.75;
    pointer-events: none;
  }
  .graph-node-icon :global(svg) {
    fill: currentColor;
    stroke: currentColor;
  }
  .graph-node-icon-orphaned {
    color: var(--warn);
    opacity: 0.9;
  }
  .graph-label {
    font-size: 8px;
    fill: var(--text);
    pointer-events: none;
  }
  .graph-label-alloc {
    font-size: 7px;
    fill: var(--text-muted);
  }
  .graph-label.graph-label-cidr {
    font-size: 7px;
    fill: var(--text-muted);
  }
  .chart-section {
    margin-bottom: 2.5rem;
  }
  .chart {
    background: var(--surface);
    border-radius: var(--radius);
    padding: 1rem 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    box-shadow: var(--shadow-sm);
  }
  .chart-row {
    display: grid;
    grid-template-columns: minmax(0, 140px) 1fr minmax(2.5rem, auto);
    align-items: center;
    gap: 0.75rem;
  }
  .chart-label {
    font-size: 0.8125rem;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chart-bar-wrap {
    height: 6px;
    background: var(--bg);
    border-radius: 3px;
    overflow: hidden;
    min-width: 0;
  }
  .chart-bar {
    height: 100%;
    border-radius: 3px;
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
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    text-align: right;
    min-width: 2.5rem;
  }
  .env-section {
    margin-bottom: 0;
  }
  .table-wrap {
    background: var(--surface);
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: var(--shadow-sm);
  }
  .table {
    width: 100%;
    border-collapse: collapse;
  }
  .table th {
    text-align: left;
    padding: 0.6rem 1rem;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
    background: var(--table-header-bg);
    border-bottom: 1px solid var(--table-row-border);
  }
  .table td {
    padding: 0.65rem 1rem;
    border-bottom: 1px solid var(--table-row-border);
  }
  .table tr:hover td {
    background: var(--table-row-hover);
  }
  .table tr:last-child td {
    border-bottom: none;
  }
  .table .name {
    font-weight: 500;
    font-size: 0.9rem;
  }
  .table .id code {
    font-family: var(--font-mono);
    font-size: 0.75rem;
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
    font-size: 0.8125rem;
    cursor: pointer;
    padding: 0.2rem 0;
  }
  .btn-link:hover {
    text-decoration: underline;
  }
  .empty-hint {
    margin-top: 2rem;
    padding: 1.5rem 0;
    color: var(--text-muted);
    font-size: 0.875rem;
  }
  .empty-hint a {
    color: var(--accent);
    text-decoration: none;
  }
  .empty-hint a:hover {
    text-decoration: underline;
  }
</style>
