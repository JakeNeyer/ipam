// @ts-check
// Basic functionality: dashboard, nav, environments, networks (requires logged-in admin).

import { test, expect } from '@playwright/test'
import { login, ensureOrgSelected } from './helpers.js'

test.describe('basic functionality', () => {
  test.beforeEach(async ({ page }) => {
    const loggedIn = await login(page)
    if (!loggedIn) {
      test.skip(true, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set')
      return
    }
    await ensureOrgSelected(page)
  })

  test('dashboard shows and has nav', async ({ page }) => {
    await page.goto('/#dashboard')
    await expect(page.locator('.nav')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('button', { name: 'Dashboard' })).toBeVisible({ timeout: 5000 })
  })

  test('navigate to Environments', async ({ page }) => {
    await page.getByRole('button', { name: 'Environments' }).click()
    await expect(page.locator('h1.page-title', { hasText: 'Environments' })).toBeVisible({ timeout: 5000 })
  })

  test('navigate to Networks', async ({ page }) => {
    await page.getByRole('button', { name: 'Networks' }).click()
    await expect(page.locator('h1.page-title', { hasText: 'Networks' })).toBeVisible({ timeout: 5000 })
  })

  test('admin can open Admin page', async ({ page }) => {
    await page.getByRole('link', { name: 'Admin' }).click()
    await expect(page.getByRole('heading', { name: 'Admin' })).toBeVisible({ timeout: 5000 })
  })

  test('navigate to Network Advisor', async ({ page }) => {
    await page.getByRole('button', { name: 'Network Advisor' }).click()
    await expect(page.locator('h1.page-title', { hasText: 'Network Advisor' })).toBeVisible({ timeout: 5000 })
    await expect(page.locator('text=Step 1').first()).toBeVisible({ timeout: 5000 })
  })

  test('user guide has Network Advisor page', async ({ page }) => {
    await page.goto('/#docs/network-advisor')
    await expect(page.locator('text=Network Advisor').first()).toBeVisible({ timeout: 10000 })
    await expect(page.locator('text=step-by-step wizard').first()).toBeVisible({ timeout: 5000 })
  })
})
