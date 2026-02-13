/**
 * Detect IP version from a CIDR string.
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8::/32"
 * @returns {4 | 6 | null}
 */
export function ipVersion(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const idx = cidr.indexOf('/')
  const network = (idx === -1 ? cidr : cidr.slice(0, idx)).trim()
  if (network.includes(':')) return 6
  const parts = network.split('.')
  if (parts.length === 4 && parts.every((p) => /^\d+$/.test(p))) return 4
  return null
}

/**
 * Returns the first and last IPv4 address of a CIDR range, or null if invalid.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {{ start: string, end: string } | null}
 */
export function cidrRange(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const v = ipVersion(cidr)
  if (v === 6) return cidrRangeIPv6(cidr)
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
  const parts = network.split('.')
  if (parts.length !== 4) return null
  let int = 0
  for (let i = 0; i < 4; i++) {
    const n = parseInt(parts[i], 10)
    if (isNaN(n) || n < 0 || n > 255) return null
    int = (int << 8) | n
  }
  const count = Math.pow(2, 32 - prefix)
  const lastInt = int + count - 1
  if (lastInt > 0xffffffff) return null
  return {
    start: intToDotted(int),
    end: intToDotted(lastInt),
  }
}

/** Expand IPv6 string to 8 hextets (16-bit each). Returns null if invalid. */
function parseIPv6ToHextets(str) {
  if (!str || typeof str !== 'string') return null
  const s = str.trim().toLowerCase()
  if (s.includes('.')) return null
  const parts = s.split(':')
  if (parts.length < 3 || parts.length > 8) return null
  const emptyIdx = parts.indexOf('')
  let expanded
  if (emptyIdx === -1) {
    if (parts.length !== 8) return null
    expanded = parts
  } else {
    const before = parts.slice(0, emptyIdx)
    const after = parts.slice(emptyIdx + 1).filter((p) => p !== '')
    const total = before.length + after.length
    if (total > 7) return null
    const zeros = 8 - total
    expanded = [...before, ...Array(zeros).fill('0'), ...after]
  }
  const hextets = expanded.map((p) => (p === '' ? 0 : parseInt(p, 16)))
  if (hextets.length !== 8 || hextets.some((h) => isNaN(h) || h < 0 || h > 0xffff)) return null
  return hextets
}

function hextetsToBigInt(hextets) {
  let n = 0n
  for (let i = 0; i < 8; i++) n = (n << 16n) | BigInt(hextets[i])
  return n
}

function bigIntToHextets(n) {
  const out = []
  let x = n
  for (let i = 0; i < 8; i++) {
    out.unshift(Number(x & 0xffffn))
    x = x >> 16n
  }
  return out
}

/** Format a BigInt as IPv4 or IPv6 address string. For IPv4, n is low 32 bits. */
export function formatAddressFromBigInt(n, version) {
  if (version === 6) return formatIPv6(bigIntToHextets(n))
  const i = Number(n & 0xffffffffn) >>> 0
  return intToDotted(i)
}

/** Align value up to next multiple of blockSize (BigInt). */
export function alignUpBigInt(value, blockSize) {
  const remainder = value % blockSize
  return remainder === 0n ? value : value + (blockSize - remainder)
}

/** Format 8 hextets to IPv6 string (compress longest run of zeros to ::). */
export function formatIPv6(hextets) {
  if (!hextets || hextets.length !== 8) return ''
  const str = hextets.map((h) => h.toString(16)).join(':')
  let bestStart = -1
  let bestLen = 0
  for (let i = 0; i < 8; i++) {
    let j = i
    while (j < 8 && hextets[j] === 0) j++
    if (j - i > bestLen) {
      bestLen = j - i
      bestStart = i
    }
  }
  if (bestLen <= 1) return str
  const left = hextets.slice(0, bestStart).map((h) => h.toString(16)).join(':')
  const right = hextets.slice(bestStart + bestLen).map((h) => h.toString(16)).join(':')
  if (!left && !right) return '::'
  if (!left) return '::' + right
  if (!right) return left + '::'
  return left + '::' + right
}

/**
 * Returns the first and last IPv6 address of a CIDR range, or null if invalid.
 * @param {string} cidr - e.g. "2001:db8::/32"
 * @returns {{ start: string, end: string } | null}
 */
