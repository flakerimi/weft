import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import MoreCases from '../fixtures/forms/MoreCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { key, px, type } from './helpers.ts'

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
function cases(props: Record<string, unknown>) {
  const m = mount(MoreCases, props)
  mounted.push(m)
  return m
}

describe('Textarea', () => {
  it('is a textarea bound with v-model, three rows by default', async () => {
    const m = cases({ case: 'textarea' })
    const area = m.el.querySelector<HTMLTextAreaElement>('.bound')!
    expect(area.tagName).toBe('TEXTAREA')
    expect(area.className).toBe('weft-textarea bound')
    expect(area.getAttribute('rows')).toBe('3')
    expect(area.value).toBe('first')
    expect(area.getAttribute('placeholder')).toBe('Notes')
    expect(area.id).toMatch(/^textarea-/)
    type(area, 'second\nline')
    expect(m.exposed.note).toBe('second\nline')
    m.exposed.note = 'from parent'
    await nextTick()
    expect(area.value).toBe('from parent')
  })

  it('takes its FormField\'s id and hint', async () => {
    const field = cases({ case: 'textarea' }).el.querySelector('.in-field')!
    await nextTick()
    const area = field.querySelector('textarea')!
    expect(field.querySelector('label')!.htmlFor).toBe(area.id)
    expect(area.getAttribute('aria-describedby')).toBe(field.querySelector('.weft-form-field__hint')!.id)
  })

  describe('autoresize', () => {
    /** Give the textarea a 20 px line and no padding or border, and say how tall its text is. */
    function sized(area: HTMLTextAreaElement, content: () => number) {
      area.style.lineHeight = '20px'
      area.style.padding = '0'
      area.style.border = '0'
      Object.defineProperty(area, 'scrollHeight', { configurable: true, get: content })
    }

    it('grows with its text and shrinks back', async () => {
      const m = cases({ case: 'textarea', autoresize: true })
      const area = m.el.querySelector<HTMLTextAreaElement>('.bound')!
      let height = 60
      sized(area, () => height)
      height = 140
      type(area, 'many\nlines')
      expect(area.style.blockSize).toBe('140px')
      expect(area.style.overflowY).toBe('hidden')
      height = 60
      m.exposed.note = 'short'
      await nextTick()
      await nextTick()
      expect(area.style.blockSize).toBe('60px')
      expect(area.dataset.autoresize).toBe('true')
    })

    it('stops at maxrows, then scrolls', () => {
      const area = cases({ case: 'textarea', autoresize: true, maxrows: 4 }).el.querySelector<HTMLTextAreaElement>('.bound')!
      sized(area, () => 300)
      type(area, 'a lot')
      expect(area.style.blockSize).toBe('80px')
      expect(area.style.overflowY).toBe('auto')
    })

    it('leaves the height alone without autoresize', () => {
      const area = cases({ case: 'textarea' }).el.querySelector<HTMLTextAreaElement>('.bound')!
      sized(area, () => 300)
      type(area, 'a lot')
      expect(area.style.blockSize).toBe('')
    })
  })
})

