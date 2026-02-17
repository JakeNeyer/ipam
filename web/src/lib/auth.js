import { writable } from 'svelte/store'
import { getMe, logout as apiLogout, getSetupStatus, getAuthConfig } from './api.js'

/** @type {import('svelte/store').Writable<{ id: string, email: string, role: string } | null>} */
export const user = writable(null)

/** @type {import('svelte/store').Writable<boolean>} */
export const authChecked = writable(false)

/** @type {import('svelte/store').Writable<boolean | null>} true = setup required, false = not required, null = not yet checked */
export const setupRequired = writable(null)

/** @type {import('svelte/store').Writable<boolean>} true when at least one OAuth provider is configured */
export const oauthEnabled = writable(false)

const SELECTED_ORG_STORAGE_KEY = 'ipam-global-admin-selected-org'

function getPersistedSelectedOrg() {
  if (typeof window === 'undefined') return null
  try {
    const raw = localStorage.getItem(SELECTED_ORG_STORAGE_KEY)
    if (!raw) return null
    const data = JSON.parse(raw)
    if (data && typeof data.id === 'string' && data.id) return { id: data.id, name: data.name ?? null }
    return null
  } catch {
    return null
  }
}

function setPersistedSelectedOrg(id, name) {
  if (typeof window === 'undefined') return
  try {
    if (id) localStorage.setItem(SELECTED_ORG_STORAGE_KEY, JSON.stringify({ id, name: name ?? null }))
    else localStorage.removeItem(SELECTED_ORG_STORAGE_KEY)
  } catch (_) {}
}

/**
 * Incremented when organizations list changes (create/update/delete on Admin page).
 * Nav subscribes and refetches organizations when this changes so the org dropdown updates.
 * @type {import('svelte/store').Writable<number>}
 */
export const organizationsRefreshTrigger = writable(0)

/**
 * Selected organization ID for global admin "switch org" context.
 * When set, list/create calls scope to this org. When null, global admin sees all orgs.
 * Persisted so it survives page refresh.
 * @type {import('svelte/store').Writable<string | null>}
 */
export const selectedOrgForGlobalAdmin = writable(null)

/**
 * Display name of the selected org (for global admin banner). Kept in sync when org is selected.
 * @type {import('svelte/store').Writable<string | null>}
 */
export const selectedOrgNameForGlobalAdmin = writable(null)

/**
 * Set the selected org for global admin (e.g. from nav dropdown or global admin dashboard).
 * Persists to localStorage so selection survives page refresh.
 * @param {string | null} id - Organization UUID or null to clear
 * @param {string | null} [name] - Display name (optional when clearing)
 */
export function setSelectedOrgForGlobalAdmin(id, name) {
  selectedOrgForGlobalAdmin.set(id)
  selectedOrgNameForGlobalAdmin.set(name ?? null)
  setPersistedSelectedOrg(id, name ?? null)
}

/**
 * Fetches current user and updates the store. Call on app load.
 * @returns {Promise<boolean>} true if logged in, false otherwise
 */
export async function checkAuth() {
  try {
    const [u, authCfg] = await Promise.all([
      getMe(),
      getAuthConfig().catch(() => null),
    ])
    user.set(u)
    if (authCfg) {
      oauthEnabled.set(
        (Array.isArray(authCfg.oauthProviders) && authCfg.oauthProviders.length > 0) ||
        authCfg.githubOAuthEnabled === true
      )
    }
    if (u && isGlobalAdmin(u)) {
      const persisted = getPersistedSelectedOrg()
      if (persisted) {
        selectedOrgForGlobalAdmin.set(persisted.id)
        selectedOrgNameForGlobalAdmin.set(persisted.name)
      }
    }
    authChecked.set(true)
    return u != null
  } catch {
    user.set(null)
    authChecked.set(true)
    return false
  }
}

/**
 * Logs out and clears the user store.
 */
export async function logout() {
  try {
    await apiLogout()
  } finally {
    user.set(null)
    selectedOrgForGlobalAdmin.set(null)
    selectedOrgNameForGlobalAdmin.set(null)
  }
}

/** True when the current user is global admin (admin with no organization). */
export function isGlobalAdmin(u) {
  return u?.role === 'admin' && (u?.organization_id == null || u?.organization_id === '')
}

/**
 * Fetches setup status and updates setupRequired store.
 * Call when not logged in to decide whether to show Setup or Login.
 * On failure (e.g. network/500), defaults to true so the user can still reach Setup.
 * @returns {Promise<boolean>} true if setup is required
 */
export async function checkSetupRequired() {
  try {
    const res = await getSetupStatus()
    const required = res?.setup_required === true
    setupRequired.set(required)
    return required
  } catch {
    // When status check fails (e.g. Postgres/network), show Setup so initial admin can be created
    setupRequired.set(true)
    return true
  }
}