export function cidrRangeIPv6(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  if (isNaN(prefix) || prefix < 0 || prefix > 128) return null
  const hextets = parseIPv6ToHextets(network)
  if (!hextets) return null
  let base = hextetsToBigInt(hextets)
  const hostBits = 128 - prefix
  const mask = (1n << BigInt(hostBits)) - 1n
  base = (base >> BigInt(hostBits)) << BigInt(hostBits)
  const count = 1n << BigInt(hostBits)
  const last = base + count - 1n
  if (last > (1n << 128n) - 1n) return null
  return {
    start: formatIPv6(bigIntToHextets(base)),
    end: formatIPv6(bigIntToHextets(last)),
  }
}

function intToDotted(int) {
  return [
    (int >>> 24) & 0xff,
    (int >>> 16) & 0xff,
    (int >>> 8) & 0xff,
    int & 0xff,
  ].join('.')
}

/**
 * Parse a dotted-decimal IPv4 address to a 32-bit int, or null if invalid.
 * @param {string} address - e.g. "10.0.0.0"
 * @returns {number | null}
 */
export function parseAddress(address) {
  if (!address || typeof address !== 'string') return null
  const parts = address.trim().split('.')
  if (parts.length !== 4) return null
  let int = 0
  for (let i = 0; i < 4; i++) {
    const n = parseInt(parts[i], 10)
    if (isNaN(n) || n < 0 || n > 255) return null
    int = (int << 8) | n
  }
  return int >>> 0
}

/**
 * Get subnet info for a network address and prefix. Normalizes the address to the network base.
 * Supports IPv4 and IPv6; for IPv6, total/usable may be string when > Number.MAX_SAFE_INTEGER.
 * @param {string} networkAddress - e.g. "10.0.0.0" or "2001:db8::"
 * @param {number} prefix - 0..32 (IPv4) or 0..128 (IPv6)
 * @param {4|6} [version] - optional; if omitted, detected from networkAddress
 * @returns {{ cidr: string, netmask?: string, first: string, last: string, usable: number|string, total: number|string } | null}
 */
export function getSubnetInfo(networkAddress, prefix, version) {
  const v = version ?? (networkAddress && networkAddress.includes(':') ? 6 : 4)
  if (v === 6) return getSubnetInfoIPv6(networkAddress, prefix)
  const addrInt = parseAddress(networkAddress)
  if (addrInt === null || prefix < 0 || prefix > 32) return null
  const size = Math.pow(2, 32 - prefix)
  const networkBase = ((addrInt >>> (32 - prefix)) << (32 - prefix)) >>> 0
  const lastInt = (networkBase + size - 1) >>> 0
  if (lastInt > 0xffffffff) return null
  const total = size
  const usable = size >= 2 ? size - 2 : 0
  return {
    cidr: `${intToDotted(networkBase)}/${prefix}`,
    netmask: netmaskFromPrefix(prefix),
    first: intToDotted(networkBase),
    last: intToDotted(lastInt),
    usable,
    total,
  }
}

/**
 * Get subnet info for IPv6. total/usable are string when > Number.MAX_SAFE_INTEGER.
 * IPv6 has no reserved network/broadcast; usable === total.
 */
export function getSubnetInfoIPv6(networkAddress, prefix) {
  if (prefix < 0 || prefix > 128) return null
  const hextets = parseIPv6ToHextets(networkAddress)
  if (!hextets) return null
  let base = hextetsToBigInt(hextets)
  const hostBits = 128 - prefix
  base = (base >> BigInt(hostBits)) << BigInt(hostBits)
  const count = 1n << BigInt(hostBits)
  const last = base + count - 1n
  if (last > (1n << 128n) - 1n) return null
  const totalNum = count <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(count) : count.toString()
  const usable = totalNum
  return {
    cidr: `${formatIPv6(bigIntToHextets(base))}/${prefix}`,
    first: formatIPv6(bigIntToHextets(base)),
    last: formatIPv6(bigIntToHextets(last)),
    usable,
    total: totalNum,
  }
}

/**
 * Parse CIDR string to base address (as 32-bit int) and prefix length. IPv4 only.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {{ baseInt: number, prefix: number } | null}
 */
