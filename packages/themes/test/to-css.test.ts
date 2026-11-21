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
  })

  it('emits control sizes that stay the same for pointer and touch', () => {
    const root = /:root, \[data-weft-scheme="light"\] \{[^}]*\}/.exec(css)![0]
    expect(root).toContain('--weft-control-sm: 28px;')
    expect(root).toContain('--weft-control-md: 36px;')
    expect(root).toContain('--weft-control-lg: 44px;')
    // A control keeps its drawn size on touch; its hit area grows instead (--weft-hit-min).
    for (const m of css.matchAll(/data-weft-input[^{]*\{[^}]*\}/g)) expect(m[0]).not.toMatch(/--weft-control-(sm|md|lg)/)
  })

  it('emits a foreground for every tone, in both schemes', () => {
    for (const tone of ['danger', 'warning', 'success']) {
      expect(css).toMatch(new RegExp(`:root, \\[data-weft-scheme="light"\\] \\{[^}]*--weft-color-${tone}-fg: #`))
      expect(css).toMatch(new RegExp(`\\[data-weft-scheme="dark"\\] \\{[^}]*--weft-color-${tone}-fg: #`))
    }
  })

  it('sizes for touch from the media query when no provider says otherwise', () => {
    expect(css).toMatch(
      /@media \(pointer: coarse\) \{\s*:root:not\(\[data-weft-input\]\) \{[^}]*--weft-control-height: 44px;[^}]*--weft-hit-min: 44px;/,
    )
  })

  it('sizes text in rem so the reader\'s browser text size still counts', () => {
    expect(css).toMatch(/\[data-weft-surface="app"\]\s*\{[^}]*--weft-text-base: 0\.875rem;/)
    expect(css).toMatch(/\[data-weft-surface="landing"\]\s*\{[^}]*--weft-text-base: 1\.125rem;/)
  })

  it.each([
    ['font.weightBody', (t: Theme) => { delete (t.font as Partial<Theme['font']>).weightBody }],
    ['space.8', (t: Theme) => { delete (t.space as Partial<Theme['space']>)[8] }],
    ['radius.card', (t: Theme) => { delete (t.radius as Partial<Theme['radius']>).card }],
    ['ease.standard', (t: Theme) => { delete (t.ease as Partial<Theme['ease']>).standard }],
    ['motion', (t: Theme) => { delete (t as Partial<Theme>).motion }],
    ['control.md', (t: Theme) => { delete (t.control as Partial<Theme['control']>).md }],
  ])('rejects a theme missing %s', (path, remove) => {
    const broken = structuredClone(neutral) as Theme
    remove(broken)
    expect(() => themeToCss(broken)).toThrow(`theme "neutral" is missing ${path}`)
  })

  it('rejects a theme with a missing colour', () => {
    const broken = structuredClone(neutral) as Theme
    delete (broken.light as Partial<Theme['light']>).primary
    expect(() => themeToCss(broken)).toThrow('theme "neutral" is missing light.primary')
  })
})
