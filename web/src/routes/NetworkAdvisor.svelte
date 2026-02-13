<script>
  import { tick, onMount } from 'svelte'
  import {
    getSubnetInfo,
    parseCidrToBigInt,
    cidrRangeToBigInt,
    formatAddressFromBigInt,
    alignUpBigInt,
  } from '../lib/cidr.js'
  import {
    totalIPsForPrefix,
  } from '../lib/networkSizing.js'
  import { formatBlockCount } from '../lib/blockCount.js'
  import { createBlock, createEnvironment, createReservedBlock, listBlocks, listReservedBlocks } from '../lib/api.js'
  import { user, selectedOrgForGlobalAdmin, isGlobalAdmin } from '../lib/auth.js'

  const RFC1918_OPTIONS = [
    { label: '10.0.0.0/8 (largest private range)', cidr: '10.0.0.0/8' },
    { label: '172.16.0.0/12 (medium private range)', cidr: '172.16.0.0/12' },
    { label: '192.168.0.0/16 (small private range)', cidr: '192.168.0.0/16' },
  ]
  const IPV6_ULA_OPTION = { label: 'fd00::/8 (ULA)', cidr: 'fd00::/8' }

  const CIDR_HINTS = [
    {
      title: 'Large private space',
      cidr: '10.0.0.0/8',
      description: 'Best for long-term growth and many environments.',
    },
    {
      title: 'Balanced private range',
      cidr: '172.16.0.0/12',
      description: 'Good middle ground for most organizations.',
    },
    {
      title: 'Compact private range',
      cidr: '192.168.0.0/16',
      description: 'Useful for smaller deployments and labs.',
    },
    {
      title: 'IPv6 ULA',
      cidr: 'fd00::/8',
      description: 'Unique Local Addresses for private IPv6 networks.',
    },
  ]
  const OTHER_CIDR_HINT = {
    title: 'Other',
    cidr: '',
    description: 'Use a custom CIDR range that matches your network plan.',
  }

  const ENV_TEMPLATES = {
    'dev-test-prod': ['Dev', 'Test', 'Prod'],
    clouds: ['AWS', 'Azure', 'GCP'],
    hybrid: ['Cloud', 'On-Prem'],
  }

  const ENV_HINTS = [
    {
      title: 'Classic SDLC',
      templateKey: 'dev-test-prod',
      description: 'Common for internal app delivery and phased promotion.',
      examples: ['Dev', 'Test', 'Prod'],
    },
    {
      title: 'Cloud-specific',
      templateKey: 'clouds',
      description: 'Useful when environments map to provider ownership.',
      examples: ['AWS', 'Azure', 'GCP'],
    },
    {
      title: 'Hybrid topology',
      templateKey: 'hybrid',
      description: 'Good for mixed cloud and datacenter infrastructure.',
      examples: ['Cloud', 'On-Prem'],
    },
  ]

  let currentStep = 1
  let poolCidr = RFC1918_OPTIONS[0].cidr
  let selectedTemplate = 'dev-test-prod'
  let environments = ENV_TEMPLATES[selectedTemplate].map((name, idx) => ({
    id: `env-${idx}`,
    name,
    networks: 4,
  }))
  let includeReservedBlocks = false
  let reservedBlocksDraft = [{ id: `reserved-${Date.now()}`, name: '', cidr: '', reason: '' }]
  let generating = false
  let generateError = ''
  let generateSuccess = ''
  let showStartOverModal = false
  let existingOccupiedIPs = 0
  let existingOccupiedRanges = []
  let loadingOccupied = false
  let lastOccupiedLoadKey = ''
  let previousBaseCidrForSizing = ''
  let hasInitialMaximized = false

  const MIN_BLOCK_IPS = 8 // /29 for IPv4 -- smallest meaningful network block

  /** Compute per-block sizing for N equal blocks in a pool. */
  function blockSizingForPool(poolRange, networks, version = 4) {
    if (!poolRange) return null
    const bits = version === 6 ? 128 : 32
    const poolSize = poolRange.end - poolRange.start + 1n
    const n = BigInt(Math.max(1, networks))
    const perBlock = poolSize / n
    if (perBlock <= 0n) return null
    let exp = 0n
    while ((1n << (exp + 1n)) <= perBlock) exp++
    const blockIPs = 1n << exp
    const prefix = Number(BigInt(bits) - exp)
    const totalIPs = blockIPs * n
    return { prefix, blockIPs, totalIPs, poolSize }
  }

  /** Max networks that fit in a pool with at least MIN_BLOCK_IPS per block. */
  function maxNetworksInPool(poolRange) {
    if (!poolRange) return 1
    const poolSize = poolRange.end - poolRange.start + 1n
    const max = poolSize / BigInt(MIN_BLOCK_IPS)
    return Math.max(1, max <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(max) : Number.MAX_SAFE_INTEGER)
  }

  /**
   * Place N equal-sized, power-of-2-aligned pools within a base range,
   * skipping over any occupied ranges (existing blocks + reserved blocks).
   */
  function computePoolRangesInFreeSpace(baseRange, occupiedRanges, n) {
    if (!baseRange || n < 1) return []
    const baseSize = baseRange.end - baseRange.start + 1n
    const occupiedIPs = BigInt(occupiedIPsWithinRange(baseRange, occupiedRanges))
    const totalFree = baseSize - occupiedIPs
    if (totalFree < BigInt(n)) return []

    // Target pool size: largest power of 2 where n pools fit in available space
    const targetPerPool = totalFree / BigInt(n)
    let exp = 0n
    while ((1n << (exp + 1n)) <= targetPerPool) exp++
    let poolSize = 1n << exp
    if (poolSize <= 0n) return []

    const bits = baseRange.version === 6 ? 128 : 32
    const prefix = Number(BigInt(bits) - exp)

    // Merge occupied ranges within base for efficient conflict checking
    const clipped = occupiedRanges
      .filter((r) => r.start <= baseRange.end && r.end >= baseRange.start)
      .map((r) => ({
        start: r.start > baseRange.start ? r.start : baseRange.start,
        end: r.end < baseRange.end ? r.end : baseRange.end,
      }))
      .sort((a, b) => (a.start < b.start ? -1 : a.start > b.start ? 1 : 0))
    const mergedOccupied = []
    for (const r of clipped) {
      if (mergedOccupied.length > 0 && r.start <= mergedOccupied[mergedOccupied.length - 1].end + 1n) {
        if (r.end > mergedOccupied[mergedOccupied.length - 1].end) mergedOccupied[mergedOccupied.length - 1].end = r.end
      } else {
        mergedOccupied.push({ start: r.start, end: r.end })
      }
    }

    // Place pools sequentially, aligning to poolSize boundaries and skipping occupied space
    const pools = []
    let cursor = baseRange.start

    for (let i = 0; i < n; i++) {
      let placed = false
      while (cursor + poolSize - 1n <= baseRange.end) {
        const aligned = alignUpBigInt(cursor, poolSize)
        if (aligned + poolSize - 1n > baseRange.end) break
        const candidateEnd = aligned + poolSize - 1n
        const conflict = mergedOccupied.find((r) => r.start <= candidateEnd && aligned <= r.end)
        if (!conflict) {
          const addrStr =
            baseRange.version === 6 ? formatAddressFromBigInt(aligned, 6) : intToAddress(Number(aligned))
          pools.push({
            start: aligned,
            end: candidateEnd,
            prefix,
            version: baseRange.version,
            cidr: `${addrStr}/${prefix}`,
          })
          cursor = aligned + poolSize
          placed = true
          break
        }
        // Skip past the conflict and try next aligned position
        cursor = conflict.end + 1n
      }
      if (!placed) break
    }

    return pools
  }

  function isPrivateCidr(cidr) {
    const parsed = parseCidrToBigInt(cidr)
    if (!parsed) return false
    if (parsed.version === 6) {
      // fd00::/8 (ULA)
      return (parsed.baseBigInt >> 120n) === 0xfdn
    }
    const i = Number(parsed.baseBigInt & 0xffffffffn) >>> 0
    if (((i & 0xff000000) >>> 0) === (0x0a000000 >>> 0)) return true
    if (((i & 0xfff00000) >>> 0) === (0xac100000 >>> 0)) return true
    if (((i & 0xffff0000) >>> 0) === (0xc0a80000 >>> 0)) return true
    return false
  }

  // Slider mapping: 0-SLIDER_STEPS maps to 1-max. Use linear when max is small so the slider isn't stuck at 1; use log scale for large max.
  const SLIDER_STEPS = 1000
  const SLIDER_LINEAR_THRESHOLD = 32 // use linear scale when max <= this so 1,2,3... are evenly spread
  function networksToSlider(networks, max) {
    if (max <= 1) return 0
    const clamped = Math.max(1, Math.min(networks, max))
    if (max <= SLIDER_LINEAR_THRESHOLD) {
      return Math.round(((clamped - 1) / (max - 1)) * SLIDER_STEPS)
    }
    return Math.round((Math.log(clamped) / Math.log(max)) * SLIDER_STEPS)
  }
  function sliderToNetworks(sliderVal, max) {
    if (max <= 1) return 1
    const t = Number(sliderVal) / SLIDER_STEPS
    if (max <= SLIDER_LINEAR_THRESHOLD) {
      return Math.max(1, Math.round(1 + (max - 1) * t))
    }
    return Math.max(1, Math.round(Math.pow(max, t)))
  }

  const advisorVersion = () => parsedStart?.version ?? 4

  /** Clamp each environment's network count to fit in its pool (accounting for occupied space). */
  function clampEnvironmentsToPool(envs) {
    if (envs.length === 0) return envs
    const baseRange = startRangeForPools ?? (parsedStart && cidrToRange(normalizeCidr(poolCidr) || poolCidr)) ?? null
    if (!baseRange) return envs
    const pools = computePoolRangesInFreeSpace(baseRange, allOccupiedRanges, envs.length)
    if (pools.length !== envs.length) return envs
    return envs.map((env, i) => {
      const max = maxNetworksInPool(pools[i])
      return { ...env, networks: Math.max(1, Math.min(Math.max(1, Math.round(Number(env.networks) || 1)), max)) }
    })
  }

  function intToAddress(intValue) {
    return [
      (intValue >>> 24) & 0xff,
      (intValue >>> 16) & 0xff,
      (intValue >>> 8) & 0xff,
      intValue & 0xff,
    ].join('.')
  }

  function normalizeCidr(cidr) {
    const parsed = parseCidrToBigInt(cidr)
    if (!parsed) return ''
    const network = cidr.indexOf('/') >= 0 ? cidr.slice(0, cidr.indexOf('/')).trim() : cidr.trim()
    const info = getSubnetInfo(network, parsed.prefix, parsed.version)
    return info?.cidr ?? ''
  }

  /** Returns { start, end, prefix, cidr, version } where start/end are bigint for range math. */
  function cidrToRange(cidr) {
    return cidrRangeToBigInt(cidr)
  }

  function rangesOverlap(a, b) {
    return a.start <= b.end && b.start <= a.end
  }

  /** Suggested reserved block CIDR within base (e.g. last /16 for /8, last /24 for /16) */
  function suggestedReservedCidrPlaceholder() {
    const baseRange = parsedStart ? cidrToRange(normalizeCidr(poolCidr) || poolCidr) : null
    if (!baseRange) return parsedStart?.version === 6 ? 'e.g. fd00::1:0/64' : 'e.g. 10.255.0.0/16'
    const baseSize = baseRange.end - baseRange.start + 1n
    const suggestedPrefix = baseRange.prefix >= 16 ? 24 : 16
    const bits = baseRange.version === 6 ? 128 : 32
    const suggestedSize = 1n << BigInt(bits - suggestedPrefix)
    if (suggestedSize > baseSize) return baseRange.cidr
    const lastStart = baseRange.end - suggestedSize + 1n
    const addrStr = baseRange.version === 6 ? formatAddressFromBigInt(lastStart, 6) : intToAddress(Number(lastStart))
    return `${addrStr}/${suggestedPrefix}`
  }

  /** Sum of IPs in ranges that overlap startRange, with overlapping ranges merged. Returns number or bigint. */
  function occupiedIPsWithinRange(startRange, ranges) {
    const clipped = ranges
      .filter((r) => rangesOverlap(r, startRange))
      .map((r) => ({
        start: r.start > startRange.start ? r.start : startRange.start,
        end: r.end < startRange.end ? r.end : startRange.end,
      }))
    if (clipped.length === 0) return startRange.version === 6 ? 0n : 0
    const sorted = clipped.slice().sort((a, b) => (a.start < b.start ? -1 : a.start > b.start ? 1 : 0))
    const merged = [{ ...sorted[0] }]
    for (let i = 1; i < sorted.length; i += 1) {
      const last = merged[merged.length - 1]
      if (sorted[i].start <= last.end + 1n) {
        last.end = sorted[i].end > last.end ? sorted[i].end : last.end
      } else {
        merged.push({ ...sorted[i] })
      }
    }
    const sum = merged.reduce((s, r) => s + (r.end - r.start + 1n), 0n)
    return startRange.version === 4 && sum <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(sum) : sum
  }

  async function loadExistingOccupiedInBase() {
    if (!parsedStart || loadingOccupied) return
    const startRange = cidrToRange(normalizeCidr(poolCidr) || poolCidr)
    if (!startRange) return
    loadingOccupied = true
    existingOccupiedIPs = startRange.version === 6 ? 0n : 0
    try {
      const opts = {}
      if (isGlobalAdmin($user) && $selectedOrgForGlobalAdmin) opts.organization_id = $selectedOrgForGlobalAdmin
      const [blocksRes, reservedRes] = await Promise.all([
        listBlocks({ limit: 500, offset: 0, ...opts }),
        $user?.role === 'admin' ? listReservedBlocks(opts) : Promise.resolve({ reserved_blocks: [] }),
      ])
      const ranges = []
      for (const b of blocksRes.blocks || []) {
        const r = cidrToRange(b.cidr)
        if (r && r.version === startRange.version) ranges.push(r)
      }
      for (const r of reservedRes.reserved_blocks || []) {
        const rg = cidrToRange(r.cidr)
        if (rg && rg.version === startRange.version) ranges.push(rg)
      }
      existingOccupiedRanges = ranges
      existingOccupiedIPs = occupiedIPsWithinRange(startRange, ranges)
    } catch {
      existingOccupiedRanges = []
      existingOccupiedIPs = startRange.version === 6 ? 0n : 0
    } finally {
      loadingOccupied = false
    }
  }

  function alignUp(value, blockSize) {
    const remainder = value % blockSize
    return remainder === 0 ? value : value + (blockSize - remainder)
  }

  function findNextFreeCidr(startRange, prefix, occupiedRanges) {
    const bits = startRange.version === 6 ? 128 : 32
    const size = 1n << BigInt(bits - prefix)
    let cursor = alignUpBigInt(startRange.start, size)
    const endMax = startRange.end
    while (cursor + size - 1n <= endMax) {
      const candidate = { start: cursor, end: cursor + size - 1n, prefix: startRange.prefix, version: startRange.version }
      const conflict = occupiedRanges.find((r) => rangesOverlap(candidate, r))
      if (!conflict) {
        const addrStr = startRange.version === 6 ? formatAddressFromBigInt(cursor, 6) : intToAddress(Number(cursor))
        return `${addrStr}/${prefix}`
      }
      cursor = alignUpBigInt(conflict.end + 1n, size)
    }
    return ''
  }

  function makeUniqueName(baseName, usedNames) {
    const base = (baseName || 'Block').trim() || 'Block'
    let candidate = base
    let suffix = 2
    while (usedNames.has(candidate.toLowerCase())) {
      candidate = `${base} ${suffix}`
      suffix += 1
    }
    usedNames.add(candidate.toLowerCase())
    return candidate
  }

  function applyTemplate(templateKey) {
    selectedTemplate = templateKey
    const base = ENV_TEMPLATES[templateKey].map((name, idx) => ({
      id: `env-${Date.now()}-${idx}`,
      name,
      networks: 4,
    }))
    environments = clampEnvironmentsToPool(base)
  }

  function addEnvironment() {
    const newEnv = {
      id: `env-${Date.now()}`,
      name: `env-${environments.length + 1}`,
      networks: 4,
    }
    environments = clampEnvironmentsToPool([...environments, newEnv])
  }

  function removeEnvironment(id) {
    const remaining = environments.filter((e) => e.id !== id)
    environments = clampEnvironmentsToPool(remaining)
  }

  function updateEnvironmentNetworks(id, rawValue) {
    const n = Math.max(1, Math.round(Number(rawValue)))
    if (!Number.isFinite(n)) return
    environments = environments.map((env, i) => {
      if (env.id !== id) return env
      const max = maxNetworksInPool(envPoolRanges?.[i])
      return { ...env, networks: Math.min(n, max) }
    })
  }

  function selectStartCidr(cidr) {
    if (cidr) poolCidr = cidr
  }

  function addReservedDraft() {
    reservedBlocksDraft = [
      ...reservedBlocksDraft,
      { id: `reserved-${Date.now()}-${reservedBlocksDraft.length}`, name: '', cidr: '', reason: '' },
    ]
  }

  function removeReservedDraft(id) {
    reservedBlocksDraft = reservedBlocksDraft.filter((entry) => entry.id !== id)
    if (reservedBlocksDraft.length === 0) {
      reservedBlocksDraft = [{ id: `reserved-${Date.now()}`, name: '', cidr: '', reason: '' }]
    }
  }

  function updateReservedDraft(id, key, value) {
    reservedBlocksDraft = reservedBlocksDraft.map((entry) =>
      entry.id === id ? { ...entry, [key]: value } : entry,
    )
  }

  async function generateAdvisorPlan() {
    if (generating) return
    generateError = ''
    generateSuccess = ''
    const normalizedStart = normalizeCidr(poolCidr)
    const startRange = cidrToRange(normalizedStart)
    if (!startRange) {
      generateError = 'Base CIDR is invalid.'
      return
    }
    const envPlans = environments
      .map((env, i) => ({
        env,
        envName: (env.name || '').trim(),
        poolRange: envPoolRanges[i] ?? null,
      }))
      .filter((item) => item.envName.length > 0)
    if (envPlans.length === 0) {
      generateError = 'Add at least one named environment to generate resources.'
      return
    }

    generating = true
    try {
      const opts = {}
      if (isGlobalAdmin($user) && $selectedOrgForGlobalAdmin) opts.organization_id = $selectedOrgForGlobalAdmin
      const [blocksResponse, reservedResponse] = await Promise.all([
        listBlocks({ limit: 500, offset: 0, ...opts }),
        $user?.role === 'admin' ? listReservedBlocks(opts) : Promise.resolve({ reserved_blocks: [] }),
      ])

      const occupiedRanges = []
      for (const block of blocksResponse.blocks || []) {
        const range = cidrToRange(block.cidr)
        if (range) occupiedRanges.push(range)
      }
      for (const reserved of reservedResponse.reserved_blocks || []) {
        const range = cidrToRange(reserved.cidr)
        if (range) occupiedRanges.push(range)
      }

      const draftsToCreate = includeReservedBlocks
        ? reservedBlocksDraft
            .map((entry) => ({
              ...entry,
              name: (entry.name || '').trim(),
              cidr: normalizeCidr(entry.cidr),
              reason: (entry.reason || '').trim(),
            }))
            .filter((entry) => entry.name || entry.cidr || entry.reason)
        : []

      if (includeReservedBlocks && $user?.role !== 'admin' && draftsToCreate.length > 0) {
        throw new Error('Only admins can create reserved blocks.')
      }

      for (const draft of draftsToCreate) {
        if (!draft.cidr) throw new Error('Reserved blocks require a valid CIDR.')
        const reservedRange = cidrToRange(draft.cidr)
        if (
          !reservedRange ||
          reservedRange.version !== startRange.version ||
          reservedRange.start < startRange.start ||
          reservedRange.end > startRange.end
        ) {
          throw new Error(`Reserved block ${draft.cidr} must be within ${normalizedStart}.`)
        }
        if (occupiedRanges.some((r) => rangesOverlap(r, reservedRange))) {
          throw new Error(`Reserved block ${draft.cidr} overlaps with existing resources.`)
        }
      }

      for (const draft of draftsToCreate) {
        await createReservedBlock({ name: draft.name, cidr: draft.cidr, reason: draft.reason })
        const reservedRange = cidrToRange(draft.cidr)
        if (reservedRange) occupiedRanges.push(reservedRange)
      }

      const usedBlockNames = new Set(
        (blocksResponse.blocks || [])
          .map((block) => (block.name || '').trim().toLowerCase())
          .filter(Boolean),
      )

      const generated = []
      let totalBlocksCreated = 0
      for (const plan of envPlans) {
        const poolRange = plan.poolRange ?? startRange
        const poolCidrForEnv = poolRange?.cidr ?? normalizedStart
        const pools = [{ name: `${plan.envName} pool`, cidr: poolCidrForEnv }]
        const envResponse = await createEnvironment(plan.envName, pools, isGlobalAdmin($user) ? $selectedOrgForGlobalAdmin : null)
        const networksCount = Math.max(1, Number(plan.env.networks) || 0)
        const sizing = blockSizingForPool(poolRange, networksCount, startRange.version)
        if (!sizing) throw new Error(`Cannot compute block size for ${plan.envName}.`)
        const blockPrefix = sizing.prefix

        for (let i = 0; i < networksCount; i += 1) {
          const cidr = findNextFreeCidr(poolRange, blockPrefix, occupiedRanges)
          if (!cidr) {
            throw new Error(
              `No available /${blockPrefix} CIDR remains inside pool ${poolCidrForEnv} for ${plan.envName}. ` +
                `Reduce networks for this environment, or use a larger base CIDR.`,
            )
          }
          const blockName = makeUniqueName(
            networksCount > 1 ? `${plan.envName} Block ${i + 1}` : `${plan.envName} Block`,
            usedBlockNames,
          )
          await createBlock(blockName, cidr, envResponse.id, null, envResponse.initial_pool_id ?? null)
          const blockRange = cidrToRange(cidr)
          if (blockRange) occupiedRanges.push(blockRange)
          generated.push({ envName: plan.envName, cidr })
          totalBlocksCreated += 1
        }
      }

      const reservedCreated = draftsToCreate.length
      const envCount = envPlans.length
      generateSuccess =
        `Created ${envCount} environment${envCount === 1 ? '' : 's'} with ${totalBlocksCreated} network block${totalBlocksCreated === 1 ? '' : 's'}` +
        (reservedCreated > 0 ? ` and ${reservedCreated} reserved block${reservedCreated === 1 ? '' : 's'}.` : '.')
    } catch (err) {
      generateError = err?.message || 'Failed to generate resources from advisor plan.'
    } finally {
      generating = false
    }
  }

  function goNext() {
    if (currentStep < 5 && canContinue) currentStep += 1
  }

  function goBack() {
    if (currentStep > 1) currentStep -= 1
  }

  $: parsedStart = parseCidrToBigInt(poolCidr)
  $: startRangeForPools = parsedStart && (normalizeCidr(poolCidr) || poolCidr)
    ? cidrToRange(normalizeCidr(poolCidr) || poolCidr)
    : null
  /** Pools: base CIDR split into N equal-sized pools placed in free space (avoiding existing blocks & reserved). */
  $: envPoolRanges = (() => {
    if (!startRangeForPools || environments.length === 0) return []
    return computePoolRangesInFreeSpace(startRangeForPools, allOccupiedRanges, environments.length)
  })()
  /** True if the base CIDR can fit all current environments as pools (each gets a pool range). */
  $: poolsFitInBase =
    !startRangeForPools || environments.length === 0 || envPoolRanges.length === environments.length
  /** True if adding one more environment would still fit in the base (so "Add environment" is allowed). */
  $: canAddMoreEnvironments = (() => {
    if (!startRangeForPools || environments.length === 0) return true
    const nextRanges = computePoolRangesInFreeSpace(startRangeForPools, allOccupiedRanges, environments.length + 1)
    return nextRanges.length === environments.length + 1
  })()
  $: startInfo = parsedStart
    ? getSubnetInfo(
        poolCidr.indexOf('/') >= 0 ? poolCidr.slice(0, poolCidr.indexOf('/')).trim() : poolCidr.trim(),
        parsedStart.prefix,
        parsedStart.version,
      )
    : null
  $: startTotalIPs = parsedStart ? totalIPsForPrefix(parsedStart.prefix, parsedStart.version) : parsedStart?.version === 6 ? '0' : 0
  $: startTotalIPsBigInt =
    parsedStart && startTotalIPs != null
      ? typeof startTotalIPs === 'string'
        ? BigInt(startTotalIPs)
        : BigInt(startTotalIPs)
      : 0n
  $: startUsableIPs = startInfo?.usable ?? 0
  $: advisorVer = advisorVersion()
  $: totalNetworkBlocks = environments.reduce((sum, env) => sum + Math.max(1, Number(env.networks) || 1), 0)
  /** Total IPs consumed by all planned blocks (BigInt for v4 and v6 safety). */
  $: totalAllocatedIPs = (() => {
    let sum = 0n
    environments.forEach((env, i) => {
      const sizing = blockSizingForPool(envPoolRanges[i], env.networks, advisorVer)
      if (sizing) sum += sizing.totalIPs
    })
    return sum
  })()
  $: usagePercent =
    startTotalIPsBigInt > 0n
      ? Math.min(100, Number((totalAllocatedIPs * 100n) / startTotalIPsBigInt))
      : 0
  /** Auto-clamp each environment's network count to fit in its pool. */
  $: if (environments.length > 0 && startRangeForPools) {
    const poolRangesForClamp = computePoolRangesInFreeSpace(startRangeForPools, allOccupiedRanges, environments.length)
    if (poolRangesForClamp.length >= environments.length) {
      let changed = false
      const next = environments.map((env, i) => {
        const max = maxNetworksInPool(poolRangesForClamp[i])
        const current = Math.max(1, Math.round(Number(env.networks) || 1))
        if (current <= max) return env
        changed = true
        return { ...env, networks: max }
      })
      if (changed) environments = next
    }
  }
  $: reservedDraftEntries = reservedBlocksDraft
    .map((entry) => ({
      ...entry,
      name: (entry.name || '').trim(),
      cidr: (entry.cidr || '').trim(),
      reason: (entry.reason || '').trim(),
    }))
    .filter((entry) => entry.name || entry.cidr || entry.reason)
  $: hasValidReservedDrafts = reservedDraftEntries.every((entry) => normalizeCidr(entry.cidr))
  /** All occupied ranges: existing blocks/reserved + any draft reserved entries. */
  $: allOccupiedRanges = (() => {
    const ranges = [...existingOccupiedRanges]
    if (includeReservedBlocks) {
      for (const e of reservedDraftEntries) {
        const cidr = normalizeCidr(e.cidr)
        if (cidr) {
          const r = cidrToRange(cidr)
          if (r) ranges.push(r)
        }
      }
    }
    return ranges
  })()
  $: reservedDraftOccupiedIPs = (() => {
    if (!includeReservedBlocks || !parsedStart) return 0
    const startRange = cidrToRange(normalizeCidr(poolCidr) || poolCidr)
    if (!startRange) return 0
    const ranges = reservedDraftEntries
      .filter((e) => normalizeCidr(e.cidr))
      .map((e) => cidrToRange(normalizeCidr(e.cidr)))
      .filter(Boolean)
    if (ranges.length === 0) return 0
    return occupiedIPsWithinRange(startRange, ranges)
  })()
  $: totalOccupiedForSizing =
    parsedStart?.version === 6
      ? BigInt(existingOccupiedIPs) + BigInt(reservedDraftOccupiedIPs)
      : Number(existingOccupiedIPs) + Number(reservedDraftOccupiedIPs)
  $: availableIPsInBase =
    parsedStart?.version === 6
      ? (startTotalIPsBigInt - totalOccupiedForSizing > 0n ? startTotalIPsBigInt - totalOccupiedForSizing : 0n)
      : Math.max(0, Number(startTotalIPs) - totalOccupiedForSizing)
  $: fitsInBaseCidr =
    startTotalIPsBigInt > 0n &&
    totalAllocatedIPs <= (typeof availableIPsInBase === 'bigint' ? availableIPsInBase : BigInt(Math.max(0, Math.floor(Number(availableIPsInBase)))))
  $: normalizedBaseCidr = parsedStart ? (normalizeCidr(poolCidr) || poolCidr) : ''
  $: if (parsedStart && environments.length > 0 && !hasInitialMaximized) {
    hasInitialMaximized = true
    environments = clampEnvironmentsToPool(environments)
  }
  $: if (normalizedBaseCidr && normalizedBaseCidr !== previousBaseCidrForSizing) {
    const hadPrevious = previousBaseCidrForSizing !== ''
    previousBaseCidrForSizing = normalizedBaseCidr
    if (hadPrevious) {
      tick().then(() => {
        environments = environments.map((env) => ({ ...env, networks: 1 }))
        environments = clampEnvironmentsToPool(environments)
      })
    }
  }
  $: occupiedLoadKey =
    currentStep >= 2 && parsedStart
      ? normalizeCidr(poolCidr) || poolCidr
      : ''
  $: if (occupiedLoadKey) {
    if (occupiedLoadKey !== lastOccupiedLoadKey && !loadingOccupied) {
      lastOccupiedLoadKey = occupiedLoadKey
      loadExistingOccupiedInBase()
    }
  } else {
    lastOccupiedLoadKey = ''
  }
  $: hasValidEnvironmentNames = environments.some((e) => (e.name || '').trim().length > 0)
  $: selectedStartHint = (() => {
    const norm = normalizeCidr(poolCidr) || poolCidr.trim()
    if (!norm) return 'other'
    const match = CIDR_HINTS.find((h) => h.cidr && normalizeCidr(h.cidr) === norm)
    return match ? match.cidr : 'other'
  })()
  $: canContinue =
    currentStep === 1 ? !!parsedStart :
    currentStep === 2 ? hasValidEnvironmentNames && poolsFitInBase :
    currentStep === 3 ? (!includeReservedBlocks || hasValidReservedDrafts) :
    true

  onMount(() => {
    const onFocus = () => {
      if (currentStep >= 2 && parsedStart) {
        lastOccupiedLoadKey = ''
      }
    }
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  })
</script>