export function parseCidrToInt(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const v = ipVersion(cidr)
  if (v === 6) return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
  const parts = network.split('.')
  if (parts.length !== 4) return null
  let int = 0
  for (let i = 0; i < 4; i++) {
    const n = parseInt(parts[i], 10)
    if (isNaN(n) || n < 0 || n > 255) return null
    int = (int << 8) | n
  }
  return { baseInt: int >>> 0, prefix }
}

/**
 * Return start/end of CIDR as BigInt and normalized CIDR. For range math (overlap, find next).
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8::/32"
 * @returns {{ start: bigint, end: bigint, prefix: number, version: 4|6, cidr: string } | null}
 */
export function cidrRangeToBigInt(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  const v = ipVersion(cidr)
  if (v === 6) {
    if (isNaN(prefix) || prefix < 0 || prefix > 128) return null
    const hextets = parseIPv6ToHextets(network)
    if (!hextets) return null
    let base = hextetsToBigInt(hextets)
    const hostBits = 128 - prefix
    base = (base >> BigInt(hostBits)) << BigInt(hostBits)
    const count = 1n << BigInt(hostBits)
    const end = base + count - 1n
    const info = getSubnetInfoIPv6(network, prefix)
    return { start: base, end, prefix, version: 6, cidr: info?.cidr ?? cidr }
  }
  if (v === 4) {
    if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
    const parsed = parseCidrToInt(cidr)
    if (!parsed) return null
    const hostBits = 32 - prefix
    const count = 1n << BigInt(hostBits)
    const start = BigInt(parsed.baseInt >>> 0)
    const end = start + count - 1n
    const info = getSubnetInfo(network, prefix, 4)
    return { start, end, prefix, version: 4, cidr: info?.cidr ?? cidr }
  }
  return null
}

/**
 * Parse CIDR string to base (BigInt), prefix, and version. Works for IPv4 and IPv6.
 * For IPv4, baseBigInt is 0n..(2^32-1)n; for IPv6 full 128-bit.
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8::/32"
 * @returns {{ baseBigInt: bigint, prefix: number, version: 4|6 } | null}
 */
export function parseCidrToBigInt(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  const v = ipVersion(cidr)
  if (v === 6) {
    if (isNaN(prefix) || prefix < 0 || prefix > 128) return null
    const hextets = parseIPv6ToHextets(network)
    if (!hextets) return null
    let base = hextetsToBigInt(hextets)
    const hostBits = 128 - prefix
    base = (base >> BigInt(hostBits)) << BigInt(hostBits)
    return { baseBigInt: base, prefix, version: 6 }
  }
  if (v === 4) {
    if (isNaN(prefix) || prefix < 0 || prefix > 32) return null
    const parsed = parseCidrToInt(cidr)
    if (!parsed) return null
    return { baseBigInt: BigInt(parsed.baseInt), prefix, version: 4 }
  }
  return null
}

/**
 * Netmask string for a given prefix length (e.g. 24 -> "255.255.255.0").
 * @param {number} prefix - 0..32
 * @returns {string}
 */
export function netmaskFromPrefix(prefix) {
  if (prefix <= 0) return '0.0.0.0'
  if (prefix >= 32) return '255.255.255.255'
  const ones = (0xffffffff >>> (32 - prefix)) >>> 0
  return intToDotted(ones)
}

/**
 * Divide a CIDR into subnets of a given (larger) prefix length. Supports IPv4 and IPv6.
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8::/48"
 * @param {number} newPrefix - must be >= current prefix, <= 32 (IPv4) or <= 128 (IPv6)
 * @returns {{ cidr: string, netmask?: string, first: string, last: string, usable: number|string, total: number|string }[] | null}
 */
