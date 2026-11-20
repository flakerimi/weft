import { expect, it } from 'vitest'
import { neutral } from '../src/neutral.ts'
import { colorKeys } from '../src/tokens.ts'

it('has every colour key as #RRGGBB in both schemes', () => {
  for (const scheme of ['light', 'dark'] as const) {
    for (const key of colorKeys) {
      expect(neutral[scheme][key], `${scheme}.${key}`).toMatch(/^#[0-9A-F]{6}$/)
    }
  }
})
