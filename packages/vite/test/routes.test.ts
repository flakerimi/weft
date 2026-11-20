import { mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, describe, expect, it } from 'vitest'
import { discoverRoutes, expandRoutes } from '../src/routes.ts'

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
})
