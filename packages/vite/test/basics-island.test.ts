// @vitest-environment happy-dom
// An island of every basic component hydrates the server's markup in place:
// the same nodes, no warning (a node or text mismatch warns), and it reacts.
// The server half is in ssr-headless.test.ts. Vue's dev check skips aria-*
// and data-* attributes, so first-render attributes are pinned by each
// component's own tests (Link's aria-current waits for mounting).
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { lucideBody } from '../src/icons.ts'
import { startIslands } from '../src/island-runtime.ts'
import Basics from './fixtures/basics-island/components/Basics.vue'
import { basicsIsland } from './fixtures/basics-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates every basic component inside an island', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  history.replaceState(null, '', '/')
  document.body.innerHTML = `<main>${basicsIsland(lucideBody('plus')!)}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*'))
  startIslands({ Basics: () => Promise.resolve(Basics as Component) })
  await settle()
  expect(island.querySelector('script')).toBeNull()
  // Hydrated, not re-rendered: every element the server drew is still there
  // (bar an icon's drawing, which Icon sets with v-html, and the props script).
  const kept = before.filter((e) => e.tagName !== 'SCRIPT' && !(e instanceof SVGElement && !(e instanceof SVGSVGElement)))
  for (const el of kept) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()

  // The link to this page is marked once mounted, which the server could not know.
  const [home, vue] = island.querySelectorAll('a.weft-link')
  expect(home!.getAttribute('aria-current')).toBe('page')
  expect(vue!.hasAttribute('aria-current')).toBe(false)

  // It reacts: a press counts, and loading blocks the next press.
  const [step, busy] = island.querySelectorAll('button')
  step!.click()
  await settle()
  expect(island.querySelector('.weft-badge')!.textContent).toBe('2')
  expect(island.querySelector('[role="progressbar"]')!.getAttribute('aria-valuenow')).toBe('2')
  expect(island.querySelector('.weft-chip__indicator')!.textContent).toBe('2')
  busy!.click()
  await settle()
  expect(step!.getAttribute('aria-busy')).toBe('true')
  step!.click()
  await settle()
  expect(island.querySelector('.weft-badge')!.textContent).toBe('2')
  expect(warn).not.toHaveBeenCalled()
})
