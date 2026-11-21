// What an overlay carries out of the subtree it opened from, since it is
// teleported into #weft-overlays under <body>: the scheme of a
// [data-weft-scheme] ancestor, and a [data-weft-no-prefetch] mark, so the
// speculation rules leave a gallery demo's overlay links alone too.
import { afterEach, describe, expect, it } from 'vitest'
import MenuCases from '../fixtures/overlays/MenuCases.vue'
import ModalCases from '../fixtures/overlays/ModalCases.vue'
import PopoverCases from '../fixtures/overlays/PopoverCases.vue'
import ReusedMark from '../fixtures/overlays/ReusedMark.vue'
import { mount } from '../mount.ts'
import { $, key, overlays, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})

const kinds = [
  { name: 'Sheet', component: ModalCases, props: { case: 'sheet' } },
  { name: 'Popover', component: PopoverCases, props: {} },
  { name: 'DropdownMenu', component: MenuCases, props: {} },
] as const

/** Mount `kind` inside an element with `attrs`, open it from its trigger, and return its .weft-overlay layer. */
async function openInside(kind: (typeof kinds)[number], attrs: Record<string, string>) {
  const m = mount(kind.component, { ...kind.props })
  unmount = m.unmount
  for (const [name, value] of Object.entries(attrs)) m.el.setAttribute(name, value)
  await settle()
  $('.trigger').focus()
  $('.trigger').click()
  await settle()
  const layer = $('.weft-overlay', overlays() ?? document)
  expect(layer).not.toBeNull()
  return layer
}

describe.each(kinds)('$name opened from a marked subtree', (kind) => {
  it('carries data-weft-no-prefetch onto its overlay layer', async () => {
    const layer = await openInside(kind, { 'data-weft-no-prefetch': '' })
    expect(layer.hasAttribute('data-weft-no-prefetch')).toBe(true)
  })

  it('leaves the mark off when nothing around it has it', async () => {
    const layer = await openInside(kind, {})
    expect(layer.hasAttribute('data-weft-no-prefetch')).toBe(false)
  })

  it('carries the scheme of the nearest data-weft-scheme ancestor', async () => {
    const layer = await openInside(kind, { 'data-weft-scheme': 'dark' })
    expect(layer.getAttribute('data-weft-scheme')).toBe('dark')
  })
})

describe('an overlay opened again from another place', () => {
  it('drops the no-prefetch mark when it reopens from an unmarked part of the page', async () => {
    const m = mount(ReusedMark)
    unmount = m.unmount
    await settle()
    $('.trigger').click()
    await settle()
    expect($('.weft-overlay', overlays()!).hasAttribute('data-weft-no-prefetch')).toBe(true)
    key('Escape', {}, $('.inside'))
    await settle()
    await settle()
    m.exposed.marked = false
    await settle()
    $('.trigger').click()
    await settle()
    expect($('.weft-overlay', overlays()!).hasAttribute('data-weft-no-prefetch')).toBe(false)
  })
})
