<script>
  import { tick } from 'svelte'
  import {
    getSubnetInfo,
    parseCidrToInt,
    parseCidrToBigInt,
    cidrRangeToBigInt,
    formatAddressFromBigInt,
    alignUpBigInt,
  } from '../lib/cidr.js'
  import {
    totalIPsForPrefix,
    suggestBlockForEnvironment,
    blockSizeForEnvironment,
    totalAllocatedBlockIPsForDisplay,
    getMaxNetworks as sizingGetMaxNetworks,
    getMaxHostsPerNetwork as sizingGetMaxHostsPerNetwork,
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
  let startCidr = RFC1918_OPTIONS[0].cidr
  let selectedTemplate = 'dev-test-prod'
  let environments = ENV_TEMPLATES[selectedTemplate].map((name, idx) => ({
    id: `env-${idx}`,
    name,
    networks: 6,
    hostsPerNetwork: 120,
    growthPercent: 0,
  }))
  let includeReservedBlocks = false
  let reservedBlocksDraft = [{ id: `reserved-${Date.now()}`, name: '', cidr: '', reason: '' }]
  let generating = false
  let generateError = ''
  let generateSuccess = ''
  let showStartOverModal = false
  let existingOccupiedIPs = 0
  let loadingOccupied = false
  let lastOccupiedLoadKey = ''
  let previousBaseCidrForSizing = ''
  let hasInitialMaximized = false

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

  function toHostStep(value) {
    return Math.max(8, Math.round(value / 8) * 8)
  }

  // Logarithmic slider mapping: internal 0–SLIDER_STEPS maps to 1–max via log scale.
  const SLIDER_STEPS = 1000
  function networksToSlider(networks, max) {
    if (max <= 1) return 0
    const clamped = Math.max(1, Math.min(networks, max))
    return Math.round((Math.log(clamped) / Math.log(max)) * SLIDER_STEPS)
  }
  function sliderToNetworks(sliderVal, max) {
    if (max <= 1) return 1
    const t = Number(sliderVal) / SLIDER_STEPS
    return Math.max(1, Math.round(Math.pow(max, t)))
  }

  const advisorVersion = () => parsedStart?.version ?? 4

  /** Set each env to 1 network and max hostsPerNetwork for the base CIDR (iterative: each env's max depends on prior envs) */
  function setEnvironmentsToMaxForBaseCidr(envs) {
    const basePrefix = parsedStart?.prefix ?? 0
    const version = advisorVersion()
    if (!basePrefix || envs.length === 0) return envs
    const result = []
    for (let i = 0; i < envs.length; i += 1) {
      const env = envs[i]
      const envWithOneNetwork = { ...env, networks: 1 }
      const currentEnvs = [...result, { ...env, networks: 1 }, ...envs.slice(i + 1)]
      const maxHosts = sizingGetMaxHostsPerNetwork(
        currentEnvs,
        basePrefix,
        env.id,
        envWithOneNetwork,
        0,
        version,
      )
      result.push({ ...env, networks: 1, hostsPerNetwork: maxHosts })
    }
    return result
  }

  function getMaxHostsPerNetworkForEnvironment(id, envOverride = null) {
    return sizingGetMaxHostsPerNetwork(
      environments,
      parsedStart?.prefix ?? 0,
      id,
      envOverride,
      0,
      advisorVersion(),
    )
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
    const baseRange = parsedStart ? cidrToRange(normalizeCidr(startCidr) || startCidr) : null
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
    const startRange = cidrToRange(normalizeCidr(startCidr) || startCidr)
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
      existingOccupiedIPs = occupiedIPsWithinRange(startRange, ranges)
    } catch {
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
      networks: 6,
      hostsPerNetwork: 120,
      growthPercent: 0,
    }))
    environments = setEnvironmentsToMaxForBaseCidr(base)
  }

  function addEnvironment() {
    const newEnv = {
      id: `env-${Date.now()}`,
      name: `env-${environments.length + 1}`,
      networks: 4,
      hostsPerNetwork: 80,
      growthPercent: 0,
    }
    environments = setEnvironmentsToMaxForBaseCidr([...environments, newEnv])
  }

  function removeEnvironment(id) {
    environments = environments.filter((e) => e.id !== id)
  }

  function updateEnvironmentSizing(id, key, rawValue) {
    const numericValue = Number(rawValue)
    if (!Number.isFinite(numericValue)) return
    environments = environments.map((env) => {
      if (env.id !== id) return env
      if (key === 'networks') {
        const networks = Math.max(1, Math.round(numericValue))
        const updated = { ...env, networks }
        const maxHosts = Math.max(8, getMaxHostsPerNetworkForEnvironment(id, updated))
        return { ...updated, hostsPerNetwork: maxHosts }
      }
      return env
    })
  }

  function getMaxNetworksForEnvironment(id, envOverride = null) {
    return sizingGetMaxNetworks(
      environments,
      parsedStart?.prefix ?? 0,
      id,
      envOverride,
      0,
      advisorVersion(),
    )
  }

  function selectStartCidr(cidr) {
    if (cidr) startCidr = cidr
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
    const normalizedStart = normalizeCidr(startCidr)
    const startRange = cidrToRange(normalizedStart)
    if (!startRange) {
      generateError = 'Starting CIDR is invalid.'
      return
    }
    const envPlans = environments
      .map((env) => ({
        env,
        envName: (env.name || '').trim(),
        suggestion: suggestBlockForEnvironment(env, startRange.version),
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
        const envResponse = await createEnvironment(plan.envName, null, isGlobalAdmin($user) ? $selectedOrgForGlobalAdmin : null)
        const networksCount = Math.max(1, Number(plan.env.networks) || 0)
        const blockPrefix = plan.suggestion.networkPrefix

        for (let i = 0; i < networksCount; i += 1) {
          const cidr = findNextFreeCidr(startRange, blockPrefix, occupiedRanges)
          if (!cidr) {
            throw new Error(
              `No available /${blockPrefix} CIDR remains inside ${normalizedStart} for ${plan.envName}. ` +
                `Existing blocks or reserved ranges may be using the space. Reduce environment sizes or use a larger base CIDR.`,
            )
          }
          const blockName = makeUniqueName(
            networksCount > 1 ? `${plan.envName} Block ${i + 1}` : `${plan.envName} Block`,
            usedBlockNames,
          )
          await createBlock(blockName, cidr, envResponse.id)
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

  $: parsedStart = parseCidrToBigInt(startCidr)
  $: startInfo = parsedStart
    ? getSubnetInfo(
        startCidr.indexOf('/') >= 0 ? startCidr.slice(0, startCidr.indexOf('/')).trim() : startCidr.trim(),
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
  $: environmentSizing = environments.map((env) => ({ env, suggestion: suggestBlockForEnvironment(env, advisorVer) }))
  $: totalRequiredIPs = environmentSizing.reduce((sum, item) => sum + item.suggestion.requiredIPs, 0)
  $: totalNetworkBlocksToCreate = environments.reduce((sum, env) => sum + Math.max(1, Number(env.networks) || 1), 0)
  $: totalRequiredBlockIPs = environmentSizing.reduce((sum, item) => sum + item.suggestion.requiredBlockIPs, 0)
  /** Sum of IPs consumed (N subnets × subnet size each; matches actual generation). BigInt for IPv6 to avoid Infinity. */
  $: totalAllocatedBlockIPsDisplay = (() => {
    const v = parsedStart?.version ?? advisorVer
    const raw = totalAllocatedBlockIPsForDisplay(environments, v)
    // Safety: if IPv6 but helper returns a number (shouldn't), coerce to BigInt for comparisons.
    if (v === 6 && typeof raw !== 'bigint') {
      const n = Number(raw)
      return Number.isFinite(n) ? BigInt(Math.trunc(n)) : 0n
    }
    return raw
  })()
  $: totalAllocatedBlockIPs =
    typeof totalAllocatedBlockIPsDisplay === 'bigint'
      ? totalAllocatedBlockIPsDisplay
      : environmentSizing.reduce((sum, item) => sum + item.suggestion.requiredBlockIPs, 0)
  $: totalPlannedUsableIPs = environmentSizing.reduce((sum, item) => sum + item.suggestion.usableIPsInPlannedSubnets, 0)
  $: aggregateFreeIPs = Math.max(0, totalPlannedUsableIPs - totalRequiredIPs)
  $: aggregateUsedPercent =
    totalPlannedUsableIPs > 0 ? Math.min(100, Math.round((totalRequiredIPs / totalPlannedUsableIPs) * 100)) : 0
  /** Base CIDR block space usage: each env takes a portion; compare allocated block IPs to base total */
  $: usagePercent =
    startTotalIPsBigInt > 0n
      ? Math.min(
          100,
          typeof totalAllocatedBlockIPsDisplay === 'bigint'
            ? Number((totalAllocatedBlockIPsDisplay * 100n) / startTotalIPsBigInt)
            : Number((BigInt(totalAllocatedBlockIPs) * 100n) / startTotalIPsBigInt),
        )
      : 0
  $: reservedDraftEntries = reservedBlocksDraft
    .map((entry) => ({
      ...entry,
      name: (entry.name || '').trim(),
      cidr: (entry.cidr || '').trim(),
      reason: (entry.reason || '').trim(),
    }))
    .filter((entry) => entry.name || entry.cidr || entry.reason)
  $: hasValidReservedDrafts = reservedDraftEntries.every((entry) => normalizeCidr(entry.cidr))
  $: reservedDraftOccupiedIPs = (() => {
    if (!includeReservedBlocks || !parsedStart) return 0
    const startRange = cidrToRange(normalizeCidr(startCidr) || startCidr)
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
    (parsedStart?.version === 6 &&
      startTotalIPsBigInt > 0n &&
      typeof totalAllocatedBlockIPsDisplay === 'bigint' &&
      totalAllocatedBlockIPsDisplay <= availableIPsInBase) ||
    (parsedStart?.version === 4 && Number(startTotalIPs) > 0 && totalAllocatedBlockIPs <= availableIPsInBase)
  $: normalizedBaseCidr = parsedStart ? (normalizeCidr(startCidr) || startCidr) : ''
  $: if (parsedStart && environments.length > 0 && !hasInitialMaximized) {
    hasInitialMaximized = true
    environments = setEnvironmentsToMaxForBaseCidr(environments)
  }
  $: if (normalizedBaseCidr && normalizedBaseCidr !== previousBaseCidrForSizing) {
    const hadPrevious = previousBaseCidrForSizing !== ''
    previousBaseCidrForSizing = normalizedBaseCidr
    if (hadPrevious) {
      tick().then(() => {
        environments = environments.map((env) => ({
          ...env,
          networks: 1,
          hostsPerNetwork: 8,
          growthPercent: 0,
        }))
        environments = setEnvironmentsToMaxForBaseCidr(environments)
      })
    }
  }
  $: occupiedLoadKey =
    (currentStep === 4 || currentStep === 5) && parsedStart
      ? `${currentStep}:${normalizeCidr(startCidr) || startCidr}`
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
    const norm = normalizeCidr(startCidr) || startCidr.trim()
    if (!norm) return 'other'
    const match = CIDR_HINTS.find((h) => h.cidr && normalizeCidr(h.cidr) === norm)
    return match ? match.cidr : 'other'
  })()
  $: canContinue =
    currentStep === 1 ? !!parsedStart :
    currentStep === 2 ? hasValidEnvironmentNames :
    currentStep === 3 ? (!includeReservedBlocks || hasValidReservedDrafts) :
    true
</script>

<div class="page">
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">Network Advisor</h1>
      <p class="page-desc">Step-by-step wizard to plan CIDR strategy, environments, and block sizing.</p>
    </div>
  </header>

  <section class="wizard-progress card">
    <div class="wizard-steps">
      <span class="step" class:active={currentStep === 1} class:done={currentStep > 1}>1. Base CIDR</span>
      <span class="step" class:active={currentStep === 2} class:done={currentStep > 2}>2. Environments</span>
      <span class="step" class:active={currentStep === 3} class:done={currentStep > 3}>3. Reserve blocks</span>
      <span class="step" class:active={currentStep === 4} class:done={currentStep > 4}>4. Block sizing</span>
      <span class="step" class:active={currentStep === 5}>5. Summary</span>
    </div>
  </section>

  <section class="card section">
    {#if currentStep === 1}
      <h2>Step 1: Choose a starting CIDR</h2>
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
          <label for="advisor-start-cidr">Starting CIDR</label>
          <input id="advisor-start-cidr" class="input" type="text" bind:value={startCidr} placeholder="e.g. 10.0.0.0/8 or fd00::/8" />
        </div>
      </div>
      {#if !parsedStart}
        <p class="error">Enter a valid CIDR (e.g. 10.0.0.0/8 or fd00::/8).</p>
      {:else if !isPrivateCidr(startCidr)}
        <p class="warn">This CIDR is valid, but it is not in a private range (RFC 1918 or IPv6 ULA).</p>
      {:else}
        <p class="ok">Private CIDR detected. Estimated usable IPs: {typeof startUsableIPs === 'number' ? startUsableIPs.toLocaleString() : formatBlockCount(startUsableIPs)}.</p>
      {/if}
    {:else if currentStep === 2}
      <h2>Step 2: Define environments</h2>
      <p class="muted">Select a suggested model to populate environments, then customize names.</p>

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

      <div class="env-grid">
        {#each environments as env}
          <div class="env-pill">
            <input class="env-name" type="text" bind:value={env.name} />
            <button type="button" class="btn btn-small btn-danger" on:click={() => removeEnvironment(env.id)} disabled={environments.length <= 1}>Remove</button>
          </div>
        {/each}
      </div>
      <div class="actions">
        <button type="button" class="btn btn-primary btn-small" on:click={addEnvironment}>Add environment</button>
      </div>
    {:else if currentStep === 3}
      <h2>Step 3: Optional reserved blocks</h2>
      <p class="muted">Reserve CIDR ranges first; they will be carved out before environment blocks.</p>
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
      <p class="muted">Estimate hosts and number of networks in each environment. Reserved blocks are deducted first.</p>
      {#if parsedStart && (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0) && !loadingOccupied}
        <p class="field-note" style="margin-bottom: 0.75rem">
          <strong>{formatBlockCount(totalOccupiedForSizing)}</strong> IPs in base CIDR are used by existing blocks
          {#if (typeof reservedDraftOccupiedIPs === 'bigint' ? reservedDraftOccupiedIPs > 0n : reservedDraftOccupiedIPs > 0)}
            and reserved blocks ({formatBlockCount(reservedDraftOccupiedIPs)})
          {/if}
          . Sliders limited to <strong>{formatBlockCount(availableIPsInBase)}</strong> available IPs.
        </p>
      {:else if parsedStart && loadingOccupied}
        <p class="field-note" style="margin-bottom: 0.75rem">Checking existing blocks…</p>
      {/if}
      <div class="advisor-grid">
        {#each environmentSizing as item}
          {@const env = item.env}
          {@const baseTotalNum = typeof startTotalIPs === 'string' ? Number(startTotalIPs) : startTotalIPs}
          {@const subnetSize = item.suggestion.networkUsableIPs}
          {@const sliderMax = Math.max(1, subnetSize > 0 && Number.isFinite(baseTotalNum) ? Math.min(1e9, Math.floor(baseTotalNum / subnetSize)) : 1)}
          {@const ipsPerNetwork = item.suggestion.networkUsableIPs}
          {@const totalIPs = item.suggestion.requiredBlockIPs}
          <article class="advisor-env-card" class:exceeded={!fitsInBaseCidr}>
            <h3>{env.name || 'Environment'}</h3>
            <div class="networks-control">
              <span class="networks-label">Networks</span>
              <input
                type="range"
                class="networks-slider"
                min="0"
                max={SLIDER_STEPS}
                step="1"
                value={networksToSlider(env.networks, sliderMax)}
                on:input={(e) => updateEnvironmentSizing(env.id, 'networks', sliderToNetworks(e.currentTarget.value, sliderMax))}
              />
              <input
                type="number"
                class="networks-input"
                min="1"
                value={env.networks}
                on:input={(e) => updateEnvironmentSizing(env.id, 'networks', e.currentTarget.value)}
              />
            </div>
            <div class="env-sizing-detail">
              <span>{typeof ipsPerNetwork === 'number' && Number.isFinite(ipsPerNetwork) ? ipsPerNetwork.toLocaleString() : formatBlockCount(ipsPerNetwork)} IPs per network</span>
              <span>{typeof totalIPs === 'number' && Number.isFinite(totalIPs) ? totalIPs.toLocaleString() : formatBlockCount(totalIPs)} IPs total</span>
            </div>
          </article>
        {/each}
      </div>
      <article class="result advisor-result-card">
        <h3>Aggregate sizing result</h3>
        <div>Required host IPs (usable): <strong>{totalRequiredIPs.toLocaleString()}</strong></div>
        <div>
          Total block IPs consumed (from base CIDR): <strong>{formatBlockCount(totalAllocatedBlockIPsDisplay)}</strong>
        </div>
        {#if parsedStart}
          <div class={fitsInBaseCidr ? 'ok' : 'warn'}>
            {fitsInBaseCidr
              ? `Fits in ${normalizeCidr(startCidr) || startCidr} (${usagePercent}% of base CIDR used)`
              : `Exceeds base CIDR — need ${formatBlockCount(totalAllocatedBlockIPsDisplay)} IPs, only ${formatBlockCount(availableIPsInBase)} available (existing & reserved deducted)`}
          </div>
        {/if}
        <div>Planned subnet capacity (all environments): <strong>{typeof totalPlannedUsableIPs === 'number' && Number.isFinite(totalPlannedUsableIPs) ? totalPlannedUsableIPs.toLocaleString() : formatBlockCount(totalPlannedUsableIPs)}</strong></div>
        <div class="ip-capacity">
          <div class="ip-capacity-head">
            <span>Block IPs used: <strong>{formatBlockCount(totalAllocatedBlockIPsDisplay)}</strong></span>
            <span>
              Base CIDR total: <strong>{formatBlockCount(startTotalIPs)}</strong>
              {#if (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0)}
                <span class="muted">(deducts existing & reserved blocks → {formatBlockCount(availableIPsInBase)} available)</span>
              {/if}
            </span>
          </div>
          <div class="ip-capacity-bar" role="presentation">
            <div class="ip-capacity-used" style="width: {usagePercent}%"></div>
          </div>
        </div>
      </article>
    {:else if currentStep === 4}
      <h2>Step 4: Optional reserved blocks</h2>
      <p class="muted">Optionally reserve CIDR ranges before generating resources from this plan.</p>
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
    {:else}
      <h2>Step 5: Advisor summary</h2>
      <p>Network blocks to be created: <strong>{totalNetworkBlocksToCreate.toLocaleString()}</strong></p>
      <p>Required host IPs across environments: <strong>{totalRequiredIPs.toLocaleString()}</strong></p>
      <p>Total block space consumed from base CIDR: <strong>{formatBlockCount(totalAllocatedBlockIPsDisplay)}</strong></p>
      {#if parsedStart}
        {#if loadingOccupied}
          <p class="muted">Checking existing blocks in base CIDR…</p>
        {:else}
          <p>
            Base CIDR: <strong>{formatBlockCount(startTotalIPs)}</strong> total IPs
            {#if (typeof totalOccupiedForSizing === 'bigint' ? totalOccupiedForSizing > 0n : totalOccupiedForSizing > 0)}
              — <strong>{formatBlockCount(totalOccupiedForSizing)}</strong> used
              ({#if (typeof existingOccupiedIPs === 'bigint' ? existingOccupiedIPs > 0n : existingOccupiedIPs > 0)}{formatBlockCount(existingOccupiedIPs)} existing{/if}
              {#if (typeof existingOccupiedIPs === 'bigint' ? existingOccupiedIPs > 0n : existingOccupiedIPs > 0) && (typeof reservedDraftOccupiedIPs === 'bigint' ? reservedDraftOccupiedIPs > 0n : reservedDraftOccupiedIPs > 0)}, {/if}
              {#if (typeof reservedDraftOccupiedIPs === 'bigint' ? reservedDraftOccupiedIPs > 0n : reservedDraftOccupiedIPs > 0)}{formatBlockCount(reservedDraftOccupiedIPs)} reserved{/if})
            {/if}
          </p>
          <p>
            Available for plan: <strong>{formatBlockCount(availableIPsInBase)}</strong> IPs
          </p>
        {/if}
        {#if !loadingOccupied}
          {#if !fitsInBaseCidr}
            <p class="warn">Plan needs {formatBlockCount(totalAllocatedBlockIPsDisplay)} IPs but only {formatBlockCount(availableIPsInBase)} available. Reduce environment sizes or choose a larger base CIDR.</p>
          {:else}
            <p class="ok">Current plan fits within available capacity.</p>
          {/if}
        {/if}
      {/if}
      <div class="summary-grid">
        {#each environments as env}
          {@const suggestion = suggestBlockForEnvironment(env, advisorVer)}
          <div class="summary-card">
            <div class="summary-title">{env.name || 'Environment'}</div>
            <div>Network blocks to be created: <strong>{Math.max(1, Number(env.networks) || 1).toLocaleString()}</strong></div>
            <div>Capacity: {typeof suggestion.usableIPs === 'number' ? suggestion.usableIPs.toLocaleString() : formatBlockCount(suggestion.usableIPs)} usable IPs</div>
          </div>
        {/each}
      </div>
      <div class="actions">
        <button type="button" class="btn btn-primary btn-small" on:click={generateAdvisorPlan} disabled={generating}>
          {generating ? 'Generating…' : 'Generate resources from plan'}
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
  .advisor-env-card.exceeded {
    border-color: var(--warn, #f59e0b);
  }
  .advisor-env-card h3 {
    margin: 0 0 0.65rem;
    font-size: 0.95rem;
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