export function divideSubnets(cidr, newPrefix) {
  const v = ipVersion(cidr)
  if (v === 6) return divideSubnetsIPv6(cidr, newPrefix)
  const parsed = parseCidrToInt(cidr)
  if (!parsed) return null
  const { baseInt, prefix } = parsed
  if (newPrefix < prefix || newPrefix > 32) return null
  const subnetSize = Math.pow(2, 32 - newPrefix)
  const count = Math.pow(2, newPrefix - prefix)
  const netmask = netmaskFromPrefix(newPrefix)
  const subnets = []
  for (let i = 0; i < count; i++) {
    const start = (baseInt + i * subnetSize) >>> 0
    const end = (start + subnetSize - 1) >>> 0
    if (end > 0xffffffff) break
    const total = subnetSize
    const usable = subnetSize >= 2 ? subnetSize - 2 : 0
    subnets.push({
      cidr: `${intToDotted(start)}/${newPrefix}`,
      netmask,
      first: intToDotted(start),
      last: intToDotted(end),
      usable,
      total,
    })
  }
  return subnets
}

/** Divide IPv6 CIDR into subnets of newPrefix. */
function divideSubnetsIPv6(cidr, newPrefix) {
  const parsed = parseCidrToBigInt(cidr)
  if (!parsed || parsed.version !== 6) return null
  const { baseBigInt, prefix } = parsed
  if (newPrefix < prefix || newPrefix > 128) return null
  const subnetSize = 1n << BigInt(128 - newPrefix)
  const count = 1n << BigInt(newPrefix - prefix)
  const subnets = []
  for (let i = 0n; i < count; i++) {
    const start = baseBigInt + i * subnetSize
    const end = start + subnetSize - 1n
    if (end > (1n << 128n) - 1n) break
    const totalNum = subnetSize <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(subnetSize) : subnetSize.toString()
    subnets.push({
      cidr: `${formatIPv6(bigIntToHextets(start))}/${newPrefix}`,
      first: formatIPv6(bigIntToHextets(start)),
      last: formatIPv6(bigIntToHextets(end)),
      usable: totalNum,
      total: totalNum,
    })
  }
  return subnets
}

/**
 * Smallest power of 2 >= n (BigInt). Used for pool sizing so each pool is a valid CIDR.
 * @param {bigint} n - required IP count (>= 0)
 * @returns {bigint}
 */
export function nextPowerOf2BigInt(n) {
  if (n <= 1n) return n <= 0n ? 1n : 1n
  let p = 1n
  while (p < n) p *= 2n
  return p
}

/** Largest power of 2 <= n (BigInt). */
function floorPowerOf2BigInt(n) {
  if (n <= 0n) return 0n
  let p = 1n
  while (p * 2n <= n) p *= 2n
  return p
}

/**
 * Prefix length for a pool of the given size (power of 2). size = 2^(bits - prefix) => prefix = bits - log2(size).
 * @param {bigint} size - power-of-2 size (e.g. 256n for /24)
 * @param {4|6} version
 * @returns {number}
 */
function prefixFromPoolSizeBigInt(size, version) {
  const bits = version === 6 ? 128 : 32
  if (size <= 1n) return bits
  let exp = 0n
  let s = size
  while (s > 1n) {
    s >>= 1n
    exp += 1n
  }
  return bits - Number(exp)
}

/**
 * Split a base range into variable-sized pool subnets to optimize IP usage.
 * Each pool is sized to the smallest power-of-2 that fits that environment's required IPs (not evenly sized).
 * If the sum of requested sizes would exceed the base, pools are capped (larger mask = smaller size) so the total never overruns.
 * @param {{ start: bigint, end: bigint, prefix: number, version: 4|6, cidr?: string }} baseRange - from cidrRangeToBigInt
 * @param {bigint[]} requiredSizes - required IP count per pool (one per environment); each is rounded up to next power-of-2
 * @returns {{ start: bigint, end: bigint, prefix: number, version: 4|6, cidr: string }[]} - one per required size; sizes adjusted so sum <= base
 */
export function splitBaseIntoPoolRangesOptimized(baseRange, requiredSizes) {
  if (!baseRange || !requiredSizes?.length) return []
  const { start, end, version } = baseRange
  const baseSize = end - start + 1n
  let nextStart = start
  const out = []
  const n = requiredSizes.length
  for (let i = 0; i < n; i++) {
    const remaining = end - nextStart + 1n
    const poolsLeft = n - i
    const maxForThis = remaining - BigInt(poolsLeft - 1)
    if (maxForThis < 1n) break
    const required = requiredSizes[i] != null ? requiredSizes[i] : 1n
    const idealSize = nextPowerOf2BigInt(required <= 0n ? 1n : required)
    const poolSize = idealSize <= maxForThis ? idealSize : floorPowerOf2BigInt(maxForThis)
    if (poolSize < 1n) break
    const subEnd = nextStart + poolSize - 1n
    if (subEnd > end) break
    const poolPrefix = prefixFromPoolSizeBigInt(poolSize, version)
    const addrStr =
      version === 6 ? formatIPv6(bigIntToHextets(nextStart)) : intToDotted(Number(nextStart & 0xffffffffn))
    const cidr = `${addrStr}/${poolPrefix}`
    out.push({ start: nextStart, end: subEnd, prefix: poolPrefix, version, cidr })
    nextStart = subEnd + 1n
  }
  return out
}

