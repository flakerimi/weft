// The weft theme is weft.base.al's: the colours of the approved design
// (docs/design/site-2026-11-21), Bricolage Grotesque and JetBrains Mono.
import { expect, it } from 'vitest'
import { contrast } from '../src/contrast.ts'
import { themeToCss } from '../src/to-css.ts'
import { colorKeys } from '../src/tokens.ts'
import { weft } from '../src/weft.ts'

it('has every colour key as #RRGGBB in both schemes', () => {
  for (const scheme of ['light', 'dark'] as const) {
    for (const key of colorKeys) {
      expect(weft[scheme][key], `${scheme}.${key}`).toMatch(/^#[0-9A-F]{6}$/)
    }
  }
})

it("uses the design's ground, ink and indigo, light and dark", () => {
  expect(weft.light).toMatchObject({ bg: '#E8ECF4', text: '#141C3A', primary: '#2B3D8F', border: '#C9D0E0' })
  expect(weft.dark).toMatchObject({ bg: '#0F1530' })
})

it('sets Bricolage Grotesque for text and JetBrains Mono for code', () => {
  expect(weft.font.sans).toMatch(/^'Bricolage Grotesque Variable', /)
  expect(weft.font.mono).toMatch(/^'JetBrains Mono Variable', /)
})

// A focus ring is a control's boundary: 3:1 against the page (WCAG 1.4.11).
it('draws focus at 3:1 or more on the page, light and dark', () => {
  for (const scheme of ['light', 'dark'] as const) {
    for (const ground of [weft[scheme].bg, weft[scheme].surface]) {
      expect(contrast(weft[scheme].focus, ground), `${scheme} focus on ${ground}`).toBeGreaterThanOrEqual(3)
    }
  }
})

it('renders to CSS, every derived colour reaching AA', () => {
  expect(() => themeToCss(weft)).not.toThrow()
})
