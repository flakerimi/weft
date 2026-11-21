import { describe, expect, it } from 'vitest'
import tokens from '../src/lisa/tokens.json' with { type: 'json' }
import { lisa } from '../src/lisa.ts'
import { themeToCss } from '../src/to-css.ts'

describe('lisa theme', () => {
  const theme = lisa(tokens)

  it('takes grounds, accent, font, radius and motion from the contract', () => {
    expect(theme.light.bg).toBe('#F1F5F9')
    expect(theme.dark.bg).toBe('#0F172A')
    // Solid fills take the contract's $solid stops: 700 on light, 300 on dark.
    expect(theme.light.primary).toBe('#414D1C')
    expect(theme.dark.primary).toBe('#AEC162')
    // Selection outlines (the focus ring) take its $selectionOutline stops: 500 on light, 300 on dark.
    expect(theme.light.focus).toBe('#6B7D2E')
    expect(theme.dark.focus).toBe('#AEC162')
    expect(theme.font.sans.startsWith('Rubik')).toBe(true)
    expect(theme.radius.control).toBe(10)
    expect(theme.motion.control).toBe(150)
  })

  it('builds for every accent in the contract', () => {
    for (const key of Object.keys(tokens.color.accent).filter((k) => /^accent-.+-500$/.test(k))) {
      expect(() => themeToCss(lisa(tokens, { accent: key.slice(7, -4) })), key).not.toThrow()
    }
  })

  it('takes control sizes from the contract', () => {
    expect(theme.control).toEqual({ sm: 28, md: 34, lg: 44 })
  })

  it('draws danger from the contract\'s red ramp, whose $solid stops reach 4.5:1 with its inks', () => {
    // semantic.error (#D6453C) reads at 4.2:1 with warm-white and 4.1:1 with ink-900: under AA either way.
    expect(theme.light.danger).toBe(tokens.color.accent['accent-red-700'].value)
    expect(theme.dark.danger).toBe(tokens.color.accent['accent-red-300'].value)
    expect(theme.light.warning).toBe(tokens.color.semantic.warning.value)
    expect(theme.light.success).toBe(tokens.color.semantic.success.value)
  })

  it('gives solid fills the ink the contract pairs with them', () => {
    const css = themeToCss(theme)
    const light = /:root, \[data-weft-scheme="light"\] \{[^}]*\}/.exec(css)![0]
    const dark = /\[data-weft-scheme="dark"\] \{[^}]*\}/.exec(css)![0]
    expect(light).toContain(`--weft-color-primary-fg: ${tokens.color.ink['warm-white'].value};`)
    expect(dark).toContain(`--weft-color-primary-fg: ${tokens.color.ink['ink-900'].value};`)
  })

  it('switches accent', () => {
    expect(lisa(tokens, { accent: 'violet' }).light.primary).toBe(
      tokens.color.accent['accent-violet-700'].value,
    )
  })

  it('rejects an unknown accent', () => {
    expect(() => lisa(tokens, { accent: 'teal-ish' })).toThrow('unknown Lisa accent "teal-ish"')
  })

  it('has no white and no black anywhere (noWhiteAnywhere)', () => {
    const css = themeToCss(theme).toUpperCase()
    expect(css).not.toContain('#FFFFFF')
    expect(css).not.toContain('#000000')
  })
})
