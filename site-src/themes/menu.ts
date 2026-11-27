// The site's Theme menu: the themes a visitor can pick besides weft, the
// site's own, each a stylesheet of its own (plugin.ts makes it) that the page
// build writes under a hashed name and a page links only when it is picked;
// the pre-paint snippet that applies the kept choice (themeSnippet); and the
// module that does what a press asks (themeModule), which a page fetches
// only once the menu is used, so the menu costs every page as little as it
// can: page budgets are never raised.

/** The themes besides weft, in the menu's order, each with the stylesheet of the fonts it names that no page has already. */
export const SITE_THEMES = [
  { name: 'neutral', label: 'Neutral' },
  { name: 'lisa', label: 'Lisa' },
  { name: 'synthwave', label: 'Synthwave', fonts: 'synthwave.css' },
  { name: 'editor', label: 'Editor' },
  { name: 'sandstorm', label: 'Sandstorm', fonts: 'sandstorm.css' },
] as const

export type SiteTheme = (typeof SITE_THEMES)[number]['name']

/**
 * The key the choice is kept under in localStorage: the theme's name and its
 * stylesheet's URL, `synthwave /assets/synthwave-<hash>.css`, so the
 * pre-paint snippet needs no list of URLs; nothing kept is weft.
 */
export const THEME_KEY = 'weft:site-theme'

/** Where the page build writes the menu's module (useFile): revalidated, so a page always gets the build's. */
export const THEME_MODULE = '/theme-menu.js'

/**
 * The pre-paint snippet (usePrePaint) of the menu. Before the first paint it
 * applies the kept choice: the attribute on <html>, and the theme's
 * stylesheet written into the head where the snippet runs, so it blocks
 * rendering as a stylesheet in the markup does, and a page never shows weft
 * first. Only a URL of this site's own (/<folder>/<file>.css) is linked. A
 * press on the menu's button or a theme's (data-site-theme-pick) fetches the
 * menu's module and hands it the press; so does a kept stylesheet that fails
 * to load (a later build renamed it), and the module links the build's. A
 * page the back-forward cache brings back reloads when the choice changed
 * meanwhile. Without storage (blocked) the menu does nothing.
 */
export function themeSnippet(): string {
  // d: the document; s: storage; v: the kept choice; k: its name and URL, if it is one;
  // m: fetch the module and hand it a press (none: mend the stylesheet).
  return `try{let d=document,s=localStorage,K=${JSON.stringify(THEME_KEY)},v=s.getItem(K),k=/^([a-z]+) (\\/[\\w.-]+\\/[\\w.-]+\\.css)$/.exec(v),
m=b=>import(${JSON.stringify(THEME_MODULE)}).then(x=>x.default(b));
weftTheme=m;
k&&(d.documentElement.dataset.siteTheme=k[1],d.write("<link rel=stylesheet href="+k[2]+" data-site-theme="+k[1]+" onerror=weftTheme()>"));
d.onclick=e=>{let b=e.target.closest(".site-theme,[data-site-theme-pick]");b&&m(b)};
onpageshow=e=>e.persisted&&s.getItem(K)!=v&&location.reload()}catch(e){}`.replace(/\n/g, '')
}

/**
 * The menu's module, given each theme's stylesheet URL, as the page build
 * named them, and the menu's CSS. Handed the menu's button the first time,
 * it fills the menu: the CSS, and a button per theme, each a copy of the
 * menu's own button (Weft's Button, as the server drew it) with the theme's
 * name, the one that is on pressed (aria-pressed); the server draws the
 * menu empty and hidden, which keeps it out of every first view. Handed a
 * theme's button, it keeps the choice, closes the
 * menu and switches once the theme's stylesheet is in, linking it once, and
 * Weft takes the attribute, the stylesheets and the kept choice off; handed
 * nothing, it links the build's stylesheet for the theme that is on and
 * keeps that. `sheet(name)` links a theme's stylesheet without putting it on
 * the page, for the gallery's stage.
 */
export function themeModule(urls: Record<string, string>, css: string): string {
  for (const [name, url] of Object.entries(urls)) {
    if (!/^[a-z]+$/.test(name) || !/^\/[\w.-]+\/[\w.-]+\.css$/.test(url)) {
      throw new Error(`site-themes: the ${name} theme's stylesheet is at ${url}, not a file of this site the pre-paint snippet links (/<folder>/<file>.css); import it by its URL, as ThemeMenu.vue does.`)
    }
  }
  return `// The site's Theme menu: what a press does (site-src/themes/menu.ts). Made by the page build.
const U = ${JSON.stringify(urls)}, K = ${JSON.stringify(THEME_KEY)}, d = document, r = d.documentElement
const THEMES = ${JSON.stringify([{ name: 'weft', label: 'Weft' }, ...SITE_THEMES.map((t) => ({ name: t.name, label: t.label }))])}
const CSS = ${JSON.stringify(css)}
const has = (t) => Object.hasOwn(U, t)
const links = () => d.querySelectorAll('link[data-site-theme]')
const mark = () => d.querySelectorAll('[data-site-theme-pick]').forEach((b) => (b.ariaPressed = String(b.dataset.siteThemePick === (r.dataset.siteTheme || 'weft'))))
const keep = (t) => {
  try {
    has(t) ? localStorage.setItem(K, t + ' ' + U[t]) : localStorage.removeItem(K)
  } catch {}
}
/** Put a theme on, weft for none, with the build's stylesheet only. */
const on = (t) => {
  if (has(t)) r.dataset.siteTheme = t
  else delete r.dataset.siteTheme
  links().forEach((l) => (l.dataset.siteTheme === t && l.getAttribute('href') === U[t]) || l.remove())
  mark()
}
const link = (t, then) => {
  const l = d.createElement('link')
  l.rel = 'stylesheet'
  l.href = U[t]
  l.dataset.siteTheme = t
  l.onload = then
  d.head.append(l)
}
/** Go to a theme once its stylesheet is in. */
const go = (t) => {
  if (!has(t) || [...links()].some((l) => l.dataset.siteTheme === t && l.getAttribute('href') === U[t] && l.sheet)) return on(t)
  link(t, () => on(t))
}
/**
 * Link a theme's stylesheet without putting it on the page: the gallery's
 * stage forces neutral or lisa on its canvas with it. Its link is the
 * stage's (data-stage-theme), which a switch of the page's theme leaves.
 */
export const sheet = (t) => {
  if (!has(t) || d.querySelector('link[data-stage-theme="' + t + '"]')) return
  const l = d.createElement('link')
  l.rel = 'stylesheet'
  l.href = U[t]
  l.dataset.stageTheme = t
  d.head.append(l)
}
/** Fill the menu the first time it is opened, from its own button; mark the theme that is on. */
const fill = (button) => {
  const menu = d.getElementById(button.getAttribute('popovertarget'))
  if (menu && !menu.children.length) {
    const style = d.createElement('style')
    style.textContent = CSS
    d.head.append(style)
    menu.hidden = false
    for (const { name, label } of THEMES) {
      const b = button.cloneNode(true)
      b.classList.remove('site-theme')
      b.removeAttribute('popovertarget')
      b.removeAttribute('aria-expanded')
      b.dataset.siteThemePick = name
      b.querySelector('.weft-button__content').textContent = label
      menu.append(b)
    }
  }
  mark()
}
export default (b) => {
  if (!b) {
    const t = r.dataset.siteTheme
    keep(t)
    return go(t)
  }
  const t = b.dataset.siteThemePick
  if (t === undefined) return fill(b)
  keep(t)
  b.closest('[popover]')?.hidePopover()
  go(t)
}
`
}
