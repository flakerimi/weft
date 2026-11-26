import { mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, describe, expect, it } from 'vitest'
import { byPattern, discoverRoutes, expandRoutes, matchPattern } from '../src/routes.ts'

const dir = (n: string) => fileURLToPath(new URL(`./fixtures/${n}`, import.meta.url))

const temps: string[] = []
afterAll(() => {
  for (const t of temps) rmSync(t, { recursive: true, force: true })
})
/** A pages folder in a temp folder holding these files (each an empty page). */
const pagesWith = (...files: string[]) => {
  const pages = join(mkdtempSync(join(tmpdir(), 'weft-routes-')), 'pages')
  temps.push(join(pages, '..'))
  for (const f of files) {
    mkdirSync(join(pages, f, '..'), { recursive: true })
    writeFileSync(join(pages, f), '<template><p /></template>\n')
  }
  return pages
}

describe('discoverRoutes', () => {
  it('maps files to paths, static before dynamic, alphabetical', async () => {
    expect(await discoverRoutes(dir('routes-basic'))).toEqual([
      { path: '/', file: 'index.vue', dynamic: false },
      { path: '/404', file: '404.vue', dynamic: false },
      { path: '/about', file: 'about.vue', dynamic: false },
      { path: '/blog', file: 'blog/index.vue', dynamic: false },
      { path: '/blog/:slug', file: 'blog/[slug].vue', dynamic: true },
    ])
  })

  it('puts dynamic routes after static ones even when they sort earlier', async () => {
    const routes = await discoverRoutes(dir('routes-order'))
    expect(routes.map((r) => r.path)).toEqual(['/', '/about', '/:id'])
  })

  it('throws when two files make the same path', async () => {
    await expect(discoverRoutes(dir('routes-clash'))).rejects.toThrow(
      'pages "about.vue" and "about/index.vue" both make /about',
    )
  })

  it('throws when two files make paths that differ only in case, on every OS', async () => {
    await expect(discoverRoutes(pagesWith('About.vue', 'about/index.vue'))).rejects.toThrow(
      new Error('pages "About.vue" and "about/index.vue" both make /about'),
    )
  })

  it('refuses a symlinked page or folder, in one sentence', async () => {
    const elsewhere = pagesWith('real.vue', 'blog/post.vue')
    const file = pagesWith('index.vue')
    symlinkSync(join(elsewhere, 'real.vue'), join(file, 'linked.vue'))
    await expect(discoverRoutes(file)).rejects.toThrow(
      new Error('pages/linked.vue is a symlink, which Weft does not follow; put the page itself in pages/.'),
    )
    const folder = pagesWith('index.vue')
    mkdirSync(join(folder, 'docs'))
    symlinkSync(join(elsewhere, 'blog'), join(folder, 'docs/blog'))
    await expect(discoverRoutes(folder)).rejects.toThrow(
      new Error('pages/docs/blog is a symlink, which Weft does not follow; put the folder itself in pages/.'),
    )
  })

  it('reads [...name] as a catch-all, which must be the last segment', async () => {
    expect(await discoverRoutes(pagesWith('docs/[...slug].vue', 'docs/index.vue'))).toEqual([
      { path: '/docs', file: 'docs/index.vue', dynamic: false },
      { path: '/docs/:...slug', file: 'docs/[...slug].vue', dynamic: true },
    ])
    expect(await discoverRoutes(pagesWith('docs/[...slug]/index.vue'))).toEqual([
      { path: '/docs/:...slug', file: 'docs/[...slug]/index.vue', dynamic: true },
    ])
    await expect(discoverRoutes(pagesWith('docs/[...slug]/edit.vue'))).rejects.toThrow(
      new Error(
        'page "docs/[...slug]/edit.vue" has the catch-all [...slug] before its last segment; a catch-all takes the rest of the path, so it must come last.',
      ),
    )
  })
})

