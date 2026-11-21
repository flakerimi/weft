import { afterEach, describe, expect, it, vi } from 'vitest'
import ListboxCases from '../fixtures/selection/ListboxCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(ListboxCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const list = () => $('[role="listbox"]')
const options = () => $$('[role="option"]')
const named = (label: string) => options().find((o) => o.querySelector('.weft-listbox__text')!.textContent!.trim() === label)!
const current = () => document.getElementById(list().getAttribute('aria-activedescendant') ?? '')
const currentIn = (el: Element) => document.getElementById(el.getAttribute('aria-activedescendant') ?? '')

describe('Listbox', () => {
  it('is an APG listbox: options with ids, one tab stop, named by its field label', async () => {
    await cases()
    expect(list().tabIndex).toBe(0)
    expect(options().map((o) => o.textContent!.trim())).toEqual(['Apple', 'Banana', 'Blueberry', 'Cherry', 'Date'])
    expect(new Set(options().map((o) => o.id)).size).toBe(5)
    expect(options().every((o) => o.getAttribute('aria-selected') === 'false')).toBe(true)
    expect(named('Blueberry').getAttribute('aria-disabled')).toBe('true')
    const label = $('label')
    expect(list().getAttribute('aria-labelledby')).toBe(label.id)
    expect(label.hasAttribute('for')).toBe(false)
    expect(list().getAttribute('aria-describedby')).toBe($('.weft-form-field__hint').id)
    expect(list().hasAttribute('aria-multiselectable')).toBe(false)
  })

  it('takes focus on the selected option, else the first', async () => {
    await cases({ initial: 'Cherry' })
    expect(named('Cherry').getAttribute('aria-selected')).toBe('true')
    list().focus()
    await settle()
    expect(current()).toBe(named('Cherry'))
    expect(named('Cherry').dataset.active).toBe('true')
  })

  it('moves with the arrows, Home and End, skipping a disabled option, and chooses with Enter and Space', async () => {
    const m = await cases()
    list().focus()
    await settle()
    expect(current()).toBe(named('Apple'))
    key('ArrowDown')
    key('ArrowDown')
    await settle()
    expect(current()).toBe(named('Cherry')) // past Blueberry, disabled
    expect(key('Enter').defaultPrevented).toBe(true)
    await settle()
    expect(m.exposed.value).toBe('Cherry')
    expect(named('Cherry').getAttribute('aria-selected')).toBe('true')
    expect($('.weft-listbox__check', named('Cherry'))).not.toBeNull()
    key('End')
    await settle()
    expect(current()).toBe(named('Date'))
    key('ArrowDown') // no loop by default
    await settle()
    expect(current()).toBe(named('Date'))
    key('Home')
    await settle()
    key(' ')
    await settle()
    expect(m.exposed.value).toBe('Apple')
    expect(named('Cherry').getAttribute('aria-selected')).toBe('false')
  })

  it('jumps to an option by typing the start of its label', async () => {
    await cases()
    list().focus()
    await settle()
    key('c')
    await settle()
    expect(current()).toBe(named('Cherry'))
    key('b') // Blueberry is disabled: the typeahead skips it
    await new Promise((r) => setTimeout(r, 600))
    key('b')
    await settle()
    expect(current()).toBe(named('Banana'))
  })

  it('selects several with multiple: aria-multiselectable, Space and Enter toggle', async () => {
    const m = await cases({ multiple: true })
    expect(list().getAttribute('aria-multiselectable')).toBe('true')
    list().focus()
    await settle()
    key(' ')
    key('ArrowDown')
    key('Enter')
    await settle()
    expect(m.exposed.value).toEqual(['Apple', 'Banana'])
    key(' ')
    await settle()
    expect(m.exposed.value).toEqual(['Apple'])
    expect(options().filter((o) => o.getAttribute('aria-selected') === 'true')).toEqual([named('Apple')])
  })

  describe('range selection with multiple (APG)', () => {
    it('Shift+Arrow moves and toggles the option reached, skipping disabled ones', async () => {
      const m = await cases({ multiple: true })
      list().focus()
      await settle()
      key(' ')
      key('ArrowDown', { shiftKey: true })
      key('ArrowDown', { shiftKey: true })
      await settle()
      expect(m.exposed.value).toEqual(['Apple', 'Banana', 'Cherry'])
      expect(current()).toBe(named('Cherry'))
    })

    it('Shift+Space chooses every option from the last one chosen to the active one', async () => {
      const m = await cases({ multiple: true })
      list().focus()
      await settle()
      key('ArrowDown')
      key(' ') // Banana: the anchor
      key('End')
      key(' ', { shiftKey: true })
      await settle()
      expect(m.exposed.value).toEqual(['Banana', 'Cherry', 'Date'])
    })

    it('Ctrl+A (or Cmd+A) chooses every option, and again none', async () => {
      const m = await cases({ multiple: true })
      list().focus()
      await settle()
      expect(key('a', { ctrlKey: true }).defaultPrevented).toBe(true)
      await settle()
      expect(m.exposed.value).toEqual(['Apple', 'Banana', 'Cherry', 'Date'])
      key('a', { metaKey: true })
      await settle()
      expect(m.exposed.value).toEqual([])
    })

    it('Shift+press chooses a range, Ctrl+press toggles one', async () => {
      const m = await cases({ multiple: true })
      named('Apple').click()
      named('Date').dispatchEvent(new MouseEvent('click', { bubbles: true, shiftKey: true }))
      await settle()
      expect(m.exposed.value).toEqual(['Apple', 'Banana', 'Cherry', 'Date'])
      named('Banana').dispatchEvent(new MouseEvent('click', { bubbles: true, ctrlKey: true }))
      await settle()
      expect(m.exposed.value).toEqual(['Apple', 'Cherry', 'Date'])
    })

    it('with selection-behavior="replace" the selection follows the arrows; Shift extends it, Ctrl+Space toggles', async () => {
      const m = await cases({ multiple: true, selectionBehavior: 'replace' })
      list().focus()
      await settle()
      key(' ')
      key('ArrowDown')
      await settle()
      expect(m.exposed.value).toEqual(['Banana'])
      key('ArrowDown', { shiftKey: true })
      key('ArrowDown', { shiftKey: true })
      await settle()
      expect(m.exposed.value).toEqual(['Banana', 'Cherry', 'Date'])
      // Ctrl+Arrow moves without choosing; Ctrl+Space toggles.
      for (let i = 0; i < 3; i++) key('ArrowUp', { ctrlKey: true })
      await settle()
      expect(current()).toBe(named('Apple'))
      expect(m.exposed.value).toEqual(['Banana', 'Cherry', 'Date'])
      key(' ', { ctrlKey: true })
      await settle()
      expect(m.exposed.value).toEqual(['Banana', 'Cherry', 'Date', 'Apple'])
      named('Cherry').click()
      await settle()
      expect(m.exposed.value).toEqual(['Cherry'])
    })
  })

  it('draws groups labelled by their headings, and separators hidden from assistive technology', async () => {
    await cases({
      items: [
        [{ type: 'label', label: 'Fruit' }, 'Apple', 'Pear'],
        [{ type: 'label', label: 'Veg' }, 'Leek'],
      ],
    })
    const groups = $$('[role="group"]', list())
    expect(groups).toHaveLength(2)
    expect(document.getElementById(groups[1]!.getAttribute('aria-labelledby')!)!.textContent).toBe('Veg')
    expect($$('.weft-listbox__separator', list()).every((s) => s.getAttribute('aria-hidden') === 'true')).toBe(true)
    expect($$('[role="separator"]', list())).toHaveLength(0)
  })

  it('follows the pointer and takes a press without losing focus', async () => {
    const m = await cases()
    list().focus()
    await settle()
    named('Date').dispatchEvent(new PointerEvent('pointermove', { bubbles: true, pointerType: 'mouse' }))
    await settle()
    expect(current()).toBe(named('Date'))
    const down = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    named('Date').dispatchEvent(down)
    expect(down.defaultPrevented).toBe(true)
    named('Date').click()
    await settle()
    expect(m.exposed.value).toBe('Date')
    expect(active()).toBe(list())
    named('Blueberry').click()
    await settle()
    expect(m.exposed.value).toBe('Date')
  })

  it('submits its value with a form under its field name', async () => {
    await cases({ initial: 'Banana' })
    const hidden = $<HTMLInputElement>('input[type="hidden"]')
    expect(hidden.name).toBe('fruit')
    expect(hidden.value).toBe('Banana')
  })

  it('does nothing while disabled: out of the tab order, aria-disabled', async () => {
    const m = await cases({ disabled: true })
    expect(list().getAttribute('aria-disabled')).toBe('true')
    expect(list().tabIndex).toBe(-1)
    key('Enter', {}, list())
    named('Apple').click()
    await settle()
    expect(m.exposed.value).toBeUndefined()
  })

  describe('with a filter', () => {
    it('has a search field that controls the list, which it filters as you type', async () => {
      const m = await cases({ filter: true })
      const input = $<HTMLInputElement>('.weft-listbox__filter input')
      expect(input.getAttribute('role')).toBe('combobox')
      expect(input.getAttribute('aria-controls')).toBe(list().id)
      expect(input.getAttribute('aria-expanded')).toBe('true')
      expect(input.getAttribute('aria-autocomplete')).toBe('list')
      expect($('label').getAttribute('for')).toBe(input.id)
      expect(list().hasAttribute('tabindex')).toBe(false)
      input.focus()
      type(input, 'an')
      await settle()
      expect(m.exposed.term).toBe('an')
      expect(options().map((o) => o.textContent!.trim())).toEqual(['Banana'])
      expect(currentIn(input)).toBe(named('Banana'))
      key('Enter')
      await settle()
      expect(m.exposed.value).toBe('Banana')
    })

    it('says so when nothing matches, or shows the empty slot with the search', async () => {
      await cases({ filter: true })
      const input = $<HTMLInputElement>('.weft-listbox__filter input')
      type(input, 'zzz')
      await settle()
      expect(options()).toHaveLength(0)
      expect($('[role="status"]').textContent!.trim()).toBe('No results')
      expect(input.hasAttribute('aria-activedescendant')).toBe(false)
      unmount!()
      reset()
      await cases({ emptySlot: true })
      type($<HTMLInputElement>('.weft-listbox__filter input'), 'kiwi')
      await settle()
      expect($('[role="status"] .none').textContent).toBe('Nothing for kiwi')
    })

    it('offers to create the search as an item when nothing matches', async () => {
      const m = await cases({ filter: true, createItem: true })
      const input = $<HTMLInputElement>('.weft-listbox__filter input')
      input.focus()
      type(input, 'Kiwi')
      await settle()
      expect(options()).toHaveLength(1)
      expect(options()[0]!.textContent!.trim()).toBe('Create “Kiwi”')
      expect(currentIn(input)).toBe(options()[0])
      key('Enter')
      await settle()
      expect(m.exposed.created).toEqual(['Kiwi'])
    })
  })

  it('is busy while loading, and says so in place of the empty list', async () => {
    await cases({ loading: true, items: [] })
    expect(list().getAttribute('aria-busy')).toBe('true')
    const status = $('[role="status"]')
    expect(status.textContent!.trim()).toBe('Loading…')
    expect($('.weft-spinner', status)).not.toBeNull()
  })

  describe('a long list', () => {
    const many = Array.from({ length: 10_000 }, (_, i) => `Item ${i + 1}`)

    it('renders only the options in view of 10,000, each knowing its place in the whole', async () => {
      await cases({ items: many })
      expect(list().dataset.virtual).toBe('true')
      expect(options().length).toBeLessThan(40)
      expect(options()[0]!.getAttribute('aria-setsize')).toBe('10000')
      expect(options()[0]!.getAttribute('aria-posinset')).toBe('1')
    })

    it('keeps aria-activedescendant on a rendered option as the keyboard goes to the end and the pointer scrolls away', async () => {
      await cases({ items: many })
      list().focus()
      await settle()
      key('End')
      await settle()
      expect(current()).not.toBeNull()
      expect(current()!.getAttribute('aria-posinset')).toBe('10000')
      expect(current()!.textContent!.trim()).toBe('Item 10000')
      expect(list().scrollTop).toBeGreaterThan(300_000)
      // Scrolled back to the top by the wheel, the active option is still there.
      list().scrollTop = 0
      list().dispatchEvent(new Event('scroll'))
      await settle()
      expect(current()!.textContent!.trim()).toBe('Item 10000')
      expect(options().length).toBeLessThan(40)
      key('PageUp')
      await settle()
      expect(current()!.textContent!.trim()).toBe('Item 9990')
      key('Home')
      await settle()
      expect(current()!.textContent!.trim()).toBe('Item 1')
      for (const ch of 'item 9') key(ch) // the typeahead finds an option that was not rendered
      await settle()
      expect(current()!.textContent!.trim()).toBe('Item 9')
    })

    it('counts aria-posinset and aria-setsize within each group', async () => {
      const half = (name: string) => [{ type: 'label', label: name }, ...Array.from({ length: 150 }, (_, i) => `${name} ${i + 1}`)]
      await cases({ items: [half('A'), half('B')] })
      expect(list().dataset.virtual).toBe('true')
      expect(options()[0]!.getAttribute('aria-setsize')).toBe('150')
      list().focus()
      await settle()
      key('End')
      await settle()
      expect(current()!.textContent!.trim()).toBe('B 150')
      expect(current()!.getAttribute('aria-posinset')).toBe('150')
      expect(current()!.getAttribute('aria-setsize')).toBe('150')
      // Its group is drawn with its heading, which names it.
      const group = current()!.closest('[role="group"]')!
      expect(document.getElementById(group.getAttribute('aria-labelledby')!)!.textContent).toBe('B')
    })

    it('starts from estimate-size, then takes the height its rows measure, so a taller row never overlaps the next', async () => {
      await cases({ items: many.slice(0, 300), virtualize: { estimateSize: 50 } })
      const spacer = () => $('.weft-listbox__spacer')
      expect(spacer().style.blockSize).toBe('15000px')
      unmount!()
      reset()
      const rect = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
        const h = this.classList.contains('weft-listbox__option') ? 60 : 0
        return { height: h, width: 100, top: 0, bottom: h, left: 0, right: 100, x: 0, y: 0, toJSON: () => ({}) } as DOMRect
      })
      try {
        await cases({ items: many.slice(0, 300) })
        await settle()
        expect(list().style.getPropertyValue('--weft-listbox-row')).toBe('60px')
        expect(spacer().style.blockSize).toBe('18000px')
        expect(options()[1]!.style.top).toBe('60px')
      } finally {
        rect.mockRestore()
      }
    })

    it('renders every option up to 200', async () => {
      await cases({ items: many.slice(0, 200) })
      expect(list().hasAttribute('data-virtual')).toBe(false)
      expect(options()).toHaveLength(200)
      expect(options()[0]!.hasAttribute('aria-setsize')).toBe(false)
    })
  })

  it('on touch, options are at least 48 px tall', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(named('Apple')).minBlockSize)).toBe(48)
    } finally {
      unstyle()
    }
  })

  it('shows a focus ring on the list and marks the active option', async () => {
    const unstyle = loadStyles()
    try {
      await cases()
      expect(ruleFor('.weft-listbox__list:focus-visible')!.style.getPropertyValue('outline-color')).toBe('var(--weft-color-focus, Highlight)')
      expect(ruleFor('.weft-listbox__option[data-active]')).toBeDefined()
    } finally {
      unstyle()
    }
  })
})