describe('InputNumber', () => {
  const field = (el: HTMLElement, sel: string) => el.querySelector<HTMLInputElement>(`${sel} input`)!

  it('is a spinbutton with its value, bounds, and two steppers out of the tab order', () => {
    const el = cases({ case: 'number' }).el
    const input = field(el, '.qty')
    expect(input.getAttribute('role')).toBe('spinbutton')
    expect(input.getAttribute('inputmode')).toBe('decimal')
    expect(input.value).toBe('2')
    expect(input.getAttribute('aria-valuenow')).toBe('2')
    expect(input.getAttribute('aria-valuemin')).toBe('0')
    expect(input.getAttribute('aria-valuemax')).toBe('10')
    expect(input.id).toMatch(/^input-number-/)
    const steps = [...el.querySelectorAll<HTMLButtonElement>('.qty button')]
    expect(steps.map((b) => [b.getAttribute('aria-label'), b.getAttribute('tabindex'), b.type])).toEqual([
      ['Decrease', '-1', 'button'],
      ['Increase', '-1', 'button'],
    ])
  })

  it('steps with Up and Down, ten steps with Page Up and Page Down, and goes to the ends with Home and End', async () => {
    const m = cases({ case: 'number' })
    const input = field(m.el, '.qty')
    expect(key(input, 'ArrowUp').defaultPrevented).toBe(true)
    expect(m.exposed.qty).toBe(3)
    key(input, 'ArrowDown')
    key(input, 'ArrowDown')
    expect(m.exposed.qty).toBe(1)
    key(input, 'PageUp')
    expect(m.exposed.qty).toBe(10)
    key(input, 'ArrowUp')
    expect(m.exposed.qty).toBe(10)
    key(input, 'Home')
    expect(m.exposed.qty).toBe(0)
    key(input, 'End')
    expect(m.exposed.qty).toBe(10)
    key(input, 'PageDown')
    expect(m.exposed.qty).toBe(0)
    await nextTick()
    expect(input.value).toBe('0')
    expect(input.getAttribute('aria-valuenow')).toBe('0')
  })

  it('reads text typed but not yet committed before it steps from it', () => {
    const m = cases({ case: 'number' })
    const input = field(m.el, '.qty')
    input.focus()
    type(input, '7')
    key(input, 'ArrowUp')
    expect(m.exposed.qty).toBe(8)
    type(input, '4')
    m.el.querySelectorAll<HTMLButtonElement>('.qty button')[0]!.click()
    expect(m.exposed.qty).toBe(3)
  })

  it('steps by decimals without floating-point dust', () => {
    const m = cases({ case: 'number' })
    const input = field(m.el, '.price')
    key(input, 'ArrowUp')
    key(input, 'ArrowUp')
    expect(m.exposed.price).toBe(0.3)
  })

  it('reads typed text when committed, keeping it within bounds, and goes back from text that is no number', async () => {
    const m = cases({ case: 'number' })
    const input = field(m.el, '.qty')
    input.focus()
    type(input, '7')
    expect(m.exposed.qty).toBe(2)
    input.blur()
    expect(m.exposed.qty).toBe(7)
    type(input, '99')
    key(input, 'Enter')
    expect(m.exposed.qty).toBe(10)
    await nextTick()
    expect(input.value).toBe('10')
    input.focus()
    type(input, 'lots')
    input.blur()
    await nextTick()
    expect(input.value).toBe('10')
    expect(m.exposed.qty).toBe(10)
    input.focus()
    type(input, '')
    input.blur()
    expect(m.exposed.qty).toBeNull()
  })

  it('reads typed text on change too, as when the field was left before the page was live', () => {
    const m = cases({ case: 'number' })
    const input = field(m.el, '.qty')
    type(input, '7')
    input.dispatchEvent(new Event('change', { bubbles: true }))
    expect(m.exposed.qty).toBe(7)
  })

  it('steps with its buttons, which disable at the bounds and keep focus where it was', async () => {
    const m = cases({ case: 'number' })
    const [down, up] = m.el.querySelectorAll<HTMLButtonElement>('.qty button')
    const press = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    down!.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(true)
    down!.click()
    down!.click()
    await nextTick()
    expect(m.exposed.qty).toBe(0)
    expect(down!.disabled).toBe(true)
    expect(up!.disabled).toBe(false)
  })

  it('starts from zero when stepped while empty', () => {
    const el = cases({ case: 'number' }).el
    const input = field(el, '.free')
    expect(input.value).toBe('')
    key(input, 'ArrowUp')
    expect(input.value).toBe('0')
  })

  it('ignores the mouse wheel unless wheel is set, and then only while focused', () => {
    const wheelUp = () => new WheelEvent('wheel', { deltaY: -100, bubbles: true, cancelable: true })
    const off = cases({ case: 'number' })
    const input = field(off.el, '.qty')
    input.focus()
    const ignored = wheelUp()
    input.dispatchEvent(ignored)
    expect(ignored.defaultPrevented).toBe(false)
    expect(off.exposed.qty).toBe(2)
    const on = cases({ case: 'number', wheel: true })
    const live = field(on.el, '.qty')
    live.dispatchEvent(wheelUp())
    expect(on.exposed.qty).toBe(2)
    live.focus()
    const taken = wheelUp()
    live.dispatchEvent(taken)
    expect(taken.defaultPrevented).toBe(true)
    expect(on.exposed.qty).toBe(3)
  })

  it('takes its FormField\'s id', async () => {
    const f = cases({ case: 'number' }).el.querySelector('.in-field')!
    await nextTick()
    expect(f.querySelector('label')!.htmlFor).toBe(f.querySelector('input')!.id)
  })
})

