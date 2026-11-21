import { afterEach, describe, expect, it } from 'vitest'
import ComboboxCases from '../fixtures/selection/ComboboxCases.vue'
import FieldSpy from '../fixtures/selection/FieldSpy.vue'
import { mount } from '../mount.ts'
import { type } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, locked, moveFocus, overlays, press, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(ComboboxCases, props)
  unmount = m.unmount
  await settle()
  return m
}
type Exposed = { value: unknown; open: boolean; term: string; log: string[]; submitted: number; input: string; calls: { term: string; signal: AbortSignal; resolve: (items: unknown[]) => void }[] }
const field = () => $<HTMLInputElement>('.weft-combobox input:not([type="hidden"])')
const list = () => $('[role="listbox"]', overlays() ?? document)
const options = () => $$('[role="option"]', overlays() ?? document)
const labels = () => options().map((o) => o.textContent!.trim())
const named = (label: string) => options().find((o) => o.querySelector('.weft-listbox__text')!.textContent!.trim() === label)!
const current = (el: Element = field()) => document.getElementById(el.getAttribute('aria-activedescendant') ?? '')
async function typeIn(text: string, el: HTMLInputElement = field()) {
  el.focus()
  type(el, text)
  await settle()
}

describe('Combobox', () => {
  it('is an APG combobox: a text field with a listbox popup, labelled by its field', async () => {
    await cases()
    const input = field()
    expect(input.getAttribute('role')).toBe('combobox')
    expect(input.getAttribute('aria-autocomplete')).toBe('list')
    expect(input.getAttribute('aria-haspopup')).toBe('listbox')
    expect(input.getAttribute('aria-expanded')).toBe('false')
    expect(input.getAttribute('autocomplete')).toBe('off')
    expect($('label').getAttribute('for')).toBe(input.id)
    expect(input.getAttribute('placeholder')).toBe('Search fruit')
    expect(input.hasAttribute('name')).toBe(false) // the search is not the value
  })

  it('opens as you type, filters, highlights the first match, and Enter chooses it without submitting', async () => {
    const m = await cases()
    await typeIn('ap')
    const e = m.exposed as unknown as Exposed
    expect(e.open).toBe(true)
    expect(e.term).toBe('ap')
    expect(field().getAttribute('aria-expanded')).toBe('true')
    expect(field().getAttribute('aria-controls')).toBe(list().id)
    expect(labels()).toEqual(['Apple', 'Apricot'])
    expect(current()).toBe(named('Apple'))
    expect(active()).toBe(field())
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Apricot'))
    const enter = key('Enter')
    await settle()
    expect(enter.defaultPrevented).toBe(true)
    expect(e.submitted).toBe(0)
    expect(e.value).toBe(2)
    expect(e.open).toBe(false)
    expect(field().value).toBe('Apricot')
    expect(active()).toBe(field())
    expect($<HTMLInputElement>('input[type="hidden"]').value).toBe('2')
  })

  it('opens with ArrowDown on the chosen option, leaves Home and End to the caret', async () => {
    await cases({ initial: 3 })
    expect(field().value).toBe('Banana')
    field().focus()
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Banana'))
    expect(labels()).toHaveLength(5) // the chosen label is not a search
    expect(key('Home').defaultPrevented).toBe(false)
    expect(key('End').defaultPrevented).toBe(false)
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Cherry')) // past Blueberry, disabled
  })

  it('closes on Escape, then clears the text and the value on a second Escape', async () => {
    const m = await cases({ initial: 1 })
    const e = m.exposed as unknown as Exposed
    await typeIn('ch')
    key('Escape')
    await settle()
    expect(e.open).toBe(false)
    expect(field().value).toBe('ch')
    expect(e.value).toBe(1)
    key('Escape')
    await settle()
    expect(field().value).toBe('')
    expect(e.value).toBeNull()
  })

  it('puts back the chosen label when focus leaves with a search typed', async () => {
    const m = await cases({ initial: 1 })
    const e = m.exposed as unknown as Exposed
    await typeIn('ban')
    field().dispatchEvent(new FocusEvent('blur', { relatedTarget: $('.after') }))
    $('.after').focus()
    await settle()
    expect(e.open).toBe(false)
    expect(field().value).toBe('Apple')
    expect(e.value).toBe(1)
  })

  it('chooses with a press, focus staying in the field', async () => {
    const m = await cases()
    await typeIn('b')
    const down = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    named('Banana').dispatchEvent(down)
    expect(down.defaultPrevented).toBe(true)
    named('Banana').click()
    await settle()
    expect((m.exposed as unknown as Exposed).value).toBe(3)
    expect(active()).toBe(field())
  })

  it('stays open on a press in the field, which is not outside the list', async () => {
    const m = await cases()
    await typeIn('a')
    press(field())
    await settle()
    expect((m.exposed as unknown as Exposed).open).toBe(true)
    press($('.after'))
    await settle()
    expect((m.exposed as unknown as Exposed).open).toBe(false)
  })

  it('toggles from its chevron, and a second press closes it', async () => {
    const m = await cases()
    const toggle = $<HTMLButtonElement>('.weft-combobox__toggle')
    expect(toggle.tabIndex).toBe(-1)
    expect(toggle.getAttribute('aria-label')).toBe('Show options')
    press(toggle)
    await settle()
    expect((m.exposed as unknown as Exposed).open).toBe(true)
    expect(active()).toBe(field())
    press(toggle)
    await settle()
    expect((m.exposed as unknown as Exposed).open).toBe(false)
  })

  it('shows "No results" when nothing matches, or the empty slot with the search', async () => {
    await cases()
    await typeIn('zz')
    expect(options()).toHaveLength(0)
    expect($('[role="status"]', overlays()!).textContent!.trim()).toBe('No results')
    expect(field().hasAttribute('aria-activedescendant')).toBe(false)
    unmount!()
    reset()
    await cases({ emptySlot: true })
    await typeIn('kiwi')
    expect($('.none', overlays()!).textContent).toBe('No fruit like kiwi')
  })

  it('offers to create what was typed, emitting create', async () => {
    const m = await cases({ createItem: true })
    await typeIn('Kiwi')
    expect(labels()).toEqual(['Create “Kiwi”'])
    key('Enter')
    await settle()
    expect((m.exposed as unknown as Exposed).log).toEqual(['create:Kiwi'])
    unmount!()
    reset()
    const n = await cases({ createItem: { when: 'always', position: 'top' } })
    await typeIn('App')
    expect(labels()).toEqual(['Create “App”', 'Apple'])
    await typeIn('apple')
    expect(labels()).toEqual(['Apple']) // an item is the search: nothing to create
    expect((n.exposed as unknown as Exposed).log).toEqual([])
  })

  it('clears with its clear button', async () => {
    const m = await cases({ initial: 5, clear: true })
    const clear = $<HTMLButtonElement>('.weft-combobox__clear')
    expect(clear.getAttribute('aria-label')).toBe('Clear')
    clear.click()
    await settle()
    const e = m.exposed as unknown as Exposed
    expect(e.value).toBeNull()
    expect(e.log).toEqual(['clear'])
    expect(field().value).toBe('')
    expect(active()).toBe(field())
    expect($('.weft-combobox__clear')).toBeNull()
  })

  describe('with multiple', () => {
    it('shows the chosen as tags with remove buttons, stays open, and clears the search after each', async () => {
      const m = await cases({ multiple: true })
      const e = m.exposed as unknown as Exposed
      await typeIn('ap')
      key('Enter')
      await settle()
      expect(e.value).toEqual([1])
      expect(e.open).toBe(true)
      expect(field().value).toBe('')
      expect(list().getAttribute('aria-multiselectable')).toBe('true')
      await typeIn('ch')
      key('Enter')
      await settle()
      const tags = $$('.weft-tag')
      expect(tags.map((t) => t.querySelector('.weft-tag__text')!.textContent)).toEqual(['Apple', 'Cherry'])
      const remove = $<HTMLButtonElement>('.weft-tag__remove', tags[0])
      expect(remove.getAttribute('aria-label')).toBe('Remove Apple')
      expect(remove.tabIndex).toBe(-1)
      remove.click()
      await settle()
      expect(e.value).toEqual([5])
      expect(e.log).toEqual(['remove:1'])
      key('Backspace')
      await settle()
      expect(e.value).toEqual([])
      expect($$('.weft-tag')).toHaveLength(0)
    })
  })

  describe('async items', () => {
    it('loads as it opens and as you type, busy meanwhile, keeping only the latest answer', async () => {
      const m = await cases({ async: true })
      const e = m.exposed as unknown as Exposed
      field().focus()
      key('ArrowDown')
      await settle()
      expect(e.calls.map((c) => c.term)).toEqual([''])
      expect(list().getAttribute('aria-busy')).toBe('true')
      expect(field().getAttribute('aria-busy')).toBe('true')
      expect($('[role="status"]', overlays()!).textContent!.trim()).toBe('Loading…')
      e.calls[0]!.resolve([{ label: 'Lemon', id: 10 }, { label: 'Lime', id: 11 }])
      await settle()
      expect(labels()).toEqual(['Lemon', 'Lime'])
      expect(list().hasAttribute('aria-busy')).toBe(false)
      await typeIn('li')
      await settle()
      await typeIn('lim')
      await settle()
      expect(e.calls.map((c) => c.term)).toEqual(['', 'li', 'lim'])
      expect(e.calls[1]!.signal.aborted).toBe(true)
      e.calls[2]!.resolve([{ label: 'Lime', id: 11 }])
      e.calls[1]!.resolve([{ label: 'Lichee', id: 12 }]) // too late: it was superseded
      await settle()
      expect(labels()).toEqual(['Lime'])
      // The app's results are shown as given, not filtered again.
      key('Enter')
      await settle()
      expect(e.value).toBe(11)
      expect(field().value).toBe('Lime')
    })
  })

  describe('as a button (Nuxt SelectMenu)', () => {
    it('opens a popup with a search field that takes focus; choosing returns focus to the button', async () => {
      const m = await cases({ trigger: 'button' })
      const e = m.exposed as unknown as Exposed
      const button = $<HTMLButtonElement>('.weft-combobox__trigger')
      expect(button.tagName).toBe('BUTTON')
      expect(button.getAttribute('aria-haspopup')).toBe('dialog')
      expect($('label').getAttribute('for')).toBe(button.id)
      expect(pressKey(button, 'Enter')).toBe(true)
      await settle()
      expect(e.open).toBe(true)
      const popup = $('.weft-popover')
      expect(popup.getAttribute('role')).toBe('dialog')
      const search = $<HTMLInputElement>('.weft-select__search input')
      expect(active()).toBe(search)
      expect(search.getAttribute('aria-controls')).toBe(list().id)
      type(search, 'cher')
      await settle()
      expect(labels()).toEqual(['Cherry'])
      expect(current(search)).toBe(named('Cherry'))
      key('Enter')
      await settle()
      expect(e.value).toBe(5)
      expect(e.open).toBe(false)
      expect(active()).toBe(button)
      expect($('.weft-select__value', button).textContent!.trim()).toBe('Cherry')
    })

    it('tells its field of a blur when focus leaves from the search field, not when a choice brings it back', async () => {
      const m = mount(FieldSpy, { control: 'combobox-button' })
      unmount = m.unmount
      await settle()
      const spy = m.exposed as unknown as { blurs: number; inputs: number }
      const button = $<HTMLButtonElement>('.weft-combobox__trigger')
      button.focus()
      button.click()
      await settle()
      key('Enter')
      await settle()
      await new Promise((r) => setTimeout(r, 5))
      expect(active()).toBe(button)
      expect(spy.inputs).toBe(1)
      expect(spy.blurs).toBe(0)
      button.click()
      await settle()
      expect(active()).toBe($('.weft-select__search input'))
      moveFocus($('.after'))
      await settle()
      await new Promise((r) => setTimeout(r, 5))
      expect($('.weft-popover')).toBeNull()
      expect(spy.blurs).toBe(1)
    })
  })

  describe('autocomplete mode', () => {
    it('binds the text itself, suggestions completing it', async () => {
      const m = await cases({ mode: 'autocomplete', initial: '' })
      const e = m.exposed as unknown as Exposed
      await typeIn('Ba')
      expect(e.value).toBe('Ba')
      expect(labels()).toEqual(['Banana'])
      key('Enter')
      await settle()
      expect(e.value).toBe('Banana')
      expect(field().value).toBe('Banana')
      expect(options().length).toBe(0)
    })
  })

  it('does not open while disabled', async () => {
    const m = await cases({ disabled: true })
    expect(field().disabled).toBe(true)
    key('ArrowDown', {}, field())
    press($('.weft-combobox__toggle'))
    await settle()
    expect((m.exposed as unknown as Exposed).open).toBe(false)
  })

  describe('on touch', () => {
    it('opens as a modal bottom sheet with a search field at the top, the field itself read-only', async () => {
      const m = await cases({ input: 'touch' })
      const e = m.exposed as unknown as Exposed
      expect(field().readOnly).toBe(true)
      field().focus()
      field().click()
      await settle()
      expect(e.open).toBe(true)
      const sheet = $('.weft-popover')
      expect(sheet.dataset.form).toBe('sheet')
      expect(sheet.getAttribute('aria-modal')).toBe('true')
      expect(locked()).toBe(true)
      const search = $<HTMLInputElement>('.weft-select__search input')
      expect(active()).toBe(search)
      type(search, 'ban')
      await settle()
      expect(labels()).toEqual(['Banana'])
      key('Enter')
      await settle()
      expect(e.value).toBe(3)
      expect(e.open).toBe(false)
      expect(active()).toBe(field())
      expect(field().value).toBe('Banana')
      expect(locked()).toBe(false)
    })

    it('becomes a floating list when the input turns to pointer while open, focus back in the field', async () => {
      const m = await cases({ input: 'touch' })
      const e = m.exposed as unknown as Exposed
      field().focus()
      field().click()
      await settle()
      e.input = 'pointer'
      await settle()
      expect(e.open).toBe(true)
      expect($('.weft-popover').dataset.form).toBe('floating')
      expect(active()).toBe(field())
      expect(field().readOnly).toBe(false)
      expect(field().getAttribute('aria-activedescendant')).toBe(current()!.id)
    })
  })
})
