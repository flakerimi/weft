import { afterEach, beforeEach, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { installResizeObserver } from './resize-observer.ts'
import { mount } from './mount.ts'
import WidthProbe from './fixtures/WidthProbe.vue'
import NarrowProbe from './fixtures/NarrowProbe.vue'

let ro: ReturnType<typeof installResizeObserver>
beforeEach(() => { ro = installResizeObserver() })
afterEach(() => ro.restore())

it('reports the width class of its element and stops observing on unmount', async () => {
  const { el, unmount } = mount(WidthProbe)
  const box = el.querySelector('.box') as HTMLElement
  ro.report(box, 500)
  await nextTick()
  expect(box.dataset.width).toBe('narrow')
  ro.report(box, 1100)
  await nextTick()
  expect(box.dataset.width).toBe('wide')
  unmount()
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