/**
 * Divide a base range into n contiguous pool subnets (one per environment).
 * Each pool is a valid power-of-2 CIDR but sizes can differ — first pools may be larger so the base is simply divided, not evenly split.
 * @param {{ start: bigint, end: bigint, prefix: number, version: 4|6, cidr?: string }} baseRange - from cidrRangeToBigInt
 * @param {number} n - number of pools (environments)
 * @returns {{ start: bigint, end: bigint, prefix: number, version: 4|6, cidr: string }[]} - n pool ranges
 */
export function divideBaseIntoPoolRanges(baseRange, n) {
  if (!baseRange || n < 1) return []
  const { start, end, version } = baseRange
  const baseSize = end - start + 1n
  if (baseSize < BigInt(n)) return []
  let nextStart = start
  let remaining = baseSize
  let prevPoolSize = baseSize // so first pool can use any size <= baseSize
  const out = []
  for (let i = 0; i < n; i++) {
    const poolsLeft = n - i
    const targetSize = remaining / BigInt(poolsLeft)
    const maxPoolSize = remaining - BigInt(poolsLeft - 1)
    let poolSize = nextPowerOf2BigInt(targetSize)
    if (poolSize > maxPoolSize) poolSize = floorPowerOf2BigInt(maxPoolSize)
    if (poolSize > prevPoolSize) poolSize = prevPoolSize
    if (poolSize <= 0n) break
    const subEnd = nextStart + poolSize - 1n
    if (subEnd > end) break
    const poolPrefix = prefixFromPoolSizeBigInt(poolSize, version)
    const addrStr =
      version === 6 ? formatIPv6(bigIntToHextets(nextStart)) : intToDotted(Number(nextStart & 0xffffffffn))
    const cidr = `${addrStr}/${poolPrefix}`
    out.push({ start: nextStart, end: subEnd, prefix: poolPrefix, version, cidr })
    nextStart = subEnd + 1n
    remaining -= poolSize
    prevPoolSize = poolSize
  }
  return out
}

/**
 * Get the parent subnet (one prefix larger) that contains this CIDR. Supports IPv4 and IPv6.
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8:0:1::/64"
 * @returns {{ cidr: string, netmask?: string, first: string, last: string, usable: number|string, total: number|string } | null}
 */
export function getParentSubnet(cidr) {
  const v = ipVersion(cidr)
  if (!v) return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const network = cidr.slice(0, idx).trim()
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  if (isNaN(prefix) || prefix <= 0) return null
  if (v === 6) return getSubnetInfoIPv6(network, prefix - 1)
  return getSubnetInfo(network, prefix - 1, 4)
}

/**
 * Get the sibling subnet (other half of the same parent). Supports IPv4 and IPv6.
 * @param {string} cidr - e.g. "10.0.0.0/24" or "2001:db8:0:1::/64"
 * @returns {string | null} sibling CIDR or null
 */
export function getSiblingCidr(cidr) {
  const v = ipVersion(cidr)
  if (!v) return null
  const idx = cidr.indexOf('/')
  if (idx === -1) return null
  const prefix = parseInt(cidr.slice(idx + 1), 10)
  if (isNaN(prefix) || prefix <= 0 || (v === 4 && prefix > 32) || (v === 6 && prefix > 128)) return null
  const parent = getParentSubnet(cidr)
  if (!parent) return null
  const halves = divideSubnets(parent.cidr, prefix)
  if (!halves || halves.length !== 2) return null
  return halves[0].cidr === cidr ? halves[1].cidr : halves[0].cidr
}
