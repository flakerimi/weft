import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import DataTableCases, { type Product, type ProductInput } from '../fixtures/data/DataTableCases.vue'
import { registerIcon } from '../../src/basics/icons.ts'
import type { EntitySchema } from '../../src/data/entity.ts'
import type { TableQuery, TableResult } from '../../src/data/table.ts'
import { mount } from '../mount.ts'
import { $, $$, overlays, reset, settle } from '../overlays/helpers.ts'

beforeAll(() => {
  for (const name of ['plus', 'pencil', 'trash']) registerIcon(name, '<path d="M0 0"/>')
})

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
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
  it('wires resource.list to Table, renders debounced search and filterDefs (with labels.anyFilter), and forwards cell slots', async () => {
    const { resource, listCalls } = makeResource()
    await cases({
      resource,
      searchDebounce: 0,
      filterDefs: [{ key: 'status', label: 'Status', items: ['draft', 'live'] }],
      customStatusCell: true,
    })
    expect(listCalls).toHaveLength(1)
    expect($$('.custom-status').map((el) => el.textContent)).toEqual(['LIVE', 'DRAFT'])

    // Filter radio options include Any ('') + Draft + Live
    const filterRadios = $$('.weft-radio__label').map((el) => el.textContent!.trim())
    expect(filterRadios).toEqual(['Any', 'Draft', 'Live'])

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

  it('opens ConfirmDialog from a row Delete button, shows an Alert when remove fails, and refreshes when it succeeds', async () => {
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
    delBtn.click()
    await settle()

    const alertDialog = () => $('[role="alertdialog"]', overlays()!)
    expect(alertDialog().textContent).toContain('Delete Brass Lamp?')

    const confirmBtn = () => $$('button', alertDialog()).find((b) => b.textContent!.trim() === 'Delete')!
    confirmBtn().click()
    await settle()

    // Failed: dialog stays open and shows the error sentence inside ConfirmDialog
    expect($('.weft-alert', alertDialog()).textContent).toContain('Product is in an open order.')

    // Retry and succeed
    shouldFail = false
    confirmBtn().click()
    await settle()
    expect(m.exposed.deletedRecords).toHaveLength(1)
    expect($('tbody').textContent).not.toContain('Brass Lamp')
  })

  it('omits New when create is absent, omits Edit/Delete when update/remove are absent, and omits Actions column when both are absent', async () => {
    const { resource } = makeResource()
    await cases({
      resource: { list: resource.list },
    })
    expect($$('button').find((b) => b.textContent!.trim() === 'New product')).toBeUndefined()
    expect($$('thead th').map((th) => th.textContent!.trim())).toEqual(['Name', 'Price', 'Status'])
    expect($$('tbody button')).toHaveLength(0)
  })
})
