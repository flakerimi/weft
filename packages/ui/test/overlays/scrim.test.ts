// The backdrop under a modal (a Dialog's, a touch sheet's) darkens the page
// in both schemes: it is mixed from black, never from the text colour, which
// is light in dark and would lighten the page there. (happy-dom drops a
// color-mix() value, so this reads the stylesheet; the site's browser test
// TestScrimDarkensInDark computes the colour in Chrome.)
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'

const css = readFileSync(join(dirname(fileURLToPath(import.meta.url)), '../../src/styles/overlay.css'), 'utf8')

it('mixes the backdrop from black, not from the theme text colour', () => {
  const rule = /\.weft-overlay-backdrop \{([^}]*)\}/.exec(css)?.[1] ?? ''
  expect(rule).toMatch(/background: color-mix\(in srgb, black \d+%, transparent\);/)
  expect(rule).not.toMatch(/--weft-color-text/)
})
