import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { createRenderServer, renderPage } from '../src/render.ts'

const root = fileURLToPath(new URL('./fixtures/site', import.meta.url))

describe('renderPage', () => {
  it('renders a static page with no islands', async () => {
    const { html, islands } = await renderPage({ root, page: 'pages/index.vue' })
    expect(html).toContain('<h1>Hello from Weft</h1>')
    expect(html).not.toContain('<weft-island')
    expect(islands).toEqual([])
  })

  it('renders an island with its id, trigger, props and server HTML', async () => {
    const { html, islands } = await renderPage({ root, page: 'pages/counter.vue' })
    const found = [...html.matchAll(/<weft-island([^>]*)>([\s\S]*?)<\/weft-island>/g)]
    expect(found).toHaveLength(1)
    const [whole, attrs, inner] = found[0]!
    expect(whole.startsWith('<weft-island data-id="Counter" data-when="visible">')).toBe(true)
    expect(attrs).not.toContain('data-media')
    const json = /^<script type="application\/json">([\s\S]*?)<\/script>/.exec(inner!)
    expect(json?.[1]).toBe('{"start":2}')
    expect(inner).toContain('n is 2')
    expect(islands).toEqual(['Counter'])
  })

  it('names the file of each island, and renders through a server it is given', async () => {
    const server = await createRenderServer(root)
    try {
      const counter = await renderPage({ root, page: 'pages/counter.vue', server })
      const plain = await renderPage({ root, page: 'pages/index.vue', server })
      expect(counter.islandFiles).toEqual({ Counter: join(root, 'components/Counter.vue') })
      expect(plain.islandFiles).toEqual({})
    } finally {
      await server.close()
    }
  })

  it('keeps island props from ending their script element', async () => {
    const { html } = await renderPage({ root, page: 'pages/escape.vue' })
    expect(html).toContain('<weft-island data-id="Counter" data-when="media" data-media="(min-width: 40rem)">')
    const json = /<script type="application\/json">([\s\S]*?)<\/script>/.exec(html)![1]!
    expect(json).not.toMatch(/[<>&]/)
    expect(JSON.parse(json)).toEqual({ start: 1, note: '</script><!--&' })
  })

  it('refuses a media island without a media query', async () => {
    const bad = fileURLToPath(new URL('./fixtures/bad-island', import.meta.url))
    await expect(renderPage({ root: bad, page: 'pages/media-without-query.vue' })).rejects.toThrow(
      'Island: when="media" needs a media query in the media prop.',
    )
  })

  it('refuses island props that JSON would drop or change, naming the island and the prop', async () => {
    const bad = fileURLToPath(new URL('./fixtures/bad-island', import.meta.url))
    class Thing {}
    const loop: Record<string, unknown> = {}
    loop.self = loop
    const sentence = (path: string, what: string) =>
      `Island: prop ${path} of island Counter is ${what}, which JSON cannot carry to the browser; pass only strings, finite numbers, booleans, null, arrays and plain objects.`
    for (const [value, path, what] of [
      [() => 1, 'extra', 'a function'],
      [Symbol('s'), 'extra', 'a symbol'],
      [10n, 'extra', 'a bigint'],
      [undefined, 'extra', 'undefined'],
      [Number.NaN, 'extra', 'NaN'],
      [new Date(0), 'extra', 'a Date'],
      [new Map(), 'extra', 'a Map'],
      [new Thing(), 'extra', 'a Thing'],
      [{ deep: [1, () => 1] }, 'extra.deep[1]', 'a function'],
      [[1, , 3], 'extra[1]', 'an empty array slot'],
      [loop, 'extra.self', 'an object that contains itself'],
    ] as const) {
      await expect(renderPage({ root: bad, page: 'pages/props.vue', props: { value } }), String(path)).rejects.toThrow(
        new Error(sentence(path, what)),
      )
    }
    // Plain data, a null-prototype object included, renders.
    const plain = Object.assign(Object.create(null) as object, { a: [1, 'x', true, null, { b: 2 }] })
    const { html } = await renderPage({ root: bad, page: 'pages/props.vue', props: { value: plain } })
    expect(/<script type="application\/json">([\s\S]*?)<\/script>/.exec(html)![1]).toBe('{"start":1,"extra":{"a":[1,"x",true,null,{"b":2}]}}')
  })

  it('refuses an island component without an SFC name', async () => {
    const bad = fileURLToPath(new URL('./fixtures/bad-island', import.meta.url))
    await expect(renderPage({ root: bad, page: 'pages/unnamed.vue' })).rejects.toThrow(
      'Island: the component has no SFC __name; an island must be a <script setup> single-file component.',
    )
  })
})
