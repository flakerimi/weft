// The site-themes Vite plugin: /.site-themes/<name>.css, the stylesheet of
// each theme of the Theme menu (menu.ts), made from the theme @weft/themes
// built. A page imports it by its URL (?url), and the page build writes it
// under a hashed name with the fonts it names; nothing inlines it.
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { join } from 'node:path'
import type { Plugin } from 'vite'
import { THEMES } from '../components/gallery/stage.ts'
import { SITE_THEMES } from './menu.ts'
import { scopeTheme } from './scope.ts'

const DIR = '.site-themes'
/** The themes the gallery's stage forces on its canvas, besides the site's. */
const STAGE_THEMES: string[] = THEMES.filter((t) => t !== 'site')

/**
 * A theme's stylesheet for the menu: the fonts it names (themes/<name>.css),
 * then its tokens (`css`, as @weft/themes built it) scoped to
 * `<html data-site-theme="<name>">`, and to an element inside with a scheme
 * of its own, light and dark following data-weft-scheme and the system as
 * the theme on the page does. Scoped, it outweighs the page's own theme,
 * whose blocks weigh nothing, wherever the link is in the head. A theme the
 * gallery's stage forces on its canvas (neutral, lisa) is scoped to that
 * canvas too, in the same file: the stage links it when it is picked there.
 */
export function siteThemeCSS(name: string, css: string): string {
  const theme = SITE_THEMES.find((t) => t.name === name)
  if (!theme) throw new Error(`site-themes: "${name}" is not a theme of the menu; add it to SITE_THEMES in site-src/themes/menu.ts.`)
  const fonts = 'fonts' in theme ? readFileSync(new URL(`./${theme.fonts}`, import.meta.url), 'utf8').trim() + '\n' : ''
  // The gallery's stage can force neutral or lisa on its canvas (components/gallery/stage.ts).
  const canvas = STAGE_THEMES.includes(name) ? `${scopeTheme(css, `.gallery-canvas[data-theme="${name}"]`)}\n` : ''
  return `${fonts}${scopeTheme(css, `:where(:root)[data-site-theme="${name}"]`, { inside: true })}\n${canvas}`
}

export function siteThemes(): Plugin {
  let root = process.cwd()
  const dir = () => join(root, DIR)
  return {
    name: 'weft-site:themes',
    enforce: 'pre',
    configResolved(config) {
      root = config.root
    },
    resolveId(id) {
      const m = /^\/\.site-themes\/([^/?]+\.css)(\?.*)?$/.exec(id)
      if (m) return join(dir(), m[1]!) + (m[2] ?? '')
    },
    load(id) {
      if (!id.startsWith(dir() + '/')) return
      const q = id.indexOf('?')
      const file = id.slice(dir().length + 1, q === -1 ? undefined : q)
      const name = file.replace(/\.css$/, '')
      const fromUI = createRequire(createRequire(join(root, 'package.json')).resolve('@weft/ui/styles/base.css'))
      return siteThemeCSS(name, readFileSync(fromUI.resolve(`@weft/themes/${name}.css`), 'utf8'))
    },
  }
}
