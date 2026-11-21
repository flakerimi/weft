// A site whose docs are Markdown: pages/docs/[...slug].vue renders each page of
// the docs collection through useContent, and the build checks the content.
import { createHash } from 'node:crypto'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest'
import { buildSite, type Manifest } from '../src/build.ts'
import type { RoutesJson } from '../src/routes.ts'

const fixture = (n: string) => fileURLToPath(new URL(`./fixtures/${n}`, import.meta.url))
const temps: string[] = []
const tempDir = () => {
  const dir = mkdtempSync(join(tmpdir(), 'weft-content-site-'))
  temps.push(dir)
  return dir
}
afterAll(() => {
  for (const dir of temps) rmSync(dir, { recursive: true, force: true })
})

const scriptTypes = (html: string) => [...html.matchAll(/<script\b([^>]*)>/g)].map((m) => /\btype="([^"]*)"/.exec(m[1]!)?.[1])

describe('buildSite on a site with Markdown docs', () => {
  let out: string
  const read = (p: string) => readFileSync(join(out, p), 'utf8')

  beforeAll(async () => {
    out = tempDir()
    await buildSite({ root: fixture('content-site'), outDir: join(out, 'dist') })
    out = join(out, 'dist')
  })

  it('makes a page per Markdown file through the catch-all route', () => {
    const routes = JSON.parse(read('routes.json')) as RoutesJson
    expect(routes.pages.map((p) => p.path)).toEqual(['/docs', '/docs/guides/serving', '/docs/guides/islands'])
  })

  it('renders the Markdown with heading ids, a table of contents, routes for .md links and the head', () => {
    const html = read('pages/docs/guides/serving/index.html')
    expect(html).toContain('<title>Serving</title>')
    expect(html).toContain('<h2 id="etags">ETags</h2>')
    expect(html).toContain('<h3 id="compression">Compression</h3>')
    expect(html).toContain('<a href="#etags">ETags</a><a href="#compression">Compression</a>')
    expect(html).toContain('<a href="/docs">the docs</a>')
    expect(read('pages/docs/index.html')).toContain('<a href="/docs/guides/islands#when-they-load">islands</a>')
    expect(html).toContain('<a class="prev" href="/docs">Docs</a>')
    expect(html).toContain('<a class="next" href="/docs/guides/islands">Islands</a>')
  })

  it('highlights code at build time, with the theme colours in the page CSS and no script', () => {
    const html = read('pages/docs/guides/serving/index.html')
    expect(html).toContain('<pre class="shiki weft"')
    expect(html).toContain('style="color:var(--weft-code-token-keyword)">func</span>')
    // code.css, imported by the page, sets the variables from the theme.
    expect(html).toMatch(/--weft-code-token-keyword:\s*var\(--weft-color-danger-text\)/)
    expect(scriptTypes(html)).toEqual(['speculationrules'])
    expect(html).not.toContain('modulepreload')
  })

  it('copies an image beside a page into the build under a content-addressed name, cached forever', () => {
    const bytes = readFileSync(join(fixture('content-site'), 'content/docs/guides/flow.png'))
    const src = `/assets/content/flow-${createHash('sha256').update(bytes).digest('hex').slice(0, 8)}.png`
    expect(read('pages/docs/guides/serving/index.html')).toContain(`<img src="${src}" alt="The flow">`)
    expect(readFileSync(join(out, src))).toEqual(bytes)
    const manifest = JSON.parse(read('manifest.json')) as Manifest
    expect(manifest.files[src]).toMatchObject({ type: 'image/png', immutable: true, br: false })
  })

  it('keeps a copy block holding </script> and <!-- inert, in the page and in the island props', () => {
    const html = read('pages/docs/guides/islands/index.html')
    // Every <script> the page opens is closed once, by its own end tag.
    expect(html.match(/<script\b/g)!.length).toBe(html.match(/<\/script>/g)!.length)
    expect(html).not.toContain('<!-- note')
    expect(html).toContain('{"code":"\\u003cscript\\u003ealert(1)\\u003c/script\\u003e\\u003c!-- note --\\u003e\\n"}')
    expect(html).toContain('&#x3C;/')
  })

  it('gives a block marked copy an island, and only that page a script', () => {
    const html = read('pages/docs/guides/islands/index.html')
    expect(html).toContain('<weft-island data-id="CopyCode"')
    expect(html).toContain('{"code":"pnpm add @weft/ui\\n"}')
    expect(scriptTypes(html)).toContain('module')
    expect(scriptTypes(read('pages/docs/index.html'))).toEqual(['speculationrules'])
  })
})

describe('buildSite on content with a broken link', () => {
  it('fails, naming the file and line', async () => {
    await expect(buildSite({ root: fixture('content-broken'), outDir: join(tempDir(), 'dist') })).rejects.toThrow(
      new Error('content/docs/index.md:6: the link to guide.md leads to no page; content/docs/guide.md does not exist.'),
    )
  })
})

describe('buildSite on an island that reads content', () => {
  it('refuses it, since the browser has no content to read', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      await expect(buildSite({ root: fixture('content-island'), outDir: join(tempDir(), 'dist') })).rejects.toThrow(
        'virtual:weft/content is read by pages on the server; an island runs in the browser, so give it what it needs in its props.',
      )
    } finally {
      error.mockRestore()
    }
  })
})
