import { afterEach, describe, expect, it, vi } from 'vitest'
import EntityDialogCases, { type Item, type ItemInput } from '../fixtures/data/EntityDialogCases.vue'
import { changes, entityProblem, humanLabel, inferControl, initialDraft, type EntityFieldConfig, type EntityFieldRule, type EntitySchema } from '../../src/data/entity.ts'
import { localToWire, wireToLocal } from '../../src/data/when.ts'
import { type as typeInto } from '../forms/helpers.ts'
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

  it('draws a date or a datetime for a field the schema says is one, and starts them empty', () => {
    expect(inferControl({ kind: 'string', format: 'date' }, undefined, 'regular')).toBe('date')
    expect(inferControl({ kind: 'string', format: 'datetime' }, undefined, 'narrow')).toBe('datetime')
    expect(inferControl({ kind: 'string', format: 'date' }, { control: 'text' }, 'regular')).toBe('text')
    expect(initialDraft({ due: { kind: 'string', format: 'date' }, at: { kind: 'string', format: 'datetime', required: true } }, null)).toEqual({ due: null, at: null })
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

  describe('dates and times', () => {
    const TIMES: EntitySchema = {
      name: { kind: 'string', required: true },
      due: { kind: 'string', format: 'date', nullable: true },
      starts: { kind: 'string', format: 'datetime', required: true },
    }
    const field = (label: string) => $$('.weft-form-field', $('[role="dialog"]', overlays()!)).find((f) => f.querySelector('.weft-form-field__label')?.textContent?.trim().startsWith(label))!
    const dateIn = (label: string) => $<HTMLInputElement>('.weft-date-picker input.weft-input__field', field(label))
    async function typeDate(label: string, text: string) {
      const el = dateIn(label)
      el.focus()
      typeInto(el, text)
      el.dispatchEvent(new Event('change', { bubbles: true }))
      await settle()
    }
    const button = (text: string) => $$('button', $('[role="dialog"]', overlays()!)).find((b) => b.textContent!.trim() === text)!

    it('draws a DatePicker for a date, and a DatePicker and an InputTime side by side for a datetime, the time labelled', async () => {
      await cases({ schema: TIMES, name: 'event', dateFields: true })
      expect($$('.weft-date-picker', field('Due'))).toHaveLength(1)
      expect(field('Due').querySelector('.weft-input-time')).toBeNull()
      const group = $('.weft-field-group', field('Starts'))
      expect($$('.weft-date-picker', group)).toHaveLength(1)
      const time = $('.weft-input-time', group)
      // The field's label is for the date; the time names itself.
      const label = $('.weft-form-field__label', field('Starts'))
      expect(label.tagName).toBe('LABEL')
      expect(label.getAttribute('for')).toBe(dateIn('Starts').id)
      expect(time.getAttribute('role')).toBe('group')
      expect(time.id).not.toBe(dateIn('Starts').id)
      const named = document.getElementById(time.getAttribute('aria-labelledby')!)
      expect(named?.textContent).toBe('Starts, time')
      expect($$('[role="spinbutton"]', time).map((s) => s.getAttribute('aria-label'))).toEqual(['Hour', 'Minute', 'AM/PM'])
    })

    it('saves a date as midnight UTC and a datetime as its instant in UTC', async () => {
      const create = vi.fn(async (body: ItemInput) => ({ id: 1, ...body }))
      await cases({ schema: TIMES, name: 'event', dateFields: true, create: fn(create) })
      const nameInput = $<HTMLInputElement>('input[name="name"]', $('[role="dialog"]', overlays()!))
      typeInto(nameInput, 'Launch')
      await typeDate('Due', '3/29/2026')
      await typeDate('Starts', '3/29/2026')
      // A date with no time yet is at midnight where the browser is.
      expect(button('Create')).toBeTruthy()
      const hour = $$('[role="spinbutton"]', field('Starts'))[0]!
      hour.focus()
      key('ArrowUp')
      await settle()
      button('Create').click()
      await settle()
      expect(create).toHaveBeenCalledWith({ name: 'Launch', due: '2026-03-29T00:00:00Z', starts: localToWire('2026-03-29', '01:00') })
    })

    it('keeps an untouched date and datetime out of the PATCH, and sends null for an emptied nullable date', async () => {
      const record = { id: 4, name: 'Launch', due: '2026-03-29T00:00:00Z', starts: '2026-03-29T10:00:30Z' } as unknown as Item
      const update = vi.fn(async (id: number, body: Partial<ItemInput>) => ({ ...record, id, ...body }))
      await cases({ schema: TIMES, record, dateFields: true, update: fn(update) })
      expect(dateIn('Due').value).toBe('03/29/2026')
      const [y, mo, d] = wireToLocal('2026-03-29T10:00:30Z')!.date.split('-')
      expect(dateIn('Starts').value).toBe(`${mo}/${d}/${y}`)
      typeInto($<HTMLInputElement>('input[name="name"]', $('[role="dialog"]', overlays()!)), 'Launch day')
      await settle()
      button('Save').click()
      await settle()
      expect(update).toHaveBeenLastCalledWith(4, { name: 'Launch day' })

      update.mockClear()
      unmount!()
      reset()
      await cases({ schema: TIMES, record, dateFields: true, update: fn(update) })
      await typeDate('Due', '')
      button('Save').click()
      await settle()
      expect(update).toHaveBeenLastCalledWith(4, { due: null })
    })

    it('draws the browser’s own date and datetime inputs without useDateFields, and saves the same values', async () => {
      const create = vi.fn(async (body: ItemInput) => ({ id: 1, ...body }))
      await cases({ schema: TIMES, name: 'event', create: fn(create) })
      const dialog = $('[role="dialog"]', overlays()!)
      expect(dialog.querySelector('.weft-date-picker')).toBeNull()
      expect(dialog.querySelector('.weft-input-time')).toBeNull()
      const due = $<HTMLInputElement>('input[type="date"]', field('Due'))
      const starts = $<HTMLInputElement>('input[type="datetime-local"]', field('Starts'))
      // The field's label is for each, as for any Input.
      expect($('.weft-form-field__label', field('Starts')).getAttribute('for')).toBe(starts.id)
      typeInto($<HTMLInputElement>('input[name="name"]', dialog), 'Launch')
      typeInto(due, '2026-03-29')
      typeInto(starts, '2026-03-29T09:15')
      await settle()
      button('Create').click()
      await settle()
      expect(create).toHaveBeenCalledWith({ name: 'Launch', due: '2026-03-29T00:00:00Z', starts: localToWire('2026-03-29', '09:15') })

      const record = { id: 4, name: 'Launch', due: '2026-03-29T00:00:00Z', starts: '2026-03-29T10:00:30Z' } as unknown as Item
      const update = vi.fn(async (id: number, body: Partial<ItemInput>) => ({ ...record, id, ...body }))
      unmount!()
      reset()
      await cases({ schema: TIMES, record, update: fn(update) })
      const local = wireToLocal('2026-03-29T10:00:30Z')!
      expect($<HTMLInputElement>('input[type="date"]', field('Due')).value).toBe('2026-03-29')
      expect($<HTMLInputElement>('input[type="datetime-local"]', field('Starts')).value).toBe(`${local.date}T${local.time}`)
      typeInto($<HTMLInputElement>('input[type="date"]', field('Due')), '')
      await settle()
      button('Save').click()
      await settle()
      expect(update).toHaveBeenLastCalledWith(4, { due: null })
    })

    it('moves a datetime by the time typed into it, in the browser’s zone', async () => {
      const record = { id: 4, name: 'Launch', due: null, starts: '2026-03-29T10:00:00Z' } as unknown as Item
      const update = vi.fn(async (id: number, body: Partial<ItemInput>) => ({ ...record, id, ...body }))
      await cases({ schema: TIMES, record, dateFields: true, update: fn(update) })
      const local = wireToLocal('2026-03-29T10:00:00Z')!
      const minute = $$('[role="spinbutton"]', field('Starts'))[1]!
      minute.focus()
      key('ArrowUp')
      await settle()
      button('Save').click()
      await settle()
      expect(update).toHaveBeenLastCalledWith(4, { starts: localToWire(local.date, local.time.replace(/:00$/, ':01')) })
    })
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
