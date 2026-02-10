// @ts-check
// Basic functionality: dashboard, nav, environments, networks (requires logged-in admin).

import { test, expect } from '@playwright/test'

test.describe('basic functionality', () => {
  test.beforeEach(async ({ page }) => {
    const email = process.env.E2E_LOGIN_EMAIL
    const password = process.env.E2E_LOGIN_PASSWORD
    if (!email || !password) {
      test.skip(true, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set')
      return
    }
    await page.goto('/#/login')
    await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
    await page.fill('input[type="email"]', email)
    await page.fill('input[type="password"]', password)
    await page.click('button[type="submit"]')
    await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
  })

  test('dashboard shows and has nav', async ({ page }) => {
    await page.goto('/#/dashboard')
    await expect(page.locator('.nav')).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('button', { name: 'Dashboard' }).or(page.locator('text=Dashboard').first())).toBeVisible({ timeout: 5000 })
  })

  test('navigate to Environments', async ({ page }) => {
    await page.goto('/#/')
    await page.getByRole('button', { name: 'Environments' }).click()
    await expect(page.locator('text=Environments').first()).toBeVisible({ timeout: 5000 })
  })

  test('navigate to Networks', async ({ page }) => {
    await page.goto('/#/')
    await page.getByRole('button', { name: 'Networks' }).click()
    await expect(page.locator('text=Networks').first()).toBeVisible({ timeout: 5000 })
  })

  test('admin can open Admin page', async ({ page }) => {
    await page.goto('/#/')
    await page.getByRole('button', { name: 'Admin' }).click()
    await expect(page.locator('text=Admin').or(page.locator('text=Users')).first()).toBeVisible({ timeout: 5000 })
  })
})
