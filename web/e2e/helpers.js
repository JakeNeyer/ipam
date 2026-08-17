// @ts-check
// Shared Playwright helpers for IPAM e2e tests.

import { expect } from '@playwright/test'

/**
 * Skip the first-login tour overlay if it is showing.
 * @param {import('@playwright/test').Page} page
 */
export async function dismissTour(page) {
  const skip = page.locator('.tour-skip')
  if (await skip.isVisible({ timeout: 2000 }).catch(() => false)) {
    await skip.click()
    await expect(skip).toBeHidden({ timeout: 5000 })
  }
}

/**
 * Log in with E2E_LOGIN_EMAIL / E2E_LOGIN_PASSWORD.
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<boolean>} false when credentials are not set
 */
export async function login(page) {
  const email = process.env.E2E_LOGIN_EMAIL
  const password = process.env.E2E_LOGIN_PASSWORD
  if (!email || !password) return false
  await page.goto('/#login')
  await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', password)
  await page.click('button[type="submit"]')
  await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
  await dismissTour(page)
  return true
}

/**
 * Global admin has no org until one is created and selected; main nav is hidden until then.
 * Creates an org via API if needed, then selects it in the sidebar.
 * @param {import('@playwright/test').Page} page
 */
export async function ensureOrgSelected(page) {
  const orgSelect = page.locator('#org-select')
  if (!(await orgSelect.isVisible().catch(() => false))) return

  const nonEmptyOptions = orgSelect.locator('option[value]:not([value=""])')
  if ((await nonEmptyOptions.count()) === 0) {
    const res = await page.request.post('/api/admin/organizations', {
      data: { name: 'E2E Org' },
      headers: { 'Content-Type': 'application/json' },
    })
    if (!res.ok()) {
      const body = await res.text()
      throw new Error(`create organization failed: ${res.status()} ${body}`)
    }
    await page.reload()
    await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
    await dismissTour(page)
  }

  const select = page.locator('#org-select')
  await expect(select).toBeVisible({ timeout: 5000 })
  await expect(select.locator('option[value]:not([value=""])').first()).toBeAttached({ timeout: 10000 })
  const values = await select.locator('option').evaluateAll((opts) =>
    opts.map((o) => /** @type {HTMLOptionElement} */ (o).value).filter(Boolean),
  )
  if (values.length === 0) throw new Error('no organizations available to select')
  await select.selectOption(values[0])
  await expect(page.getByRole('button', { name: 'Environments' })).toBeVisible({ timeout: 5000 })
}
