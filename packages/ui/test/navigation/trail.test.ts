import { afterEach, describe, expect, it } from 'vitest'
import TrailCases from '../fixtures/navigation/TrailCases.vue'
import { pageRange, pageRangeBoth } from '../../src/navigation/pagination.ts'
import { mount } from '../mount.ts'
import { press as pressKey } from '../keyboard.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
  history.replaceState(null, '', '/')
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(TrailCases, props)
  unmount = m.unmount
  await settle()
  return m
}

describe('Breadcrumb', () => {
  const crumbs = () => $$('.weft-breadcrumb__item')
  it('is a navigation landmark holding an ordered list of the path', async () => {
    await cases()
    const nav = $('nav.weft-breadcrumb')
    expect(nav.getAttribute('aria-label')).toBe('Breadcrumb')
    expect(nav.querySelector(':scope > ol')).not.toBeNull()
    expect(crumbs().map((c) => c.textContent!.trim())).toEqual(['Home', 'Docs', 'Components', 'Breadcrumb'])
    expect(crumbs()[0]!.querySelector('a')!.getAttribute('href')).toBe('/')
    expect(crumbs()[0]!.querySelector('svg.weft-icon')).not.toBeNull()
    // An item with no address is text, not a link.
    expect(crumbs()[2]!.querySelector('a')).toBeNull()
  })

  it('marks the last item as the current page, and only it', async () => {
    history.replaceState(null, '', '/docs/components/breadcrumb')
    await cases()
    const current = $$('.weft-breadcrumb [aria-current]')
    expect(current).toHaveLength(1)
    expect(current[0]!.getAttribute('aria-current')).toBe('page')
    expect(current[0]!.textContent!.trim()).toBe('Breadcrumb')
  })

  it('hides the separators from assistive technology and mirrors them right to left', async () => {
    await cases()
    const seps = $$('.weft-breadcrumb__separator')
    expect(seps).toHaveLength(3)
    for (const s of seps) expect(s.getAttribute('aria-hidden')).toBe('true')
    const unstyle = loadStyles()
    try {
      unmount!()
      await cases({ dir: 'rtl' })
      expect(getComputedStyle($('.weft-breadcrumb__separator')).scale).toBe('-1 1')
    } finally {
      unstyle()
    }
  })

  it('follows a link with Enter', async () => {
    await cases()
    const link = crumbs()[1]!.querySelector('a')!
    let followed = false
    link.addEventListener('click', (e) => {
      e.preventDefault()
      followed = true
    })
    expect(pressKey(link, 'Enter')).toBe(true)
    expect(followed).toBe(true)
  })
})

describe('pageRange', () => {
  it('gives a window of pages around the current one, shifted to fit', () => {
    expect(pageRange(5, 20, 1, false)).toEqual([4, 5, 6])
    expect(pageRange(1, 20, 2, false)).toEqual([1, 2, 3, 4, 5])
    expect(pageRange(20, 20, 2, false)).toEqual([16, 17, 18, 19, 20])
    expect(pageRange(2, 3, 2, false)).toEqual([1, 2, 3])
  })

  it('with edges, always shows the first and last pages, with ellipses for the gaps, at a constant length', () => {
    expect(pageRange(5, 20, 1, true)).toEqual([1, 'ellipsis', 4, 5, 6, 'ellipsis', 20])
    expect(pageRange(1, 20, 1, true)).toEqual([1, 2, 3, 4, 5, 'ellipsis', 20])
    expect(pageRange(4, 20, 1, true)).toEqual([1, 2, 3, 4, 5, 'ellipsis', 20])
    expect(pageRange(20, 20, 1, true)).toEqual([1, 'ellipsis', 16, 17, 18, 19, 20])
    expect(pageRange(3, 6, 1, true)).toEqual([1, 2, 3, 4, 5, 6])
  })

  it('has no pages when there is nothing to page', () => {
    expect(pageRange(1, 0, 1, true)).toEqual([])
  })
})

