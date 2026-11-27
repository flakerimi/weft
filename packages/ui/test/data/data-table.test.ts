import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import DataTableCases, { type Product, type ProductInput } from '../fixtures/data/DataTableCases.vue'
import { registerIcon } from '../../src/basics/icons.ts'
import type { EntitySchema } from '../../src/data/entity.ts'
import type { TableQuery, TableResult } from '../../src/data/table.ts'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { $, $$, overlays, reset, settle } from '../overlays/helpers.ts'

beforeAll(() => {
  for (const name of ['plus', 'pencil', 'trash']) registerIcon(name, '<path d="M0 0"/>')
})

let unmount: (() => void) | undefined
let ro: ReturnType<typeof installResizeObserver> | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  ro?.restore()
  ro = undefined
  reset()
})

const fn = <F,>(f: F) => () => f

const SCHEMA: EntitySchema = {
  name: { type: 'string', required: true },
  price: { type: 'number', required: true, min: 0 },
  sale: { type: 'number', min: 0, nullable: true },
  status: { type: 'string', required: true, enum: ['draft', 'live'] },
}

const PRODUCTS: Product[] = [
  { id: 1, name: 'Brass Lamp', price: 48, sale: 36, status: 'live' },
  { id: 2, name: 'Cast Kettle', price: 32, sale: null, status: 'draft' },
]

function makeResource(initial: Product[] = PRODUCTS) {
  let rows = initial.map((r) => ({ ...r }))
  let nextId = rows.length + 1
  const listCalls: TableQuery[] = []
  const list = vi.fn(async (q: TableQuery): Promise<TableResult<Product>> => {
    listCalls.push(q)
    let out = [...rows]
    if (q.q) {
      const term = q.q.toLowerCase()
      out = out.filter((r) => r.name.toLowerCase().includes(term))
    }
    if (q.filter?.status) {
      out = out.filter((r) => r.status === q.filter!.status)
    }
    return { items: out, total: out.length }
  })
  const create = vi.fn(async (body: ProductInput): Promise<Product> => {
    const item: Product = { id: nextId++, ...body }
    rows.push(item)
    return item
  })
  const update = vi.fn(async (id: number, body: Partial<ProductInput>): Promise<Product> => {
    const idx = rows.findIndex((r) => r.id === id)
    rows[idx] = { ...rows[idx]!, ...body }
    return rows[idx]!
  })
  const remove = vi.fn(async (id: number): Promise<void> => {
    rows = rows.filter((r) => r.id !== id)
  })
  return { resource: { list, create, update, remove }, listCalls, list, create, update, remove }
}

async function cases(props: Record<string, unknown> = {}) {
  const m = mount(DataTableCases, { schema: SCHEMA, ...props })
  unmount = m.unmount
  await settle()
  return m
}

