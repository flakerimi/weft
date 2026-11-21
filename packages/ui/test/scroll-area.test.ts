import { afterEach, describe, expect, it } from 'vitest'
import ScrollArea from '../src/layout/ScrollArea.vue'
import { mount } from './mount.ts'
import { installResizeObserver } from './resize-observer.ts'
import { loadStyles, ruleFor } from './styles.ts'
import { $, key, reset, settle } from './overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
})

/** Give an element the scroll box happy-dom does not lay out. */
function box(el: HTMLElement, size: { scrollHeight?: number; clientHeight?: number; scrollWidth?: number; clientWidth?: number }) {
  for (const [k, v] of Object.entries(size)) Object.defineProperty(el, k, { configurable: true, get: () => v })
}

async function area(props: Record<string, unknown> = {}) {
  ro = installResizeObserver()
  const m = mount(ScrollArea, props)
  unmount = m.unmount
  await settle()
  return $('.weft-scroll-area')
}

describe('ScrollArea', () => {
  it('scrolls natively in its orientation, styled thin in the theme colours', async () => {
    const unstyle = loadStyles()
    try {
      const el = await area()
      expect(el.dataset.orientation).toBe('vertical')
      // happy-dom computes no overflow-x or overflow-y, so read the rules.
      const overflow = (o: string) => {
        const rule = ruleFor(`.weft-scroll-area[data-orientation="${o}"]`)!.style
        return [rule.getPropertyValue('overflow-x') || rule.getPropertyValue('overflow'), rule.getPropertyValue('overflow-y') || rule.getPropertyValue('overflow')]
      }
      expect(overflow('vertical')).toEqual(['hidden', 'auto'])
      expect(overflow('horizontal')).toEqual(['auto', 'hidden'])
      expect(overflow('both')).toEqual(['auto', 'auto'])
      expect(ruleFor('.weft-scroll-area')!.style.getPropertyValue('scrollbar-width')).toBe('thin')
      expect(ruleFor('.weft-scroll-area')!.style.getPropertyValue('scrollbar-color')).toContain('var(--weft-color-border')
      unmount!()
      const h = await area({ orientation: 'horizontal' })
      expect(h.dataset.orientation).toBe('horizontal')
      unmount!()
      const both = await area({ orientation: 'both', maxHeight: '12rem' })
      expect(both.dataset.orientation).toBe('both')
      expect(both.style.maxBlockSize).toBe('12rem')
    } finally {
      unstyle()
    }
  })

  it('is a named region when labelled', async () => {
    const el = await area({ label: 'Release notes' })
    expect(el.getAttribute('role')).toBe('region')
    expect(el.getAttribute('aria-label')).toBe('Release notes')
    unmount!()
    const plain = await area()
    expect(plain.hasAttribute('role')).toBe(false)
  })

  it('is a tab stop while it overflows, so the keyboard scrolls it, and leaves the keys to the browser', async () => {
    const el = await area({ label: 'Notes' })
    // Before it is measured (and on the server) it is a tab stop.
    box(el, { scrollHeight: 100, clientHeight: 100 })
    ro!.report(el, 300)
    await settle()
    expect(el.hasAttribute('tabindex')).toBe(false)
    box(el, { scrollHeight: 900, clientHeight: 100 })
    ro!.report(el, 300)
    await settle()
    expect(el.getAttribute('tabindex')).toBe('0')
    el.focus()
    expect(key('ArrowDown').defaultPrevented).toBe(false)
    expect(key('PageDown').defaultPrevented).toBe(false)
  })

  it('with shadow, fades the edges that have more beyond them', async () => {
    const el = await area({ shadow: true })
    box(el, { scrollHeight: 900, clientHeight: 100 })
    el.scrollTop = 0
    el.dispatchEvent(new Event('scroll'))
    await settle()
    expect(el.dataset.fadeStart).toBeUndefined()
    expect(el.dataset.fadeEnd).toBe('true')
    el.scrollTop = 400
    el.dispatchEvent(new Event('scroll'))
    await settle()
    expect(el.dataset.fadeStart).toBe('true')
    expect(el.dataset.fadeEnd).toBe('true')
    el.scrollTop = 800
    el.dispatchEvent(new Event('scroll'))
    await settle()
    expect(el.dataset.fadeStart).toBe('true')
    expect(el.dataset.fadeEnd).toBeUndefined()
  })
})
