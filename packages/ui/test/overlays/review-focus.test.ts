// Review Focus 1 and 2, with the real components:
// 1. Overlays stacked three deep (a dialog opening a menu whose item opens a
//    tooltip): Escape closes only the top one, focus returns to the element
//    that opened each layer, scroll lock goes only with the last.
// 2. The input switching from pointer to touch while an overlay is open: it
//    re-renders in the new form keeping its state and focus, and back.
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { setMatchMedia } from '../../src/adaptive/detect.ts'
import { overlayStack } from '../../src/headless/overlayStack.ts'
import StackedCases from '../fixtures/overlays/StackedCases.vue'
import { fakeMedia } from '../fake-media.ts'
import { mount } from '../mount.ts'
import { $, $$, active, key, locked, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
let media: ReturnType<typeof fakeMedia>
beforeEach(() => {
  media = fakeMedia({ '(pointer: coarse)': false })
  setMatchMedia(media.mq)
})
afterEach(() => {
  unmount?.()
  unmount = undefined
  setMatchMedia(undefined)
  reset()
})
async function cases() {
  const m = mount(StackedCases)
  unmount = m.unmount
  await settle()
  return m
}
const touch = async (on: boolean) => {
  media.set('(pointer: coarse)', on)
  await settle()
}
const hover = (el: Element) => el.dispatchEvent(new PointerEvent('pointerenter', { pointerType: 'mouse' }))

describe('three overlays stacked: Dialog, DropdownMenu, Tooltip', () => {
  async function openAll() {
    $('.open-dialog').focus()
    $('.open-dialog').click()
    await settle()
    $('.open-menu').focus()
    $('.open-menu').click()
    await settle()
    const exportItem = $$('[role="menuitem"]')[1]!
    exportItem.focus()
    hover($('.tip-trigger').parentElement!)
    await settle()
    return exportItem
  }

  it('closes one layer per Escape, returns focus to each opener, and unlocks scroll with the last', async () => {
    await cases()
    const exportItem = await openAll()
    expect(overlayStack.ids()).toHaveLength(3)
    expect($('[role="tooltip"]').textContent).toBe('Download as CSV')
    expect(locked()).toBe(true)

    key('Escape')
    await settle()
    expect($('[role="tooltip"]')).toBeNull()
    expect($('[role="menu"]')).not.toBeNull()
    expect(active()).toBe(exportItem) // the tooltip took no focus and gives none back

    key('Escape')
    await settle()
    expect($('[role="menu"]')).toBeNull()
    expect($('[role="dialog"]')).not.toBeNull()
    expect(active()).toBe($('.open-menu'))
    expect(locked()).toBe(true) // the dialog still holds it

    key('Escape')
    await settle()
    expect($('[role="dialog"]')).toBeNull()
    expect(active()).toBe($('.open-dialog'))
    expect(locked()).toBe(false)
    expect(overlayStack.ids()).toEqual([])
  })

  it('choosing in the menu closes it and the tooltip, leaving the dialog with focus on the menu button', async () => {
    const m = await cases()
    await openAll()
    key('Enter')
    await settle()
    expect(m.exposed.log).toEqual(['export'])
    expect($('[role="menu"]')).toBeNull()
    expect($('[role="tooltip"]')).toBeNull()
    expect($('[role="dialog"]')).not.toBeNull()
    expect(active()).toBe($('.open-menu'))
    expect(locked()).toBe(true)
  })

  it('switches every open layer to its touch form and back, keeping the stack, focus and the lock', async () => {
    await cases()
    const exportItem = await openAll()
    await touch(true)
    expect($('[role="tooltip"]')).toBeNull() // no tooltips on touch
    expect($('[role="menu"]').dataset.form).toBe('sheet')
    expect($('[role="dialog"]')).not.toBeNull()
    expect(active()).toBe(exportItem)
    expect(locked()).toBe(true)
    await touch(false)
    expect($('[role="menu"]').dataset.form).toBe('floating')
    expect(active()).toBe(exportItem)
    key('Escape')
    await settle()
    expect(active()).toBe($('.open-menu'))
  })
})

describe('a dialog opened from a menu item', () => {
  it('finds focus on the menu button when it opens, and returns it there when it closes', async () => {
    await cases()
    $('.open-actions').focus()
    $('.open-actions').click()
    await settle()
    key('Enter') // Details: the menu closes, the dialog opens
    await settle()
    expect($('[role="menu"]')).toBeNull()
    expect(active()).toBe($('.inside'))
    key('Escape')
    await settle()
    expect(active()).toBe($('.open-actions'))
  })
})

describe('a live switch between pointer and touch while an overlay is open', () => {
  it('turns a popover into a bottom sheet and back, keeping what was typed and the focus', async () => {
    await cases()
    $('.open-popover').focus()
    $('.open-popover').click()
    await settle()
    const panel = $('.weft-popover')
    const field = $<HTMLInputElement>('.field')
    expect(active()).toBe(field)
    field.value = 'urgent'
    expect(panel.dataset.form).toBe('floating')

    await touch(true)
    expect($('.weft-popover')).toBe(panel) // the same element, in its sheet form
    expect(panel.dataset.form).toBe('sheet')
    expect(panel.getAttribute('aria-modal')).toBe('true')
    expect($('.weft-overlay-backdrop')).not.toBeNull()
    expect(panel.parentElement!.dataset.weftInput).toBe('touch')
    expect(locked()).toBe(true)
    expect(active()).toBe(field)
    expect($<HTMLInputElement>('.field').value).toBe('urgent')
    $('.elsewhere').focus()
    expect(active()).toBe(field) // trapped now

    await touch(false)
    expect($('.weft-popover')).toBe(panel)
    expect(panel.dataset.form).toBe('floating')
    expect(panel.hasAttribute('aria-modal')).toBe(false)
    expect($('.weft-overlay-backdrop')).toBeNull()
    expect(locked()).toBe(false)
    expect(active()).toBe(field) // the trap ended, the popover did not: focus stays
    expect(field.value).toBe('urgent')
    key('Escape')
    await settle()
    expect(active()).toBe($('.open-popover'))
  })

  it('hides a shown tooltip on touch and names nothing twice', async () => {
    await cases()
    $('.help').focus()
    await settle()
    expect($('[role="tooltip"]')).not.toBeNull()
    await touch(true)
    expect($('[role="tooltip"]')).toBeNull()
    expect($('.help').hasAttribute('aria-describedby')).toBe(false)
    expect($('.help').hasAttribute('aria-label')).toBe(false) // named by its text
    expect(active()).toBe($('.help'))
  })
})
