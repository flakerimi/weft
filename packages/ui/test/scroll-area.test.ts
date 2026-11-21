import { afterEach, describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import ScrollArea from '../src/layout/ScrollArea.vue'
import { mount } from './mount.ts'
import { installResizeObserver } from './resize-observer.ts'
import { loadStyles, ruleFor } from './styles.ts'
import { $, $$, key, reset, settle } from './overlays/helpers.ts'

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
function box(el: HTMLElement, size: { scrollHeight?: number; clientHeight?: number; scrollWidth?: number; clientWidth?: number; offsetWidth?: number; offsetHeight?: number }) {
  for (const [k, v] of Object.entries(size)) Object.defineProperty(el, k, { configurable: true, get: () => v })
}

/** Mount one; returns its root and the viewport that scrolls. */
async function area(props: Record<string, unknown> = {}) {
  ro = installResizeObserver()
  const m = mount(ScrollArea, props)
  unmount = m.unmount
  await settle()
  return { root: $('.weft-scroll-area'), view: $('.weft-scroll-area__viewport') }
}
const scrollTo = async (view: HTMLElement, top: number, left = 0) => {
  view.scrollTop = top
  view.scrollLeft = left
  view.dispatchEvent(new Event('scroll'))
  await settle()
}
const fades = (root: HTMLElement) =>
  ['blockStart', 'blockEnd', 'inlineStart', 'inlineEnd'].filter((edge) => root.dataset[`fade${edge[0]!.toUpperCase()}${edge.slice(1)}`] === 'true')

describe('ScrollArea', () => {
  it('scrolls natively in its orientation, styled thin in the theme colours', async () => {
    const unstyle = loadStyles()
    try {
      const { root, view } = await area()
      expect(root.dataset.orientation).toBe('vertical')
      // happy-dom computes no overflow-x or overflow-y, so read the rules.
      const overflow = (o: string) => {
        const rule = ruleFor(`.weft-scroll-area[data-orientation="${o}"] > .weft-scroll-area__viewport`)!.style
        return [rule.getPropertyValue('overflow-x') || rule.getPropertyValue('overflow'), rule.getPropertyValue('overflow-y') || rule.getPropertyValue('overflow')]
      }
      expect(overflow('vertical')).toEqual(['hidden', 'auto'])
      expect(overflow('horizontal')).toEqual(['auto', 'hidden'])
      expect(overflow('both')).toEqual(['auto', 'auto'])
      const bar = ruleFor('.weft-scroll-area__viewport')!.style
      expect(bar.getPropertyValue('scrollbar-width')).toBe('thin')
      expect(bar.getPropertyValue('scrollbar-color')).toContain('var(--weft-color-border')
      expect(view.parentElement).toBe(root)
      unmount!()
      const both = await area({ orientation: 'both', maxHeight: '12rem' })
      expect(both.root.dataset.orientation).toBe('both')
      expect(both.root.style.maxBlockSize).toBe('12rem')
    } finally {
      unstyle()
    }
  })

  it('is a named region when labelled', async () => {
    const { view } = await area({ label: 'Release notes' })
    expect(view.getAttribute('role')).toBe('region')
    expect(view.getAttribute('aria-label')).toBe('Release notes')
    unmount!()
    const plain = await area()
    expect(plain.view.hasAttribute('role')).toBe(false)
  })

  it('is a tab stop while it overflows, so the keyboard scrolls it, and leaves the keys to the browser', async () => {
    const { view } = await area({ label: 'Notes' })
    // Before it is measured (and on the server) it is a tab stop.
    box(view, { scrollHeight: 100, clientHeight: 100 })
    ro!.report(view, 300)
    await settle()
    expect(view.hasAttribute('tabindex')).toBe(false)
    box(view, { scrollHeight: 900, clientHeight: 100 })
    ro!.report(view, 300)
    await settle()
    expect(view.getAttribute('tabindex')).toBe('0')
    view.focus()
    expect(key('ArrowDown').defaultPrevented).toBe(false)
    expect(key('PageDown').defaultPrevented).toBe(false)
  })

  it('with shadow, fades the block edges that have more beyond them', async () => {
    const { root, view } = await area({ shadow: true })
    box(view, { scrollHeight: 900, clientHeight: 100 })
    await scrollTo(view, 0)
    expect(fades(root)).toEqual(['blockEnd'])
    await scrollTo(view, 400)
    expect(fades(root)).toEqual(['blockStart', 'blockEnd'])
    await scrollTo(view, 800)
    expect(fades(root)).toEqual(['blockStart'])
  })

  it('fades the inline edges when horizontal, and all four when both, mirrored right to left', async () => {
    const h = await area({ shadow: true, orientation: 'horizontal' })
    box(h.view, { scrollWidth: 900, clientWidth: 100 })
    await scrollTo(h.view, 0, 400)
    expect(fades(h.root)).toEqual(['inlineStart', 'inlineEnd'])
    unmount!()
    const both = await area({ shadow: true, orientation: 'both' })
    box(both.view, { scrollHeight: 900, clientHeight: 100, scrollWidth: 900, clientWidth: 100 })
    await scrollTo(both.view, 400, 400)
    expect(fades(both.root)).toEqual(['blockStart', 'blockEnd', 'inlineStart', 'inlineEnd'])
    // Right to left, scrollLeft counts down from 0: at 0 the inline start is reached.
    await scrollTo(both.view, 0, 0)
    expect(fades(both.root)).toEqual(['blockEnd', 'inlineEnd'])
    await scrollTo(both.view, 0, -800)
    expect(fades(both.root)).toEqual(['blockEnd', 'inlineStart'])
  })

  it('draws its fades over the content only: no mask, the scrollbars and the focus ring left clear', async () => {
    const unstyle = loadStyles()
    try {
      const { root, view } = await area({ shadow: true, orientation: 'both' })
      box(view, { offsetWidth: 310, clientWidth: 300, offsetHeight: 210, clientHeight: 200, scrollHeight: 900, scrollWidth: 900 })
      ro!.report(view, 310)
      await settle()
      const css = readFileSync(join(dirname(fileURLToPath(import.meta.url)), '../src/styles/scroll-area.css'), 'utf8')
      expect(css).not.toMatch(/mask/)
      // The scrollbars' thickness, measured, keeps the fades off them.
      expect(root.style.getPropertyValue('--weft-scroll-bar-inline')).toBe('10px')
      expect(root.style.getPropertyValue('--weft-scroll-bar-block')).toBe('10px')
      const fadeEls = $$('.weft-scroll-area__fade', root)
      expect(fadeEls.map((f) => f.dataset.edge)).toEqual(['block-start', 'block-end', 'inline-start', 'inline-end'])
      for (const f of fadeEls) expect(f.getAttribute('aria-hidden')).toBe('true')
      expect(ruleFor('.weft-scroll-area__fade')!.style.getPropertyValue('pointer-events')).toBe('none')
      expect(ruleFor('.weft-scroll-area__fade[data-edge="block-start"]')!.style.getPropertyValue('inset-inline-end')).toBe('var(--weft-scroll-bar-inline, 0px)')
      expect(ruleFor('.weft-scroll-area__fade[data-edge="inline-end"]')!.style.getPropertyValue('inset-block-end')).toBe('var(--weft-scroll-bar-block, 0px)')
      // The focus ring is base.css's, outside the box, where no fade reaches.
      expect(ruleFor('.weft-scroll-area__viewport:focus-visible')).toBeUndefined()
    } finally {
      unstyle()
    }
  })
})
