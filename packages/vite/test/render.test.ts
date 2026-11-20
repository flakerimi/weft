import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

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

  it('keeps island props from ending their script element', async () => {
    const { html } = await renderPage({ root, page: 'pages/escape.vue' })
    expect(html).toContain('<weft-island data-id="Counter" data-when="media" data-media="(min-width: 40rem)">')
    const json = /<script type="application\/json">([\s\S]*?)<\/script>/.exec(html)![1]!
    expect(json).not.toMatch(/[<>&]/)
    expect(JSON.parse(json)).toEqual({ start: 1, note: '</script><!--&' })
  })
})
