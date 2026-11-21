import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import TableCases, { type Person } from '../fixtures/data/TableCases.vue'
import { compareValues, nextSort, parseSort, selectRange, sortRows, tableSearch, type TableQuery, type TableResult } from '../../src/data/table.ts'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
})

const PEOPLE: Person[] = [
  { id: 1, name: 'Grace', age: 85, city: 'Arlington' },
  { id: 2, name: 'Ada', age: 36, city: 'London' },
  { id: 3, name: 'Linus', age: null, city: 'Helsinki' },
  { id: 4, name: 'Barbara', age: 36, city: 'Boston' },
  { id: 5, name: 'Alan', age: 41, city: 'Wilmslow' },
]

/** A function prop for mount(): Vapor reads a root prop given as a function as a getter of its value. */
const fn = <F,>(f: F) => () => f

async function cases(props: Record<string, unknown> = {}) {
  const m = mount(TableCases, { rows: PEOPLE, ...props })
  unmount = m.unmount
  await settle()
  return m as typeof m & { exposed: { sort: string | null; selected: (string | number)[]; page: number; errors: unknown[]; table: { refresh(): Promise<void> } } }
}

const bodyRows = () => $$('.weft-table tbody tr:not([aria-hidden])')
const names = () => bodyRows().map((r) => r.querySelector('th, td')!.textContent!.trim())
const header = (label: string) => $$('.weft-table thead th').find((th) => th.textContent!.trim() === label)!
const sortButton = (label: string) => header(label).querySelector('button')!
const rowBox = (name: string) => bodyRows().find((r) => r.textContent!.includes(name))!.querySelector<HTMLInputElement>('input[type="checkbox"]')!
const allBox = () => $<HTMLInputElement>('.weft-table thead input[type="checkbox"]')
/** A click with its modifiers; on a checkbox happy-dom then ticks it and fires change, as a browser does. */
const click = (el: HTMLElement, init: MouseEventInit = {}) => {
  el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, ...init }))
}
const tick = click

describe('table helpers', () => {
  it('read and step the wire-format sort: ascending, descending, none', () => {
    expect(parseSort('name')).toEqual({ key: 'name', desc: false })
    expect(parseSort('-since')).toEqual({ key: 'since', desc: true })
    expect(parseSort(null)).toBeNull()
    expect(nextSort(null, 'name')).toBe('name')
    expect(nextSort('name', 'name')).toBe('-name')
    expect(nextSort('-name', 'name')).toBeNull()
    expect(nextSort('-age', 'name')).toBe('name')
  })

  it('compares numbers, dates and text, empty values last', () => {
    expect(compareValues(2, 10)).toBeLessThan(0)
    expect(compareValues('item 2', 'item 10')).toBeLessThan(0)
    expect(compareValues('apple', 'Banana')).toBeLessThan(0)
    expect(compareValues(new Date(2020, 0, 1), new Date(2019, 0, 1))).toBeGreaterThan(0)
    expect(compareValues(null, 1)).toBeGreaterThan(0)
    expect(compareValues('', 'a')).toBeGreaterThan(0)
  })

  it('sorts stably in both directions, empty values last either way', () => {
    const cols = [{ key: 'age' }]
    const asc = sortRows(PEOPLE, cols, 'age').map((p) => p.name)
    expect(asc).toEqual(['Ada', 'Barbara', 'Alan', 'Grace', 'Linus'])
    const desc = sortRows(PEOPLE, cols, '-age').map((p) => p.name)
    expect(desc).toEqual(['Grace', 'Alan', 'Ada', 'Barbara', 'Linus'])
    // Unsorted, or by a key no column has: the rows as given, in a new array.
    const same = sortRows(PEOPLE, cols, 'height')
    expect(same).toEqual(PEOPLE)
    expect(same).not.toBe(PEOPLE)
  })

  it('sorts by an accessor', () => {
    const cols = [{ key: 'len', accessor: (p: Person) => p.city.length }]
    expect(sortRows(PEOPLE, cols, 'len').map((p) => p.city)).toEqual(['London', 'Boston', 'Helsinki', 'Wilmslow', 'Arlington'])
  })

  it('writes a query as the wire format search parameters', () => {
    expect(tableSearch({ page: 2, perPage: 20, sort: '-since' }).toString()).toBe('page=2&perPage=20&sort=-since')
    expect(tableSearch({ page: 1, perPage: 10 }).toString()).toBe('page=1&perPage=10')
  })

  it('selects a range between the anchor and the target, in the order shown', () => {
    expect(selectRange([], [1, 2, 3, 4, 5], 2, 4, true)).toEqual([2, 3, 4])
    expect(selectRange([9], [1, 2, 3, 4, 5], 4, 2, true)).toEqual([9, 2, 3, 4])
    expect(selectRange([1, 2, 3, 4], [1, 2, 3, 4, 5], 4, 2, false)).toEqual([1])
    expect(selectRange([], [1, 2, 3], null, 3, true)).toEqual([3])
  })
})

