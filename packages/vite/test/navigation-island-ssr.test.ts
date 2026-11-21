// The server half of the navigation island: every navigation and disclosure
// component renders under Node, with no window or document, to the markup
// navigation-island.test.ts hydrates. No dropdown or sheet that opens, no
// #weft-overlays root; hidden content is rendered, hidden until found; the
// app shell leaves its layout to CSS until it mounts.
import { fileURLToPath } from 'node:url'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'
import { navigationIsland } from './fixtures/navigation-island/markup.ts'

it('renders an island of every navigation and disclosure component', async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/navigation-island', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain(navigationIsland)
  expect(html).not.toContain('weft-overlays')
  expect(html).not.toContain('role="menu"')
  expect(html).not.toContain('role="dialog"')
  // Content that opens is on the page for find-in-page, hidden until found.
  expect(navigationIsland).toMatch(/role="region"[^>]*hidden="until-found"[^>]*>[\s\S]*?Two days/)
  expect(navigationIsland).toMatch(/class="weft-collapsible__content" hidden="until-found">[\s\S]*?Hidden details/)
  expect(navigationIsland).not.toContain('data-layout')
  // The current page and step, as far as the server knows them.
  expect(navigationIsland).toContain('aria-current="step"')
  expect(navigationIsland).toMatch(/aria-label="Page 3"[^>]*aria-current="page"|aria-current="page"[^>]*aria-label="Page 3"/)
})
