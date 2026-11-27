// The site's theme menu: each theme but the default (weft) scoped to
// <html data-site-theme="<name>">, its fonts with it, made by the site-themes
// plugin; and scopeTheme, which the gallery's stage shares.
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { describe, expect, it } from 'vitest'
import { scopeTheme } from '../themes/scope.ts'
import { SITE_THEMES, themeModule, themeSnippet } from '../themes/menu.ts'
import { siteThemeCSS } from '../themes/plugin.ts'

const themeCSS = (name: string) =>
  readFileSync(createRequire(createRequire(import.meta.url).resolve('@weft/ui/styles/base.css')).resolve(`@weft/themes/${name}.css`), 'utf8')

/** The declarations of the rule whose selector is exactly `selector`. */
function rule(css: string, selector: string): string {
  const at = css.indexOf(`${selector} {`)
  expect(at, selector).toBeGreaterThanOrEqual(0)
  return css.slice(css.indexOf('{', at) + 1, css.indexOf('}', at))
}

describe('scopeTheme', () => {
  const neutral = themeCSS('neutral')

  it("puts a theme's light and dark blocks on an element, as the gallery's canvas carries one", () => {
    const css = scopeTheme(neutral, '.gallery-canvas[data-theme="neutral"]')
    expect(rule(css, '.gallery-canvas[data-theme="neutral"]')).toContain('--weft-color-bg: #FAFAFA;')
    expect(rule(css, '.gallery-canvas[data-theme="neutral"][data-weft-scheme="dark"]')).toContain('--weft-color-bg: #09090B;')
    expect(rule(css, '.gallery-canvas[data-theme="neutral"]:not([data-weft-scheme="light"])')).toContain('--weft-color-bg: #09090B;')
    expect(css).toContain('@media (prefers-color-scheme: dark)')
    // The input and surface blocks are the same in every theme: they stay the page's.
    expect(css).not.toContain('data-weft-input')
    expect(css).not.toContain('data-weft-surface')
  })

  it('with inside, reaches an element inside it with a scheme of its own, at no more than its own weight', () => {
    const at = ':where(:root)[data-site-theme="neutral"]'
    const css = scopeTheme(neutral, at, { inside: true })
    expect(rule(css, `${at}, :where(${at}) [data-weft-scheme="light"]`)).toContain('--weft-color-bg: #FAFAFA;')
    expect(rule(css, `${at}[data-weft-scheme="dark"], :where(${at}) [data-weft-scheme="dark"]`)).toContain('--weft-color-bg: #09090B;')
    expect(rule(css, `${at}:not([data-weft-scheme="light"])`)).toContain('--weft-color-bg: #09090B;')
  })

  it('refuses CSS that is no theme, in a sentence', () => {
    expect(() => scopeTheme('.x { color: red; }', '.x')).toThrow(
      'scopeTheme: the CSS has no :where(:root, [data-weft-scheme="light"]) block, which every @weft/themes stylesheet has; give it a built theme (pnpm --filter @weft/themes build).',
    )
  })
})

