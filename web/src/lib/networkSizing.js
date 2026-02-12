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
const MAX_PREFIX_IPV6 = 128
const MIN_PREFIX = 0
const MAX_HOST_SEARCH = 2 ** 30
const SAFE_INT = 2 ** 53 - 1

function toNumber(value, fallback = 0) {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

function toInt(value, fallback = 0) {
  return Math.trunc(toNumber(value, fallback))
}

/** Clamp prefix to valid range for IP version. version: 4 or 6; default 4. */
function clampPrefix(prefix, version = 4) {
  const p = toInt(prefix, -1)
  const max = version === 6 ? MAX_PREFIX_IPV6 : MAX_PREFIX
  if (p < MIN_PREFIX || p > max) return null
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

function prefixForAtLeastTotalIPs(requiredTotal, version = 4) {
  const needed = Math.max(1, Math.ceil(toNumber(requiredTotal, 1)))
  const maxP = version === 6 ? MAX_PREFIX_IPV6 : MAX_PREFIX
  for (let prefix = maxP; prefix >= MIN_PREFIX; prefix -= 1) {
    const total = totalIPsForPrefix(prefix, version)
    const n = typeof total === 'string' ? Number(total) : total
    if (Number.isFinite(n) && n >= needed) return prefix
  }
  return MIN_PREFIX
}

function subnetPrefixForEnvironment(env, version = 4) {
  const neededIPs = requiredHostsPerSubnet(env)
  return prefixForAtLeastTotalIPs(neededIPs, version)
}

function envByIdWithOverride(environments, envId, envOverride = null) {
  if (envOverride) return envOverride
  return (environments || []).find((env) => env?.id === envId) ?? null
}

function blockFitsCapacity(env, capacity, version = 4) {
  return blockSizeForEnvironment(env, version) <= Math.max(0, toNumber(capacity, 0))
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

/**
 * Total IPs for a prefix length. version 4 or 6; default 4.
 * For IPv6, returns string when count > 2^53 to avoid precision loss (uses BigInt for large exponents).
 */
export function totalIPsForPrefix(prefix, version = 4) {
  const normalized = clampPrefix(prefix, version)
  if (normalized == null) return version === 6 ? '0' : 0
  const bits = version === 6 ? 128 : 32
  const exp = bits - normalized
  if (version === 6 && exp > 53) {
    return (1n << BigInt(exp)).toString()
  }
  const count = 2 ** exp
  if (version === 6 && count > SAFE_INT) return String(count)
  return count
}

export function usableIPsForPrefix(prefix, version = 4) {
  const total = totalIPsForPrefix(prefix, version)
  if (version === 6) return total
  const n = typeof total === 'number' ? total : Number(total)
  return n >= 2 ? n - 2 : 0
}

// --- Per-environment sizing ---

export function networkTotalIPsForEnvironment(env, version = 4) {
  const t = totalIPsForPrefix(subnetPrefixForEnvironment(env, version), version)
  return typeof t === 'number' ? t : Number(t)
}

export function suggestBlockForEnvironment(env, version = 4) {
  const networks = clampNetworks(env?.networks)
  const requiredPerSubnet = requiredHostsPerSubnet(env)
  const subnetPrefix = subnetPrefixForEnvironment(env, version)
  const subnetTotalIPs = totalIPsForPrefix(subnetPrefix, version)
  const subnetUsableIPs =
    typeof subnetTotalIPs === 'number' ? subnetTotalIPs : Number(subnetTotalIPs)

  const requiredBlockIPs = networks * subnetUsableIPs
  const blockPrefix = prefixForAtLeastTotalIPs(requiredBlockIPs, version)
  const blockUsableIPs = totalIPsForPrefix(blockPrefix, version)
  const blockUsableNum =
    typeof blockUsableIPs === 'number' ? blockUsableIPs : Number(blockUsableIPs)

  return {
    requiredIPs: networks * requiredPerSubnet,
    requiredBlockIPs,
    prefix: blockPrefix,
    usableIPs: blockUsableNum,
    networkPrefix: subnetPrefix,
    networkUsableIPs: subnetUsableIPs,
    usableIPsInPlannedSubnets: networks * subnetUsableIPs,
    requiredHostsPerNetwork: requiredPerSubnet,
  }
}

export function blockSizeForEnvironment(env, version = 4) {
  const suggestion = suggestBlockForEnvironment(env, version)
  const t = totalIPsForPrefix(suggestion.prefix, version)
  return typeof t === 'number' ? t : Number(t)
}

/** Block size as string for IPv6 (avoids Infinity); as number for IPv4. Used for BigInt sums in advisor. */
export function blockSizeForEnvironmentAsString(env, version = 4) {
  const suggestion = suggestBlockForEnvironment(env, version)
  const t = totalIPsForPrefix(suggestion.prefix, version)
  return typeof t === 'string' ? t : String(t)
}

export function totalAllocatedBlockIPs(environments, version = 4) {
  return (environments || []).reduce(
    (sum, env) => sum + blockSizeForEnvironment(env, version),
    0,
  )
}

/** Total allocated block IPs as BigInt for IPv6 (avoids Infinity); as number for IPv4. */
export function totalAllocatedBlockIPsForDisplay(environments, version = 4) {
  if (version === 6) {
    let sum = 0n
    for (const env of environments || []) {
      const s = blockSizeForEnvironmentAsString(env, version)
      try {
        sum += BigInt(s)
      } catch {
        // ignore invalid
      }
    }
    return sum
  }
  return totalAllocatedBlockIPs(environments, version)
}

/**
 * Legacy helper: planned subnet total (not rounded to environment block size).
 * Kept for compatibility with existing UI.
 */
export function environmentPlannedIPs(env, version = 4) {
  const networks = clampNetworks(env?.networks)
  return networks * networkTotalIPsForEnvironment(env, version)
}

// --- Capacity and max knobs ---

export function remainingCapacity(
  environments,
  basePrefix,
  envId,
  envOverride = null,
  existingOccupiedIPs = 0,
  version = 4,
) {
  const baseTotalIPs = totalIPsForPrefix(basePrefix, version)
  const baseNum = typeof baseTotalIPs === 'number' ? baseTotalIPs : Number(baseTotalIPs)
  if (!Number.isFinite(baseNum) || baseNum <= 0) return 0

  const occupied = Math.max(0, toNumber(existingOccupiedIPs, 0))
  const availableBaseIPs = Math.max(0, baseNum - occupied)
  const targetEnv = envByIdWithOverride(environments, envId, envOverride)

  const othersUsed = (environments || []).reduce((sum, env) => {
    if (!env) return sum
    if (targetEnv && env?.id === targetEnv.id) return sum
    if (!targetEnv && env?.id === envId) return sum
    return sum + blockSizeForEnvironment(env, version)
  }, 0)

  return Math.max(0, availableBaseIPs - othersUsed)
}

export function getMaxNetworks(
  environments,
  basePrefix,
  envId,
  envOverride = null,
  existingOccupiedIPs = 0,
  version = 4,
) {
  const env = envByIdWithOverride(environments, envId, envOverride)
  if (!env) return MIN_NETWORKS

  const capacity = remainingCapacity(
    environments,
    basePrefix,
    envId,
    envOverride,
    existingOccupiedIPs,
    version,
  )
  if (capacity <= 0) return MIN_NETWORKS

  const baseEnv = {
    ...env,
    hostsPerNetwork: normalizeHosts(env.hostsPerNetwork),
    growthPercent: normalizeGrowthPercent(env.growthPercent),
  }

  const fits = (networks) =>
    blockFitsCapacity({ ...baseEnv, networks }, capacity, version)
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
  version = 4,
) {
  const env = envByIdWithOverride(environments, envId, envOverride)
  if (!env) return MIN_HOSTS

  const capacity = remainingCapacity(
    environments,
    basePrefix,
    envId,
    envOverride,
    existingOccupiedIPs,
    version,
  )
  if (capacity <= 0) return MIN_HOSTS

  const baseEnv = {
    ...env,
    networks: clampNetworks(env.networks),
    growthPercent: normalizeGrowthPercent(env.growthPercent),
  }

  const fits = (hostsPerNetwork) =>
    blockFitsCapacity(
      { ...baseEnv, hostsPerNetwork: normalizeHosts(hostsPerNetwork) },
      capacity,
      version,
    )

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
