import { afterEach, describe, expect, it, vi } from 'vitest'
import TooltipCases from '../fixtures/overlays/TooltipCases.vue'
import { mount } from '../mount.ts'
import { $, active, key, press, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  vi.useRealTimers()
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(TooltipCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const tip = () => $('[role="tooltip"]')
const hover = (el: Element, type: 'pointerenter' | 'pointerleave', pointerType = 'mouse') =>
  el.dispatchEvent(new PointerEvent(type, { bubbles: false, pointerType }))

describe('Tooltip', () => {
  it('shows on focus at once, described by the trigger, and hides on blur', async () => {
    await cases()
    const trigger = $('.labelled')
    trigger.focus()
    await settle()
    expect(tip()).not.toBeNull()
    expect(tip().textContent).toBe('Saves a draft')
    expect(document.getElementById('weft-overlays')!.contains(tip())).toBe(true)
    expect(trigger.getAttribute('aria-describedby')).toBe(tip().id)
    expect(active()).toBe(trigger) // it never takes focus
    trigger.blur()
    await settle()
    expect(tip()).toBeNull()
    expect(trigger.hasAttribute('aria-describedby')).toBe(false)
  })

  it('shows on hover after its delay, keeps the trigger’s own description, and hides when the pointer leaves', async () => {
    vi.useFakeTimers()
    const m = mount(TooltipCases)
    unmount = m.unmount
    await vi.runAllTimersAsync()
    const trigger = $('.icon-only')
    hover(trigger.parentElement!, 'pointerenter')
    await vi.advanceTimersByTimeAsync(50)
    expect(tip()).toBeNull()
    await vi.advanceTimersByTimeAsync(60)
    expect(tip()).not.toBeNull()
    expect(m.exposed.open).toBe(true)
    expect(trigger.getAttribute('aria-describedby')).toBe(`hint ${tip().id}`)
    hover(trigger.parentElement!, 'pointerleave')
    await vi.advanceTimersByTimeAsync(300)
    expect(tip()).toBeNull()
    expect(trigger.getAttribute('aria-describedby')).toBe('hint')
  })

  it('stays while the pointer moves from the trigger onto it', async () => {
    vi.useFakeTimers()
    const m = mount(TooltipCases)
    unmount = m.unmount
    await vi.runAllTimersAsync()
    const wrap = $('.labelled').parentElement!
    hover(wrap, 'pointerenter')
    await vi.advanceTimersByTimeAsync(10)
    hover(wrap, 'pointerleave')
    hover(tip(), 'pointerenter')
    await vi.advanceTimersByTimeAsync(300)
    expect(tip()).not.toBeNull()
    hover(tip(), 'pointerleave')
    await vi.advanceTimersByTimeAsync(300)
    expect(tip()).toBeNull()
  })

  it('closes on Escape with focus left on the trigger, and on a press', async () => {
    await cases()
    const trigger = $('.labelled')
    trigger.focus()
    await settle()
    key('Escape')
    await settle()
    expect(tip()).toBeNull()
    expect(active()).toBe(trigger)
    hover(trigger.parentElement!, 'pointerenter')
    await settle()
    expect(tip()).not.toBeNull()
    press(trigger)
    await settle()
    expect(tip()).toBeNull()
  })

  it('names a trigger that has no name of its own with its text', async () => {
    await cases()
    expect($('.icon-only').getAttribute('aria-label')).toBe('Delete file')
    expect($('.labelled').hasAttribute('aria-label')).toBe(false) // named by its text
    expect($('.named').getAttribute('aria-label')).toBe('Copy') // named by its own label
  })

  describe('on touch', () => {
    it('renders nothing, on focus or a touch press', async () => {
      await cases({ input: 'touch' })
      $('.labelled').focus()
      hover($('.labelled').parentElement!, 'pointerenter', 'touch')
      await settle()
      expect(tip()).toBeNull()
      expect($('.labelled').hasAttribute('aria-describedby')).toBe(false)
      expect($('.icon-only').getAttribute('aria-label')).toBe('Delete file')
    })

    it('ignores a touch pointer even on a pointer device', async () => {
      await cases()
      hover($('.labelled').parentElement!, 'pointerenter', 'touch')
      await settle()
      expect(tip()).toBeNull()
    })
  })
})
