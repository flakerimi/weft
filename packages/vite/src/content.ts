import { createHash } from 'node:crypto'
import { existsSync } from 'node:fs'
import { readdir, readFile } from 'node:fs/promises'
import { extname, join, posix, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import GithubSlugger from 'github-slugger'
import markdownIt, { type MarkdownIt, type Token } from 'markdown-it'
import { bundledLanguagesInfo, createCssVariablesTheme, createHighlighter, type Highlighter } from 'shiki'
import { isMap, isScalar, LineCounter, parseDocument } from 'yaml'
import type { Plugin } from 'vite'

/** A frontmatter field's type in a collection's schema; with `?` after it, the field may be left out. */
export type FieldType = 'string' | 'number' | 'boolean' | 'string[]'

export interface CollectionConfig {
  /** Where the collection's pages are served, from the site root, e.g. `/docs`. Default `/<name>`. */
  base?: string
  /**
   * Frontmatter fields beyond the ones every collection has (`title` and
   * `description`, both strings, and `order`, an optional number): each
   * field's name and its type, such as `section: 'string'` or `tags: 'string[]?'`.
   */
  schema?: Record<string, FieldType | `${FieldType}?`>
}

/**
 * The default export of a site's `content.config.ts`: each collection, a
 * folder of Markdown under `content/`, by its folder's name.
 *
 *     import type { ContentConfig } from '@weft/vite'
 *     export default { docs: { schema: { section: 'string?' } } } satisfies ContentConfig
 */
export type ContentConfig = Record<string, CollectionConfig>

/** A page's frontmatter, checked against its collection's schema. */
export interface Frontmatter {
  title: string
  description: string
  order?: number
  [field: string]: unknown
}

/** A heading in a page's table of contents: its level (h2 or h3), its id and its text. */
export interface TocEntry {
  depth: 2 | 3
  id: string
  text: string
}

/** A page of a collection, as its neighbours and the collection's list show it. */
export interface ContentLink {
  slug: string
  /** Its route, e.g. `/docs/guides/islands`. */
  path: string
  title: string
  description: string
  order?: number
}

/**
 * A piece of a page's HTML. The page is cut at each code block marked `copy`
 * (```` ```sh copy ````), which comes as a part of its own with its source, for
 * the page to put a copy island beside; everything else is plain HTML.
 */
export type ContentPart = { html: string } | { html: string; code: string; lang: string }

export interface ContentPage {
  collection: string
  slug: string
  /** Its route, e.g. `/docs/guides/islands`. */
  path: string
  /** Its Markdown file, from the site root, e.g. `content/docs/guides/islands.md`. */
  file: string
  frontmatter: Frontmatter
  /** The rendered page: heading ids set, code highlighted, `.md` links turned into routes. */
  html: string
  /** The h2 and h3 headings, in order. */
  toc: TocEntry[]
  /** `html` in pieces; see ContentPart. Joined, they are `html`. */
  parts: ContentPart[]
  /** The pages before and after this one in the collection's order. */
  prev: ContentLink | null
  next: ContentLink | null
}

/** A site's content, loaded and checked. */
export interface Content {
  /** One page by its slug: its path in the collection without `.md` (`index.md` is its folder's slug, `''` at the top). */
  page(collection: string, slug: string): ContentPage
  /** Every page of the collection, ordered by `order` (pages without one last), then by slug. */
  list(collection: string): ContentLink[]
  /** The collection's slugs as a dynamic page's `getPaths()` returns them, each under `param` (default `slug`). */
  paths(collection: string, param?: string): Record<string, string>[]
  /** The images the pages show, each once: the URL the pages give it, and the file to serve there. */
  assets(): ContentAsset[]
}

/** An image of the content, served under a content-addressed name, `/assets/content/<name>-<hash><ext>`. */
export interface ContentAsset {
  path: string
  /** Its absolute path. */
  file: string
}

const BASE_FIELDS: Record<string, string> = { title: 'string', description: 'string', order: 'number?' }
const FIELD_TYPES = ['string', 'number', 'boolean', 'string[]']
const COLLECTION_KEYS = ['base', 'schema']
/** A folder or file name that is a URL segment as it is. */
const SEGMENT = /^[a-z0-9][a-z0-9_-]*$/
/** A link with a scheme (`https:`, `mailto:`) or to another host (`//host`). */
const EXTERNAL = /^(?:[a-z][a-z0-9+.-]*:|\/\/)/i
/** Shiki's languages with no grammar, which it draws as plain text. */
const PLAIN = new Set(['text', 'txt', 'plain', 'plaintext'])
/** The images a page may show from beside it; the build serves each under a content-addressed name. */
const IMAGES = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.avif', '.svg']

/** A collection as loaded: its config made whole, and its pages. */
interface Collection {
  name: string
  base: string
  schema: Map<string, { type: string; optional: boolean }>
  pages: Page[]
}

/** A page while it is loaded: the tokens are rendered once every page's ids are known. */
interface Page {
  collection: Collection
  slug: string
  path: string
  file: string
  frontmatter: Frontmatter
  tokens: Token[]
  ids: Set<string>
  toc: TocEntry[]
}

const fail = (file: string, line: number, why: string): never => {
  throw new Error(`${file}:${line}: ${why}`)
}

/**
 * Load the content of the site at `root`: every Markdown file of every
 * collection `config` (the default export of `content.config.ts`, or
 * undefined when there is none) declares under `content/`. Frontmatter is
 * checked against the collection's schema, headings get ids by GitHub's slug
 * rules, code is highlighted (with CSS variables, `--weft-code-*`, which
 * `@weft/vite/code.css` sets from the theme), and every link inside the
 * content is checked: relative `.md` links become the routes of their pages,
 * and a link to a page or heading that is not there fails, naming the file
 * and line. A relative image is one beside the page, which the build copies
 * to a content-addressed URL (see `assets()`); a missing one fails the same
 * way. External links are left as they are and never fetched.
 */
export async function loadContent(root: string, config: unknown): Promise<Content> {
  const collections = readConfig(config)
  const dir = join(root, 'content')
  const found = existsSync(dir) ? await readdir(dir, { withFileTypes: true }) : []
  const visible = found.filter((e) => !e.name.startsWith('.') && !e.name.startsWith('_'))
  if (config === undefined && visible.length) {
    throw new Error(
      'content/ has collections but the site has no content.config.ts; add one whose default export declares them, like { docs: {} }.',
    )
  }
  for (const e of visible) {
    if (e.isSymbolicLink()) fileRefused(`content/${e.name}`)
    if (!e.isDirectory()) {
      throw new Error(`content/${e.name} is not in a collection; put it in a collection's folder, like content/docs/.`)
    }
    if (!collections.has(e.name)) {
      throw new Error(`content/${e.name} is not a collection content.config.ts declares; declare it there, or move it out of content/.`)
    }
  }

  const md = markdownIt({ html: false, linkify: false, typographer: false })
  const byFile = new Map<string, Page>()
  const images = new Map<string, ContentAsset>()
  const langs = new Set<string>()
  for (const collection of collections.values()) {
    const folder = join(dir, collection.name)
    if (!existsSync(folder)) {
      throw new Error(`content.config.ts declares the collection ${collection.name}, but content/${collection.name} does not exist.`)
    }
    const bySlug = new Map<string, string>()
    const files = await collectionFiles(folder, `content/${collection.name}`)
    for (const image of files.images) {
      const file = join(root, image)
      const ext = extname(image)
      const hash = createHash('sha256').update(await readFile(file)).digest('hex').slice(0, 8)
      images.set(image, { path: `/assets/content/${posix.basename(image, ext)}-${hash}${ext}`, file })
    }
    for (const file of files.pages) {
      const page = await readPage(root, file, collection, md, langs)
      const other = bySlug.get(page.slug)
      if (other !== undefined) {
        const [a, b] = [other, file].sort()
        throw new Error(`${a} and ${b} are both the page ${page.slug || '(the top)'} of the ${collection.name} collection; remove one.`)
      }
      bySlug.set(page.slug, file)
      byFile.set(file, page)
      collection.pages.push(page)
    }
    collection.pages.sort((a, b) => (a.frontmatter.order ?? Infinity) - (b.frontmatter.order ?? Infinity) || compare(a.slug, b.slug))
  }

  const shown = new Map<string, ContentAsset>()
  for (const page of byFile.values()) checkLinks(page, byFile, collections, images, shown)

  const shiki = langs.size ? await highlighter([...langs]) : null
  md.renderer.rules.fence = (tokens, i) => highlight(shiki, tokens[i]!.content, fenceLang(tokens[i]!.info)) + '\n'
  md.renderer.rules.code_block = (tokens, i) => highlight(shiki, tokens[i]!.content, 'text') + '\n'

  const pages = new Map<string, Map<string, ContentPage>>()
  for (const collection of collections.values()) {
    const done = new Map<string, ContentPage>()
    collection.pages.forEach((page, i) => {
      const parts = render(md, page.tokens)
      done.set(page.slug, {
        collection: collection.name,
        slug: page.slug,
        path: page.path,
        file: page.file,
        frontmatter: page.frontmatter,
        html: parts.map((p) => p.html).join(''),
        toc: page.toc,
        parts,
        prev: i > 0 ? linkTo(collection.pages[i - 1]!) : null,
        next: i < collection.pages.length - 1 ? linkTo(collection.pages[i + 1]!) : null,
      })
    })
    pages.set(collection.name, done)
  }

  const of = (name: string) => {
    const collection = collections.get(name)
    if (!collection) {
      throw new Error(`useContent: content.config.ts declares no collection ${name}; add it there, with its folder content/${name}.`)
    }
    return collection
  }
  return {
    page(name, slug) {
      of(name)
      const page = pages.get(name)!.get(slug)
      if (!page) {
        throw new Error(`useContent: the ${name} collection has no page ${slug || '(the top)'} (content/${name}/${slug || 'index'}.md).`)
      }
      return page
    },
    list: (name) => of(name).pages.map(linkTo),
    paths: (name, param = 'slug') => of(name).pages.map((p) => ({ [param]: p.slug })),
    assets: () => [...shown.values()].sort((a, b) => compare(a.path, b.path)),
  }
}

const compare = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0)