describe('Slider', () => {
  const thumbs = (el: HTMLElement, sel: string) => [...el.querySelectorAll<HTMLElement>(`${sel} [role="slider"]`)]
  /** Lay the slider's track out from x 0 to 200 (happy-dom does no layout). */
  function layOut(root: Element) {
    const track = root.querySelector('.weft-slider__track')!
    track.getBoundingClientRect = () => ({ left: 0, right: 200, width: 200, top: 0, bottom: 10, height: 10, x: 0, y: 0, toJSON() {} }) as DOMRect
  }
  const pointer = (type: string, x: number) =>
    new PointerEvent(type, { clientX: x, clientY: 5, button: 0, pointerId: 1, bubbles: true, cancelable: true })

  it('is a focusable slider thumb with its value and range, named by the Slider\'s aria-label', () => {
    const el = cases({ case: 'slider' }).el
    const [thumb] = thumbs(el, '.single')
    expect(thumb!.getAttribute('aria-label')).toBe('Volume')
    expect(el.querySelector('.single')!.hasAttribute('aria-label')).toBe(false)
    expect(thumb!.getAttribute('tabindex')).toBe('0')
    expect(thumb!.getAttribute('aria-valuenow')).toBe('40')
    expect(thumb!.getAttribute('aria-valuemin')).toBe('0')
    expect(thumb!.getAttribute('aria-valuemax')).toBe('100')
    expect(thumb!.getAttribute('aria-orientation')).toBe('horizontal')
    expect(thumb!.style.insetInlineStart).toBe('40%')
  })

  it('moves with the arrows, ten steps with Page Up and Page Down, and to the ends with Home and End', async () => {
    const m = cases({ case: 'slider' })
    const [thumb] = thumbs(m.el, '.single')
    expect(key(thumb!, 'ArrowRight').defaultPrevented).toBe(true)
    expect(m.exposed.volume).toBe(41)
    key(thumb!, 'ArrowUp')
    expect(m.exposed.volume).toBe(42)
    key(thumb!, 'ArrowLeft')
    key(thumb!, 'ArrowDown')
    expect(m.exposed.volume).toBe(40)
    key(thumb!, 'PageUp')
    expect(m.exposed.volume).toBe(50)
    key(thumb!, 'PageDown')
    key(thumb!, 'PageDown')
    expect(m.exposed.volume).toBe(30)
    key(thumb!, 'End')
    expect(m.exposed.volume).toBe(100)
    key(thumb!, 'ArrowRight')
    key(thumb!, 'Home')
    expect(m.exposed.volume).toBe(0)
    // Each key press is a finished change; a key at an end changes nothing.
    expect(m.exposed.changes).toEqual([41, 42, 41, 40, 50, 40, 30, 100, 0])
    await nextTick()
    expect(thumb!.getAttribute('aria-valuenow')).toBe('0')
  })

  it('swaps Left and Right in a right-to-left document', () => {
    const m = cases({ case: 'slider', dir: 'rtl' })
    const [thumb] = thumbs(m.el, '.single')
    key(thumb!, 'ArrowLeft')
    expect(m.exposed.volume).toBe(41)
    key(thumb!, 'ArrowRight')
    key(thumb!, 'ArrowRight')
    expect(m.exposed.volume).toBe(39)
  })

  it('draws a thumb per value for a range, which cannot pass each other, each bounded by its neighbour', async () => {
    const m = cases({ case: 'slider' })
    const [low, high] = thumbs(m.el, '.range')
    expect([low!.getAttribute('aria-label'), high!.getAttribute('aria-label')]).toEqual(['Minimum', 'Maximum'])
    expect([low!.getAttribute('aria-valuemin'), low!.getAttribute('aria-valuemax')]).toEqual(['0', '60'])
    expect([high!.getAttribute('aria-valuemin'), high!.getAttribute('aria-valuemax')]).toEqual(['20', '100'])
    key(low!, 'End')
    expect(m.exposed.span).toEqual([60, 60])
    key(low!, 'ArrowRight')
    expect(m.exposed.span).toEqual([60, 60])
    key(high!, 'Home')
    expect(m.exposed.span).toEqual([60, 60])
    key(high!, 'ArrowRight')
    expect(m.exposed.span).toEqual([60, 70])
    await nextTick()
    const fill = m.el.querySelector<HTMLElement>('.range .weft-slider__fill')!
    expect([fill.style.insetInlineStart, fill.style.inlineSize]).toEqual(['60%', '10%'])
  })

  it('moves the nearest thumb to a press on the track, on a step, and drags it until let go', async () => {
    const m = cases({ case: 'slider' })
    const root = m.el.querySelector('.range')!
    layOut(root)
    const [, high] = thumbs(m.el, '.range')
    root.dispatchEvent(pointer('pointerdown', 172))
    expect(m.exposed.span).toEqual([20, 90])
    expect(document.activeElement).toBe(high)
    root.dispatchEvent(pointer('pointermove', 100))
    expect(m.exposed.span).toEqual([20, 50])
    root.dispatchEvent(pointer('pointerup', 100))
    root.dispatchEvent(pointer('pointermove', 10))
    expect(m.exposed.span).toEqual([20, 50])
    expect(m.exposed.changes).toEqual([[20, 50]])
  })

  it('names a middle thumb by its place, so three thumbs read apart', async () => {
    const f = cases({ case: 'slider' }).el.querySelector('.three-field')!
    await nextTick()
    const label = f.querySelector('.weft-form-field__label')!
    const all = [...f.querySelectorAll<HTMLElement>('[role="slider"]')]
    expect(all.map((t) => t.getAttribute('aria-label'))).toEqual(['Minimum', 'Value 2 of 3', 'Maximum'])
    expect(all[1]!.getAttribute('aria-labelledby')).toBe(`${label.id} ${all[1]!.id}`)
  })

  it('names each thumb of a range in a FormField by the field\'s label and its own end', async () => {
    const f = cases({ case: 'slider' }).el.querySelector('.range-field')!
    await nextTick()
    const label = f.querySelector('.weft-form-field__label')!
    expect(label.tagName).toBe('SPAN')
    const [low, high] = [...f.querySelectorAll<HTMLElement>('[role="slider"]')]
    expect(low!.id).not.toBe('')
    expect(low!.id).not.toBe(high!.id)
    expect(low!.getAttribute('aria-labelledby')).toBe(`${label.id} ${low!.id}`)
    expect(low!.getAttribute('aria-label')).toBe('Minimum')
    expect(high!.getAttribute('aria-labelledby')).toBe(`${label.id} ${high!.id}`)
    expect(high!.getAttribute('aria-label')).toBe('Maximum')
  })

  it('reads a press from the right in a right-to-left document', () => {
    const m = cases({ case: 'slider', dir: 'rtl' })
    const root = m.el.querySelector('.single')!
    layOut(root)
    root.dispatchEvent(pointer('pointerdown', 50))
    expect(m.exposed.volume).toBe(75)
  })

  it('runs from the bottom when vertical: Up raises it, and a press reads from the bottom edge', async () => {
    const m = cases({ case: 'slider' })
    const root = m.el.querySelector('.tall')!
    const [thumb] = thumbs(m.el, '.tall')
    expect(thumb!.getAttribute('aria-orientation')).toBe('vertical')
    expect(thumb!.style.insetBlockEnd).toBe('25%')
    key(thumb!, 'ArrowUp')
    await nextTick()
    expect(thumb!.getAttribute('aria-valuenow')).toBe('26')
    root.querySelector('.weft-slider__track')!.getBoundingClientRect = () =>
      ({ left: 0, right: 10, width: 10, top: 0, bottom: 200, height: 200, x: 0, y: 0, toJSON() {} }) as DOMRect
    root.dispatchEvent(new PointerEvent('pointerdown', { clientX: 5, clientY: 50, button: 0, pointerId: 1, bubbles: true, cancelable: true }))
    await nextTick()
    expect(thumb!.getAttribute('aria-valuenow')).toBe('75')
    expect(thumb!.style.insetBlockEnd).toBe('75%')
  })

  it('takes no focus, key or press when disabled', async () => {
    const m = cases({ case: 'slider' })
    const root = m.el.querySelector('.off')!
    const [thumb] = thumbs(m.el, '.off')
    expect(thumb!.hasAttribute('tabindex')).toBe(false)
    expect(thumb!.getAttribute('aria-disabled')).toBe('true')
    key(thumb!, 'ArrowRight')
    layOut(root)
    root.dispatchEvent(pointer('pointerdown', 150))
    await nextTick()
    expect(thumb!.getAttribute('aria-valuenow')).toBe('5')
  })

  it('in a FormField is labelled by the field and posts its value under its name', async () => {
    const f = cases({ case: 'slider' }).el.querySelector('.in-field')!
    await nextTick()
    const thumb = f.querySelector('[role="slider"]')!
    // A thumb is not a labelable element, so the field's label is no <label>.
    const label = f.querySelector('.weft-form-field__label')!
    expect(thumb.getAttribute('aria-labelledby')).toBe(label.id)
    expect(label.tagName).toBe('SPAN')
    expect(label.hasAttribute('for')).toBe(false)
    const hidden = f.querySelector<HTMLInputElement>('input[type="hidden"]')!
    expect([hidden.name, hidden.value]).toEqual(['brightness', '0'])
  })
})

