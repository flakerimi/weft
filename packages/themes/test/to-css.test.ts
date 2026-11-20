import { describe, expect, it } from 'vitest'
import { themeToCss } from '../src/to-css.ts'
import { neutral } from '../src/neutral.ts'
import type { Theme } from '../src/tokens.ts'

describe('themeToCss', () => {
  const css = themeToCss(neutral)

  it('emits colour, metric, motion and curve tokens', () => {
    expect(css).toContain('--weft-color-bg: #FAFAFA;')
    expect(css).toContain('--weft-radius-control: 8px;')
    expect(css).toContain('--weft-space-4: 16px;')
    expect(css).toContain('--weft-duration-control: 150ms;')
    expect(css).toContain('--weft-ease-standard: cubic-bezier(0.2, 0, 0, 1);')
  })

  it('emits the dark scheme for the attribute and the system preference', () => {
    expect(css).toMatch(/\[data-weft-scheme="dark"\]\s*\{[^}]*--weft-color-bg: #09090B;/)
    expect(css).toContain('@media (prefers-color-scheme: dark)')
    expect(css).toContain(':root:not([data-weft-scheme="light"])')
  })

  it('emits density for input and surface', () => {
    expect(css).toMatch(/\[data-weft-input="touch"\]\s*\{[^}]*--weft-control-height: 44px;/)
    expect(css).toMatch(/\[data-weft-input="pointer"\]\s*\{[^}]*--weft-control-height: 36px;/)
    expect(css).toMatch(/\[data-weft-surface="landing"\]\s*\{[^}]*--weft-text-base: 18px;/)
    expect(css).toMatch(/\[data-weft-surface="app"\]\s*\{[^}]*--weft-text-base: 14px;/)
  })

  it('rejects a theme with a missing colour', () => {
    const broken = structuredClone(neutral) as Theme
    delete (broken.light as Partial<Theme['light']>).primary
    expect(() => themeToCss(broken)).toThrow('theme "neutral" is missing light.primary')
  })
})
