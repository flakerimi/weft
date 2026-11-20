// The server halves: a page using every headless composable renders through
// renderToString (renderPage), and an island drawing an Icon renders the
// markup icon-island.test.ts hydrates. Node, no DOM.
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { lucideBody } from '../src/icons.ts'
import { renderPage } from '../src/render.ts'
import { ratedIsland } from './fixtures/icon-island/markup.ts'

const fixture = (n: string) => fileURLToPath(new URL(`./fixtures/${n}`, import.meta.url))

describe('server rendering', () => {
  it('renders a component using every headless composable', async () => {
    expect(typeof window).toBe('undefined')
    const { html } = await renderPage({ root: fixture('headless'), page: 'pages/index.vue' })
    expect(html).toContain('<button type="button" aria-controls="menu-v-0" aria-expanded="true">first</button>')
    expect(html).toContain('<ul id="menu-v-0" role="menu" style="position:absolute;left:0px;top:0px;" data-positioned="false"')
    expect(html.match(/role="menuitem"/g)).toHaveLength(2)
  })

  it('renders an island that draws an Icon', async () => {
    const { html } = await renderPage({ root: fixture('icon-island'), page: 'pages/index.vue' })
    expect(html).toContain(ratedIsland(lucideBody('star')!))
  })
})
