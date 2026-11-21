import { afterEach, describe, expect, it } from 'vitest'
import AnchoredPopover from '../fixtures/overlays/AnchoredPopover.vue'
import PopoverCases from '../fixtures/overlays/PopoverCases.vue'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, active, key, locked, moveFocus, overlays, press, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(PopoverCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const panel = () => $('.weft-popover', overlays() ?? document)

describe('Popover', () => {
  it('marks its trigger and renders nothing until opened', async () => {
    await cases()
    const trigger = $('.trigger')
    expect(trigger.getAttribute('aria-haspopup')).toBe('dialog')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    expect(trigger.hasAttribute('aria-controls')).toBe(false)
    expect(overlays()).toBeNull()
  })

  it('opens from its trigger into #weft-overlays, floating, and moves focus in', async () => {
    const m = await cases()
    $('.trigger').focus()
    $('.trigger').click()
    await settle()
    expect(m.exposed.open).toBe(true)
    const el = panel()
    expect(overlays()!.parentElement).toBe(document.body)
    expect(overlays()!.contains(el)).toBe(true)
    expect(el.getAttribute('role')).toBe('dialog')
    expect(el.getAttribute('aria-label')).toBe('Filters')
    expect(el.hasAttribute('aria-modal')).toBe(false)
    expect(el.dataset.form).toBe('floating')
    expect(el.classList.contains('mine')).toBe(true) // attributes go to the content
    expect($('.trigger').getAttribute('aria-expanded')).toBe('true')
    expect($('.trigger').getAttribute('aria-controls')).toBe(el.id)
    expect(el.style.position).toBe('absolute')
    expect(active()).toBe($('.field'))
    expect(locked()).toBe(false)
    expect($('.weft-overlay-backdrop')).toBeNull()
  })

  it('takes a press on its anchor element as no press outside, for the anchor owner to handle', async () => {
    const m = mount(AnchoredPopover)
    unmount = m.unmount
    await settle()
    expect(panel()).not.toBeNull()
    press($('.field'))
    await settle()
    expect(m.exposed.open).toBe(true)
    press($('.after'))
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('opens from its trigger by Enter and by Space', async () => {
    const m = await cases()
    expect(pressKey($('.trigger'), 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
    key('Escape')
    await settle()
    expect(pressKey($('.trigger'), ' ')).toBe(true)
    await settle()
    expect(m.exposed.open).toBe(true)
  })

  it('opens on hover in hover mode, leaving focus alone, and closes when the pointer leaves both', async () => {
    const m = await cases({ mode: 'hover' })
    const wrap = $('.trigger').parentElement!
    $('.before').focus()
    wrap.dispatchEvent(new PointerEvent('pointerenter', { pointerType: 'mouse' }))
    await settle()
    expect(m.exposed.open).toBe(true)
    expect(active()).toBe($('.before'))
    wrap.dispatchEvent(new PointerEvent('pointerleave', { pointerType: 'mouse' }))
    panel().dispatchEvent(new PointerEvent('pointerenter', { pointerType: 'mouse' }))
    await settle()
    expect(m.exposed.open).toBe(true)
    panel().dispatchEvent(new PointerEvent('pointerleave', { pointerType: 'mouse' }))
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('stays open when the trigger is clicked just after a hover opened it', async () => {
    const m = await cases({ mode: 'hover' })
    const wrap = $('.trigger').parentElement!
    wrap.dispatchEvent(new PointerEvent('pointerenter', { pointerType: 'mouse' }))
    await settle()
    expect(m.exposed.open).toBe(true)
    $('.trigger').click()
    await settle()
    expect(m.exposed.open).toBe(true)
    wrap.dispatchEvent(new PointerEvent('pointerleave', { pointerType: 'mouse' })) // pinned by the click
    await settle()
    expect(m.exposed.open).toBe(true)
    $('.trigger').click() // a second click closes it, as a press does
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('follows a change of arrow and offset while open', async () => {
    const html = document.documentElement
    Object.defineProperty(html, 'clientWidth', { value: 1024, configurable: true })
    Object.defineProperty(html, 'clientHeight', { value: 768, configurable: true })
    try {
      const m = await cases()
      $('.trigger').click()
      await settle()
      expect($('.weft-popover__arrow')).toBeNull()
      const top = parseFloat(panel().style.top)
      m.exposed.offset = 20
      m.exposed.arrow = true
      await settle()
      await settle()
      expect(parseFloat(panel().style.top) - top).toBe(12)
      expect($('.weft-popover__arrow').style.left).toMatch(/px$/)
    } finally {
      Reflect.deleteProperty(html, 'clientWidth')
      Reflect.deleteProperty(html, 'clientHeight')
    }
  })

  it('draws an arrow towards its trigger when asked', async () => {
    await cases({ arrow: true })
    $('.trigger').click()
    await settle()
    const arrow = $('.weft-popover__arrow', panel())
    expect(arrow.getAttribute('aria-hidden')).toBe('true')
    expect(arrow.style.left).toMatch(/px$/)
  })

  it('does not take focus when it starts open (default-open, a page that hydrates)', async () => {
    $('body').focus()
    await cases({ defaultOpen: true })
    expect(panel()).not.toBeNull()
    expect(active()).toBe(document.body)
  })

  it('closes on Escape and returns focus to the trigger', async () => {
    await cases()
    $('.trigger').focus()
    $('.trigger').click()
    await settle()
    key('Escape')
    await settle()
    expect(panel()).toBeNull()
    expect(active()).toBe($('.trigger'))
    expect($('.trigger').getAttribute('aria-expanded')).toBe('false')
  })

  it('closes on a press outside without taking focus back; a press on the trigger toggles it shut', async () => {
    const m = await cases()
    $('.trigger').click()
    await settle()
    press($('.after'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).not.toBe($('.trigger'))
    $('.trigger').click()
    await settle()
    press($('.trigger'))
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('is not modal on pointer: focus leaving it closes it and stays where it went', async () => {
    const m = await cases()
    $('.trigger').click()
    await settle()
    moveFocus($('.after'))
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe($('.after'))
  })

  it('is modal when asked: trapped and scroll-locked on pointer too', async () => {
    await cases({ modal: true })
    $('.trigger').click()
    await settle()
    expect(panel().getAttribute('aria-modal')).toBe('true')
    expect(locked()).toBe(true)
    $('.after').focus()
    expect(active()).toBe($('.field'))
    key('Escape')
    await settle()
    expect(locked()).toBe(false)
  })

  it('keeps focus where it is when autoFocus is off', async () => {
    await cases({ autoFocus: false })
    $('.trigger').focus()
    $('.trigger').click()
    await settle()
    expect(panel()).not.toBeNull()
    expect(active()).toBe($('.trigger'))
  })

  it('ignores Escape and presses outside when not dismissible; close() in the slot still closes', async () => {
    const m = await cases({ dismissible: false })
    $('.trigger').click()
    await settle()
    key('Escape')
    press($('.after'))
    await settle()
    expect(m.exposed.open).toBe(true)
    $('.done').click()
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('opens and closes with v-model:open', async () => {
    const m = await cases()
    m.exposed.open = true
    await settle()
    expect(panel()).not.toBeNull()
    m.exposed.open = false
    await settle()
    expect(panel()).toBeNull()
  })

  it('carries the direction, input and surface of where it opened into the overlay root', async () => {
    await cases({ dir: 'rtl' })
    $('.trigger').click()
    await settle()
    const layer = panel().parentElement!
    expect(layer.classList.contains('weft-overlay')).toBe(true)
    expect(layer.getAttribute('dir')).toBe('rtl')
    expect(layer.dataset.weftInput).toBe('pointer')
    expect(layer.dataset.weftSurface).toBe('app')
  })

  describe('on touch', () => {
    it('renders as a bottom sheet over a backdrop: modal, trapped, scroll-locked, with its title as heading', async () => {
      await cases({ input: 'touch' })
      $('.trigger').focus()
      $('.trigger').click()
      await settle()
      const el = panel()
      expect(el.dataset.form).toBe('sheet')
      expect(el.dataset.side).toBe('bottom')
      expect(el.classList.contains('weft-sheet')).toBe(true)
      expect(el.getAttribute('aria-modal')).toBe('true')
      expect(el.style.position).toBe('')
      expect($('.weft-overlay-backdrop')).not.toBeNull()
      expect($('h2', el).textContent).toBe('Filters')
      expect(panel().parentElement!.dataset.weftInput).toBe('touch')
      expect(locked()).toBe(true)
      expect(active()).toBe($('.field'))
      $('.after').focus()
      expect(active()).toBe($('.field'))
    })

    it('closes on a press on the backdrop and returns focus to the trigger', async () => {
      const m = await cases({ input: 'touch' })
      $('.trigger').focus()
      $('.trigger').click()
      await settle()
      press($('.weft-overlay-backdrop'), 'touch')
      await settle()
      expect(m.exposed.open).toBe(false)
      expect(locked()).toBe(false)
      expect(active()).toBe($('.trigger'))
    })
  })
})
