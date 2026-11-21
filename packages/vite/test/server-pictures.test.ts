// What a page can draw of an app with no script (a landing page's picture
// of one): a DropdownMenu with `:portal="false"` drawn open where it stands,
// floating on pointer and as a sheet over its backdrop on touch; and a
// WeftProvider's `width`, the width class drawn before anything is measured,
// so a Table inside a narrow one is drawn as its cards. A menu teleported as
// usual (portal) still draws nothing on the server.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

const section = (html: string, id: string) => {
  const start = html.indexOf(`<section id="${id}">`)
  expect(start, id).toBeGreaterThanOrEqual(0)
  return html.slice(start, html.indexOf('</section>', start))
}

it('draws open menus in place and narrow tables as cards, on the server', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/server-pictures', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })

  const pointer = section(html, 'pointer')
  expect(pointer).toMatch(/class="weft-menu"[^>]*role="menu"/)
  expect(pointer).toContain('data-form="floating"')
  expect(pointer).toContain('data-positioned="true"')
  expect(pointer).toContain('Open device')
  expect(pointer).not.toContain('weft-overlay-backdrop')

  const touch = section(html, 'touch')
  expect(touch).toContain('data-form="sheet"')
  expect(touch).toContain('weft-sheet')
  expect(touch).toContain('weft-overlay-backdrop')
  expect(touch).toContain('data-weft-input="touch"')

  expect(section(html, 'closed')).not.toContain('role="menu"')
  expect(section(html, 'portal')).not.toContain('role="menu"')
  expect(html).not.toContain('weft-overlays')

  expect(section(html, 'narrow')).toContain('data-layout="cards"')
  expect(section(html, 'wide')).toContain('data-layout="table"')
})