describe('the site themes', () => {
  it('are the five besides weft, in the menu\'s order', () => {
    expect(SITE_THEMES.map((t) => t.name)).toEqual(['neutral', 'lisa', 'synthwave', 'editor', 'sandstorm'])
    expect(SITE_THEMES.map((t) => t.label)).toEqual(['Neutral', 'Lisa', 'Synthwave', 'Editor', 'Sandstorm'])
  })

  it.each(SITE_THEMES.map((t) => t.name))('%s is scoped to <html data-site-theme>, every rule of it', (name) => {
    const css = siteThemeCSS(name, themeCSS(name))
    const at = `:where(:root)[data-site-theme="${name}"]`
    expect(rule(css, `${at}, :where(${at}) [data-weft-scheme="light"]`)).toContain('--weft-color-bg:')
    for (const [, selector] of css.matchAll(/(?:^|\})\s*([^{}@]+?)\s*\{/g)) {
      for (const one of selector!.split(',')) expect([`[data-site-theme="${name}"]`, `.gallery-canvas[data-theme="${name}"]`].some((s) => one.includes(s)), one).toBe(true)
    }
  })

  // The gallery's stage forces them on its canvas: its canvas takes them from the same stylesheet.
  it.each(['neutral', 'lisa'])('%s is scoped to the gallery\'s canvas too', (name) => {
    const css = siteThemeCSS(name, themeCSS(name))
    expect(rule(css, `.gallery-canvas[data-theme="${name}"]`)).toContain('--weft-color-bg:')
    expect(rule(css, `.gallery-canvas[data-theme="${name}"][data-weft-scheme="dark"]`)).toContain('--weft-color-bg:')
  })

  it('brings the fonts a theme names that every page does not already have, self-hosted', () => {
    const faces = (name: string) => [...siteThemeCSS(name, themeCSS(name)).matchAll(/@font-face\s*\{([^}]*)\}/g)].map((m) => m[1]!)
    const synthwave = faces('synthwave')
    expect(synthwave).toHaveLength(1)
    expect(synthwave[0]).toContain("font-family: 'Space Grotesk Variable'")
    expect(synthwave[0]).toContain("url('@fontsource-variable/space-grotesk/files/space-grotesk-latin-wght-normal.woff2')")
    const sandstorm = faces('sandstorm')
    expect(sandstorm).toHaveLength(1)
    expect(sandstorm[0]).toContain("font-family: 'Quicksand Variable'")
    expect(sandstorm[0]).toContain("url('@fontsource-variable/quicksand/files/quicksand-latin-wght-normal.woff2')")
    // JetBrains Mono is every page's already; Lisa and neutral name system fonts.
    for (const name of ['editor', 'neutral', 'lisa']) expect(faces(name), name).toEqual([])
  })

  it('refuses a theme it does not know, in a sentence', () => {
    expect(() => siteThemeCSS('monokai', themeCSS('neutral'))).toThrow(
      'site-themes: "monokai" is not a theme of the menu; add it to SITE_THEMES in site-src/themes/menu.ts.',
    )
  })
})

describe('the pre-paint snippet', () => {
  const code = themeSnippet()

  it('is a script on its own, safe inside an inline <script>', () => {
    expect(() => new Function(code)).not.toThrow()
    expect(code).not.toMatch(/<\/script|<!--/i)
    expect(code).not.toContain('\n')
  })

  it('links only a stylesheet of this site that the choice names, under weft:site-theme', () => {
    expect(code).toContain('"weft:site-theme"')
    const kept = new RegExp(/\/\^.*?\$\//.exec(code)![0].slice(1, -1))
    expect(kept.exec('synthwave /assets/synthwave-0123abcd.css')?.slice(1)).toEqual(['synthwave', '/assets/synthwave-0123abcd.css'])
    expect(kept.exec('lisa /.site-themes/lisa.css')?.slice(1)).toEqual(['lisa', '/.site-themes/lisa.css'])
    for (const bad of ['synthwave //evil.example/x.css', 'synthwave https://evil.example/x.css', 'x /assets/a b.css', 'x /assets/x.css onload=alert(1)', 'synthwave']) {
      expect(kept.test(bad), bad).toBe(false)
    }
  })
})

describe('the menu module', () => {
  const urls = Object.fromEntries(SITE_THEMES.map((t) => [t.name, `/assets/${t.name}-0123abcd.css`]))

  it('is a module that knows each theme by its stylesheet', async () => {
    const code = themeModule(urls, '.site-themes{}')
    expect(code).toContain(`const U = ${JSON.stringify(urls)}`)
    expect(code).toContain('const CSS = ".site-themes{}"')
    expect(code).toContain('"label":"Weft"')
    expect(code).toContain('"label":"Sandstorm"')
    // Its top level reads only document and its root.
    const g = globalThis as { document?: unknown }
    g.document = { documentElement: {} }
    try {
      const mod = await import(`data:text/javascript,${encodeURIComponent(code)}`)
      expect(typeof mod.default).toBe('function')
    } finally {
      delete g.document
    }
  })

  it('refuses a URL the pre-paint snippet would not link, in a sentence', () => {
    for (const lisa of ['/a b/lisa.css', '//evil.example/lisa.css', '/lisa.css']) {
      expect(() => themeModule({ ...urls, lisa }, '')).toThrow(
        `site-themes: the lisa theme's stylesheet is at ${lisa}, not a file of this site the pre-paint snippet links (/<folder>/<file>.css); import it by its URL, as ThemeMenu.vue does.`,
      )
    }
  })
})
