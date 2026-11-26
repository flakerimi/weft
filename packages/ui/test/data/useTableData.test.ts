// The Table's data source on its own: rows sorted and paged on the client,
// or a page asked of `fetch` in the wire format, with no component mounted.
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref, shallowRef } from 'vue'
import { useTableData, type TableDataOptions } from '../../src/data/useTableData.ts'
import type { TableColumn, TableFetch, TableQuery, TableResult } from '../../src/data/table.ts'

interface P {
  id?: number
  name: string
}
const PEOPLE: P[] = [
  { id: 1, name: 'Grace' },
  { id: 2, name: 'Ada' },
  { id: 3, name: 'Linus' },
  { id: 4, name: 'Barbara' },
  { id: 5, name: 'Alan' },
]
const columns: TableColumn<P>[] = [{ key: 'name', sortable: true }]
const flush = async () => {
  await nextTick()
  await new Promise((r) => setTimeout(r, 0))
  await nextTick()
}

let stop: (() => void) | undefined
afterEach(() => {
  stop?.()
  stop = undefined
  vi.restoreAllMocks()
})

function setup(over: Partial<{ rows: P[]; fetch: TableFetch<P>; perPage: number; active: boolean; needKeys: boolean; keepsPage: boolean }> = {}) {
  const rows = shallowRef(over.rows)
  const fetch = shallowRef(over.fetch)
  const sort = ref<string | null>(null)
  const page = ref(1)
  const perPage = ref(over.perPage)
  const active = ref(over.active ?? true)
  const loading = ref(false)
  const errors: unknown[] = []
  const opts: TableDataOptions<P> = {
    rows: () => rows.value,
    fetch: () => fetch.value,
    columns: () => columns,
    rowKey: () => undefined,
    sort,
    page,
    perPage: () => perPage.value,
    loading: () => loading.value,
    active: () => active.value,
    needKeys: () => !!over.needKeys,
    onError: (e) => errors.push(e),
    setPage: (p) => {
      // keepsPage: a parent that binds the page one-way and ignores the
      // update, which a Table's page shows until its next tick.
      const kept = page.value
      page.value = p
      if (over.keepsPage) void nextTick(() => (page.value = kept))
    },
  }
  const scope = effectScope()
  const data = scope.run(() => useTableData(opts))!
  stop = () => scope.stop()
  return { data, rows, fetch, sort, page, perPage, active, loading, errors }
}
const names = (shown: { row: P }[]) => shown.map((e) => e.row.name)

