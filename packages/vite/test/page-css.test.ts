// Each page inlines the CSS of what it renders, not all of @weft/ui: the
// theme, then base.css, then the stylesheets of the components it renders
// (and those they import), each once.
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'

const root = fileURLToPath(new URL('./fixtures/css-site', import.meta.url))
let parent: string
let out: string
const styleOf = (page: string) =>
  /<style>([\s\S]*?)<\/style>/.exec(readFileSync(join(out, 'pages', page, 'index.html'), 'utf8'))![1]!

/**
 * The rules of a stylesheet, each as its text with whitespace removed and, inside
 * an at-rule other than @keyframes, prefixed with the at-rule's prelude.
 */
function rules(css: string, prefix = ''): string[] {
  const out: string[] = []
  const text = css.replace(/\/\*[\s\S]*?\*\//g, '')
  let i = 0
  while (i < text.length) {
    const open = text.indexOf('{', i)
    if (open < 0) break
    const prelude = text.slice(i, open).trim()
    let depth = 1
    let j = open + 1
    while (depth > 0 && j < text.length) {
      if (text[j] === '{') depth++
      else if (text[j] === '}') depth--
      j++
    }
    const body = text.slice(open + 1, j - 1)
    if (prelude.startsWith('@') && !prelude.startsWith('@keyframes') && body.includes('{')) {
      out.push(...rules(body, prefix + prelude.replace(/\s+/g, '') + '>'))
    } else {
      out.push(prefix + (prelude + '{' + body + '}').replace(/\s+/g, ''))
    }
    i = j
  }
  return out
}

beforeAll(async () => {
  // In a folder of its own: the build stages beside outDir, where other test files look for leftovers.
  parent = mkdtempSync(join(tmpdir(), 'weft-page-css-'))
  out = join(parent, 'dist')
  await buildSite({ root, outDir: out })
}, 120_000)
afterAll(() => rmSync(parent, { recursive: true, force: true }))

describe('per-page CSS', () => {
  it('ships only the rules of the components a page renders', () => {
    const css = styleOf('button')
    expect(css).toMatch(/\.weft-button\s*\{/)
    for (const other of ['.weft-input', '.weft-slider', '.weft-form-field', '.weft-checkbox', '.weft-textarea']) {
      expect(css, other).not.toContain(other)
    }
    for (const layout of ['.weft-stack', '.weft-grid', '.weft-card', '.weft-page', '.weft-section', '.weft-container']) {
      expect(css, layout).not.toContain(layout)
    }
  })

  it("ships what a component draws with: Button's spinner and icon", () => {
    const css = styleOf('button')
    expect(css).toMatch(/\.weft-spinner\s*\{/)
    expect(css).toMatch(/\.weft-icon\s*\{/)
    expect(css).toMatch(/\[data-tone="?primary"?\]\s*\{/)
  })

  it('ships both a field and the control inside it', () => {
    const css = styleOf('field')
    expect(css).toMatch(/\.weft-form-field\s*\{/)
    expect(css).toMatch(/\.weft-input\s*\{/)
    expect(css).not.toContain('.weft-button')
  })

  it('starts with the theme, then base.css, then the components', () => {
    for (const page of ['button', 'field', 'island', 'plain']) {
      const css = styleOf(page)
      const theme = css.indexOf('--weft-color-bg:')
      const base = css.search(/box-sizing:\s*border-box/)
      expect(theme, page).toBeGreaterThanOrEqual(0)
      expect(base, page).toBeGreaterThan(theme)
      const component = css.search(/\.weft-(button|input|badge|form-field)\b/)
      if (page !== 'plain') expect(component, page).toBeGreaterThan(base)
    }
  })

  it('gives a page with no component the theme and base.css only', () => {
    const css = styleOf('plain')
    expect(css).toMatch(/box-sizing:\s*border-box/)
    expect(css).not.toMatch(/\.weft-(button|input|badge|stack|card|icon)\b/)
  })

  it('has no rule twice', () => {
    for (const page of ['button', 'field', 'island', 'plain']) {
      const all = rules(styleOf(page))
      const twice = all.filter((r, i) => all.indexOf(r) !== i)
      expect(twice, page).toEqual([])
    }
  })

  it("inlines an island's CSS once, and the components it uses, even those only the island renders", () => {
    const css = styleOf('island')
    expect(css.match(/\.tally\s*\{/g)).toHaveLength(1)
    expect(css.match(/\.weft-badge\s*\{/g)).toHaveLength(1)
    expect(css.match(/\.weft-button\s*\{/g)).toHaveLength(1)
  })

  it('orders a stylesheet after those of the components it draws with', () => {
    // Button draws a Spinner; its own rules come after the spinner's.
    const css = styleOf('button')
    expect(css.search(/\.weft-button\s*\{/)).toBeGreaterThan(css.search(/\.weft-spinner\s*\{/))
  })
})
