/**
 * Network sizing helpers used by Network Advisor.
 *
 * Model:
 * - Each environment has N planned subnets ("networks")
 * - Each subnet is sized to fit hosts + growth
 * - Environment allocation is one power-of-2 block that fits all planned subnets
 * - Base CIDR capacity checks must use actual allocated block sizes
 */

const MIN_NETWORKS = 1
const MIN_HOSTS = 8
const HOST_STEP = 8
const MAX_PREFIX = 32
const MIN_PREFIX = 0
const MAX_HOST_SEARCH = 2 ** 30

function toNumber(value, fallback = 0) {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

function toInt(value, fallback = 0) {
  return Math.trunc(toNumber(value, fallback))
}

function clampPrefix(prefix) {
  const p = toInt(prefix, -1)
  if (p < MIN_PREFIX || p > MAX_PREFIX) return null
  return p
}

function clampNetworks(value) {
  return Math.max(MIN_NETWORKS, toInt(value, MIN_NETWORKS))
}

function roundDownToStep(value, step) {
  if (step <= 1) return Math.floor(value)
  return Math.floor(value / step) * step
}

function normalizeHosts(value) {
  const raw = Math.max(MIN_HOSTS, toInt(value, MIN_HOSTS))
  return Math.max(MIN_HOSTS, roundDownToStep(raw, HOST_STEP))
}

function normalizeGrowthPercent(value) {
  return Math.max(0, toNumber(value, 0))
}

function requiredHostsPerSubnet(env) {
  const hosts = normalizeHosts(env?.hostsPerNetwork)
  const growth = normalizeGrowthPercent(env?.growthPercent)
  return Math.max(1, Math.ceil(hosts * (1 + growth / 100)))
}

function prefixForAtLeastTotalIPs(requiredTotal) {
  const needed = Math.max(1, Math.ceil(toNumber(requiredTotal, 1)))
  for (let prefix = MAX_PREFIX; prefix >= MIN_PREFIX; prefix -= 1) {
    if (totalIPsForPrefix(prefix) >= needed) return prefix
  }
  return MIN_PREFIX
}

function subnetPrefixForEnvironment(env) {
  const neededIPs = requiredHostsPerSubnet(env)
  return prefixForAtLeastTotalIPs(neededIPs)
}

function envByIdWithOverride(environments, envId, envOverride = null) {
  if (envOverride) return envOverride
  return (environments || []).find((env) => env?.id === envId) ?? null
}

function blockFitsCapacity(env, capacity) {
  return blockSizeForEnvironment(env) <= Math.max(0, toNumber(capacity, 0))
}

function maxByBinarySearch({ min, max, step, fits }) {
  if (!fits(min)) return min
  let lo = min
  let hi = max
  while (lo < hi) {
    const half = Math.ceil((lo + hi) / 2)
    const mid = step > 1 ? Math.max(min, roundDownToStep(half, step)) : half
    if (mid <= lo) {
      const next = lo + step
      if (next <= hi && fits(next)) lo = next
      else hi = lo
      continue
    }
    if (fits(mid)) lo = mid
    else hi = mid - (step > 1 ? step : 1)
  }
  return Math.max(min, step > 1 ? roundDownToStep(lo, step) : lo)
}

// --- CIDR math ---

export function totalIPsForPrefix(prefix) {
  const normalized = clampPrefix(prefix)
  if (normalized == null) return 0
  return 2 ** (32 - normalized)
}

export function usableIPsForPrefix(prefix) {
  const total = totalIPsForPrefix(prefix)
  return total >= 2 ? total - 2 : 0
}

// --- Per-environment sizing ---

export function networkTotalIPsForEnvironment(env) {
  return totalIPsForPrefix(subnetPrefixForEnvironment(env))
}

export function suggestBlockForEnvironment(env) {
  const networks = clampNetworks(env?.networks)
  const requiredPerSubnet = requiredHostsPerSubnet(env)
  const subnetPrefix = subnetPrefixForEnvironment(env)
  const subnetTotalIPs = totalIPsForPrefix(subnetPrefix)
  const subnetUsableIPs = totalIPsForPrefix(subnetPrefix)

  const requiredBlockIPs = networks * subnetTotalIPs
  const blockPrefix = prefixForAtLeastTotalIPs(requiredBlockIPs)
  const blockUsableIPs = totalIPsForPrefix(blockPrefix)

  return {
    requiredIPs: networks * requiredPerSubnet,
    requiredBlockIPs,
    prefix: blockPrefix,
    usableIPs: blockUsableIPs,
    networkPrefix: subnetPrefix,
    networkUsableIPs: subnetUsableIPs,
    usableIPsInPlannedSubnets: networks * subnetUsableIPs,
    requiredHostsPerNetwork: requiredPerSubnet,
  }
}

export function blockSizeForEnvironment(env) {
  const suggestion = suggestBlockForEnvironment(env)
  return totalIPsForPrefix(suggestion.prefix)
}

export function totalAllocatedBlockIPs(environments) {
  return (environments || []).reduce((sum, env) => sum + blockSizeForEnvironment(env), 0)
}

/**
 * Legacy helper: planned subnet total (not rounded to environment block size).
 * Kept for compatibility with existing UI.
 */
export function environmentPlannedIPs(env) {
  const networks = clampNetworks(env?.networks)
  return networks * networkTotalIPsForEnvironment(env)
}

// --- Capacity and max knobs ---

export function remainingCapacity(
  environments,
  basePrefix,
  envId,
  envOverride = null,
  existingOccupiedIPs = 0,
) {
  const baseTotalIPs = totalIPsForPrefix(basePrefix)
  if (baseTotalIPs <= 0) return 0

  const occupied = Math.max(0, toNumber(existingOccupiedIPs, 0))
  const availableBaseIPs = Math.max(0, baseTotalIPs - occupied)
  const targetEnv = envByIdWithOverride(environments, envId, envOverride)

  const othersUsed = (environments || []).reduce((sum, env) => {
    if (!env) return sum
    if (targetEnv && env?.id === targetEnv.id) return sum
    if (!targetEnv && env?.id === envId) return sum
    return sum + blockSizeForEnvironment(env)
  }, 0)

  return Math.max(0, availableBaseIPs - othersUsed)
}

export function getMaxNetworks(
  environments,
  basePrefix,
  envId,
  envOverride = null,
  existingOccupiedIPs = 0,
) {
  const env = envByIdWithOverride(environments, envId, envOverride)
  if (!env) return MIN_NETWORKS

  const capacity = remainingCapacity(
    environments,
    basePrefix,
    envId,
    envOverride,
    existingOccupiedIPs,
  )
  if (capacity <= 0) return MIN_NETWORKS

  const baseEnv = {
    ...env,
    hostsPerNetwork: normalizeHosts(env.hostsPerNetwork),
    growthPercent: normalizeGrowthPercent(env.growthPercent),
  }

  const fits = (networks) => blockFitsCapacity({ ...baseEnv, networks }, capacity)
  if (!fits(MIN_NETWORKS)) return MIN_NETWORKS

  let hi = Math.max(MIN_NETWORKS, clampNetworks(baseEnv.networks))
  while (fits(hi) && hi < capacity) {
    const next = hi * 2
    if (next <= hi) break
    hi = Math.min(capacity, next)
  }
  if (!fits(hi)) {
    return maxByBinarySearch({ min: MIN_NETWORKS, max: hi, step: 1, fits })
  }
  return hi
}

export function getMaxHostsPerNetwork(
  environments,
  basePrefix,
  envId,
  envOverride = null,
  existingOccupiedIPs = 0,
) {
  const env = envByIdWithOverride(environments, envId, envOverride)
  if (!env) return MIN_HOSTS

  const capacity = remainingCapacity(
    environments,
    basePrefix,
    envId,
    envOverride,
    existingOccupiedIPs,
  )
  if (capacity <= 0) return MIN_HOSTS

  const baseEnv = {
    ...env,
    networks: clampNetworks(env.networks),
    growthPercent: normalizeGrowthPercent(env.growthPercent),
  }

  const fits = (hostsPerNetwork) =>
    blockFitsCapacity({ ...baseEnv, hostsPerNetwork: normalizeHosts(hostsPerNetwork) }, capacity)

  if (!fits(MIN_HOSTS)) return MIN_HOSTS

  let hi = Math.max(MIN_HOSTS, normalizeHosts(env.hostsPerNetwork))
  while (fits(hi) && hi < MAX_HOST_SEARCH) {
    const next = hi * 2
    if (next <= hi) break
    hi = Math.min(MAX_HOST_SEARCH, next)
  }

  if (fits(hi)) return hi

  return maxByBinarySearch({
    min: MIN_HOSTS,
    max: hi,
    step: HOST_STEP,
    fits,
  })
}
