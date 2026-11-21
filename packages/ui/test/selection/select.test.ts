import { afterEach, describe, expect, it } from 'vitest'
import SelectCases from '../fixtures/selection/SelectCases.vue'
import SelectInDialog from '../fixtures/selection/SelectInDialog.vue'
import FieldSpy from '../fixtures/selection/FieldSpy.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, locked, overlays, press, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(SelectCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const trigger = () => $<HTMLButtonElement>('.weft-select__trigger')
const list = () => $('[role="listbox"]', overlays() ?? document)
const options = () => $$('[role="option"]', overlays() ?? document)
const named = (label: string) => options().find((o) => o.querySelector('.weft-listbox__text')!.textContent!.trim() === label)!
const current = (el: Element = trigger()) => document.getElementById(el.getAttribute('aria-activedescendant') ?? '')
async function openWith(k?: string) {
  trigger().focus()
  if (k) key(k)
  else trigger().click()
  await settle()
}

describe('Select', () => {
  it('is an APG select-only combobox: a button that says it has a listbox, labelled by its field', async () => {
    await cases()
    const t = trigger()
    expect(t.tagName).toBe('BUTTON')
    expect(t.type).toBe('button')
    expect(t.getAttribute('role')).toBe('combobox')
    expect(t.getAttribute('aria-haspopup')).toBe('listbox')
    expect(t.getAttribute('aria-expanded')).toBe('false')
    expect(t.hasAttribute('aria-controls')).toBe(false)
    expect($('label').getAttribute('for')).toBe(t.id)
    expect(t.getAttribute('aria-describedby')).toBe($('.weft-form-field__description').id)
    expect($('.weft-select__value', t).textContent!.trim()).toBe('Pick a fruit')
    expect($('.weft-select__value', t).dataset.placeholder).toBe('true')
    expect(overlays()).toBeNull()
  })

  it('opens on a press with the list floating against it, focus staying on the button', async () => {
    const m = await cases()
    await openWith()
    expect(m.exposed.open).toBe(true)
    expect(trigger().getAttribute('aria-expanded')).toBe('true')
    expect(trigger().getAttribute('aria-controls')).toBe(list().id)
    expect(list().getAttribute('aria-labelledby')).toBe($('label').id)
    expect($('.weft-popover').dataset.form).toBe('floating')
    expect($('.weft-popover').getAttribute('role')).toBe('presentation')
    expect(active()).toBe(trigger())
    expect(current()).toBe(named('Apple'))
    expect(locked()).toBe(false)
  })

  it('opens from the keyboard: ArrowDown, Enter and Space on the selected option, Home on the first, End on the last', async () => {
    await cases({ initial: 'cherry' })
    for (const k of ['ArrowDown', 'ArrowUp', 'Enter', ' ']) {
      await openWith(k)
      expect(current(), k).toBe(named('Cherry'))
      key('Escape')
      await settle()
      expect(active()).toBe(trigger())
    }
    await openWith('End')
    expect(current()).toBe(named('Date'))
    key('Escape')
    await settle()
    await openWith('Home')
    expect(current()).toBe(named('Apple'))
  })

  it('moves with the arrows past a disabled option, chooses with Enter, closes and keeps focus', async () => {
    const m = await cases()
    await openWith('ArrowDown')
    key('ArrowDown')
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Cherry'))
    expect(named('Cherry').dataset.active).toBe('true')
    const enter = key('Enter')
    await settle()
    expect(enter.defaultPrevented).toBe(true)
    expect(m.exposed.value).toBe('cherry')
    expect(m.exposed.open).toBe(false)
    expect(active()).toBe(trigger())
    expect($('.weft-select__value', trigger()).textContent!.trim()).toBe('Cherry')
    expect(trigger().hasAttribute('aria-activedescendant')).toBe(false)
  })

  it('reports no change when the chosen value is chosen again', async () => {
    const m = mount(FieldSpy, { control: 'select' })
    unmount = m.unmount
    await settle()
    const spy = m.exposed as unknown as { inputs: number; changes: number }
    for (let i = 0; i < 2; i++) {
      await openWith()
      key('Enter') // One, the first, then the chosen one
      await settle()
    }
    expect(spy.changes).toBe(1)
    expect(spy.inputs).toBe(1)
  })

  it('chooses with a press on an option and with Space', async () => {
    const m = await cases()
    await openWith()
    named('Date').click()
    await settle()
    expect(m.exposed.value).toBe('date')
    expect(m.exposed.open).toBe(false)
    await openWith()
    key('ArrowUp')
    key(' ')
    await settle()
    expect(m.exposed.value).toBe('cherry')
  })

  it('jumps by typing: closed, it opens on the match; open, it moves to it', async () => {
    await cases()
    trigger().focus()
    key('c')
    await settle()
    expect(trigger().getAttribute('aria-expanded')).toBe('true')
    expect(current()).toBe(named('Cherry'))
    await new Promise((r) => setTimeout(r, 600))
    key('b')
    await settle()
    expect(current()).toBe(named('Banana'))
  })

  it('closes on Escape without choosing, on a second press, on Tab and when focus leaves', async () => {
    const m = await cases()
    await openWith()
    key('ArrowDown')
    key('Escape')
    await settle()
    expect(m.exposed.open).toBe(false)
    expect(m.exposed.value).toBeUndefined()
    await openWith()
    press(trigger())
    await settle()
    expect(m.exposed.open).toBe(false)
    await openWith()
    key('Tab')
    await settle()
    expect(m.exposed.open).toBe(false)
    await openWith()
    trigger().dispatchEvent(new FocusEvent('blur', { relatedTarget: $('.after') }))
    trigger().dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: $('.after') }))
    $('.after').focus()
    await settle()
    expect(m.exposed.open).toBe(false)
    await openWith()
    press($('.after'))
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('selects several with multiple, staying open, and shows them joined', async () => {
    const m = await cases({ multiple: true })
    await openWith()
    expect(list().getAttribute('aria-multiselectable')).toBe('true')
    key('Enter')
    key('ArrowDown')
    key(' ')
    await settle()
    expect(m.exposed.value).toEqual(['apple', 'banana'])
    expect(m.exposed.open).toBe(true)
    expect($('.weft-select__value', trigger()).textContent!.trim()).toBe('Apple, Banana')
    named('Apple').click()
    await settle()
    expect(m.exposed.value).toEqual(['banana'])
  })

  it('submits its value under the field name, every value with multiple', async () => {
    const m = await cases({ initial: 'banana' })
    $<HTMLButtonElement>('.submit').click()
    await settle()
    expect(m.exposed.submitted).toEqual(['banana'])
    unmount!()
    reset()
    const n = await cases({ multiple: true, initial: ['apple', 'date'] })
    $<HTMLButtonElement>('.submit').click()
    await settle()
    expect(n.exposed.submitted).toEqual(['apple', 'date'])
  })

  it('does not open while disabled, by press or key', async () => {
    const m = await cases({ disabled: true })
    expect(trigger().disabled).toBe(true)
    trigger().click()
    expect(pressKey(trigger(), 'Enter')).toBe(false)
    key('ArrowDown', {}, trigger())
    await settle()
    expect(m.exposed.open).toBe(false)
  })

  it('shows a spinner and aria-busy while loading', async () => {
    await cases({ loading: true })
    expect(trigger().getAttribute('aria-busy')).toBe('true')
    expect($('.weft-spinner', trigger())).not.toBeNull()
  })

  it('virtualizes a list of 10,000, keeping the active option rendered', async () => {
    const items = Array.from({ length: 10_000 }, (_, i) => ({ label: `Row ${i + 1}`, value: i }))
    const m = await cases({ items, initial: 9_000 })
    await openWith()
    expect(options().length).toBeLessThan(40)
    expect(current()!.getAttribute('aria-posinset')).toBe('9001')
    key('End')
    await settle()
    expect(current()!.textContent!.trim()).toBe('Row 10000')
    key('Enter')
    await settle()
    expect(m.exposed.value).toBe(9_999)
  })

  it('shows the focus ring on the button and marks the active option', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      const ring = ruleFor('.weft-input.weft-select__trigger:focus-visible')!.style
      expect(ring.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      expect(ruleFor('.weft-listbox__option[data-active]')).toBeDefined()
    } finally {
      unstyle()
    }
  })

  it('mirrors its placement right to left: the list opens in the layer of a right-to-left page', async () => {
    await cases({ dir: 'rtl' })
    await openWith()
    expect($('.weft-popover').closest('.weft-overlay')!.getAttribute('dir')).toBe('rtl')
  })

  it('stacks in a Dialog: Escape closes the list, then the dialog, focus going back each time', async () => {
    const m = mount(SelectInDialog)
    unmount = m.unmount
    await settle()
    const e = m.exposed as unknown as { open: boolean; value: string | undefined }
    const opener = $<HTMLButtonElement>('.opener')
    opener.focus()
    opener.click()
    await settle()
    expect(active()).toBe(trigger())
    key('ArrowDown')
    await settle()
    expect(list()).not.toBeNull()
    expect(locked()).toBe(true)
    key('Escape')
    await settle()
    expect(e.open).toBe(true)
    expect($('[role="listbox"]')).toBeNull()
    expect(active()).toBe(trigger())
    key('Enter')
    await settle()
    key('End')
    key('Enter')
    await settle()
    expect(e.value).toBe('Large')
    expect(e.open).toBe(true)
    key('Escape')
    await settle()
    expect(e.open).toBe(false)
    expect(active()).toBe(opener)
    expect(locked()).toBe(false)
  })

  describe('on touch', () => {
    it('opens as a modal bottom sheet with a search field at the top that takes focus and filters', async () => {
      const unstyle = loadStyles()
      try {
        const m = await cases({ input: 'touch' })
        trigger().focus()
        trigger().click()
        await settle()
        const sheet = $('.weft-popover')
        expect(sheet.dataset.form).toBe('sheet')
        expect(sheet.getAttribute('role')).toBe('dialog')
        expect(sheet.getAttribute('aria-modal')).toBe('true')
        expect(sheet.getAttribute('aria-label')).toBe('Fruit')
        expect(locked()).toBe(true)
        expect(px(getComputedStyle(trigger()).blockSize)).toBe(44)
        const search = $<HTMLInputElement>('.weft-select__search input')
        // The search field comes first, under the title, then the list.
        expect(sheet.querySelector('h2')!.nextElementSibling).toBe(search.parentElement)
        expect(search.compareDocumentPosition(list()) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
        expect(active()).toBe(search)
        expect(search.getAttribute('role')).toBe('combobox')
        expect(search.getAttribute('aria-controls')).toBe(list().id)
        expect(px(getComputedStyle(named('Apple')).minBlockSize)).toBe(48)
        type(search, 'err')
        await settle()
        expect(options().map((o) => o.textContent!.trim())).toEqual(['Blueberry', 'Cherry'])
        expect(current(search)).toBe(named('Cherry'))
        key('Enter')
        await settle()
        expect(m.exposed.value).toBe('cherry')
        expect(m.exposed.open).toBe(false)
        expect(active()).toBe(trigger())
        expect(locked()).toBe(false)
        // The search starts empty next time.
        trigger().click()
        await settle()
        expect($<HTMLInputElement>('.weft-select__search input').value).toBe('')
      } finally {
        unstyle()
      }
    })

    it('with search-input false opens the sheet with no search field, focus on the list itself', async () => {
      const m = await cases({ input: 'touch', searchInput: false })
      trigger().focus()
      trigger().click()
      await settle()
      expect($('.weft-select__search')).toBeNull()
      expect(active()).toBe(list())
      expect(list().tabIndex).toBe(0)
      expect(current(list())).toBe(named('Apple'))
      key('ArrowDown')
      key('Enter')
      await settle()
      expect(m.exposed.value).toBe('banana')
      expect(active()).toBe(trigger())
    })

    it('keeps its state when the input switches to pointer while open, focus back on the button', async () => {
      const m = await cases({ input: 'touch' })
      trigger().focus()
      trigger().click()
      await settle()
      key('ArrowDown')
      await settle()
      const was = current($('.weft-select__search input'))!.id
      ;(m.exposed as { input: string }).input = 'pointer'
      await settle()
      expect($('.weft-popover').dataset.form).toBe('floating')
      expect(m.exposed.open).toBe(true)
      expect(active()).toBe(trigger())
      expect(current()!.id).toBe(was)
      expect(locked()).toBe(false)
    })
  })
})
