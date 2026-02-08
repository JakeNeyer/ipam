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
