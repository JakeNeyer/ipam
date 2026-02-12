/**
 * Helpers for block total_ips / used_ips / available_ips.
 * API returns these as strings (derive-only; supports IPv6 /64 etc.).
 */

const SAFE_INT = 2 ** 53 - 1

function toBigInt(v) {
  if (v == null || v === '') return null
  if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.floor(v))
  const s = String(v).trim()
  if (!s) return null
  try {
    return BigInt(s)
  } catch {
    return null
  }
}

/** Format large count as scientific notation (e.g. 1.84e19) for IPv6 total hosts */
function toScientific(big) {
  if (big == null || big < 0n) return '0'
  if (big === 0n) return '0'
  const s = String(big)
  const len = s.length
  if (len <= 15) return Number(big).toLocaleString()
  const lead = s.slice(0, 3)
  const exp = len - 1
  const decimal = lead.slice(0, 1) + '.' + lead.slice(1).replace(/^0+$/, '0') || '0'
  return parseFloat(decimal).toFixed(2).replace(/\.?0+$/, '') + 'e' + exp
}

/** Format count for display: locale string if safe number, scientific notation if large (e.g. IPv6), else raw string */
export function formatBlockCount(v) {
  if (v == null || v === '') return '0'
  if (typeof v === 'number' && !Number.isFinite(v)) return v === Infinity ? '∞' : String(v)
  const n = typeof v === 'number' ? v : Number(String(v).trim())
  if (Number.isFinite(n) && n <= SAFE_INT && n >= -SAFE_INT) {
    return Math.floor(n).toLocaleString()
  }
  const big = toBigInt(v)
  if (big != null && big >= 0n && big <= BigInt(SAFE_INT)) return Number(big).toLocaleString()
  if (big != null && big > BigInt(SAFE_INT)) return toScientific(big)
  return String(v)
}

/** Compare two counts (string or number) for sorting; returns -1, 0, or 1 */
export function compareBlockCount(a, b) {
  const ba = toBigInt(a)
  const bb = toBigInt(b)
  if (ba == null && bb == null) return 0
  if (ba == null) return 1
  if (bb == null) return -1
  if (ba < bb) return -1
  if (ba > bb) return 1
  return 0
}

/** Utilization percent (0–100) from total and used (strings or numbers); uses BigInt for large values */
export function utilizationPercent(total, used) {
  const t = toBigInt(total)
  const u = toBigInt(used)
  if (t == null || t === 0n) return 0
  if (u == null || u < 0n) return 0
  if (u >= t) return 100
  return Number((u * 10000n) / t) / 100
}

/** Sum an array of counts (strings or numbers); returns string */
export function sumCounts(values) {
  let sum = 0n
  for (const v of values || []) {
    const b = toBigInt(v)
    if (b != null && b >= 0n) sum += b
  }
  return sum.toString()
}
