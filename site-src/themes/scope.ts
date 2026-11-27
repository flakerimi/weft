// A whole theme on one element rather than on the page: what the gallery's
// stage does for its canvas and the site's Theme menu does for <html>.

const LIGHT = ':where(:root, [data-weft-scheme="light"])'
const DARK = ':where([data-weft-scheme="dark"])'

export interface ScopeOptions {
  /**
   * Also reach an element inside the scope with a `data-weft-scheme` of its
   * own (the landing page's night band), as the theme on the page does. Such
   * an element gets the scoped tokens at the weight of one attribute
   * selector, so a stage inside it that carries a theme of its own keeps it.
   */
  inside?: boolean
}

/**
 * The CSS of a theme as @weft/themes writes it, scoped to `selector`: its
 * light block (with the tokens every scheme shares) on the element, its dark
 * block where the element says `data-weft-scheme="dark"`, and its dark block
 * again under a dark system when the element says neither. The input and
 * surface blocks are the same in every theme and stay the page's.
 */
export function scopeTheme(css: string, selector: string, opts: ScopeOptions = {}): string {
  const body = (block: string) => {
    const at = css.indexOf(`${block} {`)
    if (at === -1) {
      throw new Error(`scopeTheme: the CSS has no ${block} block, which every @weft/themes stylesheet has; give it a built theme (pnpm --filter @weft/themes build).`)
    }
    const start = css.indexOf('{', at) + 1
    return css.slice(start, css.indexOf('}', start)).trim()
  }
  const light = body(LIGHT)
  const dark = body(DARK)
  const within = (scheme: string) => (opts.inside ? `, :where(${selector}) [data-weft-scheme="${scheme}"]` : '')
  return [
    `${selector}${within('light')} {\n  ${light}\n}`,
    `${selector}[data-weft-scheme="dark"]${within('dark')} {\n  ${dark}\n}`,
    `@media (prefers-color-scheme: dark) {\n  ${selector}:not([data-weft-scheme="light"]) {\n  ${dark}\n  }\n}`,
  ].join('\n')
}