describe('expandRoutes', () => {
  it('expands dynamic routes through getPaths', async () => {
    const routes = await discoverRoutes(dir('routes-basic'))
    const out = await expandRoutes(routes, async () => ({
      getPaths: async () => [{ slug: 'a' }, { slug: 'b' }],
    }))
    expect(out.map((r) => r.path)).toEqual(['/', '/404', '/about', '/blog', '/blog/a', '/blog/b'])
    expect(out.at(-1)).toEqual({ path: '/blog/b', file: 'blog/[slug].vue', params: { slug: 'b' } })
    expect(out[0]).toEqual({ path: '/', file: 'index.vue', params: {} })
  })

  it('throws when a dynamic page yields a path a static page owns', async () => {
    const routes = await discoverRoutes(dir('routes-dynamic'))
    await expect(
      expandRoutes(routes, async () => ({ getPaths: async () => [{ slug: 'about' }] })),
    ).rejects.toThrow('pages "about.vue" and "[slug].vue" both make /about')
  })

  it('throws when a dynamic page yields a path a static page owns in another case', async () => {
    const routes = await discoverRoutes(dir('routes-dynamic'))
    await expect(
      expandRoutes(routes, async () => ({ getPaths: async () => [{ slug: 'About' }] })),
    ).rejects.toThrow(new Error('pages "about.vue" and "[slug].vue" both make /About'))
  })

  it('throws, naming the file, when getPaths is missing', async () => {
    const routes = await discoverRoutes(dir('routes-dynamic'))
    await expect(expandRoutes(routes, async () => ({}))).rejects.toThrow(
      'page "[slug].vue" is dynamic but does not export getPaths()',
    )
  })

  it('throws when getPaths misses a param', async () => {
    const routes = await discoverRoutes(dir('routes-multi'))
    await expect(
      expandRoutes(routes, async () => ({ getPaths: async () => [{ year: '2026' }] })),
    ).rejects.toThrow('getPaths() of "[year]/[slug].vue" returned no "slug"')
  })

  it('fills every param of a multi-param route', async () => {
    const routes = await discoverRoutes(dir('routes-multi'))
    expect(routes).toEqual([{ path: '/:year/:slug', file: '[year]/[slug].vue', dynamic: true }])
    const out = await expandRoutes(routes, async () => ({
      getPaths: async () => [{ year: '2026', slug: 'hi' }],
    }))
    expect(out).toEqual([{ path: '/2026/hi', file: '[year]/[slug].vue', params: { year: '2026', slug: 'hi' } }])
  })

  it('fills a catch-all with several segments, or none for the folder itself', async () => {
    const routes = await discoverRoutes(pagesWith('docs/[...slug].vue', '[...rest].vue'))
    const out = await expandRoutes(routes, async (file) => ({
      getPaths: async () => (file === 'docs/[...slug].vue' ? [{ slug: '' }, { slug: 'guides/islands' }] : [{ rest: '' }, { rest: 'a/b/c' }]),
    }))
    expect(out).toEqual([
      { path: '/', file: '[...rest].vue', params: { rest: '' } },
      { path: '/a/b/c', file: '[...rest].vue', params: { rest: 'a/b/c' } },
      { path: '/docs', file: 'docs/[...slug].vue', params: { slug: '' } },
      { path: '/docs/guides/islands', file: 'docs/[...slug].vue', params: { slug: 'guides/islands' } },
    ])
  })

  it('refuses a catch-all value with an empty, . or .. segment', async () => {
    const routes = await discoverRoutes(pagesWith('docs/[...slug].vue'))
    for (const slug of ['a//b', '/a', 'a/', 'a/../b', './a']) {
      await expect(expandRoutes(routes, async () => ({ getPaths: async () => [{ slug }] }))).rejects.toThrow(
        new Error(
          `getPaths() of "docs/[...slug].vue" returned ${JSON.stringify(slug)} for slug, which has an empty, . or .. segment; give segments joined by /, or "" for the folder itself.`,
        ),
      )
    }
  })

  it('refuses a value of a plain param that is not one path segment', async () => {
    const routes = await discoverRoutes(dir('routes-dynamic'))
    for (const slug of ['a/b', '', '..']) {
      await expect(expandRoutes(routes, async () => ({ getPaths: async () => [{ slug }] }))).rejects.toThrow(
        new Error(
          `getPaths() of "[slug].vue" returned ${JSON.stringify(slug)} for slug, which is not one path segment; a page named [...slug].vue takes several.`,
        ),
      )
    }
  })
})

