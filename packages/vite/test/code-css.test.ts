// code.css colours highlighted code from the theme: every colour it gives a
// token reaches WCAG AA (4.5:1) on the code's ground, in every theme, light and dark.
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

describe.each(['neutral', 'lisa', 'weft', 'synthwave', 'editor', 'sandstorm'])('code.css on the %s theme', (theme) => {
  const css = read(`../../themes/dist/${theme}.css`)
  const light = block(css, ':where(:root, [data-weft-scheme="light"])')
  const schemes = { light, dark: { ...light, ...block(css, ':where([data-weft-scheme="dark"])') } }
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

// Its lines come from the theme too, as @weft/ui's do (packages/ui/test/elevation-tokens.test.ts):
// the block's border reads --weft-border-width and its focus ring --weft-focus-width, each with
// a fallback, and no border, outline or shadow in it has a literal width.
describe('code.css lines', () => {
  const rules = code.replace(/\/\*[\s\S]*?\*\//g, '')
  const declarations = [...rules.matchAll(/(?<![\w-])((?:box-shadow|border|outline)(?:-[a-z]+)*)\s*:\s*([^;{}]+)/g)]
    .map((m) => ({ property: m[1]!, value: m[2]!.trim() }))
  /** The value with every var(--weft-…) removed, fallbacks too. */
  const withoutTokens = (value: string) => {
    let out = ''
    for (let i = 0; i < value.length; ) {
      if (value.startsWith('var(--weft-', i)) {
        let depth = 1
        for (i += 'var('.length; depth > 0 && i < value.length; i++) {
          if (value[i] === '(') depth++
          else if (value[i] === ')') depth--
        }
      } else out += value[i++]
    }
    return out
  }

  it('reads the border and focus widths from the theme, with fallbacks', () => {
    expect(declarations.find((d) => d.property === 'border')?.value).toMatch(/^var\(--weft-border-width, 1px\) solid var\(--weft-color-border, [^)]+\)$/)
    expect(declarations.find((d) => d.property === 'outline')?.value).toMatch(/^var\(--weft-focus-width, 2px\) solid var\(--weft-color-focus, [^)]+\)$/)
  })

  it('has no literal line width or shadow', () => {
    const literal = declarations.filter((d) =>
      d.property === 'box-shadow'
        ? d.value !== 'none' && !d.value.startsWith('var(--weft-shadow-')
        : /^(border(-[a-z]+)*|outline(-width)?)$/.test(d.property) && !/radius|color|style/.test(d.property) && /(?<![\w.-])\d*\.?\d+px\b/.test(withoutTokens(d.value)),
    )
    expect(literal).toEqual([])
  })
})
