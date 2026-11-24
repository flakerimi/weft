// Shadows and border widths are theme tokens, so a theme can raise a flat
// design or thicken, thin or drop its lines. The three themes give the same
// defaults, which are what the components drew before the tokens existed.
import { describe, expect, it } from 'vitest'
import tokens from '../src/lisa/tokens.json' with { type: 'json' }
import { lisa } from '../src/lisa.ts'
import { neutral } from '../src/neutral.ts'
import { themeToCss } from '../src/to-css.ts'
import type { Theme } from '../src/tokens.ts'
import { weft } from '../src/weft.ts'

const defaults = {
  '--weft-shadow-control': '0 1px 2px color-mix(in srgb, CanvasText 25%, transparent)',
  '--weft-shadow-card': 'none',
  '--weft-shadow-popup': '0 8px 24px color-mix(in srgb, CanvasText 16%, transparent)',
  '--weft-shadow-dialog': '0 8px 32px color-mix(in srgb, CanvasText 24%, transparent)',
  '--weft-border-width': '1px',
  '--weft-border-width-strong': '2px',
  '--weft-focus-width': '2px',
}

const blockOf = (css: string, selector: RegExp) => selector.exec(css)?.[0] ?? ''
const light = (css: string) => blockOf(css, /:where\(:root, \[data-weft-scheme="light"\]\) \{[^}]*\}/)
const dark = (css: string) => blockOf(css, /:where\(\[data-weft-scheme="dark"\]\) \{[^}]*\}/)
const systemDark = (css: string) => blockOf(css, /:where\(:root:not\(\[data-weft-scheme="light"\]\)\) \{[^}]*\}/)

describe.each([
  ['neutral', neutral],
  ['weft', weft],
  ['lisa', lisa(tokens)],
] as const)('the %s theme', (_, theme) => {
  const css = themeToCss(theme)

  it('writes the seven shadow and border tokens with the defaults', () => {
    for (const [name, value] of Object.entries(defaults)) expect(light(css), name).toContain(`  ${name}: ${value};`)
  })

  it('writes them where the other non-colour tokens are: once, beside the radii, not again for dark', () => {
    // A shadow's CanvasText resolves in the scheme of the element it is drawn on, so dark needs no copy.
    for (const name of Object.keys(defaults)) {
      expect(css.split(`${name}:`).length - 1, name).toBe(1)
      expect(dark(css), name).not.toContain(name)
      expect(systemDark(css), name).not.toContain(name)
    }
    expect(dark(css)).not.toContain('--weft-radius-card')
  })
})

describe('a theme with its own shadows and border widths', () => {
  const own: Theme = {
    ...structuredClone(neutral),
    name: 'raised',
    shadow: {
      control: '0 1px 3px rgb(0 0 0 / 0.3)',
      card: '0 2px 8px rgb(0 0 0 / 0.1)',
      popup: '0 4px 12px rgb(0 0 0 / 0.2)',
      dialog: '0 24px 64px rgb(0 0 0 / 0.4)',
    },
    border: { width: 0, strong: 3, focus: 3 },
  }
  const css = light(themeToCss(own))

  it('writes its values', () => {
    expect(css).toContain('  --weft-shadow-control: 0 1px 3px rgb(0 0 0 / 0.3);')
    expect(css).toContain('  --weft-shadow-card: 0 2px 8px rgb(0 0 0 / 0.1);')
    expect(css).toContain('  --weft-shadow-popup: 0 4px 12px rgb(0 0 0 / 0.2);')
    expect(css).toContain('  --weft-shadow-dialog: 0 24px 64px rgb(0 0 0 / 0.4);')
    // A zero width is a theme with no hairlines, not a missing token.
    expect(css).toContain('  --weft-border-width: 0px;')
    expect(css).toContain('  --weft-border-width-strong: 3px;')
    expect(css).toContain('  --weft-focus-width: 3px;')
  })

  it.each([
    ['shadow.popup', (t: Theme) => { delete (t.shadow as Partial<Theme['shadow']>).popup }],
    ['shadow', (t: Theme) => { delete (t as Partial<Theme>).shadow }],
    ['border.width', (t: Theme) => { delete (t.border as Partial<Theme['border']>).width }],
    ['border.focus', (t: Theme) => { delete (t.border as Partial<Theme['border']>).focus }],
  ])('is refused when it lacks %s', (path, remove) => {
    const broken = structuredClone(neutral) as Theme
    remove(broken)
    expect(() => themeToCss(broken)).toThrow(`theme "neutral" is missing ${path}`)
  })
})
