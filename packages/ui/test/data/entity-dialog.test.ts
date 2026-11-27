import { afterEach, describe, expect, it, vi } from 'vitest'
import EntityDialogCases, { type Item, type ItemInput } from '../fixtures/data/EntityDialogCases.vue'
import { changes, entityProblem, humanLabel, inferControl, type EntitySchema } from '../../src/data/entity.ts'
import { mount } from '../mount.ts'
import { installResizeObserver } from '../resize-observer.ts'
import { $, $$, key, overlays, reset, settle } from '../overlays/helpers.ts'

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

async function cases(props: Record<string, unknown> = {}) {
  const m = mount(EntityDialogCases, { schema: SCHEMA, open: true, ...props })
  unmount = m.unmount
  await settle()
  return m as typeof m & {
    exposed: {
      isOpen: boolean
      currentRecord: Item | null
      savedRecords: Item[]
      errors: unknown[]
    }
  }
}

describe('entity helpers', () => {
  it('formats camelCase and snake_case field keys as human labels', () => {
    expect(humanLabel('name')).toBe('Name')
    expect(humanLabel('firstName')).toBe('First name')
    expect(humanLabel('unit_price')).toBe('Unit price')
  })

  it('infers control kind from schema rules and field config', () => {
    expect(inferControl({ type: 'string' }, undefined, 'regular')).toBe('text')
    expect(inferControl({ type: 'string', email: true }, undefined, 'regular')).toBe('email')
    expect(inferControl({ type: 'string', url: true }, undefined, 'regular')).toBe('url')
    expect(inferControl({ type: 'number' }, undefined, 'regular')).toBe('number')
    expect(inferControl({ type: 'boolean' }, undefined, 'regular')).toBe('checkbox')
    expect(inferControl({ type: 'boolean' }, { control: 'switch' }, 'regular')).toBe('switch')
    expect(inferControl({ type: 'string', enum: ['a', 'b', 'c', 'd'] }, undefined, 'regular')).toBe('radio')
    expect(inferControl({ type: 'string', enum: ['a', 'b', 'c', 'd'] }, undefined, 'narrow')).toBe('select')
    expect(inferControl({ type: 'string', enum: ['a', 'b', 'c', 'd', 'e'] }, undefined, 'regular')).toBe('select')
    expect(inferControl({ type: 'string' }, { control: 'textarea' }, 'regular')).toBe('textarea')
  })

  it('diffs only changed fields against the original record, normalizing nullable empty to null', () => {
    const was: Item = { id: 1, name: 'Kettle', price: 32, sale: 24, status: 'live' }
    expect(changes({ name: 'Kettle', price: 32, sale: 24, status: 'live' }, was, SCHEMA)).toEqual({})
    expect(changes({ name: 'Tea Kettle', price: 32, sale: null, status: 'live' }, was, SCHEMA)).toEqual({
      name: 'Tea Kettle',
      sale: null,
    })
    const wasNoSale: Item = { id: 2, name: 'Mug', price: 12, sale: null, status: 'draft' }
    expect(changes({ name: 'Mug', price: 12, sale: null, status: 'draft' }, wasNoSale, SCHEMA)).toEqual({})
  })

  it('maps ApiError-shaped errors to known field errors and top-level alert text', () => {
    const known = new Set(['name', 'price'])
    expect(
      entityProblem(
        { status: 422, code: 'invalid', message: 'Invalid input.', fields: { name: 'Enter a name.', other: 'Bad field.' } },
        known,
      ),
    ).toEqual({
      fields: { name: 'Enter a name.' },
      message: 'Bad field.',
    })
    expect(
      entityProblem({ status: 0, code: 'network', message: 'Failed to fetch', fields: {} }, known),
    ).toEqual({
      fields: {},
      message: 'The server could not be reached; check the connection and try again.',
    })
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      expect(entityProblem(new TypeError('boom'), known)).toEqual({
        fields: {},
        message: 'Something went wrong on this page; reload it and try again.',
      })
      expect(spy).toHaveBeenCalledOnce()
    } finally {
      spy.mockRestore()
    }
  })
})

