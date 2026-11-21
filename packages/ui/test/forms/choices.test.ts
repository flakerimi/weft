import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import ChoiceCases from '../fixtures/forms/ChoiceCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { key, px, settle, space } from './helpers.ts'

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
function cases(props: Record<string, unknown>) {
  const m = mount(ChoiceCases, props)
  mounted.push(m)
  return m
}
const inputOf = (root: ParentNode, sel: string) => root.querySelector<HTMLInputElement>(`${sel} input`)!

describe('Checkbox', () => {
  it('is a native checkbox labelled by its text and described by its description', () => {
    const el = cases({ case: 'checkbox' }).el
    const box = el.querySelector<HTMLElement>('.bound')!
    const input = inputOf(el, '.bound')
    expect(input.type).toBe('checkbox')
    expect(box.querySelector('label')!.htmlFor).toBe(input.id)
    const description = box.querySelector('.weft-checkbox__description')!
    expect(input.getAttribute('aria-describedby')).toBe(description.id)
    expect(box.dataset).toMatchObject({ state: 'unchecked', size: 'md', tone: 'primary' })
    expect(box.querySelector('.weft-checkbox__box')!.getAttribute('aria-hidden')).toBe('true')
  })

  it('ticks with v-model from a press or Space, and unticks again', async () => {
    const m = cases({ case: 'checkbox' })
    const input = inputOf(m.el, '.bound')
    input.click()
    await nextTick()
    expect(m.exposed.ticked).toBe(true)
    expect(input.checked).toBe(true)
    expect(m.el.querySelector<HTMLElement>('.bound')!.dataset.state).toBe('checked')
    expect(space(input)).toBe(true)
    await nextTick()
    expect(m.exposed.ticked).toBe(false)
    expect(input.checked).toBe(false)
  })

  it('stays as bound when the parent does not take the change', async () => {
    const m = cases({ case: 'checkbox', refuse: true })
    const input = inputOf(m.el, '.bound')
    input.click()
    await settle()
    expect(m.exposed.ticked).toBe(false)
    expect(input.checked).toBe(false)
  })

  it('works unbound', async () => {
    const el = cases({ case: 'checkbox' }).el
    const input = inputOf(el, '.free')
    input.click()
    await nextTick()
    expect(input.checked).toBe(true)
    expect(el.querySelector<HTMLElement>('.free')!.dataset.state).toBe('checked')
  })

  it('is indeterminate, drawn with a dash, and a press ticks it', async () => {
    const m = cases({ case: 'checkbox' })
    const input = inputOf(m.el, '.mixed')
    expect(input.indeterminate).toBe(true)
    expect(input.checked).toBe(false)
    expect(m.el.querySelector<HTMLElement>('.mixed')!.dataset.state).toBe('indeterminate')
    expect(m.el.querySelector('.mixed path')!.getAttribute('d')).toBe('M6 12h12')
    input.click()
    await nextTick()
    expect(m.exposed.all).toBe(true)
    expect(input.indeterminate).toBe(false)
    expect(input.checked).toBe(true)
  })

  it('cannot be ticked when disabled', async () => {
    const el = cases({ case: 'checkbox' }).el
    const input = inputOf(el, '.off')
    expect(input.disabled).toBe(true)
    expect(space(input)).toBe(false)
    expect(input.checked).toBe(false)
  })
})