describe('PinInput', () => {
  const boxes = (el: HTMLElement, sel: string) => [...el.querySelectorAll<HTMLInputElement>(`${sel} input:not([type="hidden"])`)]

  it('is a group of single-character boxes, each named by its place', () => {
    const el = cases({ case: 'pin' }).el
    const group = el.querySelector('.code')!
    expect(group.getAttribute('role')).toBe('group')
    const all = boxes(el, '.code')
    expect(all).toHaveLength(4)
    expect(all.map((b) => b.getAttribute('aria-label'))).toEqual(['Character 1 of 4', 'Character 2 of 4', 'Character 3 of 4', 'Character 4 of 4'])
    expect(all[0]!.getAttribute('inputmode')).toBe('numeric')
    expect(all[0]!.getAttribute('autocomplete')).toBe('one-time-code')
    expect(all[1]!.getAttribute('autocomplete')).toBe('off')
    expect(all.every((b) => b.id !== '')).toBe(true)
    expect(new Set(all.map((b) => b.id)).size).toBe(4)
    expect(boxes(el, '.masked')[0]!.type).toBe('password')
  })

  it('fills a box and moves on as each character is typed, and fires complete when all are filled', () => {
    const m = cases({ case: 'pin' })
    const all = boxes(m.el, '.code')
    all[0]!.focus()
    type(all[0]!, '4')
    expect(m.exposed.code).toBe('4')
    expect(document.activeElement).toBe(all[1])
    type(all[1]!, 'x')
    expect(all[1]!.value).toBe('')
    expect(m.exposed.code).toBe('4')
    for (const [i, d] of [[1, '2'], [2, '7'], [3, '1']] as const) type(all[i]!, d)
    expect(m.exposed.code).toBe('4271')
    expect(m.exposed.completed).toEqual(['4271'])
  })

  it('clears with Backspace, stepping back from an empty box, and with Delete', () => {
    const m = cases({ case: 'pin' })
    const all = boxes(m.el, '.code')
    type(all[0]!, '1')
    type(all[1]!, '2')
    all[2]!.focus()
    key(all[2]!, 'Backspace')
    expect(m.exposed.code).toBe('1')
    expect(document.activeElement).toBe(all[1])
    key(all[0]!, 'Delete')
    expect(m.exposed.code).toBe('')
  })

  it('fills the boxes from a paste or an autofill, digits only', async () => {
    const m = cases({ case: 'pin' })
    const all = boxes(m.el, '.code')
    const paste = new Event('paste', { bubbles: true, cancelable: true }) as ClipboardEvent
    Object.defineProperty(paste, 'clipboardData', { value: { getData: () => '98-76' } })
    all[0]!.dispatchEvent(paste)
    expect(m.exposed.code).toBe('9876')
    await nextTick()
    expect(all.map((b) => b.value)).toEqual(['9', '8', '7', '6'])
    expect(m.exposed.completed).toEqual(['9876'])
    const other = cases({ case: 'pin' })
    type(boxes(other.el, '.code')[0]!, '1234')
    expect(other.exposed.code).toBe('1234')
  })

  it('moves between boxes with the arrows, mirrored right to left, and Home and End', () => {
    const all = boxes(cases({ case: 'pin' }).el, '.code')
    all[1]!.focus()
    key(all[1]!, 'ArrowRight')
    expect(document.activeElement).toBe(all[2])
    key(all[2]!, 'End')
    expect(document.activeElement).toBe(all[3])
    key(all[3]!, 'Home')
    expect(document.activeElement).toBe(all[0])
    const rtl = boxes(cases({ case: 'pin', dir: 'rtl' }).el, '.code')
    rtl[1]!.focus()
    key(rtl[1]!, 'ArrowRight')
    expect(document.activeElement).toBe(rtl[0])
  })

  it('in a FormField is labelled by the field, and posts the joined code under its name', async () => {
    const f = cases({ case: 'pin' }).el.querySelector('.in-field')!
    await nextTick()
    expect(f.querySelector('[role="group"]')!.getAttribute('aria-labelledby')).toBe(f.querySelector('label')!.id)
    const [first, second] = boxes(f as HTMLElement, '.weft-pin-input')
    expect(first!.id).toBe(f.querySelector('label')!.htmlFor)
    expect(second!.id).toMatch(/^pin-.*-1$/)
    type(first!, '5')
    await nextTick()
    const hidden = f.querySelector<HTMLInputElement>('input[type="hidden"]')!
    expect([hidden.name, hidden.value]).toEqual(['code', '5'])
  })
})

