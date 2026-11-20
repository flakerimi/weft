import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { discoverRoutes, expandRoutes } from '../src/routes.ts'

const dir = (n: string) => fileURLToPath(new URL(`./fixtures/${n}`, import.meta.url))

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
