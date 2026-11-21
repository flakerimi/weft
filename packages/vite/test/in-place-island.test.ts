// @vitest-environment happy-dom
// An island with menus drawn open in place (:portal="false"), on pointer and
// on touch, hydrates the server's markup in place (the same nodes, no
// mismatch warning), and both stay usable: nothing around them turns inert,
// the page's scroll is not locked, and choosing an item runs it and closes
// that menu. The server half is in-place-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import InPlace from './fixtures/in-place-island/components/InPlace.vue'
import { inPlaceIsland } from './fixtures/in-place-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  document.documentElement.removeAttribute('style')
  vi.restoreAllMocks()
})

it.each(['pointer', 'touch'])('hydrates a menu open in place on %s, and it stays usable', async (input) => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${inPlaceIsland}</main><footer><a href="/">Home</a></footer>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ InPlace: () => Promise.resolve(InPlace as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()

  const menu = island.querySelector<HTMLElement>(`.${input} [role="menu"]`)!
  expect(menu).not.toBeNull()
  expect(document.getElementById('weft-overlays')).toBeNull()
  expect(document.querySelector('[inert]')).toBeNull()
  expect(document.documentElement.style.overflow).toBe('')

  const reboot = [...menu.querySelectorAll<HTMLElement>('[role="menuitem"]')].find((el) => el.textContent?.includes('Reboot'))!
  reboot.click()
  await settle()
  expect(island.querySelector('.chosen')!.textContent).toBe(`${input} reboot`)
  expect(island.querySelector(`.${input} [role="menu"]`)).toBeNull()
})
