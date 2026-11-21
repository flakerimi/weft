import { afterEach, describe, expect, it, vi } from 'vitest'
import ContextCases from '../fixtures/overlays/ContextCases.vue'
import { mount } from '../mount.ts'
import { $, $$, active, key, press, reset, settle } from './helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  vi.useRealTimers()
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(ContextCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const menu = () => $('[role="menu"]')
const contextmenu = (el: Element, x = 120, y = 80) => {
  const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: x, clientY: y })
  el.dispatchEvent(event)
  return event
}
const touch = (el: Element, type: string, x = 50, y = 50) =>
  el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerType: 'touch', pointerId: 3, clientX: x, clientY: y }))

describe('ContextMenu', () => {
  it('opens its menu at the pointer on a right press, in place of the browser’s, first item focused', async () => {
    // A viewport with room, which happy-dom (no layout) does not report.
    const html = document.documentElement
    Object.defineProperty(html, 'clientWidth', { value: 1024, configurable: true })
    Object.defineProperty(html, 'clientHeight', { value: 768, configurable: true })
    await cases()
    const event = contextmenu($('.area'))
    await settle()
    expect(event.defaultPrevented).toBe(true)
    expect(menu()).not.toBeNull()
    expect(menu().getAttribute('aria-label')).toBe('Clipboard')
    expect(active()!.textContent!.trim()).toBe('Copy')
    expect(menu().style.left).toBe('120px')
    expect(menu().style.top).toBe('86px') // 6 px under the pointer
    Reflect.deleteProperty(html, 'clientWidth')
    Reflect.deleteProperty(html, 'clientHeight')
  })

  it('works the menu by keyboard and closes back to where focus was', async () => {
    const m = await cases()
    $('.area').focus()
    contextmenu($('.area'), 0, 0) // the context menu key: no pointer position
    await settle()
    key('ArrowDown')
    key('Enter')
    await settle()
    expect(m.exposed.log).toEqual(['paste'])
    expect(menu()).toBeNull()
    expect(active()).toBe($('.area'))
  })

  it('closes on Escape and on a press outside', async () => {
    await cases()
    contextmenu($('.area'))
    await settle()
    key('Escape')
    await settle()
    expect(menu()).toBeNull()
    contextmenu($('.area'))
    await settle()
    press($('.after'))
    await settle()
    expect(menu()).toBeNull()
  })

  it('leaves the browser’s menu alone when disabled', async () => {
    await cases({ disabled: true })
    const event = contextmenu($('.area'))
    await settle()
    expect(event.defaultPrevented).toBe(false)
    expect(menu()).toBeNull()
  })

  describe('on touch', () => {
    it('opens on a long press as a bottom sheet, and the click that ends the press does not close it', async () => {
      vi.useFakeTimers()
      const m = mount(ContextCases, { input: 'touch' })
      unmount = m.unmount
      await vi.runAllTimersAsync()
      touch($('.area'), 'pointerdown')
      await vi.advanceTimersByTimeAsync(400)
      expect(menu()).toBeNull()
      await vi.advanceTimersByTimeAsync(300)
      expect(menu()).not.toBeNull()
      expect(menu().dataset.form).toBe('sheet')
      touch($('.area'), 'pointerup')
      $('.weft-overlay-backdrop').dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      await vi.advanceTimersByTimeAsync(0)
      expect(menu()).not.toBeNull()
      expect($$('[role="menuitem"]', menu())).toHaveLength(2)
    })

    it('does not open when the finger moves (a scroll) or lifts early', async () => {
      vi.useFakeTimers()
      const m = mount(ContextCases, { input: 'touch' })
      unmount = m.unmount
      await vi.runAllTimersAsync()
      touch($('.area'), 'pointerdown')
      touch($('.area'), 'pointermove', 50, 80)
      await vi.advanceTimersByTimeAsync(1000)
      expect(menu()).toBeNull()
      touch($('.area'), 'pointerdown')
      await vi.advanceTimersByTimeAsync(200)
      touch($('.area'), 'pointerup')
      await vi.advanceTimersByTimeAsync(1000)
      expect(menu()).toBeNull()
    })
  })
})
