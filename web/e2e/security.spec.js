// @ts-check
// Security: auth required for protected routes, API returns 401 without session, admin-only access.

import { test, expect } from '@playwright/test'

test.describe('security', () => {
  test('API /api/auth/me without session returns 401', async ({ request }) => {
    const res = await request.get('/api/auth/me')
    expect(res.status()).toBe(401)
  })

  test('API /api/admin/users without auth returns 401', async ({ request }) => {
    const res = await request.get('/api/admin/users')
    expect(res.status()).toBe(401)
  })

  test('API /api/setup/status is public', async ({ request }) => {
    const res = await request.get('/api/setup/status')
    expect(res.ok()).toBeTruthy()
    const body = await res.json()
    expect(body).toHaveProperty('setup_required')
  })

  test('unauthenticated visit to #/admin shows login or setup', async ({ page }) => {
    await page.goto('/#/admin')
    await page.waitForLoadState('networkidle')
    // App redirects non-auth to login/setup; or shows loading then login/setup
    await expect(page.locator('.login-form, .setup-form, text=Sign in, text=Create the initial admin').first()).toBeVisible({ timeout: 15000 })
  })

  test('direct API POST /api/setup with invalid email is rejected', async ({ request }) => {
    const res = await request.post('/api/setup', {
      data: { email: 'not-an-email', password: 'short' },
      headers: { 'Content-Type': 'application/json' },
    })
    expect(res.status()).toBe(400)
  })

  test('direct API POST /api/auth/login with invalid creds returns error', async ({ request }) => {
    const res = await request.post('/api/auth/login', {
      data: { email: 'nobody@example.com', password: 'wrong' },
      headers: { 'Content-Type': 'application/json' },
    })
    expect(res.status()).toBe(401)
  })
})