describe('DataTable', () => {
  it('wires resource.list to Table, renders debounced search and filterDefs (with labels.anyFilter and allLabel), and forwards cell slots', async () => {
    const { resource, listCalls } = makeResource()
    await cases({
      resource,
      searchDebounce: 0,
      filterDefs: [
        { key: 'status', label: 'Status', items: ['draft', 'live'], allLabel: 'All statuses' },
        { key: 'mode', label: 'Mode', items: ['auto', 'manual'], allLabel: false },
      ],
      customStatusCell: true,
    })
    expect(listCalls).toHaveLength(1)
    expect($$('.custom-status').map((el) => el.textContent)).toEqual(['LIVE', 'DRAFT'])

    // Filter radio options include 'All statuses' ('') + Draft + Live, while allLabel: false adds none
    const filterRadios = $$('.weft-radio__label').map((el) => el.textContent!.trim())
    expect(filterRadios).toEqual(['All statuses', 'Draft', 'Live', 'Auto', 'Manual'])

    // Choosing 'live' filters the table
    const liveInput = $$<HTMLInputElement>('.weft-radio__input')[2]!
    liveInput.click()
    liveInput.dispatchEvent(new Event('change', { bubbles: true }))
    await settle()
    expect(listCalls.at(-1)?.filter).toEqual({ status: 'live' })
    expect($$('.custom-status').map((el) => el.textContent)).toEqual(['LIVE'])

    // Typing in search queries with q
    const searchInput = $<HTMLInputElement>('input[type="search"]')
    searchInput.value = 'Brass'
    searchInput.dispatchEvent(new Event('input', { bubbles: true }))
    await settle()
    expect(listCalls.at(-1)?.q).toBe('Brass')
  })

  it('switches toolbar filter from RadioGroup to a labelled Select combobox when narrow', async () => {
    ro = installResizeObserver()
    const { resource } = makeResource()
    const m = await cases({
      resource,
      filterDefs: [{ key: 'status', label: 'Status', items: ['draft', 'live'] }],
    })
    const dtRoot = m.el.querySelector<HTMLElement>('.weft-data-table')!
    ro.report(dtRoot, 375)
    await settle()
    expect(dtRoot.querySelector('[role="radiogroup"]')).toBeNull()
    const combo = dtRoot.querySelector<HTMLElement>('[role="combobox"]')!
    expect(combo).not.toBeNull()
    const statusLabel = [...dtRoot.querySelectorAll('label')].find((l) => l.textContent!.trim() === 'Status')!
    expect(statusLabel.getAttribute('for')).toBe(combo.id)
    // The empty filter is the Any choice, which the Select shows, not a blank.
    expect(combo.textContent!.trim()).toBe('Any')
  })

  it('shows the all choice in a Select filter, labels.anyFilter or allLabel, at any width', async () => {
    const { resource } = makeResource()
    const m = await cases({
      resource,
      filterDefs: [
        { key: 'kind', label: 'Kind', items: ['a', 'b', 'c', 'd', 'e'] },
        { key: 'size', label: 'Size', items: ['s', 'm'], control: 'select', allLabel: 'Every size' },
      ],
      labels: { anyFilter: 'Alle' },
    })
    const combos = [...m.el.querySelectorAll<HTMLElement>('.weft-data-table [role="combobox"]')]
    expect(combos.map((c) => c.textContent!.trim())).toEqual(['Alle', 'Every size'])
  })

  it('gives row Edit and Delete buttons aria-labels with the item name, and uses md size on touch input', async () => {
    const { resource } = makeResource()
    await cases({ resource, input: 'touch' })
    const editBtns = $$<HTMLButtonElement>('button[data-weft-row-action="edit"]')
    const delBtns = $$<HTMLButtonElement>('button[data-weft-row-action="delete"]')
    expect(editBtns.map((b) => b.getAttribute('aria-label'))).toEqual(['Edit Brass Lamp', 'Edit Cast Kettle'])
    expect(delBtns.map((b) => b.getAttribute('aria-label'))).toEqual(['Delete Brass Lamp', 'Delete Cast Kettle'])
    expect(editBtns[0]!.dataset.size).toBe('md')
    expect(delBtns[0]!.dataset.size).toBe('md')
  })

  it('opens EntityDialog in create mode from the New button, calls create, and refreshes the table', async () => {
    const { resource, create, listCalls } = makeResource()
    const m = await cases({
      resource,
      defaults: { price: 15, status: 'draft' },
    })
    const newBtn = $$('button').find((b) => b.textContent!.trim() === 'New product')!
    newBtn.click()
    await settle()

    const dialog = $('[role="dialog"]', overlays()!)
    expect(dialog.textContent).toContain('New product')
    const nameInput = $<HTMLInputElement>('input[name="name"]', dialog)
    nameInput.value = 'Linen Towel'
    nameInput.dispatchEvent(new Event('input', { bubbles: true }))
    await settle()

    const submit = $$('button', dialog).find((b) => b.textContent!.trim() === 'Create')!
    submit.click()
    await settle()

    expect(create).toHaveBeenCalledWith({ name: 'Linen Towel', price: 15, sale: null, status: 'draft' })
    expect(m.exposed.savedRecords).toHaveLength(1)
    expect(listCalls.length).toBeGreaterThanOrEqual(2)
    expect($('tbody').textContent).toContain('Linen Towel')
  })

  it('opens EntityDialog in edit mode from a row Edit button, sends only changed fields, and refreshes', async () => {
    const { resource, update } = makeResource()
    await cases({ resource })
    const editBtn = $$('tbody button').find((b) => b.textContent!.trim() === 'Edit')!
    editBtn.click()
    await settle()

    const dialog = $('[role="dialog"]', overlays()!)
    expect(dialog.textContent).toContain('Edit Brass Lamp')
    const nameInput = $<HTMLInputElement>('input[name="name"]', dialog)
    nameInput.value = 'Polished Brass Lamp'
    nameInput.dispatchEvent(new Event('input', { bubbles: true }))
    await settle()

    const saveBtn = $$('button', dialog).find((b) => b.textContent!.trim() === 'Save')!
    saveBtn.click()
    await settle()

    expect(update).toHaveBeenCalledWith(1, { name: 'Polished Brass Lamp' })
    expect($('tbody').textContent).toContain('Polished Brass Lamp')
  })

  it('opens ConfirmDialog from a row Delete button, shows an Alert (role="alert") when remove fails, and restores focus to the next row Delete button (or New button) when it succeeds', async () => {
    const { resource } = makeResource()
    let shouldFail = true
    const remove = vi.fn(async (id: number) => {
      if (shouldFail) throw { status: 409, code: 'conflict', message: 'Product is in an open order.', fields: {} }
      return resource.remove(id)
    })
    const m = await cases({
      resource: { ...resource, remove },
    })
    const delBtn = $$('tbody button').find((b) => b.textContent!.trim() === 'Delete')!
    delBtn.focus()
    delBtn.click()
    await settle()

    const alertDialog = () => $('[role="alertdialog"]', overlays()!)
    expect(alertDialog().textContent).toContain('Delete Brass Lamp?')

    const confirmBtn = () => $$('button', alertDialog()).find((b) => b.textContent!.trim() === 'Delete')!
    confirmBtn().click()
    await settle()

    // Failed: dialog stays open and shows the error sentence with role="alert" inside ConfirmDialog
    const alert = $('.weft-alert', alertDialog())
    expect(alert.getAttribute('role')).toBe('alert')
    expect(alert.textContent).toContain('Product is in an open order.')

    // Retry and succeed (focus the confirm button as a real browser click does)
    shouldFail = false
    confirmBtn().focus()
    confirmBtn().click()
    await settle()
    await new Promise((r) => setTimeout(r, 20))
    await settle()
    expect(m.exposed.deletedRecords).toHaveLength(1)
    expect($('tbody').textContent).not.toContain('Brass Lamp')
    // Focus moved to the remaining row's Delete button, not body
    const remainingDelBtn = $('button[data-weft-row-action="delete"]')
    expect(document.activeElement).toBe(remainingDelBtn)

    // Delete the last remaining row: focus moves to the "New product" button
    remainingDelBtn.focus()
    remainingDelBtn.click()
    await settle()
    confirmBtn().focus()
    confirmBtn().click()
    await settle()
    await new Promise((r) => setTimeout(r, 20))
    await settle()
    expect(document.activeElement).toBe($$('button').find((b) => b.textContent!.trim() === 'New product'))
  })

  it('moves focus to a row Delete button when deleting the last row of page 2 triggers a fallback fetch of page 1', async () => {
    const items: Product[] = Array.from({ length: 11 }, (_, i) => ({
      id: i + 1,
      name: `P${String(i + 1).padStart(2, '0')}`,
      price: i + 1,
      sale: null,
      status: 'live',
    }))
    const resource = {
      list: vi.fn(async (q: { page: number; per_page: number }) => {
        await new Promise((r) => setTimeout(r, 5))
        return {
          items: items.slice((q.page - 1) * q.per_page, q.page * q.per_page),
          total: items.length,
        }
      }),
      create: vi.fn(async (b: ProductInput) => ({ id: 99, ...b })),
      update: vi.fn(async (id: number, b: Partial<ProductInput>) => ({ ...items[0]!, id, ...b })),
      remove: vi.fn(async (id: number) => {
        const idx = items.findIndex((x) => x.id === id)
        if (idx >= 0) items.splice(idx, 1)
      }),
    }
    await cases({ resource, perPage: 10 })
    await new Promise((r) => setTimeout(r, 20))
    await settle()

    // Go to page 2 (only P11)
    const page2Btn = $<HTMLElement>('.weft-table [aria-label="Page 2"]')
    page2Btn.click()
    await new Promise((r) => setTimeout(r, 20))
    await settle()
    expect($('tbody').textContent).toContain('P11')

    const delBtn = $<HTMLElement>('button[data-weft-row-action="delete"]')
    delBtn.focus()
    delBtn.click()
    await settle()
    const confirmBtn = $$('button', $('[role="alertdialog"]', overlays()!)).find((b) => b.textContent!.trim() === 'Delete')!
    confirmBtn.focus()
    confirmBtn.click()
    await new Promise((r) => setTimeout(r, 40))
    await settle()

    expect($('tbody').textContent).toContain('P01')
    expect(document.activeElement?.getAttribute('data-weft-row-action')).toBe('delete')
  })

  it('omits New and Edit when schema (and #dialog) is absent or when create/update are absent, and omits Actions column when both update and remove are absent', async () => {
    const { resource } = makeResource()
    await cases({
      schema: undefined,
      resource: { list: resource.list, create: resource.create, update: resource.update },
    })
    expect($$('button').find((b) => b.textContent!.trim() === 'New product')).toBeUndefined()
    expect($$('thead th').map((th) => th.textContent!.trim())).toEqual(['Name', 'Price', 'Status'])
    expect($$('tbody button')).toHaveLength(0)
  })
})