describe('FieldGroup', () => {
  it('is a group that gives its size to the form controls in it, unless they set their own', () => {
    const el = cases({ case: 'group' }).el
    const row = el.querySelector<HTMLElement>('.row')!
    expect(row.getAttribute('role')).toBe('group')
    expect(row.dataset).toMatchObject({ orientation: 'horizontal', size: 'sm' })
    expect(row.querySelector<HTMLElement>('.weft-input')!.dataset.size).toBe('sm')
    const sizes = [...el.querySelectorAll<HTMLElement>('.column .weft-input')].map((i) => i.dataset.size)
    expect(sizes).toEqual(['md', 'lg'])
  })

  describe('styles', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('joins its controls: only the outer corners are round, and borders overlap', () => {
      const el = cases({ case: 'group' }).el
      const [input, button] = el.querySelectorAll<HTMLElement>('.row > *')
      expect(getComputedStyle(input!).borderStartEndRadius).toBe('0')
      expect(getComputedStyle(button!).borderStartStartRadius).toBe('0')
      // Each overlaps the one before by the theme's border width (happy-dom leaves the calc() unsimplified; a browser gives -1px).
      expect(ruleFor('.weft-field-group[data-orientation="horizontal"] > * + *')!.style.getPropertyValue('margin-inline-start')).toBe('calc(-1 * var(--weft-border-width, 1px))')
      expect(getComputedStyle(button!).marginInlineStart).toMatch(/^(-1px|calc\(-1 \* 1px\))$/)
      expect(ruleFor('.weft-field-group > :focus-within')!.style.getPropertyValue('z-index')).toBe('1')
    })
  })
})

