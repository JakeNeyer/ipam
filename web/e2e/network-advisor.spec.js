// @ts-check
// Network Advisor E2E tests: wizard flow, slider behavior, sizing math, adversarial edge cases.
// Requires logged-in admin (E2E_LOGIN_EMAIL + E2E_LOGIN_PASSWORD).

import { test, expect } from '@playwright/test'

// --- Helpers ---

async function login(page) {
  const email = process.env.E2E_LOGIN_EMAIL
  const password = process.env.E2E_LOGIN_PASSWORD
  if (!email || !password) return false
  await page.goto('/#/login')
  await page.waitForSelector('.login-form', { state: 'visible', timeout: 10000 })
  await page.fill('input[type="email"]', email)
  await page.fill('input[type="password"]', password)
  await page.click('button[type="submit"]')
  await expect(page.locator('.nav')).toBeVisible({ timeout: 10000 })
  return true
}

async function goToAdvisor(page) {
  await page.goto('/#/network-advisor')
  await expect(page.locator('text=Network Advisor').first()).toBeVisible({ timeout: 5000 })
}

/** Click the Next button */
async function clickNext(page) {
  await page.click('button:has-text("Next")')
}

/** Click the Back button */
async function clickBack(page) {
  await page.click('button:has-text("Back")')
}

/** Get the visible step heading text */
function stepHeading(page) {
  return page.locator('section.card.section h2')
}

/** Get all advisor environment cards */
function envCards(page) {
  return page.locator('article.advisor-env-card')
}

/** Get the networks range slider inside an env card */
function networksSlider(card) {
  return card.locator('.networks-slider')
}

/** Get the networks number input inside an env card */
function networksInput(card) {
  return card.locator('.networks-input')
}

/** Get the sizing detail text inside an env card */
function sizingDetail(card) {
  return card.locator('.env-sizing-detail')
}

/** Set a range input to a specific value via JavaScript (avoids drag flakiness) */
async function setSliderValue(page, slider, value) {
  await slider.evaluate((el, val) => {
    const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype,
      'value',
    ).set
    nativeInputValueSetter.call(el, val)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  }, String(value))
}

/** Get the aggregate result card */
function resultCard(page) {
  return page.locator('article.advisor-result-card')
}

/** Extract a number from text content (strips commas, finds first number) */
async function extractNumber(locator) {
  const text = await locator.textContent()
  const match = text.replace(/,/g, '').match(/[\d]+/)
  return match ? Number(match[0]) : NaN
}

// --- Tests ---

