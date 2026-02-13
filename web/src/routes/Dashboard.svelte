<script>
  import { onMount } from 'svelte'
  import { get } from 'svelte/store'
  import Icon from '@iconify/svelte'
  import ErrorModal from '../lib/ErrorModal.svelte'
  import { cidrRange, parseCidrToBigInt } from '../lib/cidr.js'
  import { formatBlockCount, compareBlockCount, utilizationPercent as utilPct, sumCounts } from '../lib/blockCount.js'
  import { user, selectedOrgForGlobalAdmin, isGlobalAdmin } from '../lib/auth.js'
  import { listEnvironments, listBlocks, listAllocations, listReservedBlocks, listPools, listPoolsByOrganization, exportCSV } from '../lib/api.js'

  const GRAPH_ICON_SIZE = 12
  const GRAPH_ICON_LEFT = 6

  const NIL_UUID = '00000000-0000-0000-0000-000000000000'

  let loading = true
  let error = ''
  let errorModalMessage = ''
  let exporting = false
  let exportingDrawio = false
  const drawioIconDataURIByKey = new Map()
  let environments = []
  let blocks = []
  let allocations = []
  let reservedBlocks = []
  let pools = []

  $: totalIPs = sumCounts(blocks.map((b) => b.total_ips))
  $: usedIPs = sumCounts(blocks.map((b) => b.used_ips))
  $: utilizationPercent = utilPct(totalIPs, usedIPs)
  $: overallUtilizationDisplay =
    compareBlockCount(totalIPs, '0') <= 0 ? '0' : utilizationPercent < 1 && compareBlockCount(usedIPs, '0') > 0 ? '<1' : String(Math.round(utilizationPercent))
  $: orphanedCount = blocks.filter(
    (b) =>
      b.environment_id == null ||
      b.environment_id === '' ||
      String(b.environment_id).toLowerCase() === NIL_UUID
  ).length
  $: blockNamesWithAllocations = (() => {
    const set = new Set()
    for (const a of allocations) {
      const name = (a.block_name || '').trim().toLowerCase()
      if (name) set.add(name)
    }
    return set
  })()
  $: unusedBlockCount = blocks.filter(
    (b) => !blockNamesWithAllocations.has((b.name || '').trim().toLowerCase())
  ).length

  function blockUtilization(block) {
    if (!block) return 0
    return Math.min(100, Math.round(utilPct(block.total_ips, block.used_ips)))
  }

  function utilizationLabel(block) {
    if (!block) return '0%'
    const p = utilPct(block.total_ips, block.used_ips)
    if (compareBlockCount(block.used_ips, '0') > 0 && p < 1) return '<1%'
    return Math.round(p) + '%'
  }

  $: blocksForChart = [...blocks].sort((a, b) => compareBlockCount(b.used_ips, a.used_ips)).slice(0, 12)

  /** Total IP count for a CIDR string (for pool size). Returns string for blockCount compatibility. */
  function totalIPsForCidr(cidr) {
    const p = parseCidrToBigInt(cidr)
    if (!p) return '0'
    const bits = p.version === 6 ? 128 : 32
    const total = 1n << BigInt(bits - p.prefix)
    return total.toString()
  }

  function getEnvironmentName(envId) {
    if (envId == null || envId === '') return null
    const env = environments.find((e) => envIdsMatch(e.id, envId))
    return env?.name ?? null
  }

  /** Per-pool: total IPs in pool (from CIDR), IPs used by blocks in pool (sum of block total_ips), and usage % */
  $: poolUtilization = pools.map((pool) => {
    const poolTotal = totalIPsForCidr(pool.cidr || '')
    const poolBlocks = blocks.filter((b) => b.pool_id && String(b.pool_id).toLowerCase() === String(pool.id).toLowerCase())
    const blockIPs = sumCounts(poolBlocks.map((b) => b.total_ips))
    const pct = utilPct(poolTotal, blockIPs)
    const pctDisplay =
      compareBlockCount(poolTotal, '0') <= 0
        ? 0
        : pct < 1 && compareBlockCount(blockIPs, '0') > 0
          ? '<1'
          : Math.round(pct)
    return {
      pool,
      poolTotal,
      blockIPs,
      pct: typeof pctDisplay === 'string' ? pctDisplay : pct,
      blockCount: poolBlocks.length,
    }
  })

  function envIdsMatch(a, b) {
    if (a == null || b == null) return false
    return String(a).toLowerCase() === String(b).toLowerCase()
  }
  function isOrphanedBlock(block) {
    const id = block.environment_id
    return id == null || id === '' || String(id).toLowerCase() === NIL_UUID
  }

  function navigateToHash(hash) {
    window.location.hash = hash
    // Ensure route parsing also runs when setting the same hash value.
    window.dispatchEvent(new HashChangeEvent('hashchange'))
  }

  function goToEnvironmentBlocks(environmentId, poolId = null) {
    if (!environmentId) return
    const params = new URLSearchParams()
    params.set('environment', String(environmentId))
    if (poolId && String(poolId).indexOf('no-pool-') !== 0) params.set('pool', String(poolId))
    navigateToHash('networks?' + params.toString())
  }

  function goToOrphanedBlocks() {
    navigateToHash('networks?orphaned=1')
  }

  function goToBlock(blockName) {
    if (!blockName) return
    const params = new URLSearchParams()
    params.set('block', String(blockName))
    navigateToHash('networks?' + params.toString())
  }

  function goToAllocation(allocationName) {
    if (!allocationName) return
    const params = new URLSearchParams()
    params.set('allocation', String(allocationName))
    navigateToHash('networks?' + params.toString())
  }

  function goToReservedBlocks() {
    navigateToHash('reserved-blocks')
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
  let graphHovered = null // { type: 'env'|'pool'|'block'|'alloc', id: string } | null
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

    const poolOrder = []
    envRows.forEach((env) => {
      const key = env.id === 'orphaned' ? 'orphaned' : String(env.id).toLowerCase()
      const envBlocks = blocksByEnv.get(key) || []
      if (env.id === 'orphaned' || env.id === 'reserved') {
        poolOrder.push({ pool: { id: `no-pool-${env.id}`, name: '—', environment_id: env.id }, envId: env.id })
      } else {
        const envPools = pools.filter((p) => envIdsMatch(p.environment_id, env.id))
        envPools.forEach((p) => poolOrder.push({ pool: p, envId: env.id }))
        const hasBlocksWithoutPool = envBlocks.some((b) => !b.pool_id)
        if (hasBlocksWithoutPool) poolOrder.push({ pool: { id: `no-pool-${env.id}`, name: '— No pool —', environment_id: env.id }, envId: env.id })
      }
    })

    let blockOrder = []
    poolOrder.forEach(({ pool, envId }) => {
      const key = envId === 'orphaned' ? 'orphaned' : String(envId).toLowerCase()
      const envBlocks = blocksByEnv.get(key) || []
      const inThisPool = (pool.id || '').toString().startsWith('no-pool-')
        ? envBlocks.filter((b) => !b.pool_id)
        : envBlocks.filter((b) => b.pool_id && String(b.pool_id).toLowerCase() === String(pool.id).toLowerCase())
      inThisPool.forEach((b) => blockOrder.push({ block: b, poolId: pool.id, envId }))
    })

    const allocsByBlock = new Map()
    blockOrder.forEach(({ block }) => allocsByBlock.set((block.name || '').toLowerCase(), []))
    allocations.forEach((a) => {
      const key = (a.block_name || '').toLowerCase()
      const list = allocsByBlock.get(key) || []
      list.push(a)
    })
    allocsByBlock.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '')))

    let blockRows = blockOrder.map((item, i) => {
      const blockKey = (item.block.name || '').toLowerCase()
      const allocCount = (allocsByBlock.get(blockKey) || []).length
      return {
        id: item.block.id,
        name: item.block.name,
        cidr: item.block.cidr || '',
        environmentId: item.envId,
        poolId: item.poolId,
        y: GRAPH.padding + i * rowPitch,
        isUnused: allocCount === 0,
      }
    })
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
    poolOrder.forEach(({ pool, envId }) => {
      const items = blockOrder.filter((item) => item.poolId === pool.id && envKey(item.envId) === envKey(envId))
      items.sort((a, b) => (blockAllocCenter.get((a.block.name || '').toLowerCase()) ?? 0) - (blockAllocCenter.get((b.block.name || '').toLowerCase()) ?? 0))
      items.forEach((item) => newBlockOrder.push(item))
    })
    blockOrder = newBlockOrder
    blockRows = blockOrder.map((item, i) => {
      const blockKey = (item.block.name || '').toLowerCase()
      const allocCount = (allocsByBlock.get(blockKey) || []).length
      return {
        id: item.block.id,
        name: item.block.name,
        cidr: item.block.cidr || '',
        environmentId: item.envId,
        poolId: item.poolId,
        y: GRAPH.padding + i * rowPitch,
        isUnused: allocCount === 0,
      }
    })
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

    const poolRows = poolOrder.map(({ pool, envId }) => {
      const poolBlocks = blockRows.filter((br) => br.poolId === pool.id && envKey(br.environmentId) === envKey(envId))
      const y = poolBlocks.length > 0 ? (Math.min(...poolBlocks.map((b) => b.y)) + Math.max(...poolBlocks.map((b) => b.y))) / 2 : GRAPH.padding
      return { id: pool.id, name: pool.name || pool.id || '—', cidr: pool.cidr || '', environmentId: envId, y }
    })
    envRows.forEach((env) => {
      const envPools = poolRows.filter((pr) => envKey(pr.environmentId) === envKey(env.id))
      if (envPools.length > 0) {
        const minY = Math.min(...envPools.map((p) => p.y))
        const maxY = Math.max(...envPools.map((p) => p.y))
        env.y = (minY + maxY) / 2
      }
    })
    envRows
      .slice()
      .sort((a, b) => a.y - b.y)
      .forEach((env, i) => {
        env.y = GRAPH.padding + i * rowPitch
      })
    poolRows.forEach((pr) => {
      const poolBlocks = blockRows.filter((br) => br.poolId === pr.id)
      if (poolBlocks.length > 0) {
        const minY = Math.min(...poolBlocks.map((b) => b.y))
        const maxY = Math.max(...poolBlocks.map((b) => b.y))
        pr.y = (minY + maxY) / 2
      }
    })

    allocRows.forEach((ar) => {
      const br = blockRows.find((b) => (b.name || '').toLowerCase() === (ar.blockName || '').toLowerCase())
      if (br) ar.blockY = br.y
      else ar.blockY = ar.y
    })

    const col1X = GRAPH.padding
    const col2X = GRAPH.padding + GRAPH.nodeWidth + GRAPH.colGap
    const col3X = col2X + GRAPH.nodeWidth + GRAPH.colGap
    const col4X = col3X + GRAPH.nodeWidth + GRAPH.colGap
    envRows.forEach((r) => { r.x = col1X })
    poolRows.forEach((r) => { r.x = col2X })
    blockRows.forEach((r) => { r.x = col3X })
    allocRows.forEach((r) => { r.x = col4X })

    const edges = []
    poolRows.forEach((pr) => {
      const env = envRows.find((e) => envKey(e.id) === envKey(pr.environmentId))
      if (env) {
        const edgeType = env.id === 'orphaned' ? 'orphaned' : env.id === 'reserved' ? 'reserved' : null
        edges.push({ from: { x: col1X + GRAPH.nodeWidth, y: env.y + GRAPH.nodeHeight / 2 }, to: { x: col2X, y: pr.y + GRAPH.nodeHeight / 2 }, envId: env.id, poolId: pr.id, blockId: null, allocId: null, edgeType })
      }
    })
    blockRows.forEach((br) => {
      const pool = poolRows.find((p) => p.id === br.poolId)
      if (pool) {
        const edgeType = br.environmentId === 'orphaned' ? 'orphaned' : br.environmentId === 'reserved' ? 'reserved' : null
        edges.push({ from: { x: col2X + GRAPH.nodeWidth, y: pool.y + GRAPH.nodeHeight / 2 }, to: { x: col3X, y: br.y + GRAPH.nodeHeight / 2 }, envId: null, poolId: pool.id, blockId: br.id, allocId: null, edgeType })
      }
    })
    allocRows.forEach((ar) => {
      const br = blockRows.find((b) => (b.name || '').toLowerCase() === (ar.blockName || '').toLowerCase())
      if (br) {
        const edgeType = br.environmentId === 'orphaned' ? 'orphaned' : br.environmentId === 'reserved' ? 'reserved' : null
        edges.push({ from: { x: col3X + GRAPH.nodeWidth, y: br.y + GRAPH.nodeHeight / 2 }, to: { x: col4X, y: ar.y + GRAPH.nodeHeight / 2 }, envId: null, poolId: null, blockId: br.id, allocId: ar.id, edgeType })
      }
    })

    const width = col4X + GRAPH.nodeWidth + GRAPH.paddingRight
    const maxY = (rows) => (rows.length ? Math.max(...rows.map((r) => r.y)) + GRAPH.nodeHeight + GRAPH.padding : GRAPH.padding * 2)
    const height = Math.max(maxY(envRows), maxY(poolRows), maxY(blockRows), maxY(allocRows))
    return { envRows, poolRows, blockRows, allocRows, edges, width, height }
  })()

  function isEdgeHighlighted(edge) {
    if (!graphHovered) return false
    const id = (x) => (x != null ? String(x).toLowerCase() : '')
    if (graphHovered.type === 'env') return id(edge.envId) === id(graphHovered.id)
    if (graphHovered.type === 'pool') return id(edge.poolId) === id(graphHovered.id)
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

  function xmlEscape(text) {
    return String(text ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&apos;')
      .replaceAll('\n', '&#xa;')
  }

  function drawioIconSVG(iconKey, strokeColor = '#ffffff') {
    if (iconKey === 'layers') {
      return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12.83 2.18 8 3.56a1 1 0 0 1 0 1.83l-8 3.56a2 2 0 0 1-1.66 0l-8-3.56a1 1 0 0 1 0-1.83l8-3.56a2 2 0 0 1 1.66 0Z"/><path d="m2 12 9.17 4.08a2 2 0 0 0 1.66 0L22 12"/><path d="m2 17 9.17 4.08a2 2 0 0 0 1.66 0L22 17"/></svg>`
    }
    if (iconKey === 'network') {
      return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="16" y="16" width="6" height="6" rx="1"/><rect x="2" y="16" width="6" height="6" rx="1"/><rect x="9" y="2" width="6" height="6" rx="1"/><path d="M5 16v-2a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2v2"/><path d="M12 12V8"/></svg>`
    }
    if (iconKey === 'dashboard') {
      return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="9" rx="1"/><rect x="14" y="3" width="7" height="5" rx="1"/><rect x="14" y="12" width="7" height="9" rx="1"/><rect x="3" y="16" width="7" height="5" rx="1"/></svg>`
    }
    if (iconKey === 'ban') {
      return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>`
    }
    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${strokeColor}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/></svg>`
  }

  function getDrawioIconDataURI(iconKey) {
    const key = `${iconKey}:#ffffff`
    if (!drawioIconDataURIByKey.has(key)) {
      const svg = drawioIconSVG(iconKey, '#ffffff')
      drawioIconDataURIByKey.set(key, `data:image/svg+xml,${encodeURIComponent(svg)}`)
    }
    return drawioIconDataURIByKey.get(key)
  }

  function drawioVertexStyle(kind, id) {
    const iconKey =
      kind === 'env'
        ? (id === 'reserved' ? 'ban' : 'layers')
        : kind === 'pool'
          ? (String(id).startsWith('no-pool-') ? (id === 'no-pool-orphaned' ? 'ban' : 'ban') : 'layers')
          : kind === 'block'
            ? (id === 'reserved' ? 'ban' : 'network')
            : 'dashboard'
    const withLogo = `shape=label;image=${getDrawioIconDataURI(iconKey)};imageWidth=22;imageHeight=22;imageAlign=left;imageVerticalAlign=middle;spacingLeft=34;labelPosition=center;verticalLabelPosition=middle;`
    if (kind === 'env') {
      if (id === 'orphaned') {
        return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#cfcfcf;strokeColor=#b7b7b7;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=16;`
      }
      return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#de8f3a;strokeColor=#de8f3a;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=16;`
    }
    if (kind === 'pool') {
      if (String(id).startsWith('no-pool-')) {
        return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#cfcfcf;strokeColor=#b7b7b7;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=15;`
      }
      return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#22c55e;strokeColor=#22c55e;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=15;`
    }
    if (kind === 'block') {
      if (id === 'reserved') {
        return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#de8f3a;strokeColor=#de8f3a;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=15;`
      }
      return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#5f97d3;strokeColor=#5f97d3;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=15;`
    }
    return `${withLogo}rounded=0;whiteSpace=wrap;html=1;fillColor=#6ca2dc;strokeColor=#6ca2dc;fontColor=#ffffff;fontStyle=1;align=left;verticalAlign=middle;shadow=0;spacing=8;spacingTop=8;spacingBottom=8;fontSize=14;`
  }

  function drawioEdgeStyle(edgeType, exitX = 0.5, entryX = 0.5) {
    const base = edgeType === 'orphaned'
      ? '#8f8f8f'
      : edgeType === 'reserved'
        ? '#de8f3a'
        : '#5f97d3'
    return `edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=16;html=1;strokeColor=${base};strokeWidth=2;endArrow=none;exitX=${exitX};exitY=1;entryX=${entryX};entryY=0;`
  }

  function drawioLabel(name, cidr = '') {
    const n = String(name || '').trim() || '—'
    const c = String(cidr || '').trim()
    if (!c) return `<b>${n}</b>`
    return `<b>${n}</b><br/><span style="font-size:12px;opacity:0.92;">${c}</span>`
  }

  function buildDrawioXml() {
    const xPad = 180
    const yPad = 120
    const vertexWidth = 320
    const vertexHeight = 96
    const envGap = 240
    const blockGap = 140
    const allocGap = 120
    const levelGap = 220
    const envY = yPad
    const poolY = envY + vertexHeight + levelGap
    const blockY = poolY + vertexHeight + levelGap
    const allocY = blockY + vertexHeight + levelGap
    const cells = [
      '<mxCell id="0"/>',
      '<mxCell id="1" parent="0"/>',
    ]

    const envs = [...environments].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    const orphanedBlocks = blocks.filter(isOrphanedBlock).sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    const envRows = envs.map((e) => ({ id: String(e.id), name: e.name || String(e.id), kind: 'env' }))
    if (orphanedBlocks.length > 0) envRows.push({ id: 'orphaned', name: 'Orphaned', kind: 'env' })
    if (reservedBlocks.length > 0) envRows.push({ id: 'reserved', name: 'Reserved', kind: 'env' })

    const allocsByBlockName = new Map()
    for (const a of allocations) {
      const key = (a.block_name || '').trim().toLowerCase()
      if (!allocsByBlockName.has(key)) allocsByBlockName.set(key, [])
      allocsByBlockName.get(key).push(a)
    }
    for (const list of allocsByBlockName.values()) {
      list.sort((a, b) => (a.name || '').localeCompare(b.name || ''))
    }

    const envKey = (id) => (id === 'orphaned' ? 'orphaned' : String(id).toLowerCase())
    const envTrees = envRows.map((env) => {
      let envBlocks = []
      if (env.id === 'orphaned') {
        envBlocks = orphanedBlocks.map((b) => ({ id: String(b.id), name: b.name, cidr: b.cidr, envID: 'orphaned', pool_id: null }))
      } else if (env.id === 'reserved') {
        envBlocks = (reservedBlocks || [])
          .map((r) => ({ id: String(r.id), name: (r.name && String(r.name).trim()) || 'Reserved', cidr: r.cidr || '', envID: 'reserved', pool_id: null }))
          .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
      } else {
        envBlocks = blocks
          .filter((b) => envIdsMatch(b.environment_id, env.id))
          .map((b) => ({ id: String(b.id), name: b.name, cidr: b.cidr, envID: env.id, pool_id: b.pool_id }))
          .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
      }
      const poolList = []
      if (env.id === 'orphaned' || env.id === 'reserved') {
        poolList.push({ id: `no-pool-${env.id}`, name: '—', cidr: '', envID: env.id, blocks: envBlocks })
      } else {
        const envPools = pools.filter((p) => envIdsMatch(p.environment_id, env.id))
        envPools.forEach((p) => {
          const poolBlocks = envBlocks.filter((b) => b.pool_id && String(b.pool_id).toLowerCase() === String(p.id).toLowerCase())
          poolList.push({ id: String(p.id), name: p.name || p.id, cidr: p.cidr || '', envID: env.id, blocks: poolBlocks })
        })
        const noPoolBlocks = envBlocks.filter((b) => !b.pool_id)
        if (noPoolBlocks.length > 0) {
          poolList.push({ id: `no-pool-${env.id}`, name: '— No pool —', cidr: '', envID: env.id, blocks: noPoolBlocks })
        }
      }
      const blockTrees = []
      const blockTreesByPool = poolList.map((pool) => {
        return pool.blocks.map((b) => {
          const allocs = env.id === 'reserved'
            ? []
            : (allocsByBlockName.get((b.name || '').trim().toLowerCase()) || [])
                .map((a) => ({ id: String(a.id), name: a.name, cidr: a.cidr, blockID: b.id }))
          const allocCount = allocs.length
          const allocSpan = allocCount > 0 ? allocCount * vertexWidth + (allocCount - 1) * allocGap : 0
          const blockWidth = Math.max(vertexWidth, allocSpan)
          blockTrees.push({ ...b, allocs, blockWidth })
          return { ...b, allocs, blockWidth }
        })
      })
      const totalBlockWidth = blockTrees.reduce((sum, b) => sum + b.blockWidth, 0)
      const spanWidth =
        blockTrees.length > 0
          ? Math.max(vertexWidth, totalBlockWidth + (blockTrees.length - 1) * blockGap)
          : vertexWidth
      return { ...env, pools: poolList, blockTreesByPool, blocks: blockTrees, spanWidth }
    })

    const totalSpan = envTrees.reduce((sum, e) => sum + e.spanWidth, 0)
    const contentWidth = totalSpan + Math.max(0, envTrees.length - 1) * envGap
    const width = Math.max(1900, contentWidth + xPad * 2)
    const height = Math.max(1200, allocY + vertexHeight + yPad)
    cells.push(
      `<mxCell id="bg" value="" style="shape=rectangle;whiteSpace=wrap;html=0;fillColor=#e6e6e6;strokeColor=none;" vertex="1" parent="1" connectable="0"><mxGeometry x="0" y="0" width="${width}" height="${height}" as="geometry"/></mxCell>`,
    )

    let cursorX = xPad
    for (const env of envTrees) {
      const envCenterX = cursorX + env.spanWidth / 2
      const envNodeID = `env-${env.id}`
      cells.push(
        `<mxCell id="${xmlEscape(envNodeID)}" value="${xmlEscape(drawioLabel(env.name))}" style="${drawioVertexStyle('env', env.id)}" vertex="1" parent="1"><mxGeometry x="${Math.round(envCenterX - vertexWidth / 2)}" y="${envY}" width="${vertexWidth}" height="${vertexHeight}" as="geometry"/></mxCell>`,
      )

      let blockCursorX = cursorX
      let poolIndex = 0
      for (const pool of env.pools) {
        const poolBlocks = env.blockTreesByPool[poolIndex] || []
        const poolBlockWidth = poolBlocks.reduce((sum, b) => sum + b.blockWidth + blockGap, -blockGap) || 0
        const poolCenterX = poolBlockWidth > 0 ? blockCursorX + poolBlockWidth / 2 : blockCursorX + vertexWidth / 2
        const poolNodeID = `pool-${pool.id}`
        cells.push(
          `<mxCell id="${xmlEscape(poolNodeID)}" value="${xmlEscape(drawioLabel(pool.name, pool.cidr))}" style="${drawioVertexStyle('pool', pool.id)}" vertex="1" parent="1"><mxGeometry x="${Math.round(poolCenterX - vertexWidth / 2)}" y="${poolY}" width="${vertexWidth}" height="${vertexHeight}" as="geometry"/></mxCell>`,
        )
        const envExitX = env.pools.length <= 1 ? 0.5 : (poolIndex + 1) / (env.pools.length + 1)
        cells.push(
          `<mxCell id="${xmlEscape(`edge-env-${env.id}-pool-${pool.id}`)}" style="${drawioEdgeStyle(env.id === 'orphaned' ? 'orphaned' : env.id === 'reserved' ? 'reserved' : null, envExitX, 0.5)}" edge="1" parent="1" source="${xmlEscape(envNodeID)}" target="${xmlEscape(poolNodeID)}"><mxGeometry relative="1" as="geometry"/></mxCell>`,
        )

        for (let bi = 0; bi < poolBlocks.length; bi += 1) {
          const block = poolBlocks[bi]
          const blockCenterX = blockCursorX + block.blockWidth / 2
          const blockNodeID = `block-${block.id}`
          cells.push(
            `<mxCell id="${xmlEscape(blockNodeID)}" value="${xmlEscape(drawioLabel(block.name, block.cidr))}" style="${drawioVertexStyle('block', block.envID)}" vertex="1" parent="1"><mxGeometry x="${Math.round(blockCenterX - vertexWidth / 2)}" y="${blockY}" width="${vertexWidth}" height="${vertexHeight}" as="geometry"/></mxCell>`,
          )
          const poolExitX = poolBlocks.length <= 1 ? 0.5 : (bi + 1) / (poolBlocks.length + 1)
          cells.push(
            `<mxCell id="${xmlEscape(`edge-pool-${pool.id}-${block.id}`)}" style="${drawioEdgeStyle(block.envID === 'orphaned' ? 'orphaned' : block.envID === 'reserved' ? 'reserved' : null, poolExitX, 0.5)}" edge="1" parent="1" source="${xmlEscape(poolNodeID)}" target="${xmlEscape(blockNodeID)}"><mxGeometry relative="1" as="geometry"/></mxCell>`,
          )

          if (block.allocs.length > 0) {
            let allocCursorX = blockCursorX
            for (let ai = 0; ai < block.allocs.length; ai += 1) {
              const alloc = block.allocs[ai]
              const allocCenterX = allocCursorX + vertexWidth / 2
              const allocNodeID = `alloc-${alloc.id}`
              cells.push(
                `<mxCell id="${xmlEscape(allocNodeID)}" value="${xmlEscape(drawioLabel(alloc.name, alloc.cidr))}" style="${drawioVertexStyle('alloc', block.envID)}" vertex="1" parent="1"><mxGeometry x="${Math.round(allocCenterX - vertexWidth / 2)}" y="${allocY}" width="${vertexWidth}" height="${vertexHeight}" as="geometry"/></mxCell>`,
              )
              const blockExitX = block.allocs.length <= 1 ? 0.5 : (ai + 1) / (block.allocs.length + 1)
              cells.push(
                `<mxCell id="${xmlEscape(`edge-block-${block.id}-${alloc.id}`)}" style="${drawioEdgeStyle(block.envID === 'orphaned' ? 'orphaned' : block.envID === 'reserved' ? 'reserved' : null, blockExitX, 0.5)}" edge="1" parent="1" source="${xmlEscape(blockNodeID)}" target="${xmlEscape(allocNodeID)}"><mxGeometry relative="1" as="geometry"/></mxCell>`,
              )
              allocCursorX += vertexWidth + allocGap
            }
          }

          blockCursorX += block.blockWidth + blockGap
        }
        if (poolBlocks.length === 0) blockCursorX += vertexWidth + blockGap
        poolIndex += 1
      }
      cursorX += env.spanWidth + envGap
    }

    return `<?xml version="1.0" encoding="UTF-8"?>
<mxfile host="app.diagrams.net" modified="${new Date().toISOString()}" agent="ipam-dashboard" version="26.0.0" type="device">
  <diagram id="ipam-resource-graph" name="IPAM Resource Graph">
    <mxGraphModel dx="${width}" dy="${height}" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="${width}" pageHeight="${height}" background="#e6e6e6" math="0" shadow="0">
      <root>
        ${cells.join('\n        ')}
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>`
  }

  async function doExportDrawio() {
    exportingDrawio = true
    errorModalMessage = ''
    try {
      const xml = buildDrawioXml()
      const blob = new Blob([xml], { type: 'application/xml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'ipam-resource-graph.drawio'
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      errorModalMessage = e.message || 'Draw.io export failed'
    } finally {
      exportingDrawio = false
    }
  }

  function listOpts() {
    const u = get(user)
    const opts = {}
    if (isGlobalAdmin(u) && get(selectedOrgForGlobalAdmin)) opts.organization_id = get(selectedOrgForGlobalAdmin)
    return opts
  }

  async function load() {
    loading = true
    error = ''
    const u = get(user)
    const opts = listOpts()
    try {
      const [envsRes, blksRes, allocsRes] = await Promise.all([
        listEnvironments(opts),
        listBlocks(opts),
        listAllocations(opts),
      ])
      environments = envsRes.environments ?? envsRes.Environments ?? []
      blocks = blksRes.blocks ?? []
      allocations = allocsRes.allocations ?? []
      pools = []
      const nilUuid = '00000000-0000-0000-0000-000000000000'
      const orgId = isGlobalAdmin(u) ? get(selectedOrgForGlobalAdmin) : (u?.organization_id ?? '')
      if (orgId && String(orgId).trim() !== '' && String(orgId).toLowerCase() !== nilUuid) {
        try {
          const poolsRes = await listPoolsByOrganization(orgId)
          pools = (poolsRes.pools ?? []).map((p) => ({
            ...p,
            environment_id: p.environment_id ?? p.EnvironmentID,
          }))
        } catch (_) {
          // fall back to per-env fetch below
        }
      }
      if (pools.length === 0) {
        const envIdsToFetch = new Set(
          environments
            .map((e) => e.id ?? e.Id)
            .filter((id) => id != null && String(id).trim() !== '' && String(id).toLowerCase() !== nilUuid)
        )
        const envIdsFromBlocks = (blksRes.blocks ?? [])
          .map((b) => b.environment_id ?? b.EnvironmentID)
          .filter((id) => id != null && String(id).trim() !== '' && String(id).toLowerCase() !== nilUuid)
        envIdsFromBlocks.forEach((id) => envIdsToFetch.add(id))
        if (envIdsToFetch.size > 0) {
          const poolResults = await Promise.all([...envIdsToFetch].map((envId) => listPools(envId)))
          poolResults.forEach((res, i) => {
            const envId = [...envIdsToFetch][i]
            ;(res.pools ?? []).forEach((p) => pools.push({ ...p, environment_id: envId }))
          })
        }
      }
      if (u?.role === 'admin') {
        try {
          const r = await listReservedBlocks(opts)
          reservedBlocks = r.reserved_blocks || []
        } catch (_) {}
      }
    } catch (e) {
      error = e.message || 'Failed to load dashboard'
      errorModalMessage = error + (error.includes('8011') ? '' : ' Ensure the API is running at localhost:8011.')
    } finally {
      loading = false
    }
  }

  $: $user, $selectedOrgForGlobalAdmin, $user && (!isGlobalAdmin($user) || $selectedOrgForGlobalAdmin) && load()
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
          <span class="stat-value">{pools.length}</span>
          <span class="stat-label">Pools</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link" title="Total IPs: {formatBlockCount(totalIPs)}">
          <span class="stat-value">{formatBlockCount(totalIPs)}</span>
          <span class="stat-label">Total IPs</span>
        </a>
        <a href="#networks" class="stat-card stat-card-link" title="Allocated IPs: {formatBlockCount(usedIPs)}">
          <span class="stat-value">{formatBlockCount(usedIPs)}</span>
          <span class="stat-label">Allocated IPs</span>
        </a>
        {#if environments.length > 0}
          <a href="#block-utilization" class="stat-card stat-card-accent stat-card-link" title="Block utilization: used IPs / total IPs across all blocks">
            <span class="stat-value">{overallUtilizationDisplay}%</span>
            <span class="stat-label">Block utilization</span>
          </a>
        {/if}
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

    {#if unusedBlockCount > 0}
      <div class="unused-card">
        <span class="unused-card-message">
          <Icon icon="lucide:package" class="unused-card-icon" width="1.125rem" height="1.125rem" />
          {unusedBlockCount} unused block{unusedBlockCount === 1 ? '' : 's'} — no allocations in these blocks.
        </span>
        <a href="#networks?unused=1" class="unused-card-link">View on Networks →</a>
      </div>
    {/if}

    {#if environments.length > 0}
      <section id="block-utilization" class="chart-section">
        <h2 class="section-title">Block utilization</h2>
        {#if blocks.length > 0}
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
                <span class="chart-pct">
                  {utilizationLabel(block)}
                  <span class="chart-pct-detail" title="Used IPs / total IPs">
                    ({formatBlockCount(block.used_ips)} / {formatBlockCount(block.total_ips)})
                  </span>
                </span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="muted">No blocks yet. Block utilization will appear here when you add network blocks from the <a href="#networks">Networks</a> page or <a href="#network-advisor">Network Advisor</a>.</p>
        {/if}
      </section>

      <section id="pool-utilization" class="chart-section">
        <h2 class="section-title">Pool utilization</h2>
        {#if poolUtilization.length > 0}
          <div class="chart">
            {#each poolUtilization as item}
              {@const pctNum = typeof item.pct === 'string' ? (item.pct === '<1' ? 0.5 : 0) : item.pct}
              <div class="chart-row">
                <span class="chart-label" title="{item.pool.name || item.pool.id} ({item.pool.cidr})">
                  <span class="pool-name">{item.pool.name || item.pool.id || '—'}</span>
                </span>
                <div class="chart-bar-wrap">
                  <div
                    class="chart-bar"
                    class:high={pctNum >= 80}
                    class:mid={pctNum >= 50 && pctNum < 80}
                    style="width: {Math.min(100, pctNum)}%"
                    role="presentation"
                  ></div>
                </div>
                <span class="chart-pct">
                  {item.pct}%
                  <span class="chart-pct-detail" title="Block IPs / pool total">
                    ({formatBlockCount(item.blockIPs)} / {formatBlockCount(item.poolTotal)})
                  </span>
                </span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="muted">No pools yet. Pool utilization will appear here when you add environments with pools from the <a href="#networks">Networks</a> page or <a href="#network-advisor">Network Advisor</a>.</p>
        {/if}
      </section>
    {:else}
      <div class="dashboard-empty">
        <div class="dashboard-empty-card">
          <div class="dashboard-empty-icon">
            <Icon icon="lucide:layers" width="2.5rem" height="2.5rem" />
          </div>
          <h2 class="dashboard-empty-title">No environments yet</h2>
          <p class="dashboard-empty-message">
            Start with the <a href="#network-advisor">Network Advisor</a> to generate a plan and create resources, or create one from the <a href="#environments">Environments</a> page.
          </p>
          <div class="dashboard-empty-actions">
            <a href="#network-advisor" class="btn btn-primary">Network Advisor</a>
            <a href="#environments" class="btn btn-outline">Environments</a>
          </div>
        </div>
      </div>
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
                    <button type="button" class="btn-link" on:click={() => goToEnvironmentBlocks(env.id)}>
                      View blocks →
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {/if}

    {#if environments.length > 0 || blocks.length > 0 || allocations.length > 0 || reservedBlocks.length > 0}
      <section class="graph-section">
        <div class="section-header">
          <h2 class="section-title">Resource graph</h2>
          <button
            class="drawio-export-btn"
            type="button"
            disabled={exportingDrawio}
            on:click={doExportDrawio}
            aria-label="Export resource graph to draw.io"
            title="Export to draw.io"
          >
            <span class="drawio-logo-mark" aria-hidden="true">
              <span class="drawio-node drawio-node-center"></span>
              <span class="drawio-node drawio-node-left"></span>
              <span class="drawio-node drawio-node-right"></span>
            </span>
            <span class="drawio-wordmark" aria-hidden="true">{exportingDrawio ? 'exporting…' : 'draw.io'}</span>
          </button>
        </div>
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
                on:click={() => { if (node.id === 'orphaned') goToOrphanedBlocks(); else if (node.id === 'reserved') goToReservedBlocks(); else goToEnvironmentBlocks(node.id) }}
                on:keydown={(e) => e.key === 'Enter' && (node.id === 'orphaned' ? goToOrphanedBlocks() : node.id === 'reserved' ? goToReservedBlocks() : goToEnvironmentBlocks(node.id))}
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
            <!-- pool nodes -->
            {#each graphData.poolRows as node}
              <g
                class="graph-node-wrap"
                class:graph-node-orphaned-pool={node.environmentId === 'orphaned'}
                class:graph-node-reserved-pool={node.environmentId === 'reserved'}
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'pool', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => node.environmentId !== 'orphaned' && node.environmentId !== 'reserved' && goToEnvironmentBlocks(node.environmentId, node.id)}
                on:keydown={(e) => e.key === 'Enter' && node.environmentId !== 'orphaned' && node.environmentId !== 'reserved' && goToEnvironmentBlocks(node.environmentId, node.id)}
              >
                <rect
                  class="graph-node graph-node-pool"
                  class:graph-node-orphaned-pool-rect={node.environmentId === 'orphaned'}
                  class:graph-node-reserved-pool-rect={node.environmentId === 'reserved'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <foreignObject x={node.x + GRAPH_ICON_LEFT} y={node.y + (GRAPH.nodeHeight - GRAPH_ICON_SIZE) / 2} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE}>
                  <div xmlns="http://www.w3.org/1999/xhtml" class="graph-node-icon" class:graph-node-icon-orphaned-pool={node.environmentId === 'orphaned'} class:graph-node-icon-reserved-pool={node.environmentId === 'reserved'}>
                    <Icon icon={node.environmentId === 'orphaned' ? 'lucide:alert-triangle' : node.environmentId === 'reserved' ? 'lucide:shield-off' : 'lucide:droplets'} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE} />
                  </div>
                </foreignObject>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (11 * GRAPH.nodeHeight) / 34} class="graph-label" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (24 * GRAPH.nodeHeight) / 34} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
              </g>
            {/each}
            <!-- block nodes (name + cidr) -->
            {#each graphData.blockRows as node}
              <g
                class="graph-node-wrap"
                class:graph-node-orphaned-block={node.environmentId === 'orphaned'}
                class:graph-node-reserved-block={node.environmentId === 'reserved'}
                class:graph-node-unused-block={node.isUnused && node.environmentId !== 'orphaned' && node.environmentId !== 'reserved'}
                role="button"
                tabindex="0"
                on:mouseenter={() => (graphHovered = { type: 'block', id: node.id })}
                on:mouseleave={() => (graphHovered = null)}
                on:click={() => node.environmentId === 'reserved' ? goToReservedBlocks() : goToBlock(node.name)}
                on:keydown={(e) => e.key === 'Enter' && (node.environmentId === 'reserved' ? goToReservedBlocks() : goToBlock(node.name))}
              >
                <rect
                  class="graph-node graph-node-block"
                  class:graph-node-orphaned-block-rect={node.environmentId === 'orphaned'}
                  class:graph-node-reserved-block-rect={node.environmentId === 'reserved'}
                  class:graph-node-unused-block-rect={node.isUnused && node.environmentId !== 'orphaned' && node.environmentId !== 'reserved'}
                  x={node.x}
                  y={node.y}
                  width={GRAPH.nodeWidth}
                  height={GRAPH.nodeHeight}
                  rx="4"
                />
                <foreignObject x={node.x + GRAPH_ICON_LEFT} y={node.y + (GRAPH.nodeHeight - GRAPH_ICON_SIZE) / 2} width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE}>
                  <div xmlns="http://www.w3.org/1999/xhtml" class="graph-node-icon" class:graph-node-icon-unused={node.isUnused && node.environmentId !== 'orphaned' && node.environmentId !== 'reserved'}>
                    <Icon icon="lucide:network" width={GRAPH_ICON_SIZE} height={GRAPH_ICON_SIZE} />
                  </div>
                </foreignObject>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (11 * GRAPH.nodeHeight) / 34} class="graph-label" text-anchor="middle">{node.name}</text>
                <text x={node.x + GRAPH_TEXT_CENTER_X_OFFSET} y={node.y + (24 * GRAPH.nodeHeight) / 34} class="graph-label graph-label-cidr" text-anchor="middle">{node.cidr || '—'}</text>
                <title>{[node.isUnused ? 'Unused (no allocations).' : '', node.cidr && cidrRange(node.cidr) ? `${node.cidr} → ${cidrRange(node.cidr).start} – ${cidrRange(node.cidr).end}` : ''].filter(Boolean).join(' ')}</title>
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
                on:click={() => goToAllocation(node.name)}
                on:keydown={(e) => e.key === 'Enter' && goToAllocation(node.name)}
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
    max-width: 100%;
  }
  .loading {
    color: var(--text-muted);
    font-size: 0.9rem;
    padding: 2.5rem 0;
  }
  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.6rem;
  }
  .drawio-export-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    border: 1px solid rgba(222, 143, 58, 0.25);
    background: rgba(222, 143, 58, 0.08);
    color: rgba(222, 143, 58, 0.82);
    border-radius: 6px;
    padding: 0.3rem 0.55rem;
    font-size: 0.78rem;
    font-weight: 600;
    line-height: 1;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s, color 0.15s;
  }
  .drawio-export-btn:hover:not(:disabled) {
    background: rgba(222, 143, 58, 0.14);
    border-color: rgba(222, 143, 58, 0.4);
    color: rgba(222, 143, 58, 0.95);
  }
  .drawio-export-btn:disabled {
    opacity: 0.7;
    cursor: default;
  }
  .drawio-logo-mark {
    position: relative;
    width: 0.95rem;
    height: 0.8rem;
    display: inline-block;
  }
  .drawio-logo-mark::before,
  .drawio-logo-mark::after {
    content: '';
    position: absolute;
    height: 1px;
    background: currentColor;
    opacity: 0.8;
  }
  .drawio-logo-mark::before {
    left: 0.2rem;
    right: 0.2rem;
    top: 0.23rem;
  }
  .drawio-logo-mark::after {
    left: 0.47rem;
    width: 1px;
    top: 0.24rem;
    bottom: 0.12rem;
    height: auto;
    background: currentColor;
  }
  .drawio-node {
    position: absolute;
    width: 0.26rem;
    height: 0.26rem;
    border-radius: 3px;
    background: currentColor;
  }
  .drawio-node-center {
    top: 0;
    left: 50%;
    transform: translateX(-50%);
  }
  .drawio-node-left {
    bottom: 0;
    left: 0.12rem;
  }
  .drawio-node-right {
    bottom: 0;
    right: 0.12rem;
  }
  .drawio-wordmark {
    letter-spacing: 0.01em;
    text-transform: lowercase;
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
    container-type: inline-size;
    background: var(--surface);
    border-radius: var(--radius);
    padding: 1.125rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    box-shadow: var(--shadow-sm);
    min-width: 0;
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
    font-size: clamp(0.75rem, min(1.75rem, 15cqw), 1.75rem);
    font-weight: 600;
    color: var(--text);
    font-variant-numeric: tabular-nums;
    line-height: 1.2;
    min-height: 0;
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
  .orphaned-card-link {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--accent);
    text-decoration: none;
  }
  .orphaned-card-link:hover {
    text-decoration: underline;
  }
  .unused-card {
    margin-bottom: 2rem;
    padding: 1rem 1.25rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
    background: rgba(107, 114, 128, 0.08);
    border: 1px solid rgba(107, 114, 128, 0.35);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
  }
  .unused-card-message {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    color: var(--text-muted);
    font-weight: 500;
  }
  .unused-card-link {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--accent);
    text-decoration: none;
  }
  .unused-card-link:hover {
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
  .graph-node-wrap[role='button']:hover .graph-node-block:not(.graph-node-orphaned-block-rect):not(.graph-node-reserved-block-rect):not(.graph-node-unused-block-rect),
  .graph-node-wrap[role='button']:hover .graph-node-env:not(.graph-node-orphaned):not(.graph-node-reserved),
  .graph-node-wrap[role='button']:hover .graph-node-pool:not(.graph-node-orphaned-pool-rect):not(.graph-node-reserved-pool-rect) {
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
  .graph-node-pool:not(.graph-node-orphaned-pool-rect):not(.graph-node-reserved-pool-rect) {
    fill: rgba(34, 197, 94, 0.1);
    stroke: var(--success, #22c55e);
  }
  .graph-node-orphaned-pool-rect {
    fill: rgba(210, 153, 34, 0.08);
    stroke: var(--warn);
  }
  .graph-node-wrap[role='button']:hover .graph-node-orphaned-pool-rect {
    fill: rgba(210, 153, 34, 0.2);
    stroke: var(--warn);
  }
  .graph-node-reserved-pool-rect {
    fill: rgba(239, 68, 68, 0.08);
    stroke: var(--danger);
  }
  .graph-node-wrap[role='button']:hover .graph-node-reserved-pool-rect {
    fill: rgba(239, 68, 68, 0.2);
    stroke: var(--danger);
  }
  .graph-node-icon-orphaned-pool {
    color: var(--warn);
  }
  .graph-node-icon-reserved-pool {
    color: var(--danger);
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
  .graph-node-unused-block-rect {
    fill: rgba(107, 114, 128, 0.12);
    stroke: var(--text-muted);
  }
  .graph-node-wrap[role='button']:hover .graph-node-unused-block-rect {
    fill: rgba(107, 114, 128, 0.22);
    stroke: var(--text-muted);
  }
  .graph-node-icon-unused {
    color: var(--text-muted);
    opacity: 0.85;
  }
  .graph-node-block:not(.graph-node-orphaned-block-rect):not(.graph-node-reserved-block-rect):not(.graph-node-unused-block-rect) {
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
  .chart-pct-detail {
    font-size: 0.7rem;
    font-weight: 400;
    opacity: 0.85;
    margin-left: 0.25rem;
  }
  .chart-label .pool-name {
    font-weight: 500;
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

  .dashboard-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 28rem;
    padding: 2rem 1rem;
  }
  .dashboard-empty-card {
    text-align: center;
    max-width: 26rem;
    padding: 2.5rem 2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
  }
  .dashboard-empty-icon {
    color: var(--text-muted);
    opacity: 0.7;
    margin-bottom: 1.25rem;
  }
  .dashboard-empty-title {
    margin: 0 0 0.5rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text);
  }
  .dashboard-empty-message {
    margin: 0 0 1.5rem;
    font-size: 0.9375rem;
    line-height: 1.5;
    color: var(--text-muted);
  }
  .dashboard-empty-message a {
    color: var(--accent);
    text-decoration: none;
  }
  .dashboard-empty-message a:hover {
    text-decoration: underline;
  }
  .dashboard-empty-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    justify-content: center;
  }
  .dashboard-empty-actions .btn-outline {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text);
  }
  .dashboard-empty-actions .btn-outline:hover {
    background: var(--bg);
    border-color: var(--text-muted);
  }
</style>