describe('pageRangeBoth', () => {
  it('holds the narrow and the wide pages in one list: leaving out one form\'s entries gives the other\'s pageRange', () => {
    for (const edges of [false, true]) {
      for (let siblings = 0; siblings <= 3; siblings++) {
        for (let count = 0; count <= 24; count++) {
          for (let page = 1; page <= Math.max(1, count); page++) {
            const both = pageRangeBoth(page, count, siblings, edges)
            const at = `page ${page} of ${count}, ${siblings} siblings, edges ${edges}`
            expect(both.filter((s) => s.only !== 'wide').map((s) => s.entry), at).toEqual(pageRange(page, count, Math.min(siblings, 1), edges))
            expect(both.filter((s) => s.only !== 'narrow').map((s) => s.entry), at).toEqual(pageRange(page, count, siblings, edges))
          }
        }
      }
    }
  })

  it('marks only what differs: a page or gap in both forms is drawn once, unmarked, and a gap that ends at another page once per form', () => {
    expect(pageRangeBoth(10, 20, 2, true)).toEqual([
      { entry: 1 },
      { entry: 'ellipsis', only: 'wide' },
      { entry: 8, only: 'wide' },
      { entry: 'ellipsis', only: 'narrow' },
      { entry: 9 },
      { entry: 10 },
      { entry: 11 },
      { entry: 12, only: 'wide' },
      { entry: 'ellipsis' },
      { entry: 20 },
    ])
    expect(pageRangeBoth(5, 20, 1, true).every((s) => s.only === undefined)).toBe(true)
  })
})

describe('Pagination', () => {
  const pages = () => $$<HTMLElement>('.weft-pagination__page')
  const control = (name: string) => $<HTMLButtonElement>(`[aria-label="${name}"]`)
  it('is a navigation landmark; the current page says aria-current="page"', async () => {
    await cases()
    const nav = $('nav.weft-pagination')
    expect(nav.getAttribute('aria-label')).toBe('Pagination')
    expect(pages().map((p) => p.textContent!.trim())).toEqual(['4', '5', '6'])
    const current = pages().filter((p) => p.getAttribute('aria-current') === 'page')
    expect(current.map((p) => p.textContent!.trim())).toEqual(['5'])
    expect(pages()[0]!.getAttribute('aria-label')).toBe('Page 4')
  })

  it('moves with the page buttons and the controls, which are disabled at the ends', async () => {
    const m = await cases()
    pages()[2]!.click()
    await settle()
    expect(m.exposed.page).toBe(6)
    control('Next page').click()
    await settle()
    expect(m.exposed.page).toBe(7)
    control('Previous page').click()
    control('Previous page').click()
    await settle()
    expect(m.exposed.page).toBe(5)
    control('Last page').click()
    await settle()
    expect(m.exposed.page).toBe(20)
    expect(control('Next page').disabled).toBe(true)
    expect(control('Last page').disabled).toBe(true)
    control('First page').click()
    await settle()
    expect(m.exposed.page).toBe(1)
    expect(control('Previous page').disabled).toBe(true)
  })

  it('works from the keyboard: Enter and Space on a page button', async () => {
    const m = await cases()
    expect(pressKey(pages()[0]!, 'Enter')).toBe(true)
    await settle()
    expect(m.exposed.page).toBe(4)
    expect(pressKey(control('Next page'), ' ')).toBe(true)
    await settle()
    expect(m.exposed.page).toBe(5)
  })

  it('draws ellipses, hidden from assistive technology, with edges', async () => {
    await cases({ showEdges: true })
    expect(pages().map((p) => p.textContent!.trim())).toEqual(['1', '4', '5', '6', '20'])
    const dots = $$('.weft-pagination__ellipsis')
    expect(dots).toHaveLength(2)
    for (const d of dots) expect(d.getAttribute('aria-hidden')).toBe('true')
  })

  it('renders links when to is given, the current one with aria-current', async () => {
    await cases({ links: true })
    const links = pages()
    expect(links.every((l) => l.tagName === 'A')).toBe(true)
    expect(links[0]!.getAttribute('href')).toBe('/list?page=4')
    expect(links[1]!.getAttribute('aria-current')).toBe('page')
    expect(control('Next page').getAttribute('href')).toBe('/list?page=6')
  })

  it('disables every button when disabled', async () => {
    await cases({ disabled: true })
    expect($$<HTMLButtonElement>('.weft-pagination button').every((b) => b.disabled)).toBe(true)
  })

  it('shows fewer pages and no first and last controls when the viewport is narrow', async () => {
    const ro = installResizeObserver()
    try {
      await cases({ siblingCount: 2 })
      expect(pages()).toHaveLength(5)
      ro.report(document.documentElement, 400)
      await settle()
      expect(pages().map((p) => p.textContent!.trim())).toEqual(['4', '5', '6'])
      expect(control('First page')).toBeNull()
      expect(control('Next page')).not.toBeNull()
    } finally {
      ro.restore()
    }
  })

  it('gives each button a 44 px hit area on touch', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(pages()[0]!).blockSize)).toBeGreaterThanOrEqual(44)
    } finally {
      unstyle()
    }
  })
})
