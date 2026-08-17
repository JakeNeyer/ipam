// @ts-check
// Network Advisor E2E tests: wizard flow, slider behavior, sizing math, adversarial edge cases.
// Requires logged-in admin (E2E_LOGIN_EMAIL + E2E_LOGIN_PASSWORD).

import { test, expect } from '@playwright/test'
import { login, ensureOrgSelected } from './helpers.js'

const CIDR_INPUT = '#advisor-base-cidr'

async function goToAdvisor(page) {
  await page.goto('/#network-advisor')
  await expect(page.locator('h1.page-title', { hasText: 'Network Advisor' })).toBeVisible({ timeout: 5000 })
}

/** Click the Next button */
async function clickNext(page) {
  await page.locator('.wizard-actions button:has-text("Next")').click()
}

/** Click the Back button */
async function clickBack(page) {
  await page.locator('.wizard-actions button:has-text("Back")').click()
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

test.describe('Network Advisor', () => {
  test.beforeEach(async ({ page }) => {
    const loggedIn = await login(page)
    if (!loggedIn) {
      test.skip(true, 'E2E_LOGIN_EMAIL and E2E_LOGIN_PASSWORD must be set')
      return
    }
    await ensureOrgSelected(page)
    await goToAdvisor(page)
  })

  test.describe('wizard navigation', () => {
    test('starts at step 1 with valid default CIDR', async ({ page }) => {
      await expect(stepHeading(page)).toHaveText(/Step 1/)
      const cidrInput = page.locator(CIDR_INPUT)
      await expect(cidrInput).toHaveValue('10.0.0.0/8')
      await expect(page.locator('.ok:has-text("Base range set")')).toBeVisible()
    })

    test('can navigate forward through all 5 steps and back', async ({ page }) => {
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 2/)

      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 3/)

      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)

      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 5/)

      await clickBack(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)

      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await expect(stepHeading(page)).toHaveText(/Step 1/)
    })

    test('Next is disabled on step 1 with invalid CIDR', async ({ page }) => {
      const cidrInput = page.locator(CIDR_INPUT)
      await cidrInput.fill('not-a-cidr')
      await expect(page.locator('.error:has-text("Enter a valid CIDR")')).toBeVisible()
      const nextBtn = page.locator('.wizard-actions button:has-text("Next")')
      await expect(nextBtn).toBeDisabled()
    })

    test('Back is disabled on step 1', async ({ page }) => {
      const backBtn = page.locator('.wizard-actions button:has-text("Back")')
      await expect(backBtn).toBeDisabled()
    })
  })

  test.describe('step 1 — base CIDR', () => {
    test('selecting a hint card updates the CIDR input', async ({ page }) => {
      await page.click('.hint-card:has-text("Compact private range")')
      const cidrInput = page.locator(CIDR_INPUT)
      await expect(cidrInput).toHaveValue('192.168.0.0/16')
    })

    test('typing a custom non-RFC1918 CIDR shows warning', async ({ page }) => {
      const cidrInput = page.locator(CIDR_INPUT)
      await cidrInput.fill('203.0.113.0/24')
      await expect(page.locator('.warn:has-text("not in a private range")')).toBeVisible()
      const nextBtn = page.locator('.wizard-actions button:has-text("Next")')
      await expect(nextBtn).toBeEnabled()
    })

    test('each RFC1918 option is selectable and valid', async ({ page }) => {
      const cidrs = ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16']
      for (const cidr of cidrs) {
        const cidrInput = page.locator(CIDR_INPUT)
        await cidrInput.fill(cidr)
        await expect(page.locator('.ok:has-text("Base range set")')).toBeVisible()
      }
    })
  })

  test.describe('step 2 — environments', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page)
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

      const removeButtons = page.locator('button:has-text("Remove")')
      await removeButtons.last().click()
      await expect(page.locator('.env-name')).toHaveCount(3)
    })

    test('cannot remove last environment', async ({ page }) => {
      while ((await page.locator('.env-name').count()) > 1) {
        await page.locator('button:has-text("Remove")').first().click()
      }
      await expect(page.locator('.env-name')).toHaveCount(1)
      await expect(page.locator('button:has-text("Remove")')).toBeDisabled()
    })

    test('Next disabled when all environment names are empty', async ({ page }) => {
      const envInputs = page.locator('.env-name')
      const count = await envInputs.count()
      for (let i = 0; i < count; i++) {
        await envInputs.nth(i).fill('')
      }
      const nextBtn = page.locator('.wizard-actions button:has-text("Next")')
      await expect(nextBtn).toBeDisabled()
    })
  })

  test.describe('step 4 — block sizing', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)
    })

    test('shows one card per environment with networks slider and input', async ({ page }) => {
      const cards = envCards(page)
      await expect(cards).toHaveCount(3)
      for (let i = 0; i < 3; i++) {
        const card = cards.nth(i)
        await expect(networksSlider(card)).toBeVisible()
        await expect(networksInput(card)).toBeVisible()
        await expect(sizingDetail(card)).toBeVisible()
      }
    })

    test('networks slider updates number input and detail text', async ({ page }) => {
      const card = envCards(page).first()
      await setSliderValue(page, networksSlider(card), 500)
      const val = Number(await networksInput(card).inputValue())
      expect(val).toBeGreaterThan(1)
      const detail = await sizingDetail(card).textContent()
      expect(detail).toMatch(/per block/)
      expect(detail).toMatch(/IPs total/)
    })

    test('typing in networks input updates slider and sizing', async ({ page }) => {
      const card = envCards(page).first()
      const input = networksInput(card)
      await input.fill('10')
      await expect(input).toHaveValue('10')
      const sliderVal = Number(await networksSlider(card).inputValue())
      expect(sliderVal).toBeGreaterThan(0)
      const detail = await sizingDetail(card).textContent()
      expect(detail).toMatch(/IPs total/)
    })

    test('displays IPs per block based on network count', async ({ page }) => {
      const card = envCards(page).first()
      await networksInput(card).fill('1')
      const detail1 = await sizingDetail(card).textContent()

      await networksInput(card).fill('4')
      const detail4 = await sizingDetail(card).textContent()

      expect(detail1).toMatch(/IPs/)
      expect(detail4).toMatch(/IPs/)
      expect(detail1).not.toBe(detail4)
    })

    test('aggregate result section is visible with numbers', async ({ page }) => {
      const result = resultCard(page)
      await expect(result).toBeVisible()
      await expect(result.locator('text=Network blocks')).toBeVisible()
      await expect(result.locator('text=Block IPs consumed')).toBeVisible()
    })

    test('progress bar is visible', async ({ page }) => {
      await expect(page.locator('.ip-capacity-bar')).toBeVisible()
      await expect(page.locator('.ip-capacity-used')).toBeVisible()
    })

    test('changing one env does NOT change another env networks value', async ({ page }) => {
      const cards = envCards(page)
      const card0 = cards.nth(0)
      const card1 = cards.nth(1)

      const initialNetworks = await networksInput(card1).inputValue()
      await networksInput(card0).fill('1')
      const afterNetworks = await networksInput(card1).inputValue()
      expect(afterNetworks).toBe(initialNetworks)
    })

    test('slider can be dragged rapidly without breaking', async ({ page }) => {
      const card = envCards(page).first()
      const slider = networksSlider(card)

      for (const val of [100, 300, 500, 800, 200]) {
        await setSliderValue(page, slider, val)
      }

      const networks = Number(await networksInput(card).inputValue())
      expect(Number.isFinite(networks)).toBe(true)
      expect(networks).toBeGreaterThanOrEqual(1)
    })
  })

  test.describe('adversarial — sizing correctness', () => {
    test.beforeEach(async ({ page }) => {
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
    })

    test('tiny CIDR /28: network count is constrained', async ({ page }) => {
      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await expect(stepHeading(page)).toHaveText(/Step 1/)
      await page.locator(CIDR_INPUT).fill('192.168.1.0/28')
      await expect(page.locator(CIDR_INPUT)).toHaveValue('192.168.1.0/28')
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 4/)

      const cards = envCards(page)
      await expect(cards.first().locator('.env-pool-label')).toContainText(/\/3[0-2]/)
      for (let i = 0; i < await cards.count(); i++) {
        const input = networksInput(cards.nth(i))
        await expect.poll(async () => Number(await input.inputValue())).toBeLessThanOrEqual(2)
        await expect(input).toHaveAttribute('max', /^(1|2)$/)
      }
    })

    test('large CIDR /8: network input allows many networks', async ({ page }) => {
      const card = envCards(page).first()
      await networksInput(card).fill('200')
      await expect(networksInput(card)).toHaveValue('200')
    })

    test('all envs at 1 network fits any CIDR', async ({ page }) => {
      const cards = envCards(page)
      for (let i = 0; i < await cards.count(); i++) {
        await networksInput(cards.nth(i)).fill('1')
      }

      await expect(resultCard(page).locator('.ok:has-text("Fits in")')).toBeVisible()
    })

    test('each environment has an independent pool', async ({ page }) => {
      const cards = envCards(page)
      const card0 = cards.nth(0)
      const card1 = cards.nth(1)

      await networksInput(card1).fill('4')
      const before = await networksInput(card1).inputValue()
      await networksInput(card0).fill('50')
      await expect(networksInput(card1)).toHaveValue(before)
    })

    test('block IPs consumed stays a valid number as networks change', async ({ page }) => {
      const configs = [1, 3, 10]
      const consumedLocator = resultCard(page).locator('div:has-text("Block IPs consumed")').first()

      for (const networks of configs) {
        await networksInput(envCards(page).first()).fill(String(networks))
        const consumed = await extractNumber(consumedLocator)
        expect(consumed).toBeGreaterThan(0)
        expect(Number.isFinite(consumed)).toBe(true)
      }
    })
  })

  test.describe('adversarial — edge cases', () => {
    test('switching base CIDR mid-wizard resets sizing correctly', async ({ page }) => {
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      const ipsBefore = await extractNumber(
        resultCard(page).locator('div:has-text("Block IPs consumed")').first(),
      )
      expect(ipsBefore).toBeGreaterThan(0)

      await clickBack(page)
      await clickBack(page)
      await clickBack(page)
      await page.locator(CIDR_INPUT).fill('172.16.0.0/12')
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      const ipsAfter = await extractNumber(
        resultCard(page).locator('div:has-text("Block IPs consumed")').first(),
      )
      expect(ipsAfter).toBeGreaterThan(0)
      expect(Number.isFinite(ipsAfter)).toBe(true)
    })

    test('single environment with maximum networks shows valid numbers', async ({ page }) => {
      await clickNext(page)
      while ((await page.locator('.env-name').count()) > 1) {
        await page.locator('button:has-text("Remove")').first().click()
      }
      await clickNext(page)
      await clickNext(page)

      const card = envCards(page).first()
      await networksInput(card).fill('1000')
      const n = Number(await networksInput(card).inputValue())
      expect(n).toBeGreaterThan(1)

      const consumed = await extractNumber(
        resultCard(page).locator('div:has-text("Block IPs consumed")').first(),
      )
      expect(consumed).toBeGreaterThan(0)
      expect(Number.isFinite(consumed)).toBe(true)
    })

    test('many environments (6+) all fit in /8', async ({ page }) => {
      await clickNext(page)

      for (let i = 0; i < 3; i++) {
        await page.click('button:has-text("Add environment")')
      }
      await expect(page.locator('.env-name')).toHaveCount(6)

      const envInputs = page.locator('.env-name')
      for (let i = 0; i < 6; i++) {
        await envInputs.nth(i).fill(`Env-${i + 1}`)
      }

      await clickNext(page)
      await clickNext(page)

      const cards = envCards(page)
      await expect(cards).toHaveCount(6)
      for (let i = 0; i < 6; i++) {
        await networksInput(cards.nth(i)).fill('2')
      }

      await expect(resultCard(page).locator('.ok:has-text("Fits in")')).toBeVisible()
    })

    test('maxing networks on a small CIDR still shows a valid result', async ({ page }) => {
      await page.locator(CIDR_INPUT).fill('10.0.0.0/24')
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      const cards = envCards(page)
      for (let i = 0; i < await cards.count(); i++) {
        await networksInput(cards.nth(i)).fill('64')
      }

      const resultText = await resultCard(page).textContent()
      expect(resultText).toMatch(/Fits in|Exceeds base/)
    })
  })

  test.describe('step 5 — summary', () => {
    test('summary shows correct environment count and block info', async ({ page }) => {
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
      await expect(stepHeading(page)).toHaveText(/Step 5/)

      const summaryCards = page.locator('.summary-card')
      await expect(summaryCards).toHaveCount(3)

      await expect(page.locator('section.card.section p').filter({ hasText: 'Network blocks:' })).toBeVisible()
      await expect(page.locator('section.card.section p').filter({ hasText: 'Block IPs consumed:' })).toBeVisible()
      await expect(page.getByRole('button', { name: /Generate plan/ })).toBeVisible()
    })

    test('Start over button returns to step 1', async ({ page }) => {
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)
      await clickNext(page)

      await page.locator('.wizard-actions button:has-text("Start over")').click()
      await page.getByRole('dialog').getByRole('button', { name: 'Start over' }).click()
      await expect(stepHeading(page)).toHaveText(/Step 1/)
    })
  })
})
