// @vitest-environment happy-dom
// An island of every selection component hydrates the server's markup in
// place (the same nodes, no mismatch warning), then works: the Select opens
// its list into #weft-overlays and chooses, the long Listbox moves and
// chooses, the InputTags adds a tag, the CommandPalette runs a command. The
// server half is selection-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Selection from './fixtures/selection-island/components/Selection.vue'
import { selectionIsland } from './fixtures/selection-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))
const key = (el: Element, k: string) => el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true }))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates every selection component inside an island, then each works', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${selectionIsland}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ Selection: () => Promise.resolve(Selection as Component) })
  await settle()
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
  const out = island.querySelector('output')!
  expect(out.textContent).toBe('banana|3|vapor|')

  // The Select opens into the overlay root and chooses.
  const select = island.querySelector<HTMLButtonElement>('.weft-select__trigger')!
  select.focus()
  key(select, 'ArrowDown')
  await settle()
  const root = document.getElementById('weft-overlays')!
  expect(root.querySelector('[role="listbox"]')).not.toBeNull()
  key(select, 'ArrowDown')
  key(select, 'Enter')
  await settle()
  expect(out.textContent).toBe('cherry|3|vapor|')

  // The long Listbox: hydrated with its first rows, it moves and chooses.
  const list = island.querySelector<HTMLElement>('.weft-listbox__list')!
  list.focus()
  await settle()
  key(list, 'End')
  await settle()
  expect(document.getElementById(list.getAttribute('aria-activedescendant')!)!.textContent!.trim()).toBe('Row 300')
  key(list, 'Enter')
  await settle()
  expect(out.textContent).toBe('cherry|300|vapor|')

  // InputTags adds a tag.
  const tags = island.querySelector<HTMLInputElement>('.weft-input-tags input.weft-input__field')!
  tags.focus()
  tags.value = 'islands'
  tags.dispatchEvent(new Event('input', { bubbles: true }))
  key(tags, 'Enter')
  await settle()
  expect(out.textContent).toBe('cherry|300|vapor,islands|')

  // The CommandPalette runs its first command.
  const palette = island.querySelector<HTMLInputElement>('.weft-command-palette__input')!
  palette.focus()
  key(palette, 'Enter')
  await settle()
  expect(out.textContent).toBe('cherry|300|vapor,islands|home')
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
