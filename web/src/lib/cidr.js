/**
 * Returns the first and last IPv4 address of a CIDR range, or null if invalid.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {{ start: string, end: string } | null}
 */
export function cidrRange(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
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
 * @param {string} networkAddress - e.g. "10.0.0.0" or "10.0.0.5"
 * @param {number} prefix - 0..32
 * @returns {{ cidr: string, netmask: string, first: string, last: string, usable: number, total: number } | null}
 */
export function getSubnetInfo(networkAddress, prefix) {
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
 * Parse CIDR string to base address (as 32-bit int) and prefix length.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {{ baseInt: number, prefix: number } | null}
 */
export function parseCidrToInt(cidr) {
  if (!cidr || typeof cidr !== 'string') return null
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
 * Divide a CIDR into subnets of a given (larger) prefix length.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @param {number} newPrefix - must be >= current prefix, <= 32 (e.g. 26 for /26 subnets)
 * @returns {{ cidr: string, netmask: string, first: string, last: string, usable: number, total: number }[] | null}
 */
export function divideSubnets(cidr, newPrefix) {
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

/**
 * Get the parent subnet (one prefix larger) that contains this CIDR.
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {{ cidr: string, netmask: string, first: string, last: string, usable: number, total: number } | null}
 */
export function getParentSubnet(cidr) {
  const parsed = parseCidrToInt(cidr)
  if (!parsed || parsed.prefix <= 0) return null
  return getSubnetInfo(cidr.slice(0, cidr.indexOf('/')), parsed.prefix - 1)
}

/**
 * Get the sibling subnet (other half of the same parent).
 * @param {string} cidr - e.g. "10.0.0.0/24"
 * @returns {string | null} sibling CIDR or null
 */
export function getSiblingCidr(cidr) {
  const parsed = parseCidrToInt(cidr)
  if (!parsed || parsed.prefix <= 0 || parsed.prefix > 32) return null
  const parent = getParentSubnet(cidr)
  if (!parent) return null
  const halves = divideSubnets(parent.cidr, parsed.prefix)
  if (!halves || halves.length !== 2) return null
  return halves[0].cidr === cidr ? halves[1].cidr : halves[0].cidr
}
