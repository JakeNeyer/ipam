import { writable } from 'svelte/store'
import { getMe, logout as apiLogout, getSetupStatus } from './api.js'

/** @type {import('svelte/store').Writable<{ id: string, email: string, role: string } | null>} */
export const user = writable(null)

/** @type {import('svelte/store').Writable<boolean>} */
export const authChecked = writable(false)

/** @type {import('svelte/store').Writable<boolean | null>} true = setup required, false = not required, null = not yet checked */
export const setupRequired = writable(null)

/**
 * Fetches current user and updates the store. Call on app load.
 * @returns {Promise<boolean>} true if logged in, false otherwise
 */
export async function checkAuth() {
  try {
    const u = await getMe()
    user.set(u)
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
  }
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
