import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { installResizeObserver } from './resize-observer.ts'
import { mount } from './mount.ts'
import LayoutCases from './fixtures/LayoutCases.vue'

const first = (c: string, sel: string) => mount(LayoutCases, { case: c }).el.querySelector(sel) as HTMLElement

describe('Stack', () => {
  it('defaults to a column with gap 4', () => {
    const s = first('stack-default', '.weft-stack')
    expect(s.tagName).toBe('DIV')
    expect(s.dataset).toMatchObject({ direction: 'column', gap: '4', align: 'stretch', justify: 'start' })
    expect(s.dataset.wrap).toBeUndefined()
  })
  it('takes direction, gap, wrap, align and justify', () => {
    const s = first('stack-row', '.weft-stack')
    expect(s.dataset).toMatchObject({ direction: 'row', gap: '2', wrap: 'true', align: 'center', justify: 'between' })
  })
})

describe('Grid', () => {
  let ro: ReturnType<typeof installResizeObserver>
  beforeEach(() => { ro = installResizeObserver() })
  afterEach(() => ro.restore())

  it('picks columns for its own width', async () => {
    const g = first('grid-default', '.weft-grid')
    ro.report(g, 500)
    await nextTick()
    expect(g.dataset.width).toBe('narrow')
    expect(g.style.getPropertyValue('--weft-grid-cols')).toBe('1')
    ro.report(g, 1100)
    await nextTick()
    expect(g.dataset.width).toBe('wide')
    expect(g.style.getPropertyValue('--weft-grid-cols')).toBe('3')
  })
  it('falls back to the nearest smaller width given', async () => {
    const g = first('grid-partial', '.weft-grid')
    for (const w of [500, 800, 1100]) {
      ro.report(g, w)
      await nextTick()
      expect(g.style.getPropertyValue('--weft-grid-cols'), `${w}px`).toBe('2')
    }
  })
  it('keeps a fixed column count at every width', async () => {
    const g = first('grid-fixed', '.weft-grid')
    for (const w of [500, 800, 1100]) {
      ro.report(g, w)
      await nextTick()
      expect(g.style.getPropertyValue('--weft-grid-cols')).toBe('4')
    }
  })
})

describe('Page and Section', () => {
  it('Page renders a main with its title, description, actions and content', () => {
    const p = first('page', 'main.weft-page')
    expect(p.querySelector('h1')?.textContent).toBe('Customers')
    expect(p.textContent).toContain('Everyone who pays')
    expect(p.querySelector('.act')?.textContent).toBe('New')
    expect(p.querySelector('.body')).not.toBeNull()
  })
  it('Section renders a section with an h2', () => {
    const s = first('section', 'section.weft-section')
    expect(s.querySelector('h2')?.textContent).toBe('Billing')
  })
})

describe('Card', () => {
  it('has no header region without header or actions slots', () => {
    const c = first('card-plain', '.weft-card')
    expect(c.querySelector('.weft-card__header')).toBeNull()
    expect(c.dataset.padding).toBe('4')
    expect(c.querySelector('.card-body')).not.toBeNull()
  })
  it('renders header, actions and footer', () => {
    const c = first('card-header', '.weft-card')
    expect(c.querySelector('.weft-card__header')?.textContent).toContain('Plan')
    expect(c.querySelector('.weft-card__actions button')?.textContent).toBe('Edit')
    expect(c.querySelector('.weft-card__footer')?.textContent).toBe('Renews monthly')
    expect(c.dataset.padding).toBe('6')
  })
})

describe('Divider', () => {
  it('is an hr when horizontal', () => {
    const d = first('divider', '.weft-divider')
    expect(d.tagName).toBe('HR')
    expect(d.getAttribute('aria-orientation')).toBe('horizontal')
  })
  it('is a vertical separator', () => {
    const d = first('divider-vertical', '.weft-divider')
    expect(d.tagName).toBe('DIV')
    expect(d.getAttribute('role')).toBe('separator')
    expect(d.getAttribute('aria-orientation')).toBe('vertical')
  })
})
