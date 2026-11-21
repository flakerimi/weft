import { afterEach, describe, expect, it } from 'vitest'
import DataCases from '../fixtures/data/DataCases.vue'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles } from '../styles.ts'
import { $, $$, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(DataCases, props)
  unmount = m.unmount
  await settle()
  return m
}

describe('StatCard', () => {
  const card = () => $('.weft-stat-card')
  const delta = () => $('.weft-stat-card__delta')

  it('is a term and its value, the number formatted for the locale', async () => {
    await cases()
    const dl = $('dl', card())
    expect($('dt', dl).textContent!.trim()).toBe('Revenue')
    expect($('dd.weft-stat-card__value', dl).textContent).toBe('12,480')
    expect($('dt svg.weft-icon', dl)).not.toBeNull()
    unmount!()
    await cases({ locale: 'de-DE' })
    expect($('.weft-stat-card__value').textContent).toBe('12.480')
    unmount!()
    await cases({ value: '$1.2M' })
    expect($('.weft-stat-card__value').textContent).toBe('$1.2M')
  })

  it('says a rise as "up", in the success tone, with a sign drawn and the words for assistive technology', async () => {
    await cases({ delta: 4 })
    expect(delta().dataset.tone).toBe('success')
    expect($('[aria-hidden="true"].weft-stat-card__change', delta()).textContent).toBe('+4%')
    expect($('.weft-visually-hidden', delta()).textContent).toBe('up 4%')
    expect($('.weft-stat-card__description', delta()).textContent).toBe('vs last month')
  })

  it('says a fall as "down" in the danger tone, swapped with invert, and no change as neutral', async () => {
    await cases({ delta: -2.5 })
    expect(delta().dataset.tone).toBe('danger')
    expect($('.weft-stat-card__change', delta()).textContent).toBe('−2.5%')
    expect($('.weft-visually-hidden', delta()).textContent).toBe('down 2.5%')
    unmount!()
    await cases({ delta: -2.5, invert: true })
    expect(delta().dataset.tone).toBe('success')
    unmount!()
    await cases({ delta: 0 })
    expect(delta().dataset.tone).toBe('neutral')
    expect($('.weft-visually-hidden', delta()).textContent).toBe('no change')
  })

  it('has no delta when none is given, and a sparkline from its slot, hidden from assistive technology', async () => {
    await cases()
    expect(delta()).toBeNull()
    const spark = $('.weft-stat-card__sparkline', card())
    expect(spark.getAttribute('aria-hidden')).toBe('true')
    expect($('svg.spark', spark)).not.toBeNull()
  })

  it('is busy while loading, a skeleton in place of the value', async () => {
    await cases({ loading: true, delta: 3 })
    expect(card().getAttribute('aria-busy')).toBe('true')
    expect($('.weft-stat-card__value .weft-skeleton', card())).not.toBeNull()
    expect(delta()).toBeNull()
  })
})

describe('Timeline', () => {
  const items = () => $$('.weft-timeline > li')

  it('is an ordered list of events, each with its date, title and description', async () => {
    await cases()
    expect($('ol.weft-timeline')).not.toBeNull()
    expect(items()).toHaveLength(3)
    expect($('.weft-timeline__date', items()[0]!).textContent!.trim()).toBe('Mar 1')
    expect($('.weft-timeline__title', items()[0]!).textContent!.trim()).toBe('Ordered')
    expect($('.weft-timeline__description .desc', items()[0]!).textContent).toBe('Paid by card')
    expect($('.weft-timeline__description .desc', items()[1]!).textContent).toBe('none')
    // The indicator and the line are drawing only.
    expect($('.weft-timeline__indicator', items()[0]!).getAttribute('aria-hidden')).toBe('true')
    expect(items().map((li) => li.dataset.state)).toEqual([undefined, undefined, undefined])
    expect($$('.weft-timeline [aria-current]')).toHaveLength(0)
  })

  it('marks the items before the current one done, and the current one with aria-current="step"', async () => {
    await cases({ current: 'shipped' })
    expect(items().map((li) => li.dataset.state)).toEqual(['done', 'current', undefined])
    expect(items()[1]!.getAttribute('aria-current')).toBe('step')
    expect($('.weft-visually-hidden', items()[0]!).textContent).toBe('Done: ')
    unmount!()
    await cases({ current: 2 })
    expect(items().map((li) => li.dataset.state)).toEqual(['done', 'done', 'current'])
  })

  it('reversed, lists the newest first, the done ones after the current one', async () => {
    await cases({ current: 'shipped', reverse: true })
    expect(items().map((li) => $('.weft-timeline__title', li).textContent!.trim())).toEqual(['Delivered', 'Shipped', 'Done: Ordered'])
    expect(items().map((li) => li.dataset.state)).toEqual([undefined, 'current', 'done'])
  })

  it('lays out horizontally, and vertically when its box is narrow', async () => {
    ro = installResizeObserver()
    await cases({ orientation: 'horizontal' })
    const ol = $('ol.weft-timeline')
    expect(ol.dataset.orientation).toBe('horizontal')
    ro.report(ol, 400)
    await settle()
    expect(ol.dataset.orientation).toBe('vertical')
  })

  it('draws the done and current dots in its tone, the current one ringed', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ current: 'shipped' })
      expect($('.weft-timeline').dataset.tone).toBe('primary')
      const dots = $$('.weft-timeline__indicator').map((d) => getComputedStyle(d))
      expect(dots[0]!.backgroundColor).toBe(dots[1]!.backgroundColor)
      expect(dots[2]!.backgroundColor).not.toBe(dots[1]!.backgroundColor)
      expect(dots[1]!.boxShadow).toContain('4px')
      expect(dots[2]!.boxShadow).not.toContain('4px')
    } finally {
      unstyle()
    }
  })
})
