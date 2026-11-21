// @vitest-environment happy-dom
// An island of every date and file component hydrates the server's markup in
// place (the same nodes, no mismatch warning), then works: the Calendar moves
// and chooses, the DatePicker reads a typed date and opens its calendar into
// #weft-overlays, the InputTime steps, the FileUpload takes a drop. The server
// half is dates-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Dates from './fixtures/dates-island/components/Dates.vue'
import { datesIsland } from './fixtures/dates-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))
const key = (el: Element, k: string, init: KeyboardEventInit = {}) =>
  el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true, ...init }))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates every date and file component inside an island, then each works', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${datesIsland}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ Dates: () => Promise.resolve(Dates as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  const out = island.querySelector('output')!
  expect(out.textContent).toBe('2024-02-14|2024-03-01|2024-04-02–2024-04-05|09:30|0')

  // The Calendar moves a day and chooses it.
  const chosen = island.querySelector<HTMLButtonElement>('.weft-calendar__day[data-date="2024-02-14"]')!
  chosen.focus()
  key(chosen, 'ArrowRight')
  await settle()
  ;(document.activeElement as HTMLButtonElement).click()
  await settle()
  expect(out.textContent).toBe('2024-02-15|2024-03-01|2024-04-02–2024-04-05|09:30|0')

  // The DatePicker reads a date typed day first, and opens its calendar on it.
  const field = island.querySelector<HTMLInputElement>('.weft-date-picker:not(.weft-date-range-picker) input')!
  field.focus()
  field.value = '7/3/2024'
  field.dispatchEvent(new Event('input', { bubbles: true }))
  field.dispatchEvent(new Event('change', { bubbles: true }))
  await settle()
  expect(out.textContent).toContain('|2024-03-07|')
  key(field, 'ArrowDown', { altKey: true })
  await settle()
  const root = document.getElementById('weft-overlays')!
  expect(root.querySelector('[role="grid"]')).not.toBeNull()
  expect((document.activeElement as HTMLElement).dataset.date).toBe('2024-03-07')
  key(document.activeElement!, 'Escape')
  await settle()

  // The InputTime steps its hour.
  const hour = island.querySelector<HTMLElement>('[role="spinbutton"][aria-label="Hour"]')!
  hour.focus()
  key(hour, 'ArrowUp')
  await settle()
  expect(out.textContent).toContain('|10:30|')

  // The FileUpload takes a drop.
  const zone = island.querySelector('.weft-file-upload__zone')!
  const dt = new DataTransfer()
  dt.items.add(new File(['x'], 'a.pdf', { type: 'application/pdf' }))
  const drop = new Event('drop', { bubbles: true, cancelable: true })
  Object.defineProperty(drop, 'dataTransfer', { value: dt })
  zone.dispatchEvent(drop)
  await settle()
  expect(out.textContent!.endsWith('|1')).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
