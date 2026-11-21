// @vitest-environment happy-dom
// An island of every navigation and disclosure component hydrates the
// server's markup in place (the same nodes, no mismatch warning), then
// works: the app shell picks its layout and collapses, tabs switch, the
// pagination moves, the stepper steps, the accordion and collapsible open,
// the tree opens and chooses, the header's navigation opens a disclosure
// list. A second run hydrates the same markup on a touch screen: the first
// render does not depend on the input. The server half is
// navigation-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { setMatchMedia } from '@weft/ui'
import { startIslands } from '../src/island-runtime.ts'
import Navigation from './fixtures/navigation-island/components/Navigation.vue'
import { navigationIsland } from './fixtures/navigation-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))
const key = (el: Element, k: string) => el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true }))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
  setMatchMedia(undefined)
})

/** Parts the server draws for CSS to choose among until mounting; once mounted only the layout's own stay. */
const FORMS = '.weft-app-shell__bar, .weft-app-shell__menu, .weft-app-shell__heading, .weft-nav__menu-wrap'

/** Hydrate the island; every element the server drew stays, bar the props script, an icon's drawing (Icon sets it with v-html) and the forms not chosen. */
async function hydrate() {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  history.replaceState(null, '', '/')
  document.body.innerHTML = `<div>${navigationIsland}</div>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter(
    (e) => e.tagName !== 'SCRIPT' && !(e instanceof SVGElement && !(e instanceof SVGSVGElement)) && !e.closest(FORMS),
  )
  startIslands({ Navigation: () => Promise.resolve(Navigation as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  return { island, warn, error }
}

it('hydrates every navigation and disclosure component inside an island, then each works', async () => {
  const { island, warn, error } = await hydrate()
  const out = island.querySelector('output')!
  expect(out.textContent).toBe('one|3|0||false|')

  // The shell has picked its layout and collapses its sidebar.
  const shell = island.querySelector<HTMLElement>('.weft-app-shell')!
  expect(shell.dataset.layout).toBe('sidebar')
  island.querySelector<HTMLButtonElement>('.weft-app-shell__collapse')!.click()
  await settle()
  expect(shell.dataset.collapsed).toBe('true')

  // Tabs switch with an arrow key.
  const first = island.querySelector<HTMLButtonElement>('[role="tab"]')!
  first.focus()
  key(first, 'ArrowRight')
  await settle()
  // Pagination moves on.
  island.querySelector<HTMLButtonElement>('[aria-label="Next page"]')!.click()
  // The stepper takes its next step.
  ;[...island.querySelectorAll<HTMLButtonElement>('.weft-stepper__trigger')][1]!.click()
  // The accordion and the collapsible open.
  island.querySelector<HTMLButtonElement>('.weft-accordion__trigger')!.click()
  island.querySelector<HTMLButtonElement>('.weft-collapsible__trigger')!.click()
  await settle()
  // The tree opens its first node and chooses it.
  const src = island.querySelector<HTMLElement>('[role="treeitem"]')!
  src.click()
  await settle()
  expect(out.textContent).toBe('two|4|1|ship|true|src')
  expect(island.querySelectorAll('[role="treeitem"]')).toHaveLength(3)
  expect(island.querySelector('.weft-collapsible__content')!.hasAttribute('hidden')).toBe(false)

  // The header's navigation opens a disclosure list of links.
  const more = [...island.querySelectorAll<HTMLButtonElement>('.weft-header .weft-nav__trigger')][0]!
  more.click()
  await settle()
  expect(more.getAttribute('aria-expanded')).toBe('true')
  expect(document.getElementById(more.getAttribute('aria-controls')!)!.hidden).toBe(false)
  expect(island.querySelector('[role="menu"]')).toBeNull()
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})

it('hydrates the same markup on a touch screen, then takes the touch sizes', async () => {
  setMatchMedia(
    (query) =>
      ({ matches: query === '(pointer: coarse)', media: query, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList,
  )
  const server = /<div class="weft-tree"[^>]*>/.exec(navigationIsland)![0]
  expect(server).toContain('--weft-tree-row:32px')
  const { island, warn, error } = await hydrate()
  expect(island.querySelector<HTMLElement>('.weft-tree')!.style.getPropertyValue('--weft-tree-row')).toBe('44px')
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
