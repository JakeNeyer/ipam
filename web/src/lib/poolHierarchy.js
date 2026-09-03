/** Shared pool tree helpers used by Networks, Dashboard, and Environments. */

export const NIL_UUID = '00000000-0000-0000-0000-000000000000'

/** Case-insensitive UUID/id equality. */
export function idsMatch(a, b) {
  if (a == null || b == null || a === '' || b === '') return false
  return String(a).toLowerCase() === String(b).toLowerCase()
}

export function isOrphanedBlock(block) {
  const id = block?.environment_id
  return id == null || id === '' || String(id).toLowerCase() === NIL_UUID
}

function parentPoolId(pool) {
  if (!pool || pool.parent_pool_id == null || String(pool.parent_pool_id).trim() === '') return null
  return String(pool.parent_pool_id).toLowerCase()
}

/** Nesting depth of a pool (0 = root, 1 = child of root, …). */
export function getPoolDepth(pool, poolList) {
  if (!pool || !poolList) return 0
  let d = 0
  let p = pool
  const seen = new Set()
  while (p && parentPoolId(p)) {
    const pid = parentPoolId(p)
    if (seen.has(pid)) break
    seen.add(pid)
    const parent = poolList.find((x) => idsMatch(x.id, pid))
    if (!parent) break
    d += 1
    p = parent
  }
  return d
}

/** Order pools parent-first then children (for tables and hierarchy display). */
export function sortPoolsByHierarchy(poolList) {
  if (!poolList || !poolList.length) return []
  const id = (p) => String(p.id).toLowerCase()
  const byId = new Map(poolList.map((p) => [id(p), p]))
  const childrenMap = new Map()
  poolList.forEach((p) => {
    const pid = parentPoolId(p)
    if (!pid || !byId.has(pid)) return
    const list = childrenMap.get(pid) || []
    list.push(p)
    childrenMap.set(pid, list)
  })
  childrenMap.forEach((list) => list.sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' })))
  const result = []
  function visit(pool) {
    result.push(pool)
    ;(childrenMap.get(id(pool)) || []).forEach(visit)
  }
  const roots = poolList.filter((p) => !parentPoolId(p) || !byId.has(parentPoolId(p)))
  roots.sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
  roots.forEach(visit)
  return result
}

/** Direct child pools of a pool, sorted by name. */
export function childPoolsOf(parentId, poolList) {
  if (parentId == null || parentId === '') return []
  const pid = String(parentId).toLowerCase()
  return (poolList || [])
    .filter((p) => parentPoolId(p) === pid)
    .sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
}

/** Top-level pools in an environment (no parent, or parent not in this env). */
export function rootPoolsOf(envId, poolList) {
  const inEnv = (poolList || []).filter((p) => idsMatch(p.environment_id, envId))
  const ids = new Set(inEnv.map((p) => String(p.id).toLowerCase()))
  return inEnv
    .filter((p) => {
      const pid = parentPoolId(p)
      return !pid || !ids.has(pid)
    })
    .sort((a, b) => (a.name || '').localeCompare(b.name || '', undefined, { sensitivity: 'base' }))
}

/** Ancestor chain including `pool`, root first. */
export function ancestorPools(pool, poolList) {
  if (!pool) return []
  const chain = []
  let p = pool
  const seen = new Set()
  while (p) {
    chain.unshift(p)
    const pid = parentPoolId(p)
    if (!pid) break
    if (seen.has(pid)) break
    seen.add(pid)
    p = (poolList || []).find((x) => idsMatch(x.id, pid)) || null
  }
  return chain
}