test.describe('Network Advisor', () => {
  test.beforeEach(async ({ page }) => {
    const loggedIn = await login(page)
    if (!loggedIn) {
      test.skip(true, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set')
      return
    }
    await goToAdvisor(page)
  })

  // ==================== WIZARD NAVIGATION ====================

  test.describe('wizard navigation', () => {
    test('starts at step 1 with valid default CIDR', async ({ page }) => {
      await expect(stepHeading(page)).toHaveText(/Step 1/)
      // Default CIDR is 10.0.0.0/8
      const cidrInput = page.locator('#advisor-start-cidr')
      await expect(cidrInput).toHaveValue('10.0.0.0/8')
      // Should show private CIDR detected
      await expect(page.locator('.ok:has-text("Private CIDR detected")')).toBeVisible()
    })

    test('can navigate forward through all 5 steps and back', async ({ page }) => {
      // Step 1 → 2
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 2/)

      // Step 2 → 3
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 3/)

      // Step 3 → 4
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)

      // Step 4 → 5
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 5/)

      // Back to 4
      await clickBack(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)

      // Back to 1
      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await expect(stepHeading(page)).toHaveText(/Step 1/)
    })

    test('Next is disabled on step 1 with invalid CIDR', async ({ page }) => {
      const cidrInput = page.locator('#advisor-start-cidr')
      await cidrInput.fill('not-a-cidr')
      await expect(page.locator('.error:has-text("Enter a valid CIDR")')).toBeVisible()
      const nextBtn = page.locator('button:has-text("Next")')
      await expect(nextBtn).toBeDisabled()
    })

    test('Back is disabled on step 1', async ({ page }) => {
      const backBtn = page.locator('button:has-text("Back")')
      await expect(backBtn).toBeDisabled()
    })
  })

  // ==================== STEP 1: BASE CIDR ====================

  test.describe('step 1 — base CIDR', () => {
    test('selecting a hint card updates the CIDR input', async ({ page }) => {
      // Click the compact private space hint
      await page.click('.hint-card:has-text("Compact private space")')
      const cidrInput = page.locator('#advisor-start-cidr')
      await expect(cidrInput).toHaveValue('192.168.0.0/16')
    })

    test('typing a custom non-RFC1918 CIDR shows warning', async ({ page }) => {
      const cidrInput = page.locator('#advisor-start-cidr')
      await cidrInput.fill('203.0.113.0/24')
      await expect(page.locator('.warn:has-text("not in an RFC 1918")')).toBeVisible()
      // But Next should still be enabled (valid CIDR)
      const nextBtn = page.locator('button:has-text("Next")')
      await expect(nextBtn).toBeEnabled()
    })

    test('each RFC1918 option is selectable and valid', async ({ page }) => {
      const cidrs = ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16']
      for (const cidr of cidrs) {
        const cidrInput = page.locator('#advisor-start-cidr')
        await cidrInput.fill(cidr)
        await expect(page.locator('.ok:has-text("Private CIDR detected")')).toBeVisible()
      }
    })
  })

  // ==================== STEP 2: ENVIRONMENTS ====================

  test.describe('step 2 — environments', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page) // → step 2
    })

    test('default template has Dev, Test, Prod', async ({ page }) => {
      const envInputs = page.locator('.env-name')
      await expect(envInputs).toHaveCount(3)
      await expect(envInputs.nth(0)).toHaveValue('Dev')
      await expect(envInputs.nth(1)).toHaveValue('Test')
      await expect(envInputs.nth(2)).toHaveValue('Prod')
    })

    test('selecting Cloud-specific template switches to AWS/Azure/GCP', async ({ page }) => {
      await page.click('.hint-card:has-text("Cloud-specific")')
      const envInputs = page.locator('.env-name')
      await expect(envInputs).toHaveCount(3)
      await expect(envInputs.nth(0)).toHaveValue('AWS')
      await expect(envInputs.nth(1)).toHaveValue('Azure')
      await expect(envInputs.nth(2)).toHaveValue('GCP')
    })

    test('selecting Hybrid template switches to Cloud/On-Prem', async ({ page }) => {
      await page.click('.hint-card:has-text("Hybrid topology")')
      const envInputs = page.locator('.env-name')
      await expect(envInputs).toHaveCount(2)
      await expect(envInputs.nth(0)).toHaveValue('Cloud')
      await expect(envInputs.nth(1)).toHaveValue('On-Prem')
    })

    test('add and remove environment', async ({ page }) => {
      await page.click('button:has-text("Add environment")')
      await expect(page.locator('.env-name')).toHaveCount(4)

      // Remove the last one
      const removeButtons = page.locator('button:has-text("Remove")')
      await removeButtons.last().click()
      await expect(page.locator('.env-name')).toHaveCount(3)
    })

    test('cannot remove last environment', async ({ page }) => {
      // Remove until 1 left
      while ((await page.locator('.env-name').count()) > 1) {
        await page.locator('button:has-text("Remove")').first().click()
      }
      await expect(page.locator('.env-name')).toHaveCount(1)
      await expect(page.locator('button:has-text("Remove")')).toBeDisabled()
    })

    test('Next disabled when all environment names are empty', async ({ page }) => {
      // Clear all names
      const envInputs = page.locator('.env-name')
      const count = await envInputs.count()
      for (let i = 0; i < count; i++) {
        await envInputs.nth(i).fill('')
      }
      const nextBtn = page.locator('button:has-text("Next")')
      await expect(nextBtn).toBeDisabled()
    })
  })

  // ==================== STEP 4: BLOCK SIZING SLIDERS ====================

  test.describe('step 4 — block sizing', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4
      await expect(stepHeading(page)).toHaveText(/Step 4/)
    })

    test('shows one card per environment with networks slider and input', async ({ page }) => {
      const cards = envCards(page)
      await expect(cards).toHaveCount(3) // Dev, Test, Prod
      for (let i = 0; i < 3; i++) {
        const card = cards.nth(i)
        await expect(networksSlider(card)).toBeVisible()
        await expect(networksInput(card)).toBeVisible()
        await expect(sizingDetail(card)).toBeVisible()
      }
    })

    test('networks slider updates number input and detail text', async ({ page }) => {
      const card = envCards(page).first()
      await setSliderValue(page, networksSlider(card), 5)
      await expect(networksInput(card)).toHaveValue('5')
      // Detail should show IPs per network and total IPs
      const detail = await sizingDetail(card).textContent()
      expect(detail).toMatch(/IPs per network/)
      expect(detail).toMatch(/IPs total/)
    })

    test('typing in networks input updates slider and sizing', async ({ page }) => {
      const card = envCards(page).first()
      const input = networksInput(card)
      await input.fill('10')
      // Slider should reflect the typed value
      await expect(networksSlider(card)).toHaveValue('10')
      // Detail should update
      const detail = await sizingDetail(card).textContent()
      expect(detail).toMatch(/IPs total/)
    })

    test('displays IPs per network based on network count', async ({ page }) => {
      const card = envCards(page).first()
      await setSliderValue(page, networksSlider(card), 1)
      const detail1 = await sizingDetail(card).textContent()

      await setSliderValue(page, networksSlider(card), 4)
      const detail4 = await sizingDetail(card).textContent()

      // Both should contain valid IP numbers
      expect(detail1).toMatch(/\d+ IPs per network/)
      expect(detail4).toMatch(/\d+ IPs per network/)
    })

    test('aggregate result section is visible with numbers', async ({ page }) => {
      const result = resultCard(page)
      await expect(result).toBeVisible()
      await expect(result.locator('text=Required host IPs')).toBeVisible()
      await expect(result.locator('text=Total block IPs consumed')).toBeVisible()
      await expect(result.locator('text=Planned subnet capacity')).toBeVisible()
    })

    test('progress bar is visible', async ({ page }) => {
      await expect(page.locator('.ip-capacity-bar')).toBeVisible()
      await expect(page.locator('.ip-capacity-used')).toBeVisible()
    })

    test('changing one env does NOT change another env networks value', async ({ page }) => {
      const cards = envCards(page)
      const card0 = cards.nth(0)
      const card1 = cards.nth(1)

      // Read initial networks for card1
      const initialNetworks = await networksInput(card1).inputValue()

      // Change card0 networks drastically
      await setSliderValue(page, networksSlider(card0), 1)

      // Card1 networks should be unchanged
      const afterNetworks = await networksInput(card1).inputValue()
      expect(afterNetworks).toBe(initialNetworks)
    })

    test('slider can be dragged rapidly without breaking', async ({ page }) => {
      const card = envCards(page).first()
      const slider = networksSlider(card)

      // Rapidly change value multiple times
      for (const val of [1, 3, 5, 10, 5, 2]) {
        await setSliderValue(page, slider, val)
      }

      // After rapid changes, the value should be the last one we set
      await expect(networksInput(card)).toHaveValue('2')
    })
  })

  // ==================== ADVERSARIAL: SIZING MATH ====================

  test.describe('adversarial — sizing correctness', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4
    })

    test('tiny CIDR /28: network slider max is constrained', async ({ page }) => {
      // Go back to step 1 and set a tiny CIDR
      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      const cidrInput = page.locator('#advisor-start-cidr')
      await cidrInput.fill('192.168.1.0/28')
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4

      // /28 = 16 total IPs. With 3 envs, max networks per env is very small.
      const cards = envCards(page)
      for (let i = 0; i < await cards.count(); i++) {
        const card = cards.nth(i)
        const maxN = Number(await networksSlider(card).getAttribute('max'))
        expect(maxN).toBeLessThanOrEqual(16)
        expect(maxN).toBeGreaterThanOrEqual(1)
      }
    })

    test('large CIDR /8: network slider allows many networks', async ({ page }) => {
      // Already on /8 by default. Check first env card.
      const card = envCards(page).first()
      const maxN = Number(await networksSlider(card).getAttribute('max'))
      // /8 = 16M IPs. Max networks should be large.
      expect(maxN).toBeGreaterThan(100)
    })

    test('all envs at 1 network fits any CIDR', async ({ page }) => {
      const cards = envCards(page)
      for (let i = 0; i < await cards.count(); i++) {
        await setSliderValue(page, networksSlider(cards.nth(i)), 1)
      }

      // Should show "Fits in" message
      await expect(resultCard(page).locator('.ok:has-text("Fits in")')).toBeVisible()
    })

    test('maxing out one env reduces max networks for others', async ({ page }) => {
      // Use /16 = 65536 IPs
      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await page.locator('#advisor-start-cidr').fill('192.168.0.0/16')
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      const cards = envCards(page)
      const card0 = cards.nth(0)
      const card1 = cards.nth(1)

      // Record card1 max networks before
      const maxBefore = Number(await networksSlider(card1).getAttribute('max'))

      // Set first env to a large number of networks
      const card0Max = Number(await networksSlider(card0).getAttribute('max'))
      await setSliderValue(page, networksSlider(card0), Math.min(card0Max, 100))

      // Second env's max networks should be reduced
      const maxAfter = Number(await networksSlider(card1).getAttribute('max'))
      expect(maxAfter).toBeLessThanOrEqual(maxBefore)
      expect(maxAfter).toBeGreaterThanOrEqual(1)
    })

    test('block IPs consumed is always >= required host IPs', async ({ page }) => {
      const configs = [1, 3, 10]

      for (const networks of configs) {
        const card = envCards(page).first()
        await setSliderValue(page, networksSlider(card), networks)

        const required = await extractNumber(
          resultCard(page).locator('div:has-text("Required host IPs")').first(),
        )
        const consumed = await extractNumber(
          resultCard(page).locator('div:has-text("Total block IPs consumed")').first(),
        )

        expect(consumed).toBeGreaterThanOrEqual(required)
      }
    })
  })

  // ==================== ADVERSARIAL: EDGE CASES ====================

  test.describe('adversarial — edge cases', () => {
    test('switching base CIDR mid-wizard resets sizing correctly', async ({ page }) => {
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4

      // Note the current block IPs
      const ipsBefore = await extractNumber(
        resultCard(page).locator('div:has-text("Total block IPs consumed")').first(),
      )

      // Go back to step 1 and switch CIDR
      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await page.locator('#advisor-start-cidr').fill('172.16.0.0/12')
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      // Sizing should still be valid (not NaN or broken)
      const ipsAfter = await extractNumber(
        resultCard(page).locator('div:has-text("Total block IPs consumed")').first(),
      )
      expect(ipsAfter).toBeGreaterThan(0)
      expect(Number.isFinite(ipsAfter)).toBe(true)
    })

    test('single environment with maximum networks shows valid numbers', async ({ page }) => {
      // Remove all but one environment
      await clickNext(page) // → step 2
      while ((await page.locator('.env-name').count()) > 1) {
        await page.locator('button:has-text("Remove")').first().click()
      }
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4

      const card = envCards(page).first()
      // Max out the networks slider
      const maxN = Number(await networksSlider(card).getAttribute('max'))
      await setSliderValue(page, networksSlider(card), maxN)

      const consumed = await extractNumber(
        resultCard(page).locator('div:has-text("Total block IPs consumed")').first(),
      )
      expect(consumed).toBeGreaterThan(0)
      expect(Number.isFinite(consumed)).toBe(true)
    })

    test('many environments (6+) all fit in /8', async ({ page }) => {
      await clickNext(page) // → step 2

      // Add 3 more environments (6 total)
      for (let i = 0; i < 3; i++) {
        await page.click('button:has-text("Add environment")')
      }
      await expect(page.locator('.env-name')).toHaveCount(6)

      // Name them all
      const envInputs = page.locator('.env-name')
      for (let i = 0; i < 6; i++) {
        await envInputs.nth(i).fill(`Env-${i + 1}`)
      }

      await clickNext(page) // → step 3
      await clickNext(page) // → step 4

      // Set all to modest values
      const cards = envCards(page)
      await expect(cards).toHaveCount(6)
      for (let i = 0; i < 6; i++) {
        await setSliderValue(page, networksSlider(cards.nth(i)), 2)
      }

      // Should fit in /8
      await expect(resultCard(page).locator('.ok:has-text("Fits in")')).toBeVisible()
    })

    test('exceeding base CIDR shows warning, not crash', async ({ page }) => {
      // Use tiny /24
      await page.locator('#advisor-start-cidr').fill('10.0.0.0/24')
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4

      // Try to set large values — sliders will constrain
      const cards = envCards(page)
      for (let i = 0; i < await cards.count(); i++) {
        const maxN = Number(await networksSlider(cards.nth(i)).getAttribute('max'))
        await setSliderValue(page, networksSlider(cards.nth(i)), maxN)
      }

      // Result section should show either "Fits" or "Exceeds" — not be broken
      const resultText = await resultCard(page).textContent()
      expect(resultText).toMatch(/Fits in|Exceeds base CIDR/)
    })
  })

  // ==================== STEP 5: SUMMARY ====================

  test.describe('step 5 — summary', () => {
    test('summary shows correct environment count and block info', async ({ page }) => {
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4
      await clickNext(page) // → step 5
      await expect(stepHeading(page)).toHaveText(/Step 5/)

      // Should show summary cards for each environment
      const summaryCards = page.locator('.summary-card')
      await expect(summaryCards).toHaveCount(3) // Dev, Test, Prod

      // Should show network blocks count
      await expect(page.locator('text=Network blocks to be created')).toBeVisible()

      // Should show required host IPs
      await expect(page.locator('p:has-text("Required host IPs")')).toBeVisible()

      // Should show generate button
      await expect(page.locator('button:has-text("Generate resources from plan")')).toBeVisible()
    })

    test('Start over button returns to step 1', async ({ page }) => {
      await clickNext(page) // → step 2
      await clickNext(page) // → step 3
      await clickNext(page) // → step 4
      await clickNext(page) // → step 5

      await page.click('button:has-text("Start over")')
      await expect(stepHeading(page)).toHaveText(/Step 1/)
    })
  })
})