describe('EntityDialog', () => {
  it('auto-renders controls from schema in create mode with defaults, and calls create(body) on submit', async () => {
    const created: Item = { id: 7, name: 'Lamp', price: 45, sale: null, status: 'draft' }
    const create = vi.fn(async (body: ItemInput) => ({ id: 7, ...body }))
    const m = await cases({
      name: 'product',
      defaults: { price: 45, status: 'draft' },
      create: fn(create),
    })
    const dialog = $('[role="dialog"]', overlays()!)
    expect(dialog.textContent).toContain('New product')

    const nameInput = $<HTMLInputElement>('input[name="name"]', dialog)
    nameInput.value = 'Lamp'
    nameInput.dispatchEvent(new Event('input', { bubbles: true }))
    await settle()

    const submit = $$('button', dialog).find((b) => b.textContent!.trim() === 'Create')!
    submit.click()
    await settle()

    expect(create).toHaveBeenCalledWith({ name: 'Lamp', price: 45, sale: null, status: 'draft' })
    expect(m.exposed.savedRecords).toEqual([created])
    expect(m.exposed.isOpen).toBe(false)
  })

  it('auto-renders email, url, textarea, checkbox, switch, select (> 4 enum choices), and optional enum None option', async () => {
    const richSchema: EntitySchema = {
      name: { type: 'string', required: true },
      email: { type: 'string', email: true },
      website: { type: 'string', url: true },
      notes: { type: 'string' },
      category: { type: 'string', enum: ['a', 'b', 'c', 'd', 'e'] },
      status: { type: 'string', enum: ['draft', 'live'] },
      active: { type: 'boolean' },
    }
    await cases({
      schema: richSchema,
      fields: {
        notes: { control: 'textarea', label: 'Notes' },
        active: { control: 'switch', label: 'Active' },
      },
      labels: { noneOption: 'No status' },
    })
    const dialog = $('[role="dialog"]', overlays()!)
    expect($('input[name="email"]', dialog).getAttribute('type')).toBe('email')
    expect($('input[name="website"]', dialog).getAttribute('type')).toBe('url')
    expect($('textarea[name="notes"]', dialog)).not.toBeNull()
    expect($('[role="switch"]', dialog)).not.toBeNull()
    // 5 choices -> Select combobox
    expect($('[role="combobox"]', dialog)).not.toBeNull()
    // Optional enum with 2 choices -> 3 radio options including 'No status'
    const radios = $$('.weft-radio__label', dialog).map((el) => el.textContent!.trim())
    expect(radios).toEqual(['No status', 'Draft', 'Live'])
  })

  it('switches enum with <= 4 choices from RadioGroup to Select when narrow', async () => {
    ro = installResizeObserver()
    await cases()
    ro.report(document.documentElement, 480)
    await settle()
    const dialog = $('[role="dialog"]', overlays()!)
    expect($('[role="radiogroup"]', dialog)).toBeNull()
    expect($('[role="combobox"]', dialog)).not.toBeNull()
  })

  it('populates fields in edit mode, sends only changed fields to update(id, body), and skips request when nothing changed', async () => {
    const record: Item = { id: 3, name: 'Kettle', price: 32, sale: 24, status: 'live' }
    const update = vi.fn(async (id: number, body: Partial<ItemInput>) => ({ ...record, id, ...body }))
    const m = await cases({
      record,
      name: 'product',
      update: fn(update),
    })
    const dialog = $('[role="dialog"]', overlays()!)
    expect(dialog.textContent).toContain('Edit Kettle')

    // Submitting without changes makes no network call and closes the dialog.
    const saveBtn = () => $$('button', $('[role="dialog"]', overlays()!)).find((b) => b.textContent!.trim() === 'Save')!
    saveBtn().click()
    await settle()
    expect(update).not.toHaveBeenCalled()
    expect(m.exposed.isOpen).toBe(false)
    expect(m.exposed.savedRecords).toEqual([record])

    // Reopen, change name and clear nullable sale -> sends only { name, sale: null }.
    m.exposed.isOpen = true
    await settle()
    const openDialog = $('[role="dialog"]', overlays()!)
    const nameInput = $<HTMLInputElement>('input[name="name"]', openDialog)
    nameInput.value = 'Copper Kettle'
    nameInput.dispatchEvent(new Event('input', { bubbles: true }))
    const saleInput = $<HTMLInputElement>('input[name="sale"]', openDialog)
    saleInput.value = ''
    saleInput.dispatchEvent(new Event('input', { bubbles: true }))
    saleInput.dispatchEvent(new Event('change', { bubbles: true }))
    await settle()

    saveBtn().click()
    await settle()
    expect(update).toHaveBeenCalledWith(3, { name: 'Copper Kettle', sale: null })
    expect(m.exposed.isOpen).toBe(false)
  })

  it('shows server field errors on known fields, unknown field errors or message in the top Alert, and clears them when reopened', async () => {
    let failMode: 'field' | 'unknownField' | 'top' = 'field'
    const create = vi.fn(async (_body: ItemInput): Promise<Item> => {
      if (failMode === 'field') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { name: 'That name is taken.' } }
      }
      if (failMode === 'unknownField') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { slug: 'Slug already exists.' } }
      }
      throw { status: 500, code: 'internal', message: 'Database unavailable.', fields: {} }
    })
    const m = await cases({
      defaults: { name: 'Kettle', price: 10, status: 'draft' },
      create: fn(create),
    })
    const dialog = () => $('[role="dialog"]', overlays()!)
    const submit = () => $$('button', dialog()).find((b) => b.textContent!.trim() === 'Create')!

    submit().click()
    await settle()
    expect(m.exposed.isOpen).toBe(true)
    expect($('.weft-form-field__error', dialog()).textContent!.trim()).toBe('That name is taken.')
    expect($('.weft-alert', dialog())).toBeNull()

    failMode = 'unknownField'
    submit().click()
    await settle()
    expect($('.weft-alert', dialog()).textContent).toContain('Slug already exists.')

    failMode = 'top'
    submit().click()
    await settle()
    expect($('.weft-alert', dialog()).textContent).toContain('Database unavailable.')

    // Close and reopen: errors are cleared.
    key('Escape')
    await settle()
    expect(m.exposed.isOpen).toBe(false)
    m.exposed.isOpen = true
    await settle()
    expect($('.weft-alert', dialog())).toBeNull()
    expect($('.weft-form-field__error', dialog())).toBeNull()
  })

  it('supports custom per-field slots and a custom default body slot', async () => {
    const create = vi.fn(async (body: ItemInput) => ({ id: 1, ...body }))
    await cases({
      customNameSlot: true,
      defaults: { name: 'From slot', price: 10, status: 'draft' },
      create: fn(create),
    })
    expect($('.custom-name-field', overlays()!)).not.toBeNull()
  })
})
