/**
 * Pool usage: used = child pool IPs + block IPs.
 * Shared by Dashboard, Networks, and Environments.
 */
import { parseCidrToBigInt } from './cidr.js'
import { sumCounts, utilizationPercent } from './blockCount.js'

/** Total IP count for a CIDR string. Returns string for blockCount compatibility. */
export function totalIPsForCidr(cidr) {
  const p = parseCidrToBigInt(cidr)
  if (!p) return '0'
  const bits = p.version === 6 ? 128 : 32
  const total = 1n << BigInt(bits - p.prefix)
  return total.toString()
}

/**
 * Used IPs for a pool = IPs in direct child pools + IPs in blocks in this pool.
 * @param {object} pool - pool with id, cidr
 * @param {object[]} allPools - all pools (must include children)
 * @param {object[]} blocks - blocks with pool_id, total_ips
 */
export function poolUsedIPs(pool, allPools, blocks) {
  const childPools = (allPools || []).filter(
    (p) => p.parent_pool_id != null && String(p.parent_pool_id).toLowerCase() === String(pool.id).toLowerCase()
  )
  const childPoolIPs = sumCounts(childPools.map((p) => totalIPsForCidr(p.cidr || '')))
  const poolBlocks = (blocks || []).filter(
    (b) => b.pool_id && String(b.pool_id).toLowerCase() === String(pool.id).toLowerCase()
  )
  const blockIPs = sumCounts(poolBlocks.map((b) => b.total_ips))
  return sumCounts([childPoolIPs, blockIPs])
}

/** Utilization percent (0–100) for a pool given allPools and blocks. */
export function poolUtilizationPercent(pool, allPools, blocks) {
  const total = totalIPsForCidr(pool.cidr || '')
  const used = poolUsedIPs(pool, allPools, blocks)
  return utilizationPercent(total, used)
}
