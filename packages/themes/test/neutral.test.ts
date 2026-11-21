import { expect, it } from 'vitest'
import { contrast } from './contrast.ts'
import { neutral } from '../src/neutral.ts'
import { colorKeys } from '../src/tokens.ts'

it('has every colour key as #RRGGBB in both schemes', () => {
  for (const scheme of ['light', 'dark'] as const) {
    for (const key of colorKeys) {
      expect(neutral[scheme][key], `${scheme}.${key}`).toMatch(/^#[0-9A-F]{6}$/)
    }
  }
})

it('draws text on every tone at 4.5:1 or better (WCAG AA), in both schemes', () => {
  for (const scheme of ['light', 'dark'] as const) {
    const c = neutral[scheme]
    for (const tone of ['danger', 'warning', 'success'] as const) {
      expect(contrast(c[tone], c[`${tone}-fg`]), `${scheme}.${tone}`).toBeGreaterThanOrEqual(4.5)
    }
  }
})
