import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useFloating, type FloatingOptions } from '../../src/headless/useFloating.ts'
import { $, html, reset } from './dom.ts'

afterEach(reset)

// The viewport floating-ui keeps the element inside (happy-dom reports 0 by 0).
beforeAll(() => {
  Object.defineProperty(document.documentElement, 'clientWidth', { configurable: true, value: 1024 })
  Object.defineProperty(document.documentElement, 'clientHeight', { configurable: true, value: 768 })
})

type Rect = { x: number; y: number; width: number; height: number }

/** happy-dom has no layout: give an element the box a browser would. */
function place(el: HTMLElement, r: Rect) {
  el.getBoundingClientRect = () =>
    ({ ...r, top: r.y, left: r.x, right: r.x + r.width, bottom: r.y + r.height, toJSON() {} }) as DOMRect
  Object.defineProperty(el, 'offsetWidth', { configurable: true, value: r.width })
  Object.defineProperty(el, 'offsetHeight', { configurable: true, value: r.height })
  el.style.width = `${r.width}px`
  el.style.height = `${r.height}px`
}

const settle = async () => {
  for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0))
}

function setup(opts: Partial<FloatingOptions> = {}, wrap = (s: string) => s) {
  const root = html(wrap('<button class="ref">r</button>'))
  const floatEl = document.createElement('div')
  floatEl.className = 'float'
  document.body.appendChild(floatEl) // teleported, as overlays are
  const reference = ref<HTMLElement | null>($('.ref', root))
  const floating = ref<HTMLElement | null>(floatEl)
  place(reference.value!, { x: 100, y: 100, width: 80, height: 20 })
  place(floatEl, { x: 0, y: 0, width: 50, height: 30 })
  const scope = effectScope()
  const api = scope.run(() =>
    useFloating(reference, floating, { placement: 'bottom-start', offset: 8, flip: true, shift: true, ...opts }),
  )!
  return { api, scope, reference, floating, floatEl }
}

describe('useFloating', () => {
  it('positions the floating element against its reference', async () => {
    const { api, scope } = setup()
    await settle()
    expect(api.styles.value).toMatchObject({ position: 'absolute', left: '100px', top: '128px' })
    expect(api.placement.value).toBe('bottom-start')
    scope.stop()
  })

  it('uses the fixed strategy when asked', async () => {
    const { api, scope } = setup({ strategy: 'fixed' })
    await settle()
    expect(api.styles.value.position).toBe('fixed')
    scope.stop()
  })

  it('mirrors start and end when the reference sits under dir="rtl", even when the floating element does not', async () => {
    const { api, scope } = setup({}, (s) => `<div dir="rtl">${s}</div>`)
    await settle()
    // bottom-start in rtl aligns the right edges: 100 + 80 - 50.
    expect(api.styles.value).toMatchObject({ left: '130px', top: '128px' })
    scope.stop()
  })

  it('follows a reactive placement', async () => {
    const placement = ref<'bottom-start' | 'top-start'>('bottom-start')
    const { api, scope } = setup({ placement, flip: false })
    await settle()
    placement.value = 'top-start'
    await settle()
    expect(api.styles.value).toMatchObject({ top: `${100 - 8 - 30}px` })
    expect(api.placement.value).toBe('top-start')
    scope.stop()
  })

  it('sets the available size and the reference width when size is on', async () => {
    const { floatEl, scope } = setup({ size: true })
    await settle()
    expect(floatEl.style.getPropertyValue('--weft-floating-reference-width')).toBe('80px')
    expect(floatEl.style.getPropertyValue('--weft-floating-available-height')).toMatch(/^\d+(\.\d+)?px$/)
    scope.stop()
  })

  it('does nothing until both elements exist, then follows the page', async () => {
    const { api, scope, floating, floatEl, reference } = setup()
    floating.value = null
    await settle()
    floating.value = floatEl
    await settle()
    expect(api.styles.value.left).toBe('100px')
    place(reference.value!, { x: 300, y: 100, width: 80, height: 20 })
    window.dispatchEvent(new Event('resize'))
    await settle()
    expect(api.styles.value.left).toBe('300px')
    scope.stop()
  })

  it('stops following the page when its scope ends', async () => {
    const { api, scope, reference } = setup()
    await settle()
    scope.stop()
    place(reference.value!, { x: 300, y: 100, width: 80, height: 20 })
    window.dispatchEvent(new Event('resize'))
    await settle()
    expect(api.styles.value.left).toBe('100px')
  })

  it('updates on demand', async () => {
    const { api, scope, reference } = setup()
    await settle()
    place(reference.value!, { x: 10, y: 100, width: 80, height: 20 })
    await api.update()
    await nextTick()
    expect(api.styles.value.left).toBe('10px')
    scope.stop()
  })
})
