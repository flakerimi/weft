import { afterEach, describe, expect, it, vi } from 'vitest'
import EntityDialogCases, { type Item, type ItemInput } from '../fixtures/data/EntityDialogCases.vue'
import { changes, entityProblem, humanLabel, inferControl, initialDraft, type EntityFieldConfig, type EntityFieldRule, type EntitySchema } from '../../src/data/entity.ts'
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
      slotPropsSeen: { record: Item | null; rule: EntityFieldRule; field: EntityFieldConfig } | null
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

  it('initializes non-scalar list, record and object fields to [] and {}, and supports a defaults factory function', () => {
    const schema: EntitySchema = {
      name: { kind: 'string', required: true },
      tags: { kind: 'list' },
      meta: { kind: 'record' },
      extra: { kind: 'object' },
    }
    expect(initialDraft(schema, null, () => ({ name: 'From factory' }))).toEqual({
      name: 'From factory',
      tags: [],
      meta: {},
      extra: {},
    })
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

  it('maps ApiError-shaped errors to known field errors and top-level alert text, distinguishing code==="network" from code==="http"', () => {
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
    expect(
      entityProblem({ status: 502, code: 'http', message: 'Bad Gateway', fields: {} }, known),
    ).toEqual({
      fields: {},
      message: 'The server could not answer right now; try again in a moment.',
    })
    expect(
      entityProblem({ status: 200, code: 'http', message: 'The answer was not JSON.', fields: {} }, known),
    ).toEqual({
      fields: {},
      message: 'The server could not answer right now; try again in a moment.',
    })
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      expect(entityProblem(new TypeError('boom'), known)).toEqual({
        fields: {},
        message: 'Something went wrong on this page; reload it and try again.',
      })
      expect(spy).toHaveBeenCalledWith('The page failed:', expect.any(TypeError))
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
      defaults: fn(() => ({ price: 45, status: 'draft' })),
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

  it('auto-renders email, url, textarea, checkbox, switch, select (> 4 enum choices), optional enum None option, and skips unslotted non-scalar fields', async () => {
    const richSchema: EntitySchema = {
      name: { type: 'string', required: true },
      email: { type: 'string', email: true },
      website: { type: 'string', url: true },
      notes: { type: 'string' },
      category: { type: 'string', enum: ['a', 'b', 'c', 'd', 'e'] },
      status: { type: 'string', enum: ['draft', 'live'] },
      active: { type: 'boolean' },
      count: { type: 'number' },
      tags: { type: 'list' },
    }
    await cases({
      schema: richSchema,
      fields: [
        { name: 'name' },
        { name: 'email' },
        { name: 'website' },
        { name: 'notes', control: 'textarea', label: 'Notes' },
        { name: 'category' },
        { name: 'status' },
        { name: 'active', control: 'switch', label: 'Active' },
        { name: 'count' },
        { name: 'tags' },
      ],
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
    // Optional number in create mode does not get the required asterisk
    const countField = $$('.weft-form-field', dialog).find((el) => el.textContent!.includes('Count'))!
    expect(countField.querySelector('.weft-form-field__required')).toBeNull()
    // Non-scalar list field without a slot is not rendered as a text input
    expect(dialog.querySelector('input[name="tags"]')).toBeNull()
  })

  it('switches enum with <= 4 choices from RadioGroup to Select when narrow, keeping the Select combobox labelled', async () => {
    ro = installResizeObserver()
    await cases()
    ro.report(document.documentElement, 480)
    await settle()
    const dialog = $('[role="dialog"]', overlays()!)
    expect($('[role="radiogroup"]', dialog)).toBeNull()
    const combo = $('[role="combobox"]', dialog)
    expect(combo).not.toBeNull()
    const label = $$('label', dialog).find((l) => l.textContent!.includes('Status'))!
    expect(label.getAttribute('for')).toBe(combo.id)

    // Widening and narrowing again while open keeps the combobox labelled
    ro.report(document.documentElement, 1200)
    await settle()
    expect($('[role="radiogroup"]', dialog)).not.toBeNull()
    ro.report(document.documentElement, 400)
    await settle()
    const comboAfter = $('[role="combobox"]', dialog)
    const labelAfter = $$('label', dialog).find((l) => l.textContent!.includes('Status'))!
    expect(labelAfter.getAttribute('for')).toBe(comboAfter.id)
  })

  it('populates fields in edit mode, falls back edit title to title then resource name, sends only changed fields, and closes without emitting saved when nothing changed', async () => {
    const record: Item = { id: 3, name: 'Kettle', price: 32, sale: 24, status: 'live' }
    const update = vi.fn(async (id: number, body: Partial<ItemInput>) => ({ ...record, id, ...body }))
    const m = await cases({
      record,
      name: 'product',
      update: fn(update),
    })
    const dialog = $('[role="dialog"]', overlays()!)
    expect(dialog.textContent).toContain('Edit Kettle')

    // Submitting without changes makes no network call, closes the dialog, and does NOT emit saved.
    const saveBtn = () => $$('button', $('[role="dialog"]', overlays()!)).find((b) => b.textContent!.trim() === 'Save')!
    saveBtn().click()
    await settle()
    expect(update).not.toHaveBeenCalled()
    expect(m.exposed.isOpen).toBe(false)
    expect(m.exposed.savedRecords).toEqual([])

    // Edit title fallback: empty name falls back to title, then to resource name (not id)
    m.exposed.currentRecord = { id: 9, name: '', title: 'Draft Title', price: 10, sale: null, status: 'draft' }
    m.exposed.isOpen = true
    await settle()
    expect($('[role="dialog"]', overlays()!).textContent).toContain('Edit Draft Title')
    m.exposed.currentRecord = { id: 9, name: '', price: 10, sale: null, status: 'draft' }
    await settle()
    expect($('[role="dialog"]', overlays()!).textContent).toContain('Edit product')

    // Restore Kettle, change name and clear nullable sale -> sends only { name, sale: null } and emits saved.
    m.exposed.currentRecord = record
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
    expect(m.exposed.savedRecords).toHaveLength(1)
  })

  it('shows server field errors on known fields (and focuses the first invalid control after busy ends), unknown, hidden or omitted field errors in the top Alert with role="alert", and clears them when reopened', async () => {
    let failMode: 'field' | 'unknownField' | 'hiddenField' | 'omittedField' | 'top' = 'field'
    const create = vi.fn(async (_body: ItemInput): Promise<Item> => {
      if (failMode === 'field') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { name: 'That name is taken.' } }
      }
      if (failMode === 'unknownField') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { slug: 'Slug already exists.' } }
      }
      if (failMode === 'hiddenField') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { notes: 'Hidden note error.' } }
      }
      if (failMode === 'omittedField') {
        throw { status: 422, code: 'invalid', message: 'Invalid input.', fields: { internalCode: 'Omitted code error.' } }
      }
      throw { status: 500, code: 'internal', message: 'Database unavailable.', fields: {} }
    })
    const m = await cases({
      schema: {
        ...SCHEMA,
        notes: { type: 'string', required: true },
        internalCode: { type: 'string', required: true },
      },
      // Only listed fields render in that order; hidden or omitted required schema fields do not block submit, and their server errors go to the top Alert.
      fields: [
        { name: 'name' },
        { name: 'price' },
        { name: 'sale' },
        { name: 'status' },
        { name: 'notes', hidden: true },
      ],
      defaults: { name: 'Kettle', price: 10, status: 'draft' },
      create: fn(create),
    })
    const dialog = () => $('[role="dialog"]', overlays()!)
    expect($$('.weft-form-field__label', dialog()).map((l) => l.textContent!.replace('*', '').trim())).toEqual([
      'Name',
      'Price',
      'Sale',
      'Status',
    ])
    expect(dialog().querySelector('input[name="internalCode"]')).toBeNull()
    expect(dialog().querySelector('input[name="notes"]')).toBeNull()
    const submit = () => $$('button', dialog()).find((b) => b.textContent!.trim() === 'Create')!

    submit().focus()
    submit().click()
    await settle()
    expect(m.exposed.isOpen).toBe(true)
    expect($('.weft-form-field__error', dialog()).textContent!.trim()).toBe('That name is taken.')
    expect(document.activeElement).toBe($('input[name="name"]', dialog()))
    expect($('.weft-alert', dialog())).toBeNull()

    failMode = 'unknownField'
    submit().click()
    await settle()
    const alert1 = $('.weft-alert', dialog())
    expect(alert1.getAttribute('role')).toBe('alert')
    expect(alert1.textContent).toContain('Slug already exists.')

    failMode = 'hiddenField'
    submit().click()
    await settle()
    expect($('.weft-alert', dialog()).textContent).toContain('Hidden note error.')

    failMode = 'omittedField'
    submit().click()
    await settle()
    expect($('.weft-alert', dialog()).textContent).toContain('Omitted code error.')

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

  it('supports custom per-field slots with { state, record, rule, field } and a custom default body slot', async () => {
    const record: Item = { id: 5, name: 'From slot', price: 10, sale: null, status: 'draft' }
    const create = vi.fn(async (body: ItemInput) => ({ id: 1, ...body }))
    const m = await cases({
      record,
      customNameSlot: true,
      fields: [{ name: 'name', label: 'Product Name' }],
      create: fn(create),
    })
    expect($('.custom-name-field', overlays()!)).not.toBeNull()
    expect(m.exposed.slotPropsSeen).toEqual({
      record,
      rule: SCHEMA.name,
      field: { name: 'name', label: 'Product Name' },
    })
  })
})