describe('runtime pages', () => {
  it('expands a page that exports runtime = true to one shell, its params empty', async () => {
    const routes = await discoverRoutes(pagesWith('products/[id].vue', 'products/new.vue', 'docs/[...rest].vue'))
    const out = await expandRoutes(routes, async () => ({ runtime: true }))
    expect(out).toEqual([
      { path: '/products/new', file: 'products/new.vue', params: {} },
      { path: '/docs/:...rest', file: 'docs/[...rest].vue', params: { rest: '' }, runtime: true },
      { path: '/products/:id', file: 'products/[id].vue', params: { id: '' }, runtime: true },
    ])
  })

  it('takes any other value of runtime for no export', async () => {
    const routes = await discoverRoutes(pagesWith('products/[id].vue'))
    for (const runtime of [false, 'true', 1, null]) {
      expect(await expandRoutes(routes, async () => ({ runtime, getPaths: async () => [{ id: 'a' }] }))).toEqual([
        { path: '/products/a', file: 'products/[id].vue', params: { id: 'a' } },
      ])
    }
  })

  it('runtime and getPaths together fail', async () => {
    const routes = await discoverRoutes(pagesWith('products/[id].vue'))
    await expect(expandRoutes(routes, async () => ({ runtime: true, getPaths: async () => [] }))).rejects.toThrow(
      new Error('page "products/[id].vue" exports both runtime and getPaths; keep one.'),
    )
  })

  it('neither fails as before', async () => {
    const routes = await discoverRoutes(pagesWith('products/[id].vue'))
    for (const mod of [{}, { runtime: false }]) {
      await expect(expandRoutes(routes, async () => mod)).rejects.toThrow(
        new Error('page "products/[id].vue" is dynamic but does not export getPaths()'),
      )
    }
  })

  it('refuses two runtime pages of the same shape, and allows one an exact page overlaps', async () => {
    const routes = await discoverRoutes(pagesWith('a/[x].vue', 'a/[y].vue'))
    await expect(expandRoutes(routes, async () => ({ runtime: true }))).rejects.toThrow(
      new Error('pages "a/[x].vue" and "a/[y].vue" both match paths like /a/1; keep one.'),
    )
    const rest = await discoverRoutes(pagesWith('docs/[...a].vue', 'docs/[...b]/index.vue'))
    await expect(expandRoutes(rest, async () => ({ runtime: true }))).rejects.toThrow(
      new Error('pages "docs/[...a].vue" and "docs/[...b]/index.vue" both match paths like /docs/1; keep one.'),
    )
    // Of another shape, or a page with getPaths, is allowed: the order, and then the exact page, decide.
    const other = await discoverRoutes(pagesWith('a/[x].vue', 'a/[...y].vue', '[s]/[t].vue', 'a/b.vue'))
    expect((await expandRoutes(other, async () => ({ runtime: true }))).map((r) => r.path)).toEqual(['/a/b', '/:s/:t', '/a/:...y', '/a/:x'])
  })

  it('orders patterns so that a catch-all never hides a more specific page', () => {
    // More static segments first; then a catch-all last; then fewer params, but
    // the longer of two catch-alls first; then by path.
    const paths = [
      '/:a', '/:section/:id', '/docs/:...rest', '/products/:id', '/products/:id/edit', '/blog/:slug',
      '/a/b/:...c', '/:...all', '/docs/:a/:b', '/docs/:a/:...r', '/a/:x/:...r', '/a/:x/:y/:z',
    ]
    expect(paths.map((path) => ({ path })).sort(byPattern).map((r) => r.path)).toEqual([
      '/products/:id/edit',
      '/a/b/:...c',
      '/blog/:slug',
      '/products/:id',
      '/docs/:a/:b',
      '/a/:x/:y/:z',
      '/a/:x/:...r',
      '/docs/:a/:...r',
      '/docs/:...rest',
      '/:a',
      '/:section/:id',
      '/:...all',
    ])
    // So each of them answers some path: the first that matches is never a pattern listed earlier that covers it.
    const sorted = paths.map((path) => ({ path })).sort(byPattern)
    for (const [path, want] of [
      ['/docs/x/y', '/docs/:a/:b'],
      ['/docs/x/y/z', '/docs/:a/:...r'],
      ['/docs/x', '/docs/:a/:...r'],
      ['/docs', '/docs/:...rest'],
      ['/a/x/y/z', '/a/:x/:y/:z'],
      ['/a/x/y', '/a/:x/:...r'],
      ['/x/y', '/:section/:id'],
      ['/x', '/:a'],
      ['/x/y/z', '/:...all'],
    ] as const) {
      expect(sorted.find((r) => matchPattern(r.path, path) !== null)?.path, path).toBe(want)
    }
  })

  it('matches a path against a pattern as warp.Serve does', () => {
    expect(matchPattern('/products/:id', '/products/42')).toEqual({ id: '42' })
    expect(matchPattern('/:section/:id', '/a/b')).toEqual({ section: 'a', id: 'b' })
    expect(matchPattern('/docs/:...rest', '/docs')).toEqual({ rest: '' })
    expect(matchPattern('/docs/:...rest', '/docs/a/b')).toEqual({ rest: 'a/b' })
    expect(matchPattern('/:...rest', '/')).toEqual({ rest: '' })
    // Cleaned as path.Clean cleans it: empty and . segments dropped, .. taking one back.
    expect(matchPattern('/products/:id', '/products//42/')).toEqual({ id: '42' })
    expect(matchPattern('/products/:id', '/products/x/../42')).toEqual({ id: '42' })
    expect(matchPattern('/products/:id', '/products/./42')).toEqual({ id: '42' })
    expect(matchPattern('/products/:id', '/products/../42')).toBeNull()
    for (const path of ['/products', '/products/', '/products/1/x', '/Products/1', '/product/1', '/']) {
      expect(matchPattern('/products/:id', path), path).toBeNull()
    }
  })
})