function linkTo(page: Page): ContentLink {
  const { title, description, order } = page.frontmatter
  return { slug: page.slug, path: page.path, title, description, ...(order === undefined ? {} : { order }) }
}

/** The collections `config` declares, with their bases and whole schemas; a config it cannot read is refused. */
function readConfig(config: unknown): Map<string, Collection> {
  const out = new Map<string, Collection>()
  if (config === undefined) return out
  const plain = (v: unknown): v is Record<string, unknown> =>
    typeof v === 'object' && v !== null && !Array.isArray(v) && [Object.prototype, null].includes(Object.getPrototypeOf(v))
  if (!plain(config)) {
    throw new Error('content.config.ts: the default export is not a set of collections, like { docs: {} }.')
  }
  const bases = new Map<string, string>()
  for (const [name, value] of Object.entries(config)) {
    if (!SEGMENT.test(name)) {
      throw new Error(`content.config.ts: ${JSON.stringify(name)} is not a collection name; name it as its folder, in lowercase letters, digits, - and _.`)
    }
    const why = `content.config.ts: the ${name} collection`
    if (!plain(value)) throw new Error(`${why} is not an object like { schema: { … } }.`)
    const unknown = Object.keys(value).find((k) => !COLLECTION_KEYS.includes(k))
    if (unknown) throw new Error(`${why} has ${unknown}, which a collection does not take; it takes base and schema.`)
    const base = value.base ?? `/${name}`
    if (typeof base !== 'string' || !/^(?:\/|(?:\/[^/\s?#]+)+)$/.test(base)) {
      throw new Error(`${why}'s base is ${JSON.stringify(base)}; give a path from the site root with no trailing /, like /docs.`)
    }
    const other = bases.get(base)
    if (other) throw new Error(`content.config.ts: the ${other} and ${name} collections both have the base ${base}; give each its own.`)
    bases.set(base, name)
    const schema = new Map<string, { type: string; optional: boolean }>()
    const fields = { ...BASE_FIELDS }
    if (value.schema !== undefined) {
      if (!plain(value.schema)) throw new Error(`${why}'s schema is not an object of fields, like { section: 'string' }.`)
      for (const [field, type] of Object.entries(value.schema)) {
        if (field in BASE_FIELDS) throw new Error(`${why}'s schema names ${field}, which every collection has; leave it out.`)
        if (typeof type !== 'string' || !FIELD_TYPES.includes(type.replace(/\?$/, ''))) {
          throw new Error(
            `${why}'s schema gives ${field} the type ${JSON.stringify(type)}; use string, number, boolean or string[], with ? when optional.`,
          )
        }
        fields[field] = type
      }
    }
    for (const [field, type] of Object.entries(fields)) {
      schema.set(field, { type: type.replace(/\?$/, ''), optional: type.endsWith('?') })
    }
    out.set(name, { name, base, schema, pages: [] })
  }
  return out
}

const fileRefused = (shown: string): never => {
  throw new Error(`${shown} is a symlink, which Weft does not follow; put the file itself there.`)
}

/** The Markdown files and images under `folder`, as paths from the site root (`shown` is the folder's). */
async function collectionFiles(folder: string, shown: string, out = { pages: [] as string[], images: [] as string[] }) {
  const entries = await readdir(folder, { withFileTypes: true })
  for (const e of entries.sort((a, b) => compare(a.name, b.name))) {
    if (e.name.startsWith('.') || e.name.startsWith('_')) continue
    const path = `${shown}/${e.name}`
    if (e.isSymbolicLink()) fileRefused(path)
    if (e.isDirectory()) {
      if (!SEGMENT.test(e.name)) segmentRefused(path)
      await collectionFiles(join(folder, e.name), path, out)
    } else if (e.name.endsWith('.md') || IMAGES.includes(extname(e.name))) {
      if (!SEGMENT.test(e.name.slice(0, -extname(e.name).length))) segmentRefused(path)
      ;(e.name.endsWith('.md') ? out.pages : out.images).push(path)
    } else {
      throw new Error(`${path} is neither a .md file nor an image; put other files in public/.`)
    }
  }
  return out
}

const segmentRefused = (path: string): never => {
  throw new Error(`${path} has a name that is not a plain URL segment; use lowercase letters, digits, - and _.`)
}

/** Read one page: its frontmatter checked, its Markdown parsed, its headings given ids and its code blocks checked. */
async function readPage(root: string, file: string, collection: Collection, md: MarkdownIt, langs: Set<string>): Promise<Page> {
  // A byte order mark, as some editors write, is not part of the frontmatter.
  const text = (await readFile(join(root, file), 'utf8')).replace(/^\uFEFF/, '').replace(/\r\n?/g, '\n')
  const { frontmatter, body } = readFrontmatter(text, file, collection)
  const segments = file.slice(`content/${collection.name}/`.length, -'.md'.length).split('/')
  if (segments.at(-1) === 'index') segments.pop()
  const slug = segments.join('/')
  const { base } = collection
  const path = base === '/' ? `/${slug}` : slug ? `${base}/${slug}` : base

  const tokens = md.parse(body, {})
  const slugger = new GithubSlugger()
  const ids = new Set<string>()
  const toc: TocEntry[] = []
  tokens.forEach((token, i) => {
    if (token.type === 'heading_open') {
      const text = plainText(tokens[i + 1]!)
      const id = slugger.slug(text)
      if (!id) return
      token.attrSet('id', id)
      ids.add(id)
      const depth = Number(token.tag.slice(1))
      if (depth === 2 || depth === 3) toc.push({ depth, id, text })
    } else if (token.type === 'fence') {
      const line = token.map![0] + 1
      const [lang = '', ...words] = token.info.trim().split(/\s+/)
      const extra = words.find((w) => w !== 'copy')
      if (extra) fail(file, line, `the code block has ${JSON.stringify(extra)} after its language; the one word Weft reads there is copy.`)
      if (words.length && token.level !== 0) {
        fail(file, line, 'a copy code block must be at the top level of the page, not inside a list or a quote.')
      }
      const known = fenceLang(token.info)
      if (known === 'ansi') {
        fail(file, line, "the code block's language ansi draws terminal colours, which Weft's code colours do not cover; use text.")
      }
      if (!PLAIN.has(known) && !LANGS.has(known)) {
        fail(file, line, `the code block's language ${lang} is not one Shiki knows; use one of its language ids, or text.`)
      }
      langs.add(PLAIN.has(known) ? 'text' : LANGS.get(known)!)
    } else if (token.type === 'code_block') {
      langs.add('text')
    }
  })
  return { collection, slug, path, file, frontmatter, tokens, ids, toc }
}

/** Shiki's language ids and aliases, each to its id. */
const LANGS = new Map(bundledLanguagesInfo.flatMap((l) => [l.id, ...(l.aliases ?? [])].map((name) => [name, l.id] as const)))

/** The language a fence's info string names, in lowercase (`JSON` is `json`), `text` when it names none. */
const fenceLang = (info: string) => (info.trim().split(/\s+/)[0] || 'text').toLowerCase()

/** A heading's text as GitHub reads it for its slug: its text and code, without the markup. */
function plainText(inline: Token): string {
  return (inline.children ?? []).map((c) => (c.type === 'text' || c.type === 'code_inline' ? c.content : c.type === 'softbreak' ? ' ' : '')).join('')
}

/**
 * The frontmatter, checked against the collection's schema, and the body with
 * the frontmatter's lines left blank, so that its lines keep their numbers.
 */
function readFrontmatter(text: string, file: string, collection: Collection): { frontmatter: Frontmatter; body: string } {
  const lines = text.split('\n')
  let yaml = ''
  let body = text
  if (lines[0]!.trimEnd() === '---') {
    const end = lines.findIndex((l, i) => i > 0 && l.trimEnd() === '---')
    if (end === -1) fail(file, 1, 'the frontmatter opened here is never closed by a --- line.')
    yaml = lines.slice(1, end).join('\n')
    body = '\n'.repeat(end + 1) + lines.slice(end + 1).join('\n')
  }
  // The frontmatter starts on the file's second line.
  const counter = new LineCounter()
  const lineOf = (offset: number) => counter.linePos(offset).line + 1
  const doc = parseDocument(yaml, { lineCounter: counter, prettyErrors: false })
  const error = doc.errors[0]
  if (error) fail(file, lineOf(error.pos[0]), `the frontmatter is not valid YAML (${error.message}).`)
  const out: Record<string, unknown> = {}
  const name = collection.name
  if (doc.contents !== null) {
    if (!isMap(doc.contents)) fail(file, lineOf(doc.contents.range?.[0] ?? 0), 'the frontmatter is not a set of fields (name: value).')
    for (const pair of isMap(doc.contents) ? doc.contents.items : []) {
      const key = isScalar(pair.key) ? String(pair.key.value) : String(pair.key)
      const keyLine = lineOf(isScalar(pair.key) ? (pair.key.range?.[0] ?? 0) : 0)
      const field = collection.schema.get(key)
      if (!field) fail(file, keyLine, `${key} is not in the ${name} collection's schema; fix the name, or add it to the schema in content.config.ts.`)
      const value: unknown = (pair.value as { toJSON?: () => unknown } | null)?.toJSON?.() ?? null
      const valueLine = pair.value && typeof pair.value === 'object' && 'range' in pair.value ? lineOf((pair.value.range as number[])[0]!) : keyLine
      if (!fits(value, field!.type)) {
        fail(file, valueLine, `${key} is ${JSON.stringify(value)}, but the ${name} collection's schema makes it ${TYPE_NAMES[field!.type]}.`)
      }
      out[key] = value
    }
  }
  for (const [field, { optional }] of collection.schema) {
    if (!optional && !(field in out)) fail(file, 1, `the frontmatter has no ${field}, which every page of the ${name} collection needs.`)
  }
  return { frontmatter: out as Frontmatter, body }
}

const TYPE_NAMES: Record<string, string> = { string: 'a string', number: 'a number', boolean: 'true or false', 'string[]': 'a list of strings' }

function fits(value: unknown, type: string): boolean {
  if (type === 'string[]') return Array.isArray(value) && value.every((v) => typeof v === 'string')
  if (type === 'number') return typeof value === 'number' && Number.isFinite(value)
  return typeof value === type
}

/** The tokens of a page in document order, each with the line it is on, children included. */
function* withLines(tokens: Token[]): Generator<[Token, number]> {
  let line = 1
  for (const token of tokens) {
    if (token.map) line = token.map[0] + 1
    yield [token, line]
    let inner = line
    for (const child of token.children ?? []) {
      if (child.type === 'softbreak' || child.type === 'hardbreak') inner++
      yield [child, inner]
    }
  }
}

/**
 * Check every link and image of `page`. Relative `.md` links are rewritten to
 * their pages' routes, and relative images to the URLs of `images` (by path
 * from the site root); each image shown is added to `shown`.
 */
function checkLinks(
  page: Page,
  byFile: Map<string, Page>,
  collections: Map<string, Collection>,
  images: Map<string, ContentAsset>,
  shown: Map<string, ContentAsset>,
) {
  const { file } = page
  const decode = (s: string) => {
    try {
      return decodeURIComponent(s)
    } catch {
      return s
    }
  }
  const checkHash = (line: number, href: string, target: Page, hash: string) => {
    if (!hash || target.ids.has(hash)) return
    const where = target === page ? 'this page has' : `${target.file} has`
    fail(file, line, `the link to ${href} leads to no heading; ${where} no heading with the id ${hash}.`)
  }
  for (const [token, line] of withLines(page.tokens)) {
    if (token.type === 'image') {
      const src = String(token.attrGet('src') ?? '')
      if (!src || EXTERNAL.test(src) || src.startsWith('/')) continue
      const written = decode(src)
      const path = written.replace(/[?#].*$/, '')
      if (!IMAGES.includes(posix.extname(path))) {
        fail(file, line, `the image ${written} is not a ${IMAGES.slice(0, -1).join(', ')} or ${IMAGES.at(-1)} file.`)
      }
      const to = posix.normalize(posix.join(posix.dirname(file), path))
      if (!to.startsWith('content/')) {
        fail(file, line, `the image ${written} leads outside content/; put it beside the page, or in public/ with a path from the site root.`)
      }
      const image = images.get(to)
      if (!image) fail(file, line, `the image ${written} leads to no file; ${to} does not exist.`)
      token.attrSet('src', image!.path)
      shown.set(image!.path, image!)
      continue
    }
    if (token.type !== 'link_open') continue
    const raw = String(token.attrGet('href') ?? '')
    if (!raw || EXTERNAL.test(raw)) continue
    const href = decode(raw)
    const hashAt = raw.indexOf('#')
    const hash = hashAt === -1 ? '' : decode(raw.slice(hashAt + 1))
    const beforeHash = hashAt === -1 ? raw : raw.slice(0, hashAt)
    const queryAt = beforeHash.indexOf('?')
    const query = queryAt === -1 ? '' : beforeHash.slice(queryAt)
    const path = decode(queryAt === -1 ? beforeHash : beforeHash.slice(0, queryAt))
    if (path === '') {
      checkHash(line, href, page, hash)
    } else if (path.startsWith('/')) {
      // A path from the site root into a collection: the page it names must be there.
      const collection = [...collections.values()]
        .filter((c) => c.base === '/' || path === c.base || path.startsWith(c.base + '/'))
        .sort((a, b) => b.base.length - a.base.length)[0]
      if (!collection) continue
      const slug = collection.base === '/' ? path.slice(1) : path.slice(collection.base.length + 1)
      const target = collection.pages.find((p) => p.slug === slug)
      if (!target) fail(file, line, `the link to ${href} leads to no page; the ${collection.name} collection has no page ${slug}.`)
      checkHash(line, href, target!, hash)
    } else {
      if (!path.endsWith('.md')) {
        fail(file, line, `the link to ${href} is relative but not to a .md file; link to the page's .md file, or give a path from the site root.`)
      }
      const to = posix.normalize(posix.join(posix.dirname(file), path))
      if (!to.startsWith('content/')) fail(file, line, `the link to ${href} leads outside content/; link to the .md file of a page in a collection.`)
      const target = byFile.get(to)
      if (!target) fail(file, line, `the link to ${href} leads to no page; ${to} does not exist.`)
      checkHash(line, href, target!, hash)
      token.attrSet('href', target!.path + query + (hashAt === -1 ? '' : raw.slice(hashAt)))
    }
  }
}

/** The page's HTML in parts, cut at each top-level code block marked copy. */
function render(md: MarkdownIt, tokens: Token[]): ContentPart[] {
  const parts: ContentPart[] = []
  let from = 0
  const flush = (to: number) => {
    if (to > from) parts.push({ html: md.renderer.render(tokens.slice(from, to), md.options, {}) })
  }
  tokens.forEach((token, i) => {
    if (token.type !== 'fence' || !token.info.trim().split(/\s+/).includes('copy')) return
    flush(i)
    parts.push({ html: md.renderer.render([token], md.options, {}), code: token.content, lang: fenceLang(token.info) })
    from = i + 1
  })
  flush(tokens.length)
  return parts
}

/**
 * Code colours as CSS variables (`--weft-code-token-keyword` and the rest),
 * so that one rendering reads in light and dark: `code.css` sets them from the
 * theme's colours.
 */
export const CODE_THEME = createCssVariablesTheme({ name: 'weft', variablePrefix: '--weft-code-', variableDefaults: {}, fontStyle: true })

let shared: Promise<Highlighter> | undefined
/** The one Shiki highlighter of the process, with `langs` loaded. */
async function highlighter(langs: string[]): Promise<Highlighter> {
  shared ??= createHighlighter({ themes: [CODE_THEME], langs: [] })
  const shiki = await shared
  const missing = langs.filter((l) => !PLAIN.has(l) && !shiki.getLoadedLanguages().includes(l))
  if (missing.length) await shiki.loadLanguage(...(missing as Parameters<Highlighter['loadLanguage']>))
  return shiki
}

/** A code block as Shiki draws it; Shiki is started only when the content has a code block. */
function highlight(shiki: Highlighter | null, code: string, lang: string): string {
  return shiki!.codeToHtml(code.replace(/\n$/, ''), { lang: PLAIN.has(lang) ? 'text' : LANGS.get(lang)!, theme: 'weft' })
}

/**
 * Source a page reads itself (an example's file, say), highlighted as the
 * content's code blocks are: by Shiki at build time, with the colours as
 * `--weft-code-*` variables that `code.css` sets. `lang` is a Shiki language
 * id or alias in any case, or `text`; one it does not know is refused.
 */
export async function highlightCode(code: string, lang: string): Promise<string> {
  const known = lang.toLowerCase()
  if (!PLAIN.has(known) && !LANGS.has(known)) {
    throw new Error(`highlightCode: the language ${lang} is not one Shiki knows; use one of its language ids, or text.`)
  }
  const id = PLAIN.has(known) ? 'text' : LANGS.get(known)!
  return highlight(await highlighter([id]), code, id)
}

const contentId = 'virtual:weft/content'
const resolvedContentId = '\0' + contentId
/** An absolute file path as Vite takes it in an import, on every OS. */
const fsImport = (file: string) => {
  const posixPath = file.split(sep).join('/')
  return '/@fs' + (posixPath.startsWith('/') ? '' : '/') + posixPath
}

/**
 * Serves `virtual:weft/content` to pages on the server: `useContent`,
 * `useCollection` and `contentPaths`, and `highlightCode` for source a page
 * reads itself, over the site's content loaded once, with
 * `content.config.ts` (when the site has one) as its config. An island runs in
 * the browser, where there is no content to read, so it is refused there.
 */
export function weftContent(): Plugin {
  let root = process.cwd()
  return {
    name: 'weft:content',
    configResolved(config) {
      root = config.root
    },
    resolveId(id) {
      if (id === contentId) return resolvedContentId
    },
    load(id, options) {
      if (id !== resolvedContentId) return
      if (!options?.ssr) {
        throw new Error(`${contentId} is read by pages on the server; an island runs in the browser, so give it what it needs in its props.`)
      }
      const config = join(root, 'content.config.ts')
      return [
        existsSync(config) ? `import config from ${JSON.stringify(fsImport(config))}` : 'const config = undefined',
        `import { loadContent, highlightCode } from ${JSON.stringify(fsImport(fileURLToPath(import.meta.url)))}`,
        `const content = await loadContent(${JSON.stringify(root)}, config)`,
        'export const useContent = (collection, slug) => content.page(collection, slug)',
        'export const useCollection = (collection) => content.list(collection)',
        'export const contentPaths = (collection, param) => content.paths(collection, param)',
        'export { highlightCode }',
        // For buildSite, which copies the images the pages show into the build.
        'export const contentAssets = () => content.assets()',
        '',
      ].join('\n')
    },
  }
}
