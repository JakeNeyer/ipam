import assert from 'node:assert/strict'

import {
  totalIPsForPrefix,
  usableIPsForPrefix,
  suggestBlockForEnvironment,
  blockSizeForEnvironmentAsString,
  totalAllocatedBlockIPsForDisplay,
} from '../src/lib/networkSizing.js'

function test(name, fn) {
  try {
    fn()
    process.stdout.write(`ok - ${name}\n`)
  } catch (err) {
    process.stderr.write(`not ok - ${name}\n`)
    process.stderr.write((err && err.stack) ? `${err.stack}\n` : `${String(err)}\n`)
    process.exitCode = 1
  }
}

// --- totalIPsForPrefix / usableIPsForPrefix ---

test('totalIPsForPrefix IPv4 /24 => 256', () => {
  assert.equal(totalIPsForPrefix(24, 4), 256)
})

test('totalIPsForPrefix IPv6 /64 => 2^64 as string', () => {
  assert.equal(totalIPsForPrefix(64, 6), '18446744073709551616')
})

test('totalIPsForPrefix IPv6 /8 => 2^120 as string (no Infinity)', () => {
  assert.equal(totalIPsForPrefix(8, 6), '1329227995784915872903807060280344576')
})

test('usableIPsForPrefix IPv6 equals total', () => {
  assert.equal(usableIPsForPrefix(64, 6), '18446744073709551616')
})

// --- suggestBlockForEnvironment (IPv6) ---

test('suggestBlockForEnvironment IPv6: 1 network, 8 hosts => /125 block and subnet', () => {
  const env = { id: 'e1', networks: 1, hostsPerNetwork: 8, growthPercent: 0 }
  const s = suggestBlockForEnvironment(env, 6)
  assert.equal(s.networkPrefix, 125)
  assert.equal(s.prefix, 125)
  assert.equal(s.requiredIPs, 8)
  assert.equal(s.requiredBlockIPs, 8)
})

test('suggestBlockForEnvironment IPv6: 2 networks, 8 hosts => /124 block, /125 subnet', () => {
  const env = { id: 'e1', networks: 2, hostsPerNetwork: 8, growthPercent: 0 }
  const s = suggestBlockForEnvironment(env, 6)
  assert.equal(s.networkPrefix, 125)
  assert.equal(s.prefix, 124)
  assert.equal(s.requiredIPs, 16)
  assert.equal(s.requiredBlockIPs, 16)
})

// --- BigInt-safe advisor totals ---

test('blockSizeForEnvironmentAsString IPv6 returns full integer string', () => {
  // With 1 network sized to /125, block prefix should also be /125 => 8 IPs.
  const env = { id: 'e1', networks: 1, hostsPerNetwork: 8, growthPercent: 0 }
  assert.equal(blockSizeForEnvironmentAsString(env, 6), '8')
})

test('totalAllocatedBlockIPsForDisplay IPv6 returns BigInt sum', () => {
  const envs = [
    { id: 'e1', networks: 1, hostsPerNetwork: 8, growthPercent: 0 }, // /125 => 8
    { id: 'e2', networks: 2, hostsPerNetwork: 8, growthPercent: 0 }, // /124 => 16
  ]
  const sum = totalAllocatedBlockIPsForDisplay(envs, 6)
  assert.equal(typeof sum, 'bigint')
  assert.equal(sum, 24n)
})

if (process.exitCode) {
  process.stderr.write('\nSizing tests failed.\n')
} else {
  process.stdout.write('\nAll sizing tests passed.\n')
}

