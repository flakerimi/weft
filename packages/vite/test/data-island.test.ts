// @vitest-environment happy-dom
// An island of every feedback and data component hydrates the server's
// markup in place (the same nodes, no mismatch warning), then works: the
// remote table fetches once mounted, the table sorts, chooses and pages,
// the toggles press, the alert and the banner close, the error state
// retries. The server half is data-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Data from './fixtures/data-island/components/Data.vue'
import { dataIsland } from './fixtures/data-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  localStorage.clear()
  vi.restoreAllMocks()
})

it('hydrates every feedback and data component inside an island, then each works', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<div>${dataIsland}</div>`
  const island = document.querySelector('weft-island')!
  // Every element the server drew stays, bar the props script and the remote table's skeleton rows, which its answer replaces.
  const before = Array.from(island.querySelectorAll('*')).filter(
    (e) => e.tagName !== 'SCRIPT' && !e.closest('.remote tbody') && !(e instanceof SVGElement && !(e instanceof SVGSVGElement)),
  )
  startIslands({ Data: () => Promise.resolve(Data as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  const output = island.querySelector('output')!
  /** The output's text, the arrays' line breaks dropped. */
  const out = { get textContent() { return output.textContent!.replace(/\s+/g, '') } }
  // The remote table asked once, after mounting, and shows its first page.
  expect(out.textContent).toBe('|[]|1|false|left|true|0')
  const remote = island.querySelector('.remote')!
  expect(remote.querySelector('table')!.hasAttribute('aria-busy')).toBe(false)
  expect([...remote.querySelectorAll('tbody th')].map((th) => th.textContent!.trim())).toEqual(['Grace', 'Ada'])

  // The local table sorts, chooses every row shown and pages.
  const local = island.querySelector('.weft-table:not(.remote)')!
  local.querySelector<HTMLButtonElement>('thead .weft-table__sort')!.click()
  await settle()
  local.querySelector<HTMLInputElement>('thead input[type="checkbox"]')!.click()
  await settle()
  expect(out.textContent).toBe('name|[2,3]|1|false|left|true|0')
  local.querySelector<HTMLButtonElement>('[aria-label="Page 2"]')!.click()
  await settle()
  expect([...local.querySelectorAll('tbody th')].map((th) => th.textContent!.trim())).toEqual(['Grace'])

  // The toggles press; the alert and the banner close; the error state retries.
  ;[...island.querySelectorAll<HTMLButtonElement>('button')].find((b) => b.textContent!.trim() === 'Bold')!.click()
  ;[...island.querySelectorAll<HTMLButtonElement>('.weft-toggle-group button')][1]!.click()
  island.querySelector<HTMLButtonElement>('.weft-alert__close')!.click()
  island.querySelector<HTMLButtonElement>('.weft-banner__close')!.click()
  ;[...island.querySelectorAll<HTMLButtonElement>('.weft-error-state button')][0]!.click()
  await settle()
  expect(out.textContent).toBe('name|[2,3]|1|true|right|false|1')
  expect(island.querySelector('.weft-banner')).toBeNull()
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
