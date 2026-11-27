import { afterEach, beforeEach, expect, it } from 'vitest'
import { createVaporApp, nextTick } from 'vue'
import { widthPrePaint } from '../src/adaptive/useWidth.ts'
import { ID_SCOPE } from '../src/headless/useId.ts'
import { installResizeObserver } from './resize-observer.ts'
import { mount } from './mount.ts'
import WidthProbe from './fixtures/WidthProbe.vue'
import NarrowProbe from './fixtures/NarrowProbe.vue'

let ro: ReturnType<typeof installResizeObserver>
beforeEach(() => { ro = installResizeObserver() })
afterEach(() => {
  delete (window as { __weftWidthRo?: unknown }).__weftWidthRo
  ro.restore()
})

it('reports the width class of its element, marks it data-weft-width, and stops observing on unmount', async () => {
  const { el, unmount } = mount(WidthProbe)
  const box = el.querySelector('.box') as HTMLElement
  expect(box.hasAttribute('data-weft-width')).toBe(true)
  ro.report(box, 500)
  await nextTick()
  expect(box.dataset.width).toBe('narrow')
  ro.report(box, 1100)
  await nextTick()
  expect(box.dataset.width).toBe('wide')
  unmount()
  expect(box.hasAttribute('data-weft-width')).toBe(false)
  expect(ro.liveCount()).toBe(0)
})

it("starts at its WeftProvider's width until it is measured", async () => {
  const { el, unmount } = mount(NarrowProbe)
  const box = el.querySelector('.box') as HTMLElement
  expect(box.dataset.width).toBe('narrow')
  ro.report(box, 1100)
  await nextTick()
  expect(box.dataset.width).toBe('wide')
  unmount()
})

it('starts wide with no WeftProvider saying otherwise', () => {
  const { el, unmount } = mount(WidthProbe)
  expect((el.querySelector('.box') as HTMLElement).dataset.width).toBe('wide')
  unmount()
})

it('widthPrePaint sets data-width synchronously on unhydrated elements (including inside [inert]), updates on resize, and hands off once useWidth mounts', async () => {
  const staticEl = document.createElement('div')
  staticEl.setAttribute('data-width', 'wide')
  staticEl.getBoundingClientRect = () => ({ width: 420 }) as DOMRect
  const inertWrap = document.createElement('div')
  inertWrap.setAttribute('inert', '')
  const inertChild = document.createElement('div')
  inertChild.setAttribute('data-width', 'wide')
  inertChild.getBoundingClientRect = () => ({ width: 277 }) as DOMRect
  inertWrap.appendChild(inertChild)
  document.body.append(staticEl, inertWrap)
  try {
    new Function(widthPrePaint)()
    // Synchronously measured before the ResizeObserver fires (CLS 0, and [inert] measured too).
    expect(staticEl.getAttribute('data-width')).toBe('narrow')
    expect(inertChild.getAttribute('data-width')).toBe('narrow')
    ro.report(staticEl, 768)
    expect(staticEl.getAttribute('data-width')).toBe('regular')
    ro.report(staticEl, 1280)
    expect(staticEl.getAttribute('data-width')).toBe('wide')
    ro.report(inertChild, 1280)
    expect(inertChild.getAttribute('data-width')).toBe('wide')

    // Once a component mounts, useWidth reads the existing data-width, sets data-weft-width, and unobserves from __weftWidthRo.
    const { el, unmount } = mount(WidthProbe)
    const box = el.querySelector('.box') as HTMLElement
    expect(box.hasAttribute('data-weft-width')).toBe(true)
    ro.report(box, 420)
    await nextTick()
    expect(box.dataset.width).toBe('narrow')
    unmount()
  } finally {
    staticEl.remove()
    inertWrap.remove()
  }
})

it('widthPrePaint batches all getBoundingClientRect reads before any setAttribute writes, and starts useWidth from the pre-hydrated island attribute without writing "wide"', () => {
  const order: string[] = []
  const el1 = document.createElement('div')
  const el2 = document.createElement('div')
  el1.setAttribute('data-width', 'wide')
  el2.setAttribute('data-width', 'wide')
  el1.getBoundingClientRect = () => { order.push('read:1'); return { width: 320 } as DOMRect }
  el2.getBoundingClientRect = () => { order.push('read:2'); return { width: 720 } as DOMRect }
  const origSet1 = el1.setAttribute.bind(el1)
  const origSet2 = el2.setAttribute.bind(el2)
  el1.setAttribute = (k: string, v: string) => { if (k === 'data-width') order.push(`write:1:${v}`); origSet1(k, v) }
  el2.setAttribute = (k: string, v: string) => { if (k === 'data-width') order.push(`write:2:${v}`); origSet2(k, v) }
  const island = document.createElement('weft-island')
  island.append(el1, el2)
  document.body.appendChild(island)
  try {
    new Function(widthPrePaint)()
    expect(order).toEqual(['read:1', 'read:2', 'write:1:narrow', 'write:2:regular'])
    const host = document.createElement('div')
    document.body.appendChild(host)
    const app = createVaporApp(WidthProbe as never)
    app.provide(ID_SCOPE, { prefix: 'i0', next: 0 })
    app.mount(host)
    expect((host.querySelector('.box') as HTMLElement).dataset.width).toBe('narrow')
    app.unmount()
    host.remove()
  } finally {
    island.remove()
  }
})
