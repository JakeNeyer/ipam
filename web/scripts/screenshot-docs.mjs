#!/usr/bin/env node
/**
 * Capture screenshots for the user guide (light and dark mode).
 *
 * 1. Logs in once (requires LOGIN_EMAIL and LOGIN_PASSWORD).
 * 2. For each app page (Dashboard, Networks, etc.), captures in light and dark mode.
 * 3. Does not capture the landing page—only authenticated app pages.
 *
 * Run with the dev server up: npm run dev (in another terminal), then:
 *   LOGIN_EMAIL=admin@localhost LOGIN_PASSWORD=yourpassword node scripts/screenshot-docs.mjs
 *
 * Requires: npm install -D playwright && npx playwright install chromium
 */
import { chromium } from 'playwright'
import { mkdir } from 'fs/promises'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const baseUrl = process.env.BASE_URL || 'http://localhost:5173'
const outDir = join(__dirname, '..', 'public', 'images')
const THEME_STORAGE_KEY = 'ipam-theme'
const loginEmail = process.env.LOGIN_EMAIL || ''
const loginPassword = process.env.LOGIN_PASSWORD || ''

/** App pages to capture (user docs). Landing page is not in this list. */
const captures = [
  { url: `${baseUrl}/#`, name: 'Dashboard', base: 'dashboard' },
  { url: `${baseUrl}/#networks`, name: 'Networks', base: 'networks' },
  { url: `${baseUrl}/#environments`, name: 'Environments', base: 'environments' },
  { url: `${baseUrl}/#admin`, name: 'Admin', base: 'admin' },
  { url: `${baseUrl}/#reserved-blocks`, name: 'Reserved blocks', base: 'reserved-blocks' },
  { url: `${baseUrl}/#subnet-calculator`, name: 'Subnet calculator', base: 'subnet-calculator' },
  {
    url: `${baseUrl}/#`,
    name: 'Command palette',
    base: 'command-palette',
    async beforeScreenshot(page) {
      await page.keyboard.press(process.platform === 'darwin' ? 'Meta+k' : 'Control+k')
      await page.waitForSelector('.palette', { state: 'visible', timeout: 5000 })
      await page.waitForTimeout(500)
    },
  },
  {
    url: `${baseUrl}/#networks`,
    name: 'CIDR wizard',
    base: 'cidr-wizard',
    async beforeScreenshot(page) {
      await page.click('button:has-text("Create block")')
      await page.waitForSelector('.cidr-wizard', { state: 'visible', timeout: 5000 })
      await page.waitForTimeout(500)
    },
  },
]

const NAV_TIMEOUT_MS = 15000

function setThemeInStorage(page, theme) {
  return page.evaluate(
    ({ key, value }) => {
      localStorage.setItem(key, value)
      document.documentElement.setAttribute('data-theme', 'wintry')
      document.documentElement.classList.toggle('dark', value === 'dark')
    },
    { key: THEME_STORAGE_KEY, value: theme }
  )
}

async function signOut(page) {
  await page.goto(`${baseUrl}/#`, { waitUntil: 'load', timeout: 20000 })
  await page.waitForTimeout(1500)
  const nav = page.locator('nav')
  const inApp = await nav.isVisible().catch(() => false)
  if (!inApp) return
  await page.locator('.settings-trigger').click()
  await page.waitForSelector('.settings-popover', { state: 'visible', timeout: 3000 }).catch(() => null)
  await page.getByRole('menuitem', { name: /sign out/i }).click()
  await page.waitForSelector('.login-form, .landing', { timeout: 8000 }).catch(() => null)
}

async function login(page) {
  if (!loginEmail || !loginPassword) {
    console.warn('LOGIN_EMAIL and LOGIN_PASSWORD are required. Set them to capture app pages.')
    return false
  }
  await signOut(page)
  await page.goto(`${baseUrl}/#login`, { waitUntil: 'load', timeout: 20000 })
  await page.waitForTimeout(2000)

  const loginForm = page.locator('.login-form')
  const nav = page.locator('nav')

  await page.waitForSelector('.login-form, .setup-form, nav', { state: 'visible', timeout: 45000 }).catch(() => null)
  await page.waitForTimeout(500)

  let formVisible = await loginForm.isVisible().catch(() => false)

  if (!formVisible) {
    const hasNav = await nav.isVisible().catch(() => false)
    if (hasNav) {
      console.log('Already in app (nav visible).')
      return true
    }
    const setupVisible = await page.locator('.setup-form').isVisible().catch(() => false)
    if (setupVisible) {
      console.warn('Setup page is shown; run IPAM setup first (create initial admin), then run this script.')
      return false
    }
    const landingVisible = await page.locator('.landing').isVisible().catch(() => false)
    if (landingVisible) {
      await page.getByRole('button', { name: /log in/i }).first().click().catch(() => null)
      await page.waitForTimeout(1500)
      formVisible = await loginForm.waitFor({ state: 'visible', timeout: 10000 }).catch(() => false)
    }
  }

  if (!formVisible) {
    const stillLoading = await page.locator('.loading-message').isVisible().catch(() => false)
    if (stillLoading) {
      console.warn('App still loading; /api/auth/me (401) and /api/setup/status must complete. Ensure the API is running and reachable.')
    } else {
      console.warn('Login form did not appear. A 401 from /api/auth/me is normal when not logged in; the app also needs /api/setup/status to succeed so it can show the login form.')
    }
    return false
  }

  await page.locator('input[type="email"]').fill(loginEmail)
  await page.locator('input[type="password"]').fill(loginPassword)
  await page.locator('button.login-submit').click()
  await page.waitForSelector('nav', { state: 'visible', timeout: NAV_TIMEOUT_MS })
  console.log('Logged in.')
  return true
}

async function waitForApp(page) {
  await page.waitForSelector('nav', { state: 'visible', timeout: NAV_TIMEOUT_MS })
}

async function main() {
  await mkdir(outDir, { recursive: true })

  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext({
    viewport: { width: 1200, height: 800 },
    deviceScaleFactor: 2,
  })
  const page = await context.newPage()

  const loggedIn = await login(page)
  if (!loggedIn) {
    await browser.close()
    console.error('Cannot capture: login required. Set LOGIN_EMAIL and LOGIN_PASSWORD.')
    process.exit(1)
  }

  for (const capture of captures) {
    const { url, name, base, beforeScreenshot } = capture
    for (const theme of ['light', 'dark']) {
      try {
        await setThemeInStorage(page, theme)
        await page.goto(url, { waitUntil: 'networkidle', timeout: 20000 })
        await waitForApp(page)
        await page.waitForTimeout(600)

        if (typeof beforeScreenshot === 'function') {
          await beforeScreenshot(page)
        }

        const filename = `${base}-${theme}.png`
        const filepath = join(outDir, filename)
        await page.screenshot({ path: filepath, fullPage: false })
        console.log('Captured:', name, theme, '->', filename)
      } catch (err) {
        console.error('Failed', name, theme, ':', err.message)
      }
    }
  }

  await browser.close()
  console.log('Done. Screenshots in public/images/')
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
