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
    // How long a toast stays before it leaves on its own.
    expect(css).toContain('--weft-duration-toast: 4000ms;')
    expect(css).toContain('--weft-ease-standard: cubic-bezier(0.2, 0, 0, 1);')
  })

  it('emits the dark scheme for the attribute and the system preference', () => {
    expect(css).toMatch(/\[data-weft-scheme="dark"\]\s*\{[^}]*--weft-color-bg: #09090B;/)
    expect(css).toContain('@media (prefers-color-scheme: dark)')
    expect(css).toContain(':root:not([data-weft-scheme="light"])')
  })

  it('emits the hit area for input: 44 px on touch, 24 px on pointer', () => {
    expect(css).toMatch(/\[data-weft-input="touch"\]\s*\{[^}]*--weft-hit-min: 44px;/)
    expect(css).toMatch(/\[data-weft-input="pointer"\]\s*\{[^}]*--weft-hit-min: 24px;/)
    // A drawn thumb or handle is at least 24 px on touch, and as drawn on pointer.
    expect(css).toMatch(/\[data-weft-input="touch"\]\s*\{[^}]*--weft-thumb-min: 24px;/)
    expect(css).toMatch(/\[data-weft-input="pointer"\]\s*\{[^}]*--weft-thumb-min: 0px;/)
    expect(css).toMatch(/@media \(pointer: coarse\) \{\s*:root:not\(\[data-weft-input\]\) \{[^}]*--weft-thumb-min: 24px;/)
    // Heights come from --weft-control-sm/md/lg alone; there is no second height token to drift from them.
    expect(css).not.toContain('--weft-control-height')
  })

  it('emits control sizes that stay the same for pointer and touch', () => {
    const root = /:root, \[data-weft-scheme="light"\] \{[^}]*\}/.exec(css)![0]
    expect(root).toContain('--weft-control-sm: 28px;')
    expect(root).toContain('--weft-control-md: 36px;')
    expect(root).toContain('--weft-control-lg: 44px;')
    // A control keeps its drawn size on touch; its hit area grows instead (--weft-hit-min).
    for (const m of css.matchAll(/data-weft-input[^{]*\{[^}]*\}/g)) expect(m[0]).not.toMatch(/--weft-control-(sm|md|lg)/)
  })

  it('chooses each tone\'s text at build: the better of the theme\'s two inks, and a tinted text', () => {
    const light = /:root, \[data-weft-scheme="light"\] \{[^}]*\}/.exec(css)![0]
    const dark = /\[data-weft-scheme="dark"\] \{[^}]*\}/.exec(css)![0]
    // Light: white-ish ink on blue and red, dark ink on amber and green.
    expect(light).toContain(`--weft-color-primary-fg: ${neutral.dark.text};`)
    expect(light).toContain(`--weft-color-danger-fg: ${neutral.dark.text};`)
    expect(light).toContain(`--weft-color-warning-fg: ${neutral.light.text};`)
    expect(light).toContain(`--weft-color-success-fg: ${neutral.light.text};`)
    // Dark: every tone is light enough for the dark ink.
    for (const tone of ['primary', 'danger', 'warning', 'success']) {
      expect(dark).toContain(`--weft-color-${tone}-fg: ${neutral.light.text};`)
      expect(light).toMatch(new RegExp(`--weft-color-${tone}-text: #[0-9A-F]{6};`))
      expect(dark).toMatch(new RegExp(`--weft-color-${tone}-text: #[0-9A-F]{6};`))
    }
  })

  it('refuses a tone neither ink reaches 4.5:1 on, in one sentence', () => {
    const grey = structuredClone(neutral)
    grey.light.warning = '#767676'
    expect(() => themeToCss(grey)).toThrow(
      'theme "neutral": neither its light text (#FAFAFA) nor its dark text (#18181B) reaches 4.5:1 on light.warning (#767676); pick a warning colour one of them can be read on.',
    )
  })

  it('refuses a colour it cannot check, in one sentence', () => {
    const named = structuredClone(neutral)
    named.dark.danger = 'red'
    expect(() => themeToCss(named)).toThrow('theme "neutral": dark.danger is "red"; give it as #RRGGBB so its contrast can be checked.')
  })

  it('sizes for touch from the media query when no provider says otherwise', () => {
    expect(css).toMatch(
      /@media \(pointer: coarse\) \{\s*:root:not\(\[data-weft-input\]\) \{[^}]*--weft-hit-min: 44px;/,
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
