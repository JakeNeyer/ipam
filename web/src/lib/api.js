const API_BASE = '/api'

const FETCH_OPTS = { credentials: 'include' }

/**
 * Parses API error response body into a user-friendly message.
 * Handles JSON bodies with message/error/detail fields or plain text.
 */
export function parseErrorMessage(body, fallback = 'Something went wrong') {
  if (!body || typeof body !== 'string') return fallback
  const trimmed = body.trim()
  if (!trimmed) return fallback
  if (trimmed.startsWith('{')) {
    try {
      const data = JSON.parse(trimmed)
      const msg = data.message ?? data.error ?? data.detail ?? data.msg
      if (typeof msg === 'string' && msg.length > 0) return msg
      if (Array.isArray(data.errors) && data.errors.length > 0) {
        const first = data.errors[0]
        return typeof first === 'string' ? first : (first.message ?? first.error ?? fallback)
      }
    } catch {
      // not valid JSON, use raw text below
    }
  }
  return trimmed.length > 200 ? trimmed.slice(0, 200) + '…' : trimmed
}

async function handleError(res) {
  const text = await res.text().catch(() => res.statusText)
  throw new Error(parseErrorMessage(text, res.statusText))
}

async function get(path, params = {}) {
  const q = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v))
  })
  const query = q.toString()
  const url = query ? `${API_BASE}${path}?${query}` : `${API_BASE}${path}`
  const res = await fetch(url, FETCH_OPTS)
  if (!res.ok) await handleError(res)
  return res.json()
}

async function post(path, body) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...FETCH_OPTS,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await handleError(res)
  return res.json()
}

async function put(path, body) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...FETCH_OPTS,
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await handleError(res)
  return res.json()
}

async function del(path) {
  const res = await fetch(`${API_BASE}${path}`, { ...FETCH_OPTS, method: 'DELETE' })
  if (!res.ok) await handleError(res)
  if (res.status === 204) return
  const text = await res.text()
  if (!text) return
  return JSON.parse(text)
}

/**
 * @param {{ limit?: number, offset?: number, name?: string }} opts
 * @returns {{ environments: Array, total: number }}
 */
export async function listEnvironments(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  const data = await get('/environments', params)
  return { environments: data.environments ?? [], total: data.total ?? 0 }
}

export async function createEnvironment(name, initialBlock = null) {
  const body = { name }
  if (initialBlock && initialBlock.name && initialBlock.cidr) {
    body.initial_block = { name: initialBlock.name, cidr: initialBlock.cidr }
  }
  return post('/environments', body)
}

export async function getEnvironment(id) {
  const data = await get('/environments/' + id)
  return data
}

export async function updateEnvironment(id, name) {
  return put('/environments/' + id, { name })
}

export async function deleteEnvironment(id) {
  return del('/environments/' + id)
}

/**
 * @param {{ limit?: number, offset?: number, name?: string, environment_id?: string, orphaned_only?: boolean }} opts
 * @returns {{ blocks: Array, total: number }}
 */
export async function listBlocks(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  if (opts.environment_id != null && opts.environment_id !== '') params.environment_id = opts.environment_id
  if (opts.orphaned_only) params.orphaned_only = 'true'
  const data = await get('/blocks', params)
  return { blocks: data.blocks ?? [], total: data.total ?? 0 }
}

export async function createBlock(name, cidr, environmentId = null) {
  const body = { name, cidr }
  if (environmentId) body.environment_id = environmentId
  return post('/blocks', body)
}

const NIL_UUID = '00000000-0000-0000-0000-000000000000'

export async function updateBlock(id, name, environmentId = null) {
  const body = { name }
  if (environmentId !== undefined && environmentId !== null) {
    body.environment_id = environmentId === '' ? NIL_UUID : environmentId
  }
  return put('/blocks/' + id, body)
}

export async function deleteBlock(id) {
  return del('/blocks/' + id)
}

export async function suggestBlockCidr(blockId, prefix) {
  const data = await get(`/blocks/${blockId}/suggest-cidr?prefix=${encodeURIComponent(prefix)}`)
  return data?.cidr ?? ''
}

export async function suggestEnvironmentBlockCidr(environmentId, prefix) {
  const data = await get(`/environments/${environmentId}/suggest-block-cidr?prefix=${encodeURIComponent(prefix)}`)
  return data?.cidr ?? ''
}

/**
 * @param {{ limit?: number, offset?: number, name?: string, block_name?: string }} opts
 * @returns {{ allocations: Array, total: number }}
 */
export async function listAllocations(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  if (opts.block_name != null && opts.block_name !== '') params.block_name = opts.block_name
  const data = await get('/allocations', params)
  return { allocations: data.allocations ?? [], total: data.total ?? 0 }
}

export async function createAllocation(name, block_name, cidr) {
  return post('/allocations', { name, block_name, cidr })
}

export async function deleteAllocation(id) {
  return del('/allocations/' + id)
}

/**
 * Fetches the full CSV export and triggers a file download.
 * Throws on non-OK response.
 */
export async function login(email, password) {
  const data = await post('/auth/login', { email, password })
  return data?.user ?? null
}

export async function logout() {
  await post('/auth/logout', {})
}

export async function getMe() {
  const res = await fetch(`${API_BASE}/auth/me`, FETCH_OPTS)
  if (res.status === 401) return null
  if (!res.ok) throw new Error(parseErrorMessage(await res.text()))
  const data = await res.json()
  return data?.user ?? null
}

/**
 * Marks the onboarding tour as completed for the current user. Persisted on the backend.
 */
export async function completeTour() {
  const res = await fetch(`${API_BASE}/auth/me/tour-completed`, {
    ...FETCH_OPTS,
    method: 'POST',
  })
  if (!res.ok) await handleError(res)
  if (res.status === 204) return
  return res.json()
}

/**
 * Returns whether initial setup is required (no users exist).
 * @returns {{ setup_required: boolean }}
 */
export async function getSetupStatus() {
  const res = await fetch(`${API_BASE}/setup/status`, FETCH_OPTS)
  if (!res.ok) throw new Error(parseErrorMessage(await res.text()))
  return res.json()
}

/**
 * Creates the initial admin. Only succeeds when no users exist.
 * @returns {{ user: { id, email, role } }}
 */
export async function setup(email, password) {
  const data = await post('/setup', { email, password })
  return data?.user ?? null
}

export async function listUsers() {
  const res = await fetch(`${API_BASE}/admin/users`, FETCH_OPTS)
  if (!res.ok) await handleError(res)
  const data = await res.json()
  return { users: data.users ?? [] }
}

export async function createUser(email, password, role = 'user') {
  const data = await post('/admin/users', { email, password, role })
  return data?.user ?? null
}

export async function exportCSV() {
  const res = await fetch(`${API_BASE}/export/csv`, FETCH_OPTS)
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(parseErrorMessage(text, res.statusText))
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'ipam-export.csv'
  a.click()
  URL.revokeObjectURL(url)
}
