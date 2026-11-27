// What a page's HTML document is made of, for the page build and the dev
// server alike: the theme and base CSS, the head tags, the icon tags, the
// pre-paint script and the document around the rendered body.
import { readdirSync, statSync } from 'node:fs'
import { readdir, readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import type { PageHead } from './render.ts'

export type Theme = 'neutral' | 'lisa' | 'weft'

export const THEMES: Theme[] = ['neutral', 'lisa', 'weft']

/** Cross-document view transitions, the last rule of every page's CSS. */
export const VIEW_TRANSITION = '@view-transition { navigation: auto; }'

/**
 * The CSS every page starts with: the theme stylesheet, then @weft/ui's
 * base.css, and their files. Each component brings its own stylesheet,
 * collected per page. `bundles` are the stylesheets that hold every
 * component's CSS (weft.css, built and source, and the group sheets of
 * @imports it is made of), which a page may not import. `who` starts the
 * sentence an error is given in. @weft/ui is resolved as Node resolves it
 * from a file in `root`: `<root>/package.json` need not exist, and a
 * `node_modules` in a folder above is found.
 */
export async function readStyles(
  root: string,
  theme: Theme,
  who = 'buildSite',
): Promise<{ css: string; files: string[]; bundles: string[] }> {
  const read = async (from: string, spec: string, fix: string) => {
    try {
      const file = createRequire(from).resolve(spec)
      return [file, await readFile(file, 'utf8')] as const
    } catch (e) {
      const why = e instanceof Error ? e.message.split('\n')[0] : String(e)
      throw new Error(`${who}: cannot read ${spec} (${why}); ${fix}.`)
    }
  }
  const [baseFile, base] = await read(join(root, 'package.json'), '@weft/ui/styles/base.css', 'add @weft/ui to the site\'s dependencies')
  // The theme comes from @weft/ui's own @weft/themes, the version its components expect.
  const [themeFile, themeCss] = await read(baseFile, `@weft/themes/${theme}.css`, 'run pnpm --filter @weft/themes build')
  const styles = dirname(baseFile)
  const bundles: string[] = []
  for (const name of (await readdir(styles)).filter((n) => n.endsWith('.css')).sort()) {
    const text = await readFile(join(styles, name), 'utf8')
    if (text.replace(/\/\*[\s\S]*?\*\//g, '').replace(/@import\s+'[^']*';/g, '').trim() === '') bundles.push(join(styles, name))
  }
  try {
    bundles.push(createRequire(baseFile).resolve('@weft/ui/weft.css'))
  } catch {
    // Not built: then no page can import it either.
  }
  return { css: `${themeCss}\n${base}`, files: [themeFile, baseFile], bundles }
}

export const escapeHTML = (s: string) =>
  s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)

/** `<title>`, description and Open Graph tags for what the page set with useHead. */
export function headTags({ title, description, image }: PageHead): string[] {
  const meta = (attr: string, key: string, value: string | undefined) =>
    value === undefined ? [] : [`<meta ${attr}="${key}" content="${escapeHTML(value)}">`]
  return [
    ...(title === undefined ? [] : [`<title>${escapeHTML(title)}</title>`]),
    ...meta('name', 'description', description),
    ...meta('property', 'og:title', title),
    ...meta('property', 'og:description', description),
    ...meta('property', 'og:image', image),
  ]
}

/**
 * The icon files a site's `public/` folder may hold, each with its tag, in
 * the order they go: the .ico first and the SVG after it, since a browser
 * takes the last icon it understands, then the home-screen icon.
 */
const ICONS: [file: string, tag: string][] = [
  ['favicon.ico', '<link rel="icon" href="/favicon.ico" sizes="32x32">'],
  ['favicon.svg', '<link rel="icon" href="/favicon.svg" type="image/svg+xml">'],
  ['apple-touch-icon.png', '<link rel="apple-touch-icon" href="/apple-touch-icon.png">'],
]

/**
 * The icon tags for the files the site at `root` has in `public/`, the
 * folder the build copies to its root: one for each that is there, a file
 * by its exact name (or a symlink to one), and none for the others, nor for
 * an entry it cannot stat. Read each time, so the dev server sees a file
 * dropped in.
 */
export function iconTags(root: string): string[] {
  const dir = join(root, 'public')
  let names: Set<string>
  try {
    names = new Set(readdirSync(dir))
  } catch {
    return []
  }
  // An entry that cannot be read, such as a symlink loop or one without permission, is no file.
  const isFile = (name: string) => {
    if (!names.has(name)) return false
    try {
      return statSync(join(dir, name)).isFile()
    } catch {
      return false
    }
  }
  return ICONS.filter(([file]) => isFile(file)).map(([, tag]) => tag)
}

/**
 * The page's pre-paint snippets in one blocking inline script, or none.
 * Each snippet is wrapped in `(function(){…})()` so its top-level variables
 * never leak into the global scope or collide with another snippet's or the
 * page's own scripts. A snippet that would end the script early is refused in
 * a sentence that `who` starts.
 */
export function prePaintTag(snippets: string[], who = 'buildSite'): string[] {
  if (!snippets.length) return []
  for (const code of snippets) {
    if (/<\/script|<!--/i.test(code)) throw new Error(`${who}: a pre-paint snippet holds "</script" or "<!--", which would break the page's <head>: ${code.slice(0, 60)}`)
  }
  return [`<script data-weft-prepaint>${snippets.map((c) => `(function(){${c}})()`).join(';\n')}</script>`]
}

export function pageHTML(p: {
  body: string
  head: PageHead
  /** Pre-paint snippets (usePrePaint), run before the CSS and the first paint. */
  prePaint: string[]
  /** The icon tags (iconTags) for the site's public/ folder. */
  icons: string[]
  /** The woff2 fonts the page's CSS names, preloaded. */
  fonts: string[]
  css: string
  stylesheet: string | null
  /** The tags after the CSS, in order: the speculation rules, the modules to preload, the page's module scripts. */
  tags: string[]
  /** Who starts the sentence of an error. */
  who?: string
  /** A runtime page's pattern (`/products/:id`), written on `<html>` as data-weft-route for useParams(). */
  route?: string
}): string {
  return [
    '<!doctype html>',
    p.route === undefined ? '<html lang="en">' : `<html lang="en" data-weft-route="${escapeHTML(p.route)}">`,
    '<head>',
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    ...prePaintTag(p.prePaint, p.who),
    ...headTags(p.head),
    ...p.icons,
    ...p.fonts.map((f) => `<link rel="preload" href="${f}" as="font" type="font/woff2" crossorigin>`),
    p.stylesheet === null ? `<style>${p.css}</style>` : `<link rel="stylesheet" href="${p.stylesheet}">`,
    ...p.tags,
    '</head>',
    `<body>${p.body}</body>`,
    '</html>',
    '',
  ].join('\n')
}
