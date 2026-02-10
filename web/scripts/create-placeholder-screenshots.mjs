#!/usr/bin/env node
/**
 * Create minimal placeholder PNGs so user guide images display until real
 * screenshots are captured (run: npm run screenshot-docs with dev server up).
 */
import { writeFileSync, mkdirSync } from 'fs'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const outDir = join(__dirname, '..', 'public', 'images')

// Minimal 1x1 PNG (valid PNG file)
const MINI_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64'
)

const bases = ['dashboard', 'networks', 'environments', 'admin', 'reserved-blocks', 'subnet-calculator', 'command-palette', 'cidr-wizard']
const themes = ['light', 'dark']

mkdirSync(outDir, { recursive: true })
for (const base of bases) {
  for (const theme of themes) {
    const path = join(outDir, `${base}-${theme}.png`)
    writeFileSync(path, MINI_PNG)
    console.log('Created:', `${base}-${theme}.png`)
  }
}
console.log('Placeholder images created. Run "npm run screenshot-docs" (with dev server up) to replace with real screenshots.')