describe('Table', () => {
  it('is a plain table with a caption, column headers and row headers', async () => {
    await cases()
    const table = $('table.weft-table__table')
    expect(table.getAttribute('role')).toBeNull()
    expect(table.querySelector('caption')!.textContent).toBe('People')
    const ths = $$('thead th', table)
    expect(ths.map((t) => t.getAttribute('scope'))).toEqual(['col', 'col', 'col'])
    expect(ths.map((t) => t.textContent!.trim())).toEqual(['Name', 'Age', 'City'])
    // The row-header column's cells are th scope="row"; the others td.
    const first = bodyRows()[0]!
    expect(first.querySelector('th')!.getAttribute('scope')).toBe('row')
    expect(first.querySelector('th')!.textContent!.trim()).toBe('Grace')
    expect($$('td', first).map((c) => c.textContent!.trim())).toEqual(['85', 'Arlington'])
    expect($$('td', first)[0]!.dataset.align).toBe('end')
    expect(names()).toEqual(['Grace', 'Ada', 'Linus', 'Barbara', 'Alan'])
    // Not a grid: no cell is a tab stop.
    expect($$('.weft-table [tabindex]')).toHaveLength(0)
  })

  it('sorts by a header: ascending, descending, then as given, with aria-sort on that header only', async () => {
    const m = await cases()
    expect($$('.weft-table th[aria-sort]')).toHaveLength(0)
    // Only sortable headers hold a button.
    expect(header('City').querySelector('button')).toBeNull()
    click(sortButton('Name'))
    await settle()
    expect(header('Name').getAttribute('aria-sort')).toBe('ascending')
    expect(m.exposed.sort).toBe('name')
    expect(names()).toEqual(['Ada', 'Alan', 'Barbara', 'Grace', 'Linus'])
    click(sortButton('Name'))
    await settle()
    expect(header('Name').getAttribute('aria-sort')).toBe('descending')
    expect(m.exposed.sort).toBe('-name')
    expect(names()).toEqual(['Linus', 'Grace', 'Barbara', 'Alan', 'Ada'])
    click(sortButton('Name'))
    await settle()
    expect(header('Name').hasAttribute('aria-sort')).toBe(false)
    expect(m.exposed.sort).toBeNull()
    expect(names()).toEqual(['Grace', 'Ada', 'Linus', 'Barbara', 'Alan'])
    // Another column: stable, and its header alone says how it sorts.
    click(sortButton('Age'))
    await settle()
    expect(names()).toEqual(['Ada', 'Barbara', 'Alan', 'Grace', 'Linus'])
    expect($$('.weft-table th[aria-sort]')).toEqual([header('Age')])
  })

  it('follows a bound sort', async () => {
    const m = await cases()
    ;(m.exposed as { sort: string | null }).sort = '-age'
    await settle()
    expect(header('Age').getAttribute('aria-sort')).toBe('descending')
    expect(names()).toEqual(['Grace', 'Alan', 'Ada', 'Barbara', 'Linus'])
  })

  it('chooses rows with checkboxes named by their row header, the header checkbox showing some, all or none', async () => {
    const m = await cases({ selectable: true })
    expect($$('thead th').map((t) => t.getAttribute('scope'))).toEqual(['col', 'col', 'col', 'col'])
    expect(allBox().getAttribute('aria-label')).toBe('Select all')
    expect(rowBox('Ada').getAttribute('aria-label')).toBe('Select Ada')
    tick(rowBox('Ada'))
    await settle()
    expect(m.exposed.selected).toEqual([2])
    expect(rowBox('Ada').checked).toBe(true)
    expect(bodyRows()[1]!.dataset.selected).toBe('true')
    expect(allBox().indeterminate).toBe(true)
    expect(allBox().checked).toBe(false)
    tick(allBox())
    await settle()
    expect(m.exposed.selected).toEqual([2, 1, 3, 4, 5])
    expect(allBox().checked).toBe(true)
    expect(allBox().indeterminate).toBe(false)
    tick(allBox())
    await settle()
    expect(m.exposed.selected).toEqual([])
    expect(allBox().checked).toBe(false)
  })

  it('chooses a range with Shift+click, from the last row clicked, in the order shown', async () => {
    const m = await cases({ selectable: true })
    click(sortButton('Name'))
    await settle()
    // Ada, Alan, Barbara, Grace, Linus
    tick(rowBox('Alan'))
    await settle()
    tick(rowBox('Grace'), { shiftKey: true })
    await settle()
    expect([...m.exposed.selected].sort()).toEqual([1, 4, 5])
    expect(rowBox('Barbara').checked).toBe(true)
    // The last row clicked is the next anchor: Shift+click on a chosen row clears back to it.
    tick(rowBox('Barbara'), { shiftKey: true })
    await settle()
    expect([...m.exposed.selected].sort()).toEqual([5])
  })

  it('shows the empty text, or the empty slot, when there are no rows', async () => {
    await cases({ rows: [] })
    const cell = $('.weft-table__empty')
    expect(cell.textContent!.trim()).toBe('Nothing to show.')
    expect(cell.getAttribute('colspan')).toBe('3')
    unmount!()
    await cases({ rows: [], slots: true })
    expect($('.weft-table__empty .no-people').textContent).toBe('No people yet')
  })

  it('draws a cell from its slot, given the row and the value', async () => {
    await cases({ slots: true })
    expect($('tbody .city').textContent).toBe('Arlington (1)')
  })

  it('is busy while loading: skeleton rows hidden from assistive technology, and a status', async () => {
    await cases({ rows: [], loading: true })
    const table = $('table.weft-table__table')
    expect(table.getAttribute('aria-busy')).toBe('true')
    const skeletons = $$('tbody tr.weft-table__skeleton')
    expect(skeletons).toHaveLength(5)
    for (const r of skeletons) expect(r.getAttribute('aria-hidden')).toBe('true')
    expect(skeletons[0]!.querySelectorAll('td')).toHaveLength(3)
    expect(skeletons[0]!.querySelector('.weft-skeleton')).not.toBeNull()
    expect($('.weft-table [role="status"]').textContent).toBe('Loading')
    expect($('.weft-table__empty')).toBeNull()
  })

  it('keeps the rows it has while loading again, busy', async () => {
    await cases({ loading: true })
    expect($('table.weft-table__table').getAttribute('aria-busy')).toBe('true')
    expect(names()).toHaveLength(5)
    expect($$('tbody tr.weft-table__skeleton')).toHaveLength(0)
  })

  it('pages the rows on the client with a Pagination, going back to the first page when the sort changes', async () => {
    const m = await cases({ perPage: 2 })
    expect(names()).toEqual(['Grace', 'Ada'])
    const nav = $('.weft-table nav.weft-pagination')
    expect(nav).not.toBeNull()
    click($('[aria-label="Next page"]', nav))
    await settle()
    expect(m.exposed.page).toBe(2)
    expect(names()).toEqual(['Linus', 'Barbara'])
    click($('[aria-label="Page 3"]', nav))
    await settle()
    expect(names()).toEqual(['Alan'])
    click(sortButton('Name'))
    await settle()
    expect(m.exposed.page).toBe(1)
    expect(names()).toEqual(['Ada', 'Alan'])
    // The header checkbox chooses the rows shown, on this page.
    unmount!()
    const n = await cases({ perPage: 2, selectable: true })
    tick(allBox())
    await settle()
    expect(n.exposed.selected).toEqual([1, 2])
  })

  it('has no Pagination when every row fits one page', async () => {
    await cases({ perPage: 10 })
    expect($('.weft-table .weft-pagination')).toBeNull()
  })

  describe('with more than 200 rows', () => {
    const many: Person[] = Array.from({ length: 1000 }, (_, i) => ({ id: i + 1, name: `Person ${i + 1}`, age: i % 90, city: 'Town' }))

    it('renders only the rows in view, with spacers, the row count and each row index', async () => {
      await cases({ rows: many, rowHeight: 40 })
      const table = $('table.weft-table__table')
      expect($('.weft-table').dataset.virtual).toBe('true')
      expect(table.getAttribute('aria-rowcount')).toBe('1001')
      const rows = bodyRows()
      expect(rows.length).toBeGreaterThan(5)
      expect(rows.length).toBeLessThan(40)
      expect(rows[0]!.getAttribute('aria-rowindex')).toBe('2')
      expect($('thead tr').getAttribute('aria-rowindex')).toBe('1')
      const spacers = $$('tbody tr[aria-hidden="true"].weft-table__spacer')
      expect(spacers.length).toBeGreaterThan(0)
      // Scrolling shows other rows.
      const scroller = $('.weft-table__scroller')
      Object.defineProperty(scroller, 'clientHeight', { configurable: true, get: () => 400 })
      scroller.scrollTop = 20_000
      scroller.dispatchEvent(new Event('scroll'))
      await settle()
      const now = names()
      expect(now).toContain('Person 501')
      expect(now).not.toContain('Person 1')
      const top = $$('tbody tr.weft-table__spacer')[0]!
      expect(px(top.querySelector('td')!.style.blockSize)).toBe(40 * Number(bodyRows()[0]!.getAttribute('aria-rowindex')!) - 80)
    })

    it('does not window 200 rows or fewer', async () => {
      await cases({ rows: many.slice(0, 200) })
      expect($('.weft-table').dataset.virtual).toBeUndefined()
      expect(bodyRows()).toHaveLength(200)
      expect($('table.weft-table__table').hasAttribute('aria-rowcount')).toBe(false)
    })
  })

  describe('when narrow', () => {
    async function narrow(props: Record<string, unknown> = {}) {
      ro = installResizeObserver()
      const m = await cases(props)
      ro.report($('.weft-table'), 400)
      await settle()
      return m
    }

    it('renders each row as a card of label and value pairs from the same columns', async () => {
      await narrow({ slots: true })
      expect($('table')).toBeNull()
      expect($('.weft-table').dataset.layout).toBe('cards')
      const list = $('ul.weft-table__cards')
      expect(list.getAttribute('aria-label')).toBe('People')
      const cards = $$(':scope > li', list)
      expect(cards).toHaveLength(5)
      const pairs = $$('dl > div', cards[0]!)
      expect(pairs.map((p) => p.querySelector('dt')!.textContent!.trim())).toEqual(['Name', 'Age', 'City'])
      expect(pairs.map((p) => p.querySelector('dd')!.textContent!.trim())).toEqual(['Grace', '85', 'Arlington (1)'])
      // Back to a table when wide again.
      ro!.report($('.weft-table'), 900)
      await settle()
      expect($('table')).not.toBeNull()
    })

    it('keeps selection and sorting in the cards', async () => {
      const m = await narrow({ selectable: true })
      const boxes = $$<HTMLInputElement>('.weft-table__card input[type="checkbox"]')
      expect(boxes[1]!.getAttribute('aria-label')).toBe('Select Ada')
      tick(boxes[1]!)
      await settle()
      expect(m.exposed.selected).toEqual([2])
      expect($$('.weft-table__card')[1]!.dataset.selected).toBe('true')
      const all = $<HTMLInputElement>('.weft-table__toolbar input[type="checkbox"]')
      expect(all.indeterminate).toBe(true)
      tick(all)
      await settle()
      expect(m.exposed.selected).toHaveLength(5)
      // The sortable columns are buttons above the cards, saying how they sort.
      const sorts = $$<HTMLButtonElement>('.weft-table__toolbar .weft-table__sort')
      expect(sorts.map((b) => b.textContent!.trim())).toEqual(['Name', 'Age'])
      click(sorts[0]!)
      await settle()
      expect(m.exposed.sort).toBe('name')
      expect($$('.weft-table__card dd')[0]!.textContent!.trim()).toBe('Ada')
      expect($$<HTMLButtonElement>('.weft-table__toolbar .weft-table__sort')[0]!.textContent).toContain('sorted ascending')
    })

    it('shows the empty text in place of the cards', async () => {
      await narrow({ rows: [] })
      expect($('.weft-table__empty').textContent!.trim()).toBe('Nothing to show.')
    })
  })

  describe('with keyboard', () => {
    it('is a grid: one tab stop, arrows move between cells, Home and End along the row', async () => {
      await cases({ keyboard: true, selectable: true })
      const table = $('table.weft-table__table')
      expect(table.getAttribute('role')).toBe('grid')
      const stops = $$('.weft-table [tabindex="0"]')
      expect(stops).toHaveLength(1)
      // The first item: the select-all checkbox.
      expect(stops[0]).toBe(allBox())
      allBox().focus()
      key('ArrowRight')
      expect(active()).toBe(sortButton('Name'))
      key('ArrowDown')
      expect(active()!.textContent!.trim()).toBe('Grace')
      key('ArrowRight')
      expect(active()!.textContent!.trim()).toBe('85')
      key('End')
      expect(active()!.textContent!.trim()).toBe('Arlington')
      key('Home')
      expect(active()).toBe(rowBox('Grace'))
      key('ArrowDown')
      expect(active()).toBe(rowBox('Ada'))
      expect($$('.weft-table [tabindex="0"]')).toEqual([rowBox('Ada')])
    })

    it('mirrors the arrows right to left', async () => {
      await cases({ keyboard: true, dir: 'rtl' })
      sortButton('Name').focus()
      key('ArrowLeft')
      expect(active()).toBe(sortButton('Age'))
    })

    it('moves into a cell with Enter and back out with Escape', async () => {
      await cases({ keyboard: true, slots: true, columns: [{ key: 'name', label: 'Name' }, { key: 'city', label: 'City' }] })
      const cells = $$('tbody td')
      // A cell whose slot holds a control: Enter moves into it.
      const cell = cells[1]!
      const edit = document.createElement('button')
      edit.className = 'inner'
      edit.textContent = 'Edit'
      cell.append(edit)
      cell.focus()
      key('Enter')
      expect(active()!.className).toBe('inner')
      key('Escape')
      expect(active()).toBe(cell)
    })

    it('is a plain table without the prop', async () => {
      await cases({ selectable: true })
      expect($('table').getAttribute('role')).toBeNull()
      expect(allBox().hasAttribute('tabindex')).toBe(false)
    })
  })

  describe('with fetch', () => {
    function server(total = 45) {
      const all: Person[] = Array.from({ length: total }, (_, i) => ({ id: i + 1, name: `P${String(i + 1).padStart(2, '0')}`, age: i, city: 'X' }))
      const calls: TableQuery[] = []
      const fetch = vi.fn(async (q: TableQuery): Promise<TableResult<Person>> => {
        calls.push(q)
        const sorted = sortRows(all, [{ key: 'name' }, { key: 'age' }], q.sort)
        return { items: sorted.slice((q.page - 1) * q.perPage, q.page * q.perPage), total: all.length, page: q.page, perPage: q.perPage }
      })
      return { fetch, calls }
    }

    it('asks for the first page once mounted, busy until it answers, then pages with Pagination', async () => {
      const { fetch, calls } = server()
      const m = mount(TableCases, { rows: undefined, fetch: fn(fetch), perPage: 20 })
      unmount = m.unmount
      // Before the answer: busy, skeleton rows.
      await nextTick()
      expect($('table').getAttribute('aria-busy')).toBe('true')
      expect($$('tbody tr.weft-table__skeleton')).toHaveLength(20)
      await settle()
      expect(calls).toEqual([{ page: 1, perPage: 20 }])
      expect($('table').hasAttribute('aria-busy')).toBe(false)
      expect(names()).toHaveLength(20)
      expect(names()[0]).toBe('P01')
      click($('.weft-table [aria-label="Page 3"]'))
      await settle()
      expect(calls.at(-1)).toEqual({ page: 3, perPage: 20 })
      expect(names()).toEqual(['P41', 'P42', 'P43', 'P44', 'P45'])
    })

    it('sends the sort to the server, in the wire format, from page 1, and does not sort on the client', async () => {
      const { fetch, calls } = server()
      const m = await cases({ rows: undefined, fetch: fn(fetch), perPage: 10 })
      click($('.weft-table [aria-label="Page 2"]'))
      await settle()
      click(sortButton('Name'))
      await settle()
      click(sortButton('Name'))
      await settle()
      expect(calls.at(-1)).toEqual({ page: 1, perPage: 10, sort: '-name' })
      expect(m.exposed.page).toBe(1)
      expect(names()[0]).toBe('P45')
    })

    it('ignores an answer that a newer query overtook', async () => {
      const answers: ((r: TableResult<Person>) => void)[] = []
      const fetch = (_q: TableQuery) => new Promise<TableResult<Person>>((resolve) => answers.push(resolve))
      await cases({ rows: undefined, fetch: fn(fetch), perPage: 10 })
      click(sortButton('Name'))
      await settle()
      expect(answers).toHaveLength(2)
      answers[1]!({ items: [PEOPLE[1]!], total: 1 })
      await settle()
      answers[0]!({ items: [PEOPLE[0]!], total: 1 })
      await settle()
      expect(names()).toEqual(['Ada'])
    })

    it('says when fetch failed, emits the error and tries again', async () => {
      let fail = true
      const fetch = vi.fn(async () => {
        if (fail) throw new Error('offline')
        return { items: PEOPLE, total: 5 }
      })
      const m = await cases({ rows: undefined, fetch: fn(fetch), perPage: 10 })
      expect(m.exposed.errors).toHaveLength(1)
      const cell = $('.weft-table__error')
      expect(cell.getAttribute('role')).toBe('alert')
      expect(cell.textContent).toContain('The rows could not be loaded.')
      fail = false
      click($('button', cell))
      await settle()
      expect(names()).toHaveLength(5)
      expect($('.weft-table__error')).toBeNull()
      // refresh() asks again.
      await m.exposed.table.refresh()
      expect(fetch).toHaveBeenCalledTimes(3)
    })
  })

  describe('styles', () => {
    it('keeps the header in view as the rows scroll, and gives each row a 44 px hit on touch', async () => {
      const unstyle = loadStyles()
      try {
        await cases({ selectable: true, input: 'touch' })
        expect(getComputedStyle($('thead th')).position).toBe('sticky')
        expect(px(getComputedStyle(bodyRows()[0]!.querySelector('td')!).blockSize)).toBeGreaterThanOrEqual(44)
        expect(ruleFor('.weft-table__table[role="grid"] td:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      } finally {
        unstyle()
      }
    })
  })
})
