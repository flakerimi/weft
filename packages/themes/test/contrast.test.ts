// Every pair of text and ground a toned component draws, read from the CSS a
// theme renders to, reaches WCAG AA (4.5:1): solid text on its tone, and
// tinted text (soft, outline, ghost) on the page, on cards and on the tinted
// fills laid over them, at rest and on hover; and solid text on the hover and
// active (pressed, current) fills of every tone, neutral included.
import { describe, expect, it } from 'vitest'
import tokens from '../src/lisa/tokens.json' with { type: 'json' }
import { contrast, mix } from '../src/contrast.ts'
import { lisa, type LisaTokens } from '../src/lisa.ts'
import { neutral } from '../src/neutral.ts'
import { themeToCss } from '../src/to-css.ts'
import { weft } from '../src/weft.ts'
import { tints, tones } from '../src/tokens.ts'
import type { Theme } from '../src/tokens.ts'

/** The custom properties of the first block whose selector is `selector`. */
function block(css: string, selector: string): Record<string, string> {
  const start = css.indexOf(`${selector} {`)
  expect(start, selector).toBeGreaterThanOrEqual(0)
  const body = css.slice(start, css.indexOf('}', start))
  return Object.fromEntries([...body.matchAll(/--weft-([\w-]+): ([^;]+);/g)].map((m) => [m[1]!, m[2]!]))
}

const accents = Object.keys(tokens.color.accent).filter((k) => /^accent-.+-500$/.test(k)).map((k) => k.slice(7, -4))
const themes: Theme[] = [neutral, weft, ...accents.map((accent) => lisa(tokens as LisaTokens, { accent }))]

describe.each(themes.map((t) => [t.name, t] as const))('%s', (_, theme) => {
  const css = themeToCss(theme)
  const schemes = {
    light: block(css, ':where(:root, [data-weft-scheme="light"])'),
    dark: { ...block(css, ':where(:root, [data-weft-scheme="light"])'), ...block(css, ':where([data-weft-scheme="dark"])') },
  }

  for (const [scheme, v] of Object.entries(schemes)) {
    const grounds = ['bg', 'surface', 'surface-raised'].map((g) => [g, v[`color-${g}`]!] as const)

    it.each(tones)(`${scheme}: solid %s text reads on its tone`, (tone) => {
      expect(contrast(v[`color-${tone}`]!, v[`color-${tone}-fg`]!)).toBeGreaterThanOrEqual(4.5)
    })

    it.each(tones)(`${scheme}: soft, outline and ghost %s text reads on every ground and tint`, (tone) => {
      const text = v[`color-${tone}-text`]!
      const fill = v[`color-${tone}`]!
      for (const [name, ground] of grounds) {
        for (const [tint, share] of [['none', 0], ...Object.entries(tints)] as const) {
          const behind = mix(fill, ground, share / 100)
          expect(contrast(text, behind), `${text} on ${name} with ${tint} ${share}%`).toBeGreaterThanOrEqual(4.5)
        }
      }
    })

    // Neutral's solid fill is the text colour, with the page's background on it.
    const solids: [tone: string, rest: string, fg: string][] = [
      ['neutral', v['color-text']!, v['color-bg']!],
      ...tones.map((t): [string, string, string] => [t, v[`color-${t}`]!, v[`color-${t}-fg`]!]),
    ]

    it.each(solids)(`${scheme}: solid %s text reads on its hover and active fills, which differ from rest`, (tone, rest, fg) => {
      for (const state of ['hover', 'active']) {
        const fill = v[`color-${tone}-${state}`]
        expect(fill, `${tone} ${state}`).toMatch(/^#[0-9A-F]{6}$/i)
        expect(contrast(fill!, fg), `${fg} on ${tone} ${state} ${fill}`).toBeGreaterThanOrEqual(4.5)
        expect(contrast(fill!, rest), `${tone} ${state} ${fill} against rest ${rest}`).toBeGreaterThan(1.1)
      }
    })

    it(`${scheme}: neutral text reads on every ground and tint of itself`, () => {
      for (const [name, ground] of grounds) {
        for (const share of [0, ...Object.values(tints)]) {
          expect(contrast(v['color-text']!, mix(v['color-text']!, ground, share / 100)), `${name} ${share}%`).toBeGreaterThanOrEqual(4.5)
        }
      }
    })
  }
})
