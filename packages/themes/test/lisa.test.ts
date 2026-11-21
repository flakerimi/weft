import { describe, expect, it } from 'vitest'
import tokens from '../src/lisa/tokens.json' with { type: 'json' }
import { lisa } from '../src/lisa.ts'
import { themeToCss } from '../src/to-css.ts'

describe('lisa theme', () => {
  const theme = lisa(tokens)

  it('takes grounds, accent, font, radius and motion from the contract', () => {
    expect(theme.light.bg).toBe('#F1F5F9')
    expect(theme.dark.bg).toBe('#0F172A')
    expect(theme.light.primary).toBe('#6B7D2E')
    expect(theme.font.sans.startsWith('Rubik')).toBe(true)
    expect(theme.radius.control).toBe(10)
    expect(theme.motion.control).toBe(150)
  })

  it('takes control sizes from the contract', () => {
    expect(theme.control).toEqual({ sm: 28, md: 34, lg: 44 })
  })

  it('draws each tone\'s text in the contract\'s ink that reads best on it', () => {
    expect(theme.light['danger-fg']).toBe('#F8FAFC')
    expect(theme.light['warning-fg']).toBe('#0F172A')
    expect(theme.light['success-fg']).toBe('#0F172A')
    expect(theme.dark['warning-fg']).toBe(theme.light['warning-fg'])
  })

  it('switches accent', () => {
    expect(lisa(tokens, { accent: 'violet' }).light.primary).toBe(
      tokens.color.accent['accent-violet-500'].value,
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
