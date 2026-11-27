// @vitest-environment happy-dom
// An island whose FormFields hold a Tooltip and a Checkbox in their label
// slots hydrates the server's markup in place: the same nodes, no mismatch
// warning, so the browser counts the label's ids as the server did. The
// server half is label-slot-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import LabelSlot from './fixtures/label-slot-island/components/LabelSlot.vue'
import { labelSlotIsland } from './fixtures/label-slot-island/markup.ts'

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates FormFields with components in their label slots', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${labelSlotIsland}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ LabelSlot: () => Promise.resolve(LabelSlot as Component) })
  await new Promise((r) => setTimeout(r, 0))
  for (const el of before) expect(el.isConnected, el.outerHTML.slice(0, 120)).toBe(true)
  expect(warn.mock.calls.map((c) => String(c[0]).slice(0, 160))).toEqual([])
  expect(error).not.toHaveBeenCalled()
  const name = [...island.querySelectorAll<HTMLLabelElement>('label.weft-form-field__label')].find((l) => l.textContent!.includes('Name'))!
  expect(name.control).toBe(island.querySelectorAll('input.weft-input__field')[1])
})
