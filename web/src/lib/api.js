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
 * @param {{ limit?: number, offset?: number, name?: string, organization_id?: string }} opts
 * @returns {{ environments: Array, total: number }}
 */
export async function listEnvironments(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  if (opts.organization_id != null && opts.organization_id !== '') params.organization_id = opts.organization_id
  const data = await get('/environments', params)
  return { environments: data.environments ?? [], total: data.total ?? 0 }
}

/**
 * @param {string} name
 * @param {{ name: string, cidr: string } | null} [initialBlock]
 * @param {string | null} [organizationId] - Global admin: create env in this org
 */
export async function createEnvironment(name, initialBlock = null, organizationId = null) {
  const body = { name }
  if (initialBlock && initialBlock.name && initialBlock.cidr) {
    body.initial_block = { name: initialBlock.name, cidr: initialBlock.cidr }
  }
  if (organizationId != null && organizationId !== '') body.organization_id = organizationId
  return post('/environments', body)
}

function envIdPath(id) {
  return '/environments/' + encodeURIComponent(String(id))
}

export async function getEnvironment(id) {
  const data = await get(envIdPath(id))
  return data
}

export async function updateEnvironment(id, name) {
  return put(envIdPath(id), { name })
}

export async function deleteEnvironment(id) {
  return del(envIdPath(id))
}

/**
 * @param {{ limit?: number, offset?: number, name?: string, environment_id?: string, organization_id?: string, orphaned_only?: boolean }} opts
 * @returns {{ blocks: Array, total: number }}
 */
export async function listBlocks(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  if (opts.environment_id != null && opts.environment_id !== '') params.environment_id = opts.environment_id
  if (opts.organization_id != null && opts.organization_id !== '') params.organization_id = opts.organization_id
  if (opts.orphaned_only) params.orphaned_only = 'true'
  const data = await get('/blocks', params)
  return { blocks: data.blocks ?? [], total: data.total ?? 0 }
}

/**
 * Create a network block. For orphan blocks (no environment), organizationId is required for global admin.
 * @param {string} name
 * @param {string} cidr
 * @param {string|null} [environmentId] - environment UUID or null for orphan block
 * @param {string|null} [organizationId] - required for orphan blocks when global admin; org-scoped users use their org
 */
export async function createBlock(name, cidr, environmentId = null, organizationId = null) {
  const body = { name, cidr }
  if (environmentId) body.environment_id = environmentId
  if (!environmentId && organizationId) body.organization_id = organizationId
  return post('/blocks', body)
}

// Used only for block environment_id "unset"; never send for global-admin / organization_id (omit field instead).
const NIL_UUID = '00000000-0000-0000-0000-000000000000'

/**
 * Update a network block. When setting to orphan (no environment), organizationId is required for global admin.
 */
