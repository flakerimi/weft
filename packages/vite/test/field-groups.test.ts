// A FormField's label over every control it can wrap, as the server draws
// it, so a page that never hydrates is right too. Over a labelable element
// (an input, a textarea, a button) the label is a <label> whose `for` is that
// element's id. Over a control that is none (a radio group's fieldset, a
// slider's thumbs, a listbox, a group of time segments) it is no <label> at
// all, since a <label> must label a form control: the control is
// aria-labelledby its id. No <label> on the page has a `for` naming a missing
// or unlabelable element, or neither a `for` nor a control inside it.
import { fileURLToPath } from 'node:url'
import { Window } from 'happy-dom'
import { beforeAll, describe, expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

const LABELABLE = new Set(['BUTTON', 'INPUT', 'METER', 'OUTPUT', 'PROGRESS', 'SELECT', 'TEXTAREA'])
const labelable = (el: Element | null) => !!el && LABELABLE.has(el.tagName) && !(el.tagName === 'INPUT' && (el as HTMLInputElement).type === 'hidden')

// The elements each control that is no labelable element names by the label's id.
const named: Record<string, string> = {
  RadioGroup: 'fieldset',
  CheckboxGroup: 'fieldset',
  Slider: '[role="slider"]',
  'Slider range': '[role="slider"]',
  Listbox: '[role="listbox"]',
  InputTime: '.weft-input-time[role="group"]',
  'InputTime range': '.weft-input-time[role="group"]',
  'InputTime touch range': '.weft-input-time[role="group"]',
  ToggleGroup: '.weft-toggle-group[role="group"]',
}

let doc: Document
beforeAll(async () => {
  expect(typeof window).toBe('undefined')
  const root = fileURLToPath(new URL('./fixtures/field-groups', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  const win = new Window()
  win.document.body.innerHTML = html
  doc = win.document as unknown as Document
})

const fields = () => [...doc.querySelectorAll('[data-control]')].map((el) => {
  const [kind, ...name] = el.getAttribute('data-control')!.split(' ')
  return { el, kind: kind as 'named' | 'native', name: name.join(' ') }
})

describe('a FormField on the server', () => {
  it('draws every control in the fixture', () => {
    expect(fields().map((f) => f.name).sort()).toEqual([
      ...Object.keys(named),
      'Checkbox', 'Combobox', 'DatePicker', 'DatePicker touch', 'DateRangePicker', 'FileUpload', 'Input', 'InputNumber',
      'InputTags', 'InputTime touch', 'Listbox filter', 'PinInput', 'Select', 'Switch', 'Textarea', 'Toggle', 'select',
    ].sort())
  })

  it('labels a labelable control with a <label> for its id', () => {
    for (const { el, kind, name } of fields()) {
      if (kind !== 'native') continue
      const labels = el.querySelectorAll('.weft-form-field__label')
      expect(labels, name).toHaveLength(1)
      const label = labels[0]!
      expect(label.tagName, name).toBe('LABEL')
      const target = doc.getElementById(label.getAttribute('for') ?? '')
      expect(labelable(target), `${name}: the label is for ${target?.outerHTML.slice(0, 80) ?? 'nothing'}`).toBe(true)
      expect(el.contains(target), name).toBe(true)
    }
  })

  it('names a control that is no labelable element by the label\'s id, with no <label>', () => {
    for (const { el, kind, name } of fields()) {
      if (kind !== 'named') continue
      const labels = el.querySelectorAll('.weft-form-field__label')
      expect(labels, name).toHaveLength(1)
      const label = labels[0]!
      expect(label.tagName, name).not.toBe('LABEL')
      expect(label.hasAttribute('for'), name).toBe(false)
      expect(label.id, name).not.toBe('')
      const controls = [...el.querySelectorAll(named[name]!)]
      expect(controls.length, name).toBeGreaterThan(0)
      for (const c of controls) expect(c.getAttribute('aria-labelledby')?.split(' '), `${name}: ${c.outerHTML.slice(0, 80)}`).toContain(label.id)
      expect(el.querySelector('.weft-form-field__head label'), name).toBeNull()
    }
  })

  it('names no control by the id of a label it does not draw', () => {
    const unlabelled = doc.querySelectorAll('[data-unlabelled] .weft-form-field')
    expect(unlabelled.length).toBe(16)
    for (const field of unlabelled) expect(field.querySelector('.weft-form-field__label')).toBeNull()
    for (const el of doc.querySelectorAll('[aria-labelledby]')) {
      for (const id of el.getAttribute('aria-labelledby')!.split(' ')) {
        expect(doc.getElementById(id), `${el.outerHTML.slice(0, 100)}: no element ${id}`).not.toBeNull()
      }
    }
  })

  it('draws no <label> that labels no form control', () => {
    const labels = [...doc.querySelectorAll('label')]
    expect(labels.length).toBeGreaterThan(0)
    for (const label of labels) {
      const what = label.outerHTML.slice(0, 100)
      if (label.hasAttribute('for')) expect(labelable(doc.getElementById(label.getAttribute('for')!)), what).toBe(true)
      else expect([...label.querySelectorAll('*')].some(labelable), what).toBe(true)
    }
  })
})
