// Each page inlines the CSS of what it renders, not all of @weft/ui: the
// theme, then base.css, then the stylesheets of the components it renders
// (and those they import), each once.
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { createRequire } from 'node:module'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'
import { createRenderServer, renderPage } from '../src/render.ts'

const root = fileURLToPath(new URL('./fixtures/css-site', import.meta.url))
const pages = ['button', 'field', 'island', 'barrel-island', 'link', 'plain']
let parent: string
let out: string
/** Each page's CSS before minifying, which could merge a rule given twice: the theme, base.css, then the render's. */
const unminified: Record<string, string> = {}
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

  const require = createRequire(join(root, 'package.json'))
  const base = require.resolve('@weft/ui/styles/base.css')
  const theme = createRequire(base).resolve('@weft/themes/neutral.css')
  const server = await createRenderServer(root, 'production')
  try {
    for (const page of pages) {
      const { css } = await renderPage({ root, page: `pages/${page}.vue`, server, providedCSS: [theme, base] })
      unminified[page] = [theme, base].map((f) => readFileSync(f, 'utf8')).join('\n') + '\n' + css
    }
  } finally {
    await server.close()
  }
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
    for (const page of pages) {
      const css = styleOf(page)
      const theme = css.indexOf('--weft-color-bg:')
      const base = css.search(/box-sizing:\s*border-box/)
      expect(theme, page).toBeGreaterThanOrEqual(0)
      expect(base, page).toBeGreaterThan(theme)
      const component = css.search(/\.weft-(?!visually-hidden|provider)[a-z]/)
      if (page !== 'plain') expect(component, page).toBeGreaterThan(base)
    }
  })

  it('gives a page with no component the theme and base.css only', () => {
    const css = styleOf('plain')
    expect(css).toMatch(/box-sizing:\s*border-box/)
    expect(css).not.toMatch(/\.weft-(button|input|badge|stack|card|icon)\b/)
  })

  it('has no rule twice, before minifying', () => {
    for (const page of pages) {
      const all = rules(unminified[page]!)
      expect(all.length, page).toBeGreaterThan(10)
      const twice = all.filter((r, i) => all.indexOf(r) !== i)
      expect(twice, page).toEqual([])
    }
  })

  it("inlines an island's CSS once, and the components it uses, even those only the island renders", () => {
    const css = styleOf('island')
    expect(css.match(/\.tally\s*\{/g)).toHaveLength(1)
    expect(css.match(/\.weft-badge\s*\{/g)).toHaveLength(1)
    expect(css.match(/\.weft-button\s*\{/g)).toHaveLength(1)
    // Drawn only once hydrated, never on the server.
    expect(css.match(/\.weft-kbd\s*\{/g)).toHaveLength(1)
    // Imported by path, so nothing else of @weft/ui comes with them.
    expect(css).not.toContain('.weft-slider')
  })

  it('inlines, once, what an island draws only after hydrating, even through the @weft/ui index', () => {
    const css = styleOf('barrel-island')
    // The rule for the track itself, not those for a track inside a switch in some state.
    expect(css.match(/\}\.weft-switch__track\{/g)).toHaveLength(1)
    // The index brings every component's stylesheet: why islands import components by path.
    expect(css).toMatch(/\.weft-slider\s*\{/)
  })

  it('keeps what the browser targets need when minifying', () => {
    // Safari before 18 needs the prefix; a text-decoration shorthand with a thickness is not valid there.
    expect(styleOf('button')).toMatch(/-webkit-user-select:\s*none/)
    expect(styleOf('link')).toMatch(/text-decoration-thickness:\s*1px/)
  })

  it('orders a stylesheet after those of the components it draws with', () => {
    // Button draws a Spinner; its own rules come after the spinner's.
    const css = styleOf('button')
    expect(css.search(/\.weft-button\s*\{/)).toBeGreaterThan(css.search(/\.weft-spinner\s*\{/))
  })
})