describe('useTableData', () => {
  describe('with rows', () => {
    it('sorts and slices the rows to the page, keyed by id', () => {
      const { data, sort, page } = setup({ rows: PEOPLE, perPage: 2 })
      expect(names(data.shown.value)).toEqual(['Grace', 'Ada'])
      expect(data.shown.value.map((e) => e.key)).toEqual([1, 2])
      expect(data.total.value).toBe(5)
      expect(data.pageCount.value).toBe(3)
      sort.value = 'name'
      page.value = 2
      expect(names(data.shown.value)).toEqual(['Barbara', 'Grace'])
    })

    it('keeps the page inside the pages there are', () => {
      const { data, page } = setup({ rows: PEOPLE, perPage: 2 })
      page.value = 9
      expect(data.currentPage.value).toBe(3)
      expect(names(data.shown.value)).toEqual(['Alan'])
    })

    it('shows every row without perPage, keyed by place when rows have no id', () => {
      const { data } = setup({ rows: PEOPLE.map(({ name }) => ({ name })) })
      expect(data.shown.value).toHaveLength(5)
      expect(data.shown.value.map((e) => e.key)).toEqual([0, 1, 2, 3, 4])
      expect(data.pageCount.value).toBe(1)
    })

    it('is busy only while loading is set', () => {
      const { data, loading } = setup({ rows: PEOPLE })
      expect(data.busy.value).toBe(false)
      loading.value = true
      expect(data.busy.value).toBe(true)
    })
  })

  describe('with fetch', () => {
    function server() {
      const calls: TableQuery[] = []
      const fetch = vi.fn(async (q: TableQuery): Promise<TableResult<P>> => {
        calls.push({ ...q })
        return { items: PEOPLE.slice((q.page - 1) * q.per_page, q.page * q.per_page), total: PEOPLE.length }
      })
      return { fetch, calls }
    }

    it('asks nothing until active, busy meanwhile, then asks for the page in the wire format', async () => {
      const { fetch, calls } = server()
      const { data, active } = setup({ fetch, active: false })
      await flush()
      expect(calls).toEqual([])
      expect(data.busy.value).toBe(true)
      expect(data.perPage.value).toBe(20)
      active.value = true
      await flush()
      expect(calls).toEqual([{ page: 1, per_page: 20 }])
      expect(data.busy.value).toBe(false)
      expect(names(data.shown.value)).toHaveLength(5)
      expect(data.total.value).toBe(5)
    })

    it('asks once when the sort and the page change together, and never twice for the same query', async () => {
      const { fetch, calls } = server()
      const { sort, page } = setup({ fetch, perPage: 2 })
      await flush()
      page.value = 2
      await flush()
      sort.value = '-name'
      page.value = 1
      await flush()
      expect(calls).toEqual([
        { page: 1, per_page: 2 },
        { page: 2, per_page: 2 },
        { page: 1, per_page: 2, sort: '-name' },
      ])
      // Writing the same values again asks nothing.
      sort.value = '-name'
      page.value = 1
      await flush()
      expect(calls).toHaveLength(3)
    })

    it('keeps the answer to the latest query only', async () => {
      const answers: ((r: TableResult<P>) => void)[] = []
      const fetch = (_q: TableQuery) => new Promise<TableResult<P>>((resolve) => answers.push(resolve))
      const { data, sort } = setup({ fetch })
      await flush()
      sort.value = 'name'
      await flush()
      answers[1]!({ items: [PEOPLE[1]!], total: 1 })
      await flush()
      answers[0]!({ items: [PEOPLE[0]!], total: 1 })
      await flush()
      expect(names(data.shown.value)).toEqual(['Ada'])
    })

    it('reports a failure, not busy, and loads again on load()', async () => {
      let fail = true
      const fetch = async () => {
        if (fail) throw new Error('offline')
        return { items: PEOPLE, total: 5 }
      }
      const { data, errors } = setup({ fetch })
      await flush()
      expect(errors).toHaveLength(1)
      expect(data.failed.value).toBeInstanceOf(Error)
      expect(data.busy.value).toBe(false)
      fail = false
      await data.load()
      expect(data.failed.value).toBeNull()
      expect(data.shown.value).toHaveLength(5)
    })

    it('does not sort on the client', async () => {
      const { fetch } = server()
      const { data, sort } = setup({ fetch })
      await flush()
      sort.value = 'name'
      await flush()
      expect(names(data.shown.value)).toEqual(['Grace', 'Ada', 'Linus', 'Barbara', 'Alan'])
    })

    // A server whose rows can be removed, as a table's Delete does.
    function shrinking(count: number) {
      const rows = { count }
      const calls: number[] = []
      const fetch = vi.fn(async (q: TableQuery): Promise<TableResult<P>> => {
        calls.push(q.page)
        const all = Array.from({ length: rows.count }, (_, i) => ({ id: i + 1, name: `R${i + 1}` }))
        return { items: all.slice((q.page - 1) * q.per_page, q.page * q.per_page), total: rows.count }
      })
      return { fetch, calls, rows }
    }

    it('moves to the last page when the page asked is past it, as after deleting the last row of the last page', async () => {
      const { fetch, calls, rows } = shrinking(11)
      const { data, page } = setup({ fetch, perPage: 10 })
      await flush()
      page.value = 2
      await flush()
      expect(names(data.shown.value)).toEqual(['R11'])
      rows.count = 10
      await data.load()
      await flush()
      expect(page.value).toBe(1)
      expect(calls).toEqual([1, 2, 2, 1])
      expect(names(data.shown.value)).toHaveLength(10)
      expect(data.pageCount.value).toBe(1)
    })

    it('moves to the last page when the total shrinks by more than a page', async () => {
      const { fetch, calls, rows } = shrinking(35)
      const { data, page } = setup({ fetch, perPage: 10 })
      await flush()
      page.value = 4
      await flush()
      rows.count = 12
      await data.load()
      await flush()
      expect(page.value).toBe(2)
      expect(calls).toEqual([1, 4, 4, 2])
      expect(names(data.shown.value)).toEqual(['R11', 'R12'])
    })

    it('moves once, then shows the page empty, for a parent that keeps its page', async () => {
      const { fetch, calls, rows } = shrinking(11)
      const { data, page } = setup({ fetch: (q) => (calls.length >= 20 ? new Promise<never>(() => {}) : fetch(q)), perPage: 10, keepsPage: true })
      await flush()
      page.value = 2
      await flush()
      expect(names(data.shown.value)).toEqual(['R11'])
      rows.count = 10
      await data.load()
      for (let i = 0; i < 10; i++) await flush()
      expect(calls).toEqual([1, 2, 2, 1])
      expect(page.value).toBe(2)
      expect(data.shown.value).toEqual([])
      expect(data.total.value).toBe(10)
      expect(data.busy.value).toBe(false)
      // Asking again moves once more, and no more.
      await data.load()
      for (let i = 0; i < 10; i++) await flush()
      expect(calls).toEqual([1, 2, 2, 1, 2, 1])
      expect(data.shown.value).toEqual([])
    })

    it('does not loop when the server answers empty with a total on every page', async () => {
      const calls: number[] = []
      const fetch = async (q: TableQuery): Promise<TableResult<P>> => {
        calls.push(q.page)
        return { items: [], total: 25 }
      }
      const { page } = setup({ fetch, perPage: 10 })
      await flush()
      page.value = 5
      await flush()
      await flush()
      expect(page.value).toBe(3)
      expect(calls).toEqual([1, 5, 3])
    })

    it('does not move the page for an answer that is no longer the latest', async () => {
      const answers: ((r: TableResult<P>) => void)[] = []
      const fetch = (_q: TableQuery) => new Promise<TableResult<P>>((resolve) => answers.push(resolve))
      const { page } = setup({ fetch, perPage: 10 })
      await flush()
      page.value = 4
      await flush()
      answers[1]!({ items: [], total: 5 })
      answers[0]!({ items: [], total: 5 }) // the first page's, stale
      await flush()
      expect(page.value).toBe(1)
      page.value = 3
      await flush()
      page.value = 2
      await flush()
      answers.at(-2)!({ items: [], total: 5 }) // page 3's, stale
      await flush()
      expect(page.value).toBe(2)
    })

    it('warns once when rows to be chosen have no key of their own', async () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
      const fetch = async () => ({ items: [{ name: 'A' }, { name: 'B' }], total: 2 })
      const { data } = setup({ fetch, needKeys: true })
      await flush()
      expect(data.shown.value.map((e) => e.key)).toEqual(['1:0', '1:1'])
      void data.shown.value
      expect(warn).toHaveBeenCalledTimes(1)
      expect(String(warn.mock.calls[0]![0])).toContain('rowKey')
    })
  })
})
