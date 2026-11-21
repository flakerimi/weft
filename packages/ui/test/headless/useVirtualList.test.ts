import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref, shallowRef } from 'vue'
import { useVirtualList } from '../../src/headless/useVirtualList.ts'

afterEach(() => {
  document.body.innerHTML = ''
})

/** A scroller of `height` px (happy-dom lays nothing out, so its clientHeight is given by hand). */
function scroller(height = 0) {
  const el = document.createElement('div')
  Object.defineProperty(el, 'clientHeight', { configurable: true, get: () => height })
  document.body.appendChild(el)
  return el
}

function setup(opts: { count?: number; height?: number; enabled?: boolean; pinned?: number; el?: HTMLElement | null; leading?: number; stuck?: number } = {}) {
  const count = ref(opts.count ?? 10_000)
  const enabled = ref(opts.enabled ?? true)
  const pinned = ref(opts.pinned ?? -1)
  const el = shallowRef<HTMLElement | null>(opts.el === undefined ? scroller(opts.height) : opts.el)
  const scope = effectScope()
  const list = scope.run(() =>
    useVirtualList(el, {
      count: () => count.value,
      itemHeight: () => 32,
      overscan: 4,
      enabled: () => enabled.value,
      pinned: () => pinned.value,
      ...(opts.leading === undefined ? {} : { leading: () => opts.leading! }),
      ...(opts.stuck === undefined ? {} : { stuck: () => opts.stuck! }),
    }),
  )!
  return { list, count, enabled, pinned, el, scope }
}

const scrollTo = async (el: HTMLElement, top: number) => {
  el.scrollTop = top
  el.dispatchEvent(new Event('scroll'))
  await nextTick()
}
const range = (from: number, to: number) => Array.from({ length: to - from }, (_, i) => from + i)

describe('useVirtualList', () => {
  it('renders every index when it is off', () => {
    const { list, scope } = setup({ count: 5, enabled: false })
    expect(list.indices.value).toEqual([0, 1, 2, 3, 4])
    expect(list.virtual.value).toBe(false)
    scope.stop()
  })

  it('scrolls the scroller, not the page, to show a row when every row is rendered', () => {
    const { list, el, scope } = setup({ count: 50, enabled: false, height: 100 })
    const node = el.value!
    const row = document.createElement('div')
    row.dataset.index = '30'
    node.appendChild(row)
    node.getBoundingClientRect = () => ({ top: 0, bottom: 100 }) as DOMRect
    row.getBoundingClientRect = () => ({ top: 960, bottom: 992 }) as DOMRect
    list.scrollTo(30)
    expect(node.scrollTop).toBe(892)
    row.getBoundingClientRect = () => ({ top: -40, bottom: -8 }) as DOMRect
    list.scrollTo(30)
    expect(node.scrollTop).toBe(852)
    scope.stop()
  })

  it('renders only the rows in view and the overscan, of 10,000, with the whole height as a spacer', async () => {
    const { list, scope } = setup({ height: 320 })
    await nextTick()
    expect(list.virtual.value).toBe(true)
    expect(list.total.value).toBe(320_000)
    // 10 rows fill 320 px; 4 more below, none above the first.
    expect(list.indices.value).toEqual(range(0, 14))
    expect(list.offset(9_999)).toBe(319_968)
    scope.stop()
  })

  it('assumes a 320 px viewport before layout and on the server, so the first render is the same in both', () => {
    const { list, scope } = setup({ el: null })
    expect(list.indices.value).toEqual(range(0, 14))
    scope.stop()
  })

  it('follows the scroller as it scrolls', async () => {
    const { list, el, scope } = setup({ height: 320 })
    await nextTick()
    await scrollTo(el.value!, 16_000) // row 500 at the top
    expect(list.indices.value).toEqual(range(496, 514))
    scope.stop()
  })

  it('always renders the pinned index (the active item), even far out of view', async () => {
    const { list, pinned, el, scope } = setup({ height: 320, pinned: 9_000 })
    await nextTick()
    expect(list.indices.value).toEqual([...range(0, 14), 9_000])
    pinned.value = 5
    expect(list.indices.value).toEqual(range(0, 14))
    await scrollTo(el.value!, 16_000)
    pinned.value = 2
    expect(list.indices.value).toEqual([2, ...range(496, 514)])
    scope.stop()
  })

  it('scrolls an index into view: down to show it at the bottom, up to show it at the top, not at all when it shows', async () => {
    const { list, el, scope } = setup({ height: 320 })
    await nextTick()
    list.scrollTo(9_999)
    expect(el.value!.scrollTop).toBe(320_000 - 320)
    expect(list.indices.value).toContain(9_999)
    expect(list.indices.value.at(-1)).toBe(9_999)
    list.scrollTo(100)
    expect(el.value!.scrollTop).toBe(3_200)
    list.scrollTo(105)
    expect(el.value!.scrollTop).toBe(3_200)
    list.scrollTo(0)
    expect(el.value!.scrollTop).toBe(0)
    expect(list.indices.value).toEqual(range(0, 14))
    scope.stop()
  })

  it('counts what comes before the rows (a caption and a header) and what sticks over the view\'s top', async () => {
    // A 24 px caption and a 36 px header above row 0; the header sticks, covering the view's top 36 px.
    const { list, el, scope } = setup({ height: 320, leading: 60, stuck: 36 })
    await nextTick()
    list.scrollTo(20) // rows 700–732: its bottom at the view's bottom
    expect(el.value!.scrollTop).toBe(732 - 320)
    // Row 11 is at 412–444, under the header (412–448); 12 (444–476) shows below it.
    expect(list.indices.value).toEqual(range(7, 25))
    list.scrollTo(12) // 444, 4 px under the header: just below it
    expect(el.value!.scrollTop).toBe(444 - 36)
    list.scrollTo(0) // 60: the header over the caption
    expect(el.value!.scrollTop).toBe(24)
    list.scrollTo(9_999)
    expect(el.value!.scrollTop).toBe(60 + 320_000 - 320)
    expect(list.indices.value.at(-1)).toBe(9_999)
    scope.stop()
  })

  it('keeps the window inside the list when the count shrinks under it', async () => {
    const { list, count, el, scope } = setup({ height: 320 })
    await nextTick()
    await scrollTo(el.value!, 160_000)
    count.value = 20
    expect(list.indices.value).toEqual(range(6, 20))
    expect(list.total.value).toBe(640)
    scope.stop()
  })

  it('stops listening when its scope ends', async () => {
    const { list, el, scope } = setup({ height: 320 })
    await nextTick()
    const node = el.value!
    scope.stop()
    await scrollTo(node, 16_000)
    expect(list.indices.value).toEqual(range(0, 14))
  })
})