describe('touch', () => {
  let unload: () => void
  beforeEach(() => { unload = loadStyles() })
  afterEach(() => unload())
  const sizes = (input: string) => {
    const el = cases({ case: 'adaptive', input }).el
    const style = (sel: string) => getComputedStyle(el.querySelector(sel)!)
    return {
      box: px(style('.n').blockSize),
      stepper: px(style('.n .weft-input-number__step').inlineSize),
      slider: px(style('.s').blockSize),
      pin: px(style('.p .weft-pin-input__box').blockSize),
    }
  }

  it('makes InputNumber\'s steppers large buttons on touch: 44 px square in a 44 px box', () => {
    expect(sizes('touch')).toMatchObject({ box: 44, stepper: 44 })
    expect(sizes('pointer')).toMatchObject({ box: 36, stepper: 28 })
    const hit = ruleFor('.weft-input-number__step::before')!
    expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
  })

  it('grows the slider thumb\'s and track\'s hit areas to 44 px on touch, keeping the drawn thumb', () => {
    expect(sizes('touch').slider).toBe(44)
    expect(sizes('pointer').slider).toBe(24)
    for (const sel of ['.weft-slider__thumb::before', '.weft-slider__track::before']) {
      expect(ruleFor(sel)!.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    }
    expect(ruleFor('.weft-slider__thumb::before')!.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    expect(ruleFor('.weft-slider__thumb')!.style.getPropertyValue('inline-size')).toBe('var(--weft-slider-drawn, 16px)')
  })

  it('draws the slider thumb 24 px on touch (16 px on pointer), keeping the 44 px hit area', () => {
    const thumb = (input: string, sel: string) => {
      const el = cases({ case: 'small', input }).el
      return px(getComputedStyle(el.querySelector(`${sel} .weft-slider__thumb`)!).inlineSize)
    }
    expect(thumb('touch', '.s')).toBe(24)
    expect(thumb('pointer', '.s')).toBe(12)
    expect(thumb('touch', '.l')).toBe(24)
    expect(thumb('pointer', '.l')).toBe(20)
    const el = cases({ case: 'adaptive', input: 'touch' }).el
    expect(px(getComputedStyle(el.querySelector('.s .weft-slider__thumb')!).inlineSize)).toBe(24)
    expect(px(getComputedStyle(cases({ case: 'adaptive', input: 'pointer' }).el.querySelector('.s .weft-slider__thumb')!).inlineSize)).toBe(16)
  })

  describe('sm controls on touch', () => {
    const small = (input: string) => {
      const el = cases({ case: 'small', input }).el
      const style = (sel: string) => getComputedStyle(el.querySelector(sel)!)
      return {
        input: px(style('.i').blockSize),
        number: px(style('.n').blockSize),
        pin: px(style('.p .weft-pin-input__box').blockSize),
        pinWide: px(style('.p .weft-pin-input__box').inlineSize),
        slider: px(style('.s').blockSize),
      }
    }

    it('keeps the drawn box of an sm Input and InputNumber, and makes sm PinInput boxes and Sliders 44 px', () => {
      expect(small('touch')).toEqual({ input: 28, number: 28, pin: 44, pinWide: 44, slider: 44 })
      expect(small('pointer')).toEqual({ input: 28, number: 28, pin: 28, pinWide: 28, slider: 24 })
    })

    it('isolates the input box, so its hit area behind the field is painted above any ancestor background', () => {
      const el = cases({ case: 'small', input: 'touch' }).el
      expect(getComputedStyle(el.querySelector('.i')!).isolation).toBe('isolate')
      expect(getComputedStyle(el.querySelector('.n')!).isolation).toBe('isolate')
      expect(ruleFor('.weft-input::before')!.style.getPropertyValue('z-index')).toBe('-1')
    })

    it('gives sm InputNumber steppers a 44 px hit area on touch', () => {
      const el = cases({ case: 'small', input: 'touch' }).el
      expect(px(getComputedStyle(el.querySelector('.n .weft-input-number__step')!).inlineSize)).toBe(44)
    })

    it('keeps an sm Textarea at least 44 px tall on touch, whatever its rows', () => {
      const height = (input: string) =>
        px(getComputedStyle(cases({ case: 'small', input }).el.querySelector('.t')!).minBlockSize)
      expect(height('touch')).toBe(44)
      expect(height('pointer')).toBe(36)
    })
  })

  it('makes PinInput boxes 44 px on touch', () => {
    expect(sizes('touch').pin).toBe(44)
    expect(sizes('pointer').pin).toBe(36)
  })

  it('rings a focused slider thumb, textarea and pin box', () => {
    for (const sel of ['.weft-slider__thumb:focus-visible', '.weft-textarea:focus-visible', '.weft-pin-input__box:focus-visible']) {
      expect(ruleFor(sel)!.style.getPropertyValue('outline'), sel).toContain('var(--weft-color-focus')
    }
  })
})
