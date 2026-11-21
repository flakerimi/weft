// code.css colours highlighted code from the theme: every colour it gives a
// token reaches WCAG AA (4.5:1) on the code's ground, in both themes, light and dark.
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { CODE_THEME } from '../src/content.ts'

const read = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')
const code = read('../src/code.css')

/** The custom properties of the block whose selector is `selector`, by name without `--`. */
function block(css: string, selector: string): Record<string, string> {
  const start = css.indexOf(`${selector} {`)
  expect(start, selector).toBeGreaterThanOrEqual(0)
  const body = css.slice(start, css.indexOf('}', start))
  return Object.fromEntries([...body.matchAll(/--([\w-]+):\s*([^;]+);/g)].map((m) => [m[1]!, m[2]!.trim()]))
}

const luminance = (hex: string) => {
  const [r, g, b] = [1, 3, 5]
    .map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!
}
const contrast = (a: string, b: string) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi! + 0.05) / (lo! + 0.05)
}

const vars = block(code, '.shiki')

it('sets every variable the code theme can draw with', () => {
  const emitted = [...new Set(JSON.stringify(CODE_THEME).match(/--weft-code-[\w-]+/g))].map((v) => v.slice(2))
  // ansi is refused (its terminal colours have no AA pair on every ground), so its variables are never drawn.
  const drawn = emitted.filter((v) => !v.startsWith('weft-code-ansi-'))
  expect(drawn).toContain('weft-code-token-keyword')
  for (const v of drawn) expect(Object.keys(vars), v).toContain(v)
})

describe.each(['neutral', 'lisa'])('code.css on the %s theme', (theme) => {
  const css = read(`../../themes/dist/${theme}.css`)
  const light = block(css, ':root, [data-weft-scheme="light"]')
  const schemes = { light, dark: { ...light, ...block(css, '[data-weft-scheme="dark"]') } }
  /** A code variable's colour in a scheme, through the theme token it names. */
  const colour = (scheme: Record<string, string>, name: string) => {
    const token = /^var\(--([\w-]+)\)$/.exec(vars[name]!)?.[1]
    expect(token, `${name} is ${vars[name]}`).toBeDefined()
    expect(scheme[token!], `${token} in the ${theme} theme`).toMatch(/^#[0-9a-f]{6}$/i)
    return scheme[token!]!
  }

  for (const [name, scheme] of Object.entries(schemes)) {
    it(`${name}: every token colour reads on the code's ground`, () => {
      const ground = colour(scheme, 'weft-code-background')
      const tokens = Object.keys(vars).filter((v) => v !== 'weft-code-background')
      expect(tokens).toContain('weft-code-token-keyword')
      for (const v of tokens) expect(contrast(colour(scheme, v), ground), v).toBeGreaterThanOrEqual(4.5)
    })
  }
})