describe('CheckboxGroup', () => {
  it('is a fieldset of boxes, ticked by the bound list, each posting under its name', () => {
    const el = cases({ case: 'group' }).el
    const group = el.querySelector<HTMLFieldSetElement>('.items')!
    expect(group.tagName).toBe('FIELDSET')
    expect(group.querySelector('legend')!.textContent).toBe('Fruit')
    const boxes = [...group.querySelectorAll<HTMLInputElement>('input')]
    expect(boxes.map((b) => [b.value, b.checked, b.disabled, b.name])).toEqual([
      ['apple', true, false, 'fruit'],
      ['pear', false, false, 'fruit'],
      ['fig', false, true, 'fruit'],
    ])
    expect([...group.querySelectorAll('label')].map((l) => l.textContent)).toEqual(['apple', 'pear', 'Fig'])
  })

  it('adds a ticked value to the list and removes an unticked one', async () => {
    const m = cases({ case: 'group' })
    const [apple, pear] = m.el.querySelectorAll<HTMLInputElement>('.items input')
    pear!.click()
    await nextTick()
    expect(m.exposed.fruits).toEqual(['apple', 'pear'])
    apple!.click()
    await nextTick()
    expect(m.exposed.fruits).toEqual(['pear'])
    // The slotted group shares the list.
    const slotted = [...m.el.querySelectorAll<HTMLInputElement>('.slotted input')]
    expect(slotted.map((b) => b.checked)).toEqual([false, true])
    slotted[0]!.click()
    await nextTick()
    expect(m.exposed.fruits).toEqual(['pear', 'apple'])
  })

  it('in a FormField is labelled by the field, described by its hint and error, and invalid', () => {
    const el = cases({ case: 'group' }).el
    const field = el.querySelector('.in-field')!
    const group = field.querySelector('fieldset')!
    expect(group.getAttribute('aria-labelledby')).toBe(field.querySelector('label.weft-form-field__label')!.id)
    expect(group.getAttribute('aria-describedby')).toBe(
      `${field.querySelector('.weft-form-field__hint')!.id} ${field.querySelector('.weft-form-field__error')!.id}`,
    )
    expect(group.getAttribute('aria-invalid')).toBe('true')
    // The boxes keep ids of their own, not the field's.
    const ids = [...el.querySelectorAll('input')].map((i) => i.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('RadioGroup', () => {
  const radios = (root: ParentNode, sel = '.plans') => [...root.querySelectorAll<HTMLInputElement>(`${sel} input`)]

  it('is a radiogroup of native radios sharing a name, with one tab stop on the chosen radio', async () => {
    const el = cases({ case: 'radio' }).el
    await nextTick()
    const group = el.querySelector('.plans')!
    expect(group.getAttribute('role')).toBe('radiogroup')
    const all = radios(el)
    expect(new Set(all.map((r) => r.name)).size).toBe(1)
    expect(all.map((r) => r.checked)).toEqual([false, true, false, false])
    expect(all.map((r) => r.getAttribute('tabindex'))).toEqual(['-1', '0', '-1', '-1'])
    expect(all[1]!.getAttribute('aria-describedby')).toBe(group.querySelector('.weft-radio__description')!.id)
    // With nothing chosen, the first radio takes the tab stop.
    expect(radios(el, '.none').map((r) => r.getAttribute('tabindex'))).toEqual(['0', '-1', '-1'])
  })

  it('moves to the next radio and chooses it with Down or Right, skipping disabled ones and wrapping', async () => {
    const m = cases({ case: 'radio' })
    await nextTick()
    const all = radios(m.el)
    all[1]!.focus()
    expect(key(all[1]!, 'ArrowDown').defaultPrevented).toBe(true)
    await nextTick()
    expect(document.activeElement).toBe(all[3])
    expect(m.exposed.plan).toBe('max')
    expect(all[3]!.checked).toBe(true)
    expect(all.map((r) => r.getAttribute('tabindex'))).toEqual(['-1', '-1', '-1', '0'])
    key(all[3]!, 'ArrowRight')
    await nextTick()
    expect(document.activeElement).toBe(all[0])
    expect(m.exposed.plan).toBe('free')
  })

  it('moves back with Up or Left', async () => {
    const m = cases({ case: 'radio' })
    await nextTick()
    const all = radios(m.el)
    all[1]!.focus()
    key(all[1]!, 'ArrowUp')
    await nextTick()
    expect(m.exposed.plan).toBe('free')
    key(all[0]!, 'ArrowLeft')
    await nextTick()
    expect(m.exposed.plan).toBe('max')
  })

  it('mirrors Left and Right in a right-to-left document', async () => {
    const m = cases({ case: 'radio', dir: 'rtl' })
    await nextTick()
    const all = radios(m.el)
    all[1]!.focus()
    key(all[1]!, 'ArrowLeft')
    await nextTick()
    expect(m.exposed.plan).toBe('max')
    key(all[3]!, 'ArrowRight')
    await nextTick()
    expect(m.exposed.plan).toBe('pro')
  })

  it('chooses nothing when focus only arrives, and chooses on a press or Space', async () => {
    const m = cases({ case: 'radio' })
    await nextTick()
    const none = radios(m.el, '.none')
    none[0]!.focus()
    await nextTick()
    expect(none.map((r) => r.checked)).toEqual([false, false, false])
    expect(space(none[2]!)).toBe(true)
    await nextTick()
    expect(none.map((r) => r.checked)).toEqual([false, false, true])
    radios(m.el)[0]!.click()
    await nextTick()
    expect(m.exposed.plan).toBe('free')
    expect(m.el.querySelector('.echo')!.textContent).toBe('free')
  })

  it('in a FormField is labelled by the field, invalid, required and described by the error', () => {
    const field = cases({ case: 'radio' }).el.querySelector('.in-field')!
    const group = field.querySelector('[role="radiogroup"]')!
    expect(group.getAttribute('aria-labelledby')).toBe(field.querySelector('.weft-form-field__label')!.id)
    expect(group.getAttribute('aria-invalid')).toBe('true')
    expect(group.getAttribute('aria-required')).toBe('true')
    expect(group.getAttribute('aria-describedby')).toBe(field.querySelector('.weft-form-field__error')!.id)
  })
})

describe('Switch', () => {
  it('is a native checkbox with role switch, labelled and described', () => {
    const el = cases({ case: 'switch' }).el
    const input = inputOf(el, '.bound')
    expect(input.type).toBe('checkbox')
    expect(input.getAttribute('role')).toBe('switch')
    expect(el.querySelector<HTMLLabelElement>('.bound label')!.htmlFor).toBe(input.id)
    expect(input.getAttribute('aria-describedby')).toBe(el.querySelector('.bound .weft-switch__description')!.id)
  })

  it('turns on and off with v-model, by press or Space, and emits change', async () => {
    const m = cases({ case: 'switch' })
    const input = inputOf(m.el, '.bound')
    input.click()
    await nextTick()
    expect(m.exposed.on).toBe(true)
    expect(m.el.querySelector<HTMLElement>('.bound')!.dataset.state).toBe('checked')
    space(input)
    await nextTick()
    expect(m.exposed.on).toBe(false)
    expect(m.exposed.changes).toBe(2)
  })

  it('while loading is busy and unavailable but focusable, and neither a press nor Space toggles it', async () => {
    const m = cases({ case: 'switch' })
    const input = inputOf(m.el, '.busy')
    expect(input.getAttribute('aria-busy')).toBe('true')
    expect(input.getAttribute('aria-disabled')).toBe('true')
    expect(input.disabled).toBe(false)
    input.click()
    space(input)
    m.el.querySelector<HTMLLabelElement>('.busy label')!.click()
    await nextTick()
    expect(m.exposed.busy).toBe(true)
    expect(input.checked).toBe(true)
    expect(document.activeElement).toBe(input)
    expect(m.el.querySelector('.busy .weft-switch__thumb .weft-spinner')).not.toBeNull()
  })

  it('takes its FormField\'s id, so the field label names it', () => {
    const field = cases({ case: 'switch' }).el.querySelector('.in-field')!
    expect(field.querySelector<HTMLLabelElement>('.weft-form-field__label')!.htmlFor).toBe(field.querySelector('input')!.id)
  })
})

describe('touch', () => {
  let unload: () => void
  beforeEach(() => { unload = loadStyles() })
  afterEach(() => unload())

  it.each(['checkbox', 'switch', 'radio'])('lays the %s input over its drawing at least --weft-hit-min square', (name) => {
    const hit = ruleFor(`.weft-${name}__input`)!
    expect(hit.style.getPropertyValue('position')).toBe('absolute')
    expect(hit.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    expect(hit.style.getPropertyValue('opacity')).toBe('0')
  })

  it('gives each row at least the hit height, so neighbouring hit areas do not overlap', () => {
    const rows = (input: string) => {
      const el = cases({ case: 'adaptive', input }).el
      return ['.c', '.s', '.r .weft-radio'].map((sel) => px(getComputedStyle(el.querySelector(sel)!).minBlockSize))
    }
    expect(rows('touch')).toEqual([44, 44, 44])
    expect(rows('pointer')).toEqual([24, 24, 24])
  })

  it.each(['checkbox__box', 'switch__track', 'radio__circle'])('rings the drawn %s when its input has keyboard focus', (drawn) => {
    const name = drawn.split('__')[0]
    const ring = ruleFor(`.weft-${name}__input:focus-visible + .weft-${drawn}`)!
    expect(ring.style.getPropertyValue('outline')).toContain('var(--weft-color-focus')
  })
})
