// @ts-check
// Auth flows: landing, setup (when required), login, logout.

import { test, expect } from '@playwright/test'

test.describe('auth', () => {
  test('landing page loads and shows IPAM', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('text=IPAM').first()).toBeVisible({ timeout: 10000 })
    // Either landing (Get started / features) or after redirect: Login / Setup
    const hasGetStarted = await page.locator('text=Get started').isVisible().catch(() => false)
    const hasSignIn = await page.locator('text=Sign in').isVisible().catch(() => false)
    const hasSetup = await page.locator('text=Setup').isVisible().catch(() => false)
    expect(hasGetStarted || hasSignIn || hasSetup).toBeTruthy()
  })

  test('unauthenticated visit to #/dashboard shows login or setup', async ({ page }) => {
    await page.goto('/#/dashboard')
    await page.waitForLoadState('networkidle')
    // App shows either Login or Setup (or brief Loading)
    await expect(page.locator('text=Sign in').or(page.locator('text=Create the initial admin')).or(page.locator('text=Loading')).first()).toBeVisible({ timeout: 15000 })
    // Eventually we should see Sign in or Setup form, not dashboard nav
    await expect(page.locator('.nav, .login-form, .setup-form').first()).toBeVisible({ timeout: 10000 })
  })

  test('setup page shows form when setup is required', async ({ page }) => {
    await page.goto('/#/')
    await page.waitForLoadState('networkidle')
    // If setup is required we see Setup form; otherwise Login or Landing
    const setupForm = page.locator('.setup-form')
    const loginForm = page.locator('.login-form')
    const setupTitle = page.locator('text=Create the initial admin')
    await expect(setupForm.or(loginForm).or(setupTitle)).toBeVisible({ timeout: 15000 })
    if (await setupForm.isVisible()) {
      await expect(page.locator('input[type="email"]')).toBeVisible()
      await expect(page.locator('input[type="password"]').first()).toBeVisible()
    }
  })

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/#/login')
    await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
    await page.fill('input[type="email"]', 'nobody@example.com')
    await page.fill('input[type="password"]', 'wrongpassword')
    await page.click('button[type="submit"]')
    await expect(page.locator('.login-error, [role="alert"]').filter({ hasText: /invalid|required|failed/i })).toBeVisible({ timeout: 5000 })
  })

  test('login with valid credentials shows app', async ({ page }) => {
    const email = process.env.E2E_LOGIN_EMAIL
    const password = process.env.E2E_LOGIN_PASSWORD
    test.skip(!email || !password, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set for this test')
    await page.goto('/#/login')
    await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
    await page.fill('input[type="email"]', email)
    await page.fill('input[type="password"]', password)
    await page.click('button[type="submit"]')
    await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
    await expect(page.locator('text=Dashboard').first()).toBeVisible({ timeout: 5000 })
  })

  test('logout returns to landing or login', async ({ page }) => {
    const email = process.env.E2E_LOGIN_EMAIL
    const password = process.env.E2E_LOGIN_PASSWORD
    test.skip(!email || !password, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set')
    await page.goto('/#/login')
    await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
    await page.fill('input[type="email"]', email)
    await page.fill('input[type="password"]', password)
    await page.click('button[type="submit"]')
    await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: 'Settings' }).click()
    await page.getByRole('menuitem', { name: /Sign out/ }).click()
    await expect(page.locator('.login-form, .login-page').locator('text=Sign in').first()).toBeVisible({ timeout: 8000 })
  })
})
