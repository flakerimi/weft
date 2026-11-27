// Each page inlines the CSS of what it renders, not all of @weft/ui: the
// theme, then base.css, then the stylesheets of the components it renders
// (and those they import), each once.
import { mkdtempSync, readFileSync, rmSync, symlinkSync } from 'node:fs'
import { createRequire } from 'node:module'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildSite } from '../src/build.ts'
import { createRenderServer, renderPage } from '../src/render.ts'

const root = fileURLToPath(new URL('./fixtures/css-site', import.meta.url))
const pages = ['button', 'field', 'island', 'barrel-island', 'barrel-button-island', 'dialog-island', 'on-demand-island', 'link', 'nav', 'plain']
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

  it("ships what a component draws with, and only what it draws: a still Button's tone, not its spinner", () => {
    const css = styleOf('button')
    expect(css).toMatch(/\[data-tone="?primary"?\]\s*\{/)
    // Outside an island nothing is drawn later: a Button that is not loading never shows its Spinner.
    expect(css).not.toMatch(/\.weft-spinner\s*\{/)
  })

  it('ships no CSS for the forms a NavigationMenu imports but a page without islands never draws', () => {
    const css = styleOf('nav')
    expect(css).toMatch(/\.weft-nav\s*\{/)
    // A rule for the class itself; navigation-menu.css names .weft-overlay-trigger in a selector of its own.
    for (const other of ['weft-sheet', 'weft-tooltip', 'weft-badge', 'weft-overlay', 'weft-close-button', 'weft-spinner', 'weft-button']) {
      expect(css, other).not.toMatch(new RegExp(`\\.${other}[\\s{,:[]`))
    }
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
    // A name taken from the index brings that component's stylesheet, not every one the index exports.
    expect(css).not.toMatch(/\.weft-slider\s*\{/)
    expect(css).not.toMatch(/\.weft-button\s*\{/)
  })

  it('gives an island that takes { Button } from the @weft/ui index what a Button may draw, and nothing else', () => {
    // An island may draw what its components import later (a Button's spinner once loading): more than a still Button.
    const island = new Set(rules(unminified['barrel-button-island']!))
    for (const rule of rules(unminified.button!)) expect(island, rule).toContain(rule)
    const css = styleOf('barrel-button-island')
    expect(css.match(/\.weft-button\s*\{/g)).toHaveLength(1)
    expect(css).toMatch(/\.weft-spinner\s*\{/)
    for (const other of ['.weft-dialog', '.weft-table', '.weft-input', '.weft-switch', '.weft-menu', '.weft-toast', '.weft-calendar']) {
      expect(css, other).not.toContain(other)
    }
  })

  it('inlines the CSS of a Dialog an island opens after hydrating, and of the Button inside it', () => {
    const html = readFileSync(join(out, 'pages', 'dialog-island', 'index.html'), 'utf8')
    // The server drew only the trigger: the dialog exists only once the island runs.
    expect(html).not.toContain('role="dialog"')
    const css = styleOf('dialog-island')
    // Its own rule once (its entry animation adds one under @starting-style).
    expect(css.match(/\}\.weft-dialog\{/g)).toHaveLength(1)
    expect(css).toMatch(/\.weft-overlay-backdrop\s*\{/)
    expect(css).toMatch(/\.weft-overlay__close\s*\{/)
    expect(css.match(/\.weft-button\s*\{/g)).toHaveLength(1)
    expect(css).toMatch(/\.weft-spinner\s*\{/)
    // By path, so no other overlay comes with it.
    expect(css).not.toContain('.weft-menu')
    expect(css).not.toContain('.weft-toast')
  })

  it('ships the CSS of what an island loads on demand, which the server never drew', () => {
    const html = readFileSync(join(out, 'pages', 'on-demand-island', 'index.html'), 'utf8').replace(/<style>[\s\S]*?<\/style>/, '')
    expect(html).not.toContain('class="later"')
    expect(html).not.toContain('weft-popover')
    const css = styleOf('on-demand-island')
    // A component of the site's, loaded with import().
    expect(css).toMatch(/\.later\s*\{/)
    // The DatePicker's Popover and Calendar, which it loads on demand from @weft/ui.
    expect(css).toMatch(/\.weft-popover\[data-form="?floating"?\]\s*\{/)
    expect(css).toMatch(/\.weft-calendar\s*\{/)
    expect(css).toMatch(/\.weft-sheet\s*\{/)
  })

  it('keeps what the browser targets need when minifying', () => {
    // Safari before 18 needs the prefix; a text-decoration shorthand with a thickness is not valid there.
    expect(styleOf('button')).toMatch(/-webkit-user-select:\s*none/)
    expect(styleOf('link')).toMatch(/text-decoration-thickness:\s*1px/)
  })

  it('orders a stylesheet after those of the components it draws with', () => {
    // Button draws a Spinner; its own rules come after the spinner's.
    const css = styleOf('dialog-island')
    expect(css.search(/\.weft-button\s*\{/)).toBeGreaterThan(css.search(/\.weft-spinner\s*\{/))
  })

  it('keeps component CSS when the site root is reached through a symlink (such as macOS tmpdir)', async () => {
    const linkParent = mkdtempSync(join(tmpdir(), 'weft-css-link-'))
    try {
      const symlinkRoot = join(linkParent, 'site-link')
      symlinkSync(root, symlinkRoot)
      const { css } = await renderPage({ root: symlinkRoot, page: 'pages/button.vue' })
      expect(css).toMatch(/\.weft-button\s*\{/)
    } finally {
      rmSync(linkParent, { recursive: true, force: true })
    }
  })
})