export async function updateBlock(id, name, environmentId = null, organizationId = null) {
  const body = { name }
  if (environmentId !== undefined && environmentId !== null) {
    body.environment_id = environmentId === '' ? NIL_UUID : environmentId
  }
  if (environmentId === '' || environmentId === null) {
    if (organizationId) body.organization_id = organizationId
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
 * @param {{ limit?: number, offset?: number, name?: string, block_name?: string, environment_id?: string, organization_id?: string }} opts
 * @returns {{ allocations: Array, total: number }}
 */
export async function listAllocations(opts = {}) {
  const limit = opts.limit ?? 500
  const params = { limit: limit || 500, offset: opts.offset ?? 0 }
  if (opts.name != null && opts.name !== '') params.name = opts.name
  if (opts.block_name != null && opts.block_name !== '') params.block_name = opts.block_name
  if (opts.environment_id != null && opts.environment_id !== '') params.environment_id = opts.environment_id
  if (opts.organization_id != null && opts.organization_id !== '') params.organization_id = opts.organization_id
  const data = await get('/allocations', params)
  return { allocations: data.allocations ?? [], total: data.total ?? 0 }
}

export async function createAllocation(name, block_name, cidr) {
  return post('/allocations', { name, block_name, cidr })
}

export async function updateAllocation(id, name) {
  return put('/allocations/' + id, { name })
}

export async function deleteAllocation(id) {
  return del('/allocations/' + id)
}

/**
 * Auth config (no auth required). Returns { oauth_providers: string[], github_oauth_enabled: boolean }.
 */
export async function getAuthConfig() {
  const data = await get('/auth/config')
  const providers = Array.isArray(data?.oauth_providers) ? data.oauth_providers : []
  return {
    oauthProviders: providers,
    githubOAuthEnabled: data?.github_oauth_enabled === true || providers.includes('github'),
  }
}

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
 * List API tokens for the current user.
 * @returns {{ tokens: Array<{ id: string, name: string, created_at: string, expires_at?: string | null, organization_id?: string }> }}
 */
export async function listTokens() {
  const data = await get('/auth/me/tokens')
  return { tokens: data.tokens ?? [] }
}

/**
 * Create an API token. The raw token is only returned once.
 * @param {string} name
 * @param {{ expires_at?: string | null, organization_id?: string | null }} [options] - Optional. expires_at: RFC3339; organization_id: global admin only, scopes token to this org.
 * @returns {{ token: { id: string, name: string, token: string, created_at: string, expires_at?: string | null, organization_id?: string } }}
 */
export async function createToken(name, options = {}) {
  const body = { name }
  if (options.expires_at) body.expires_at = options.expires_at
  if (options.organization_id != null && options.organization_id !== '') body.organization_id = options.organization_id
  const data = await post('/auth/me/tokens', body)
  return data
}

/**
 * Delete an API token by id.
 * @param {string} id
 */
export async function deleteToken(id) {
  const res = await fetch(`${API_BASE}/auth/me/tokens/${id}`, {
    ...FETCH_OPTS,
    method: 'DELETE',
  })
  if (!res.ok) await handleError(res)
  if (res.status === 204) return
}

/**
 * List reserved blocks (blacklisted CIDR ranges). Admin only.
 * @returns {{ reserved_blocks: Array<{ id: string, cidr: string, reason: string, created_at: string }> }}
 */
/**
 * @param {{ organization_id?: string }} [opts] - Global admin: scope to this org
 */
export async function listReservedBlocks(opts = {}) {
  const params = {}
  if (opts.organization_id != null && opts.organization_id !== '') params.organization_id = opts.organization_id
  const data = await (Object.keys(params).length ? get('/reserved-blocks', params) : get('/reserved-blocks'))
  return { reserved_blocks: data.reserved_blocks ?? [] }
}

/**
 * Create a reserved block. Admin only.
 * @param {{ name?: string, cidr: string, reason?: string }} body - name (optional), cidr, reason (optional)
 * @returns {{ id: string, name: string, cidr: string, reason: string, created_at: string }}
 */
export async function createReservedBlock(body) {
  const b = {
    name: (body.name ?? '').trim(),
    cidr: (body.cidr ?? '').trim(),
  }
  if (body.reason != null) b.reason = String(body.reason).trim()
  const data = await post('/reserved-blocks', b)
  return data
}

/**
 * Delete a reserved block by id. Admin only.
 * @param {string} id
 */
export async function deleteReservedBlock(id) {
  const res = await fetch(`${API_BASE}/reserved-blocks/${id}`, {
    ...FETCH_OPTS,
    method: 'DELETE',
  })
  if (!res.ok) await handleError(res)
  if (res.status === 204) return
  const text = await res.text()
  if (!text) return
  return JSON.parse(text)
}

/**
 * Update a reserved block name by id. Admin only.
 * @param {string} id
 * @param {string} name
 */
export async function updateReservedBlock(id, name) {
  return put('/reserved-blocks/' + encodeURIComponent(String(id)), { name: String(name ?? '') })
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

/**
 * List organizations. Global admin only.
 * @returns {{ organizations: Array<{ id: string, name: string, created_at: string }> }}
 */
export async function listOrganizations() {
  const data = await get('/admin/organizations')
  return { organizations: data.organizations ?? [] }
}

/**
 * Create an organization. Global admin only.
 * @param {string} name
 * @returns {{ organization: { id: string, name: string, created_at: string } }}
 */
export async function createOrganization(name) {
  const data = await post('/admin/organizations', { name })
  return data
}

/**
 * Update an organization's name. Global admin only.
 * @param {string} id - Organization UUID
 * @param {string} name
 * @returns {{ organization: { id: string, name: string, created_at: string } }}
 */
export async function updateOrganization(id, name) {
  const data = await fetch(`${API_BASE}/admin/organizations/${encodeURIComponent(id)}`, {
    ...FETCH_OPTS,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!data.ok) await handleError(data)
  return data.json()
}

/**
 * Delete an organization. Global admin only. Cascades: all environments, blocks, allocations,
 * reserved blocks, signup links, and users in the org are permanently deleted.
 * @param {string} id - Organization UUID
 */
export async function deleteOrganization(id) {
  await del('/admin/organizations/' + encodeURIComponent(id))
}

/**
 * Create a user. Global admin can pass organizationId; org admin creates in their org.
 * @param {string} email
 * @param {string} password
 * @param {string} [role='user']
 * @param {string|null} [organizationId=null] - Required when global admin; ignored for org admin
 */
export async function createUser(email, password, role = 'user', organizationId = null) {
  const body = { email, password, role }
  if (organizationId != null && organizationId !== '') {
    body.organization_id = organizationId
  }
  const data = await post('/admin/users', body)
  return data?.user ?? null
}

export async function updateUserRole(id, role) {
  const data = await fetch(`${API_BASE}/admin/users/${encodeURIComponent(id)}/role`, {
    ...FETCH_OPTS,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  })
  if (!data.ok) await handleError(data)
  const res = await data.json()
  return res?.user ?? null
}

export async function deleteUser(id) {
  await del('/admin/users/' + encodeURIComponent(id))
}

/**
 * Update a user's organization. Global admin only.
 * @param {string} userId
 * @param {string} organizationId - UUID; use '' or null for global admin (no org). We omit the field for "no org" so the zero UUID is never sent.
 */
export async function updateUserOrganization(userId, organizationId) {
  const body = {}
  if (organizationId != null && organizationId !== '') {
    body.organization_id = organizationId
  }
  const data = await fetch(`${API_BASE}/admin/users/${encodeURIComponent(userId)}/organization`, {
    ...FETCH_OPTS,
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!data.ok) await handleError(data)
  const res = await data.json()
  return res?.user ?? null
}

/**
 * List signup invites (admin only).
 * @returns {{ invites: Array<{ id, created_at, expires_at, used_at?, used_by_email? }> }}
 */
export async function listSignupInvites() {
  const data = await get('/admin/signup-invites')
  return { invites: data?.invites ?? [] }
}

/**
 * Create a time-bound signup invite link (admin only). Global admin can pass organizationId and role.
 * @param {number} [expiresInHours=24]
 * @param {string|null} [organizationId=null] - Global admin only; org the new user will join
 * @param {string} [role='user'] - Global admin only; 'user' or 'admin'
 * @returns {{ invite_url: string, token: string, expires_at: string }}
 */
export async function createSignupInvite(expiresInHours = 24, organizationId = null, role = 'user') {
  const body = { expires_in_hours: expiresInHours }
  if (organizationId != null && organizationId !== '') {
    body.organization_id = organizationId
  }
  if (role === 'admin') {
    body.role = 'admin'
  }
  const data = await post('/admin/signup-invites', body)
  return data
}

/**
 * Revoke a signup invite (admin only). Fails if invite not found.
 * @param {string} id - invite id
 */
export async function revokeSignupInvite(id) {
  await del('/admin/signup-invites/' + encodeURIComponent(id))
}

/**
 * Validate a signup invite token (no auth).
 * @param {string} token
 * @returns {{ valid: boolean, expires_at?: string }}
 */
export async function validateSignupInvite(token) {
  const data = await get('/signup/validate', { token })
  return data
}

/**
 * Register a new user with an invite token (no auth). Sets session on success.
 * @param {string} token
 * @param {string} email
 * @param {string} password
 * @returns {{ user: { id, email, role, tour_completed } }}
 */
export async function registerWithInvite(token, email, password) {
  const data = await post('/signup/register', { token, email, password })
  return data
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