<div class="page">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Network Advisor</h1>
      <p class="page-desc">Plan from a base CIDR: define environments (each with a pool), then size network blocks.</p>
    </div>
  </header>

  <section class="wizard-progress card">
    <div class="wizard-steps">
      <span class="step" class:active={currentStep === 1} class:done={currentStep > 1}>1. Base CIDR</span>
      <span class="step" class:active={currentStep === 2} class:done={currentStep > 2}>2. Environments & pools</span>
      <span class="step" class:active={currentStep === 3} class:done={currentStep > 3}>3. Reserved blocks</span>
      <span class="step" class:active={currentStep === 4} class:done={currentStep > 4}>4. Network blocks</span>
      <span class="step" class:active={currentStep === 5}>5. Summary</span>
    </div>
  </section>

  <section class="card section">
    {#if currentStep === 1}
      <h2>Step 1: Choose base CIDR</h2>
      <p class="muted">The overall address space. Environments (and their pools) and network blocks will be carved from this range.</p>
      <div class="hint-grid">
        {#each CIDR_HINTS as hint}
          <button
            type="button"
            class="hint-card hint-action"
            class:selected={selectedStartHint === hint.cidr}
            on:click={() => selectStartCidr(hint.cidr)}
            title={`Use ${hint.cidr}`}
          >
            <h3>{hint.title}</h3>
            <p>{hint.description}</p>
            <div class="hint-chips">
              <span class="chip">{hint.cidr}</span>
            </div>
          </button>
        {/each}
        <button
          type="button"
          class="hint-card hint-action"
          class:selected={selectedStartHint === 'other'}
          on:click={() => selectStartCidr(OTHER_CIDR_HINT.cidr)}
          title="Use a custom CIDR"
        >
          <h3>{OTHER_CIDR_HINT.title}</h3>
          <p>{OTHER_CIDR_HINT.description}</p>
          <div class="hint-chips">
            <span class="chip">custom input</span>
          </div>
        </button>
      </div>
      <div class="start-cidr-input">
        <div class="form-row">
          <label for="advisor-base-cidr">Base CIDR</label>
          <input id="advisor-base-cidr" class="input" type="text" bind:value={poolCidr} placeholder="e.g. 10.0.0.0/8 or fd00::/8" />
        </div>
      </div>
      {#if !parsedStart}
        <p class="error">Enter a valid CIDR (e.g. 10.0.0.0/8 or fd00::/8).</p>
      {:else if !isPrivateCidr(poolCidr)}
        <p class="warn">This CIDR is valid, but it is not in a private range (RFC 1918 or IPv6 ULA).</p>
      {:else}
        <p class="ok">Base range set. Estimated usable IPs: {typeof startUsableIPs === 'number' ? startUsableIPs.toLocaleString() : formatBlockCount(startUsableIPs)}.</p>
      {/if}
    {:else if currentStep === 2}
      <h2>Step 2: Define environments (with pools)</h2>
      <p class="muted">Each environment gets one pool -- the base CIDR is split into pools sized by each environment's needs (equal split). Select a template or add environments, then customize names. The pool shown next to each environment is that environment's share. You can add more pools per environment later on the Networks page.</p>

      <div class="hint-grid">
        {#each ENV_HINTS as hint}
          <button
            type="button"
            class="hint-card hint-action"
            class:selected={selectedTemplate === hint.templateKey}
            on:click={() => applyTemplate(hint.templateKey)}
            title={`Use ${hint.title} environments`}
          >
            <h3>{hint.title}</h3>
            <p>{hint.description}</p>
            <div class="hint-chips">
              {#each hint.examples as ex}
                <span class="chip">{ex}</span>
              {/each}
            </div>
          </button>
        {/each}
      </div>

      {#if parsedStart && environments.length > 0 && !poolsFitInBase}
        <p class="warn">Too many environments for this base CIDR -- only {envPoolRanges.length} pool{envPoolRanges.length === 1 ? '' : 's'} fit. Remove environment{environments.length - envPoolRanges.length === 1 ? '' : 's'} or choose a larger base so all pools fit.</p>
      {/if}
      <div class="env-grid">
        {#each environments as env, i}
          {@const poolRange = envPoolRanges[i]}
          <div class="env-pill env-pill-with-pool">
            <input class="env-name" type="text" bind:value={env.name} placeholder="Environment name" />
            {#if poolRange}
              <span class="env-pool-cidr" title="Pool CIDR (sized for this environment's needs)">{poolRange.cidr}</span>
            {:else if parsedStart && environments.length > 0}
              <span class="env-pool-cidr muted">--</span>
            {/if}
            <button type="button" class="btn btn-small btn-danger" on:click={() => removeEnvironment(env.id)} disabled={environments.length <= 1}>Remove</button>
          </div>
        {/each}
      </div>
      <div class="actions">
        <button type="button" class="btn btn-primary btn-small" on:click={addEnvironment} disabled={!canAddMoreEnvironments} title={canAddMoreEnvironments ? '' : 'Base CIDR cannot fit more pools. Use a larger base or remove an environment.'}>Add environment</button>
      </div>
    {:else if currentStep === 3}
      <h2>Step 3: Optional reserved blocks</h2>
      <p class="muted">Reserve CIDR ranges within the base; they will be carved out before environment pools and network blocks.</p>
      <label class="reserve-toggle">
        <input type="checkbox" bind:checked={includeReservedBlocks} />
        <span>Include reserved blocks in generated plan</span>
      </label>
      {#if includeReservedBlocks}
        {#if $user?.role !== 'admin'}
          <p class="warn">Only admins can create reserved blocks. You can continue without adding them.</p>
        {/if}
        <div class="reserve-grid">
          {#each reservedBlocksDraft as entry}
            <div class="reserve-row">
              <input
                class="input reserve-input"
                type="text"
                placeholder="Name (optional)"
                value={entry.name}
                on:input={(e) => updateReservedDraft(entry.id, 'name', e.currentTarget.value)}
              />
              <input
                class="input reserve-input reserve-cidr"
                type="text"
                placeholder="CIDR (e.g. {suggestedReservedCidrPlaceholder()})"
                value={entry.cidr}
                on:input={(e) => updateReservedDraft(entry.id, 'cidr', e.currentTarget.value)}
              />
              <input
                class="input reserve-input"
                type="text"
                placeholder="Reason (optional)"
                value={entry.reason}
                on:input={(e) => updateReservedDraft(entry.id, 'reason', e.currentTarget.value)}
              />
              <button type="button" class="btn btn-small btn-danger" on:click={() => removeReservedDraft(entry.id)}>Remove</button>
            </div>
          {/each}
        </div>
        <div class="actions">
          <button type="button" class="btn btn-primary btn-small" on:click={addReservedDraft}>Add reserved block</button>
        </div>
        {#if !hasValidReservedDrafts}
          <p class="error">Every reserved block entry must include a valid CIDR.</p>
        {/if}
      {/if}
    {:else if currentStep === 4}
      <h2>Step 4: Size network blocks</h2>
      <p class="muted">Choose how many network blocks each environment needs. The base CIDR is split into equal pools -- block size is automatically maximized to fit.</p>
      {#if parsedStart && (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0) && !loadingOccupied}
        <p class="field-note" style="margin-bottom: 0.75rem">
          <strong>{formatBlockCount(totalOccupiedForSizing)}</strong> IPs used by existing blocks.
          <strong>{formatBlockCount(availableIPsInBase)}</strong> available.
        </p>
      {:else if parsedStart && loadingOccupied}
        <p class="field-note" style="margin-bottom: 0.75rem">Checking existing blocks...</p>
      {/if}
      <div class="advisor-grid">
        {#each environments as env, i}
          {@const poolRange = envPoolRanges[i]}
          {@const sliderMax = maxNetworksInPool(poolRange)}
          {@const sizing = blockSizingForPool(poolRange, env.networks, advisorVer)}
          <article class="advisor-env-card">
            <h3>{env.name || 'Environment'}</h3>
            {#if poolRange}
              <p class="env-pool-label muted">Pool: {poolRange.cidr}</p>
            {/if}
            <div class="networks-control">
              <span class="networks-label">Networks</span>
              <input
                type="range"
                class="networks-slider"
                min="0"
                max={SLIDER_STEPS}
                step="1"
                value={networksToSlider(env.networks, sliderMax)}
                on:input={(e) => updateEnvironmentNetworks(env.id, sliderToNetworks(e.currentTarget.value, sliderMax))}
              />
              <input
                type="number"
                class="networks-input"
                min="1"
                value={env.networks}
                on:input={(e) => updateEnvironmentNetworks(env.id, e.currentTarget.value)}
              />
            </div>
            {#if sizing}
              <div class="env-sizing-detail">
                <span>/{sizing.prefix} per block ({formatBlockCount(sizing.blockIPs)} IPs)</span>
                <span>{formatBlockCount(sizing.totalIPs)} IPs total</span>
              </div>
            {/if}
          </article>
        {/each}
      </div>
      <article class="result advisor-result-card">
        <h3>Aggregate sizing</h3>
        <div>Network blocks: <strong>{totalNetworkBlocks}</strong></div>
        <div>Block IPs consumed: <strong>{formatBlockCount(totalAllocatedIPs)}</strong></div>
        {#if parsedStart}
          <div class={fitsInBaseCidr ? 'ok' : 'warn'}>
            {fitsInBaseCidr
              ? `Fits in base ${normalizeCidr(poolCidr) || poolCidr} (${usagePercent}% used)`
              : `Exceeds base -- ${formatBlockCount(totalAllocatedIPs)} needed, ${formatBlockCount(availableIPsInBase)} available`}
          </div>
          <div class="ip-capacity">
            <div class="ip-capacity-head">
              <span>Used: <strong>{formatBlockCount(totalAllocatedIPs)}</strong></span>
              <span>
                Available: <strong>{formatBlockCount(availableIPsInBase)}</strong>
                {#if (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0)}
                  <span class="muted">(of {formatBlockCount(startTotalIPs)} base, minus existing)</span>
                {:else}
                  <span class="muted">(of {formatBlockCount(startTotalIPs)} base)</span>
                {/if}
              </span>
            </div>
            <div class="ip-capacity-bar" role="presentation">
              <div class="ip-capacity-used" style="width: {usagePercent}%"></div>
            </div>
          </div>
        {/if}
      </article>
    {:else}
      <h2>Step 5: Advisor summary</h2>
      <p>Network blocks: <strong>{totalNetworkBlocks}</strong></p>
      <p>Block IPs consumed: <strong>{formatBlockCount(totalAllocatedIPs)}</strong></p>
      {#if parsedStart}
        {#if loadingOccupied}
          <p class="muted">Checking existing blocks...</p>
        {:else}
          <p>
            Base: <strong>{formatBlockCount(startTotalIPs)}</strong> total
            {#if (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0)}
              -- <strong>{formatBlockCount(totalOccupiedForSizing)}</strong> used by existing blocks
            {/if}
            -- <strong>{formatBlockCount(availableIPsInBase)}</strong> available
          </p>
        {/if}
        {#if !loadingOccupied}
          {#if !fitsInBaseCidr}
            <p class="warn">Plan needs {formatBlockCount(totalAllocatedIPs)} IPs but only {formatBlockCount(availableIPsInBase)} available.</p>
          {:else}
            <p class="ok">Plan fits within available capacity ({usagePercent}% used).</p>
          {/if}
        {/if}
      {/if}
      <div class="summary-grid">
        {#each environments as env, i}
          {@const poolRange = envPoolRanges[i]}
          {@const sizing = blockSizingForPool(poolRange, env.networks, advisorVer)}
          <div class="summary-card">
            <div class="summary-title">{env.name || 'Environment'}</div>
            {#if poolRange}
              <div class="muted" style="font-size: 0.8rem;">Pool: {poolRange.cidr}</div>
            {/if}
            <div>Blocks: <strong>{Math.max(1, Number(env.networks) || 1)}</strong></div>
            {#if sizing}
              <div>/{sizing.prefix} each ({formatBlockCount(sizing.blockIPs)} IPs per block)</div>
            {/if}
          </div>
        {/each}
      </div>
      <div class="actions">
        <button type="button" class="btn btn-primary btn-small" on:click={generateAdvisorPlan} disabled={generating}>
          {generating ? 'Generating...' : 'Generate plan'}
        </button>
      </div>
      {#if generateError}
        <p class="error">{generateError}</p>
      {/if}
      {#if generateSuccess}
        <p class="ok">{generateSuccess}</p>
      {/if}
    {/if}
  </section>

  <div class="wizard-actions">
    <button type="button" class="btn btn-small" on:click={goBack} disabled={currentStep === 1}>Back</button>
    {#if currentStep < 5}
      <button type="button" class="btn btn-primary btn-small" on:click={goNext} disabled={!canContinue}>Next</button>
    {:else}
      <button type="button" class="btn btn-outline-danger btn-small" on:click={() => (showStartOverModal = true)}>Start over</button>
    {/if}
  </div>
</div>

{#if showStartOverModal}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal-backdrop" role="presentation" on:click={() => (showStartOverModal = false)}>
    <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-noninteractive-element-interactions -->
    <div class="modal-dialog" role="dialog" aria-modal="true" aria-labelledby="start-over-title" on:click|stopPropagation>
      <h3 id="start-over-title" class="modal-title">Start over?</h3>
      <p class="modal-body">This will discard your current plan and return to Step 1. Any unsaved changes will be lost.</p>
      <div class="modal-actions">
        <button type="button" class="btn btn-small" on:click={() => (showStartOverModal = false)}>Cancel</button>
        <button type="button" class="btn btn-danger btn-small" on:click={() => { showStartOverModal = false; currentStep = 1 }}>Start over</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    padding: 0;
  }
  .section {
    margin-bottom: 1rem;
  }
  .card {
    padding: 1rem 1.25rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  h2 {
    margin: 0 0 0.35rem;
    font-size: 1rem;
  }
  h3 {
    margin: 0;
    font-size: 0.95rem;
  }
  .muted {
    margin: 0 0 0.75rem;
    color: var(--text-muted);
    font-size: 0.88rem;
  }
  .wizard-progress {
    margin-bottom: 1rem;
  }
  .wizard-steps {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    gap: 0.5rem;
  }
  .step {
    padding: 0.45rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-muted);
    font-size: 0.82rem;
    text-align: center;
    background: var(--bg);
  }
  .step.active {
    border-color: var(--accent);
    color: var(--accent);
    background: var(--accent-dim);
    font-weight: 600;
  }
  .step.done {
    border-color: var(--success, #22c55e);
    color: var(--success, #22c55e);
  }
  .form-row {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    max-width: 22rem;
  }
  .form-row label {
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  .input {
    padding: 0.5rem 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-mono);
  }
  .hint-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 0.65rem;
    margin-bottom: 0.8rem;
  }
  .start-cidr-input {
    margin-top: 1rem;
    padding-left: 0.25rem;
  }
  .hint-card {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.7rem;
    background: var(--bg);
    text-align: left;
  }
  .hint-action {
    width: 100%;
    cursor: pointer;
    color: inherit;
    font: inherit;
  }
  .hint-action:hover {
    border-color: var(--accent);
    background: var(--accent-dim);
  }
  .hint-card.selected {
    border-color: var(--accent);
    box-shadow: inset 0 0 0 1px var(--accent);
  }
  .hint-card p {
    margin: 0.35rem 0 0.55rem;
    color: var(--text-muted);
    font-size: 0.83rem;
  }
  .hint-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  .chip {
    font-size: 0.75rem;
    padding: 0.14rem 0.4rem;
    border-radius: 999px;
    border: 1px solid var(--border);
    color: var(--text-muted);
  }
  .env-grid {
    display: grid;
    gap: 0.5rem;
  }
  .env-pill {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .env-pill-with-pool {
    flex-wrap: wrap;
  }
  .env-pool-cidr {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  .env-name {
    flex: 1;
    min-width: 0;
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-sans);
  }
  .actions {
    margin-top: 0.75rem;
  }
  .advisor-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 0.75rem;
  }
  .advisor-env-card {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.75rem;
    background: var(--bg);
  }
  .advisor-env-card h3 {
    margin: 0 0 0.25rem;
    font-size: 0.95rem;
  }
  .advisor-env-card .env-pool-label {
    margin: 0 0 0.5rem;
    font-size: 0.78rem;
  }
  .advisor-env-card input[type="range"] {
    width: 100%;
  }
  .networks-control {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .networks-label {
    font-size: 0.82rem;
    color: var(--text-muted);
    white-space: nowrap;
    flex-shrink: 0;
  }
  .networks-slider {
    flex: 1;
    min-width: 0;
  }
  .networks-input {
    width: 3.5rem;
    padding: 0.3rem 0.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.82rem;
    text-align: center;
    flex-shrink: 0;
    -moz-appearance: textfield;
  }
  .networks-input::-webkit-inner-spin-button,
  .networks-input::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
  .env-sizing-detail {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    font-size: 0.78rem;
    color: var(--text-muted);
  }
  .field-note {
    margin: -0.25rem 0 0.65rem;
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .result {
    margin-top: 0.35rem;
    font-size: 0.85rem;
    display: grid;
    gap: 0.2rem;
  }
  .reserve-toggle {
    display: inline-flex;
    gap: 0.45rem;
    align-items: center;
    margin-bottom: 0.8rem;
    font-size: 0.86rem;
  }
  .reserve-grid {
    display: grid;
    gap: 0.5rem;
  }
  .reserve-row {
    display: grid;
    grid-template-columns: minmax(130px, 1fr) minmax(180px, 1.2fr) minmax(170px, 1fr) auto;
    gap: 0.45rem;
    align-items: center;
  }
  .reserve-input {
    font-family: var(--font-sans);
  }
  .reserve-cidr {
    font-family: var(--font-mono);
  }
  .ip-capacity {
    margin-top: 0.55rem;
  }
  .ip-capacity-head {
    display: flex;
    justify-content: space-between;
    gap: 0.6rem;
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-bottom: 0.25rem;
  }
  .ip-capacity-bar {
    height: 0.45rem;
    border-radius: 999px;
    overflow: hidden;
    background: var(--border);
  }
  .ip-capacity-used {
    height: 100%;
    background: var(--accent);
    border-radius: 999px;
    transition: width 0.2s ease;
  }
  .summary-grid {
    margin-top: 0.75rem;
    display: grid;
    gap: 0.6rem;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }
  .summary-card {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
    padding: 0.65rem;
    font-size: 0.85rem;
  }
  .summary-title {
    font-weight: 600;
    margin-bottom: 0.25rem;
  }
  .wizard-actions {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    margin-top: 0.25rem;
  }
  .ok {
    color: var(--success, #22c55e);
  }
  .warn {
    color: var(--warn, #f59e0b);
  }
  .error {
    color: var(--danger, #ef4444);
  }

  /* Start-over confirmation modal */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(2px);
  }
  .modal-dialog {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.5rem;
    max-width: 420px;
    width: 90%;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  }
  .modal-title {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text);
  }
  .modal-body {
    margin: 0 0 1.25rem;
    font-size: 0.9rem;
    color: var(--text-muted);
    line-height: 1.5;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
</style>
