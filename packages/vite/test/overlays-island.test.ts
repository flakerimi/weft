// @vitest-environment happy-dom
// An island of every overlay component hydrates the server's markup in place
// (the same nodes, no mismatch warning), then its overlays open into
// #weft-overlays, created after hydration: the popover that starts open, a
// Dialog with a Button inside, a menu. The server half is
// overlays-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Overlays from './fixtures/overlays-island/components/Overlays.vue'
import { overlaysIsland } from './fixtures/overlays-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))
const key = (el: Element, k: string) => el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true }))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates every overlay component inside an island, then opens them into the overlay root', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${overlaysIsland}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ Overlays: () => Promise.resolve(Overlays as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()

  const root = document.getElementById('weft-overlays')!
  expect(root.parentElement).toBe(document.body)
  // Opened at once in the browser: the popover that starts open, and the Toaster's region.
  expect(root.querySelector('.weft-popover .field')).not.toBeNull()
  expect(root.querySelector('.weft-toaster[role="region"]')).not.toBeNull()
  // Triggers carry their ARIA once hydrated.
  expect(island.querySelector('.open-menu')!.getAttribute('aria-haspopup')).toBe('menu')
  expect(island.querySelector('.open-dialog')!.getAttribute('aria-haspopup')).toBe('dialog')
  key(root.querySelector('.field')!, 'Escape')
  await settle()
  expect(root.querySelector('.weft-popover')).toBeNull()

  const opener = island.querySelector<HTMLButtonElement>('.open-dialog')!
  opener.focus()
  opener.click()
  await settle()
  const dialog = root.querySelector('[role="dialog"]')!
  expect(dialog.querySelector('button.weft-button.done')).not.toBeNull()
  dialog.querySelector<HTMLButtonElement>('.done')!.click()
  await settle()
  expect(root.querySelector('[role="dialog"]')).toBeNull()
  expect(document.activeElement).toBe(opener)

  island.querySelector<HTMLButtonElement>('.open-menu')!.click()
  await settle()
  key(document.activeElement!, 'Enter')
  await settle()
  expect(island.querySelector('output')!.textContent).toBe('archive')
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
