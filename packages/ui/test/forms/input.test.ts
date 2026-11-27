import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { neutral } from '@weft/themes'
import { registerIcon } from '../../src/basics/icons.ts'
import Input from '../../src/forms/Input.vue'
import InputCases from '../fixtures/forms/InputCases.vue'
import LabelSlotCases from '../fixtures/forms/LabelSlotCases.vue'
import { counts } from '../fixtures/forms/setup-counts.ts'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, settle, type } from './helpers.ts'

registerIcon('search', '<path d="M11 11"></path>')
registerIcon('mail', '<path d="M4 4"></path>')

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
function render(props: Record<string, unknown> = {}) {
  const m = mount(Input, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
function cases(props: Record<string, unknown>) {
  const m = mount(InputCases, props)
  mounted.push(m)
  return m
}

describe('Input', () => {
  it('is a text input in a box, md by default', () => {
    const box = render()
    expect(box.className).toBe('weft-input')
    expect(box.dataset.size).toBe('md')
    const input = box.querySelector('input')!
    expect(input.className).toBe('weft-input__field')
    expect(input.type).toBe('text')
    expect(input.value).toBe('')
  })

  it('gives attributes to the input, and class and style to the box', () => {
    const box = render({ class: 'wide', style: 'color: red', placeholder: 'Search', autocomplete: 'email', 'aria-label': 'Find', name: 'q' })
    expect(box.classList.contains('wide')).toBe(true)
    expect(box.getAttribute('style')).toContain('color: red')
    const input = box.querySelector('input')!
    expect(input.getAttribute('placeholder')).toBe('Search')
    expect(input.getAttribute('autocomplete')).toBe('email')
    expect(input.getAttribute('aria-label')).toBe('Find')
    expect(input.getAttribute('name')).toBe('q')
    expect(input.hasAttribute('class') && input.classList.contains('wide')).toBe(false)
    expect(input.hasAttribute('style')).toBe(false)
  })

  it('takes size, type, disabled, readonly and required', () => {
    const box = render({ size: 'lg', type: 'password', disabled: true, readonly: true, required: true })
    const input = box.querySelector('input')!
    expect(box.dataset).toMatchObject({ size: 'lg', disabled: 'true' })
    expect(input.type).toBe('password')
    expect(input.disabled).toBe(true)
    expect(input.readOnly).toBe(true)
    expect(input.required).toBe(true)
  })

  it('draws icons before and after the text, hidden from assistive technology', () => {
    const box = render({ icon: 'search', trailingIcon: 'mail' })
    expect(box.querySelector('.weft-input__leading svg')!.getAttribute('aria-hidden')).toBe('true')
    expect(box.querySelector('.weft-input__trailing svg')!.innerHTML).toBe('<path d="M4 4"></path>')
    expect(render().querySelector('.weft-input__leading, .weft-input__trailing')).toBeNull()
  })

  it('takes leading and trailing slots', () => {
    const box = cases({ case: 'ends' }).el.querySelector('.weft-input')!
    expect(box.querySelector('.weft-input__leading .lead')).not.toBeNull()
    expect(box.querySelector('.weft-input__trailing .clear')).not.toBeNull()
  })

  it('shows a spinner and is busy while loading, still editable', () => {
    const box = render({ loading: true, trailingIcon: 'mail' })
    const input = box.querySelector('input')!
    expect(input.getAttribute('aria-busy')).toBe('true')
    expect(input.disabled).toBe(false)
    expect(box.querySelector('.weft-input__trailing .weft-spinner')).not.toBeNull()
    expect(box.querySelector('.weft-input__trailing svg')).toBeNull()
  })

  describe('v-model', () => {
    it('shows the bound text and sends what is typed back', async () => {
      const m = cases({ case: 'model' })
      const input = m.el.querySelector<HTMLInputElement>('.bound input')!
      expect(input.value).toBe('hello')
      type(input, 'hello there')
      await nextTick()
      expect(m.el.querySelector('.echo')!.textContent).toBe('hello there')
    })

    it('follows the bound text when the parent changes it', async () => {
      const m = cases({ case: 'model' })
      m.exposed.text = 'from parent'
      await nextTick()
      expect(m.el.querySelector<HTMLInputElement>('.bound input')!.value).toBe('from parent')
    })

    it('works unbound, keeping what is typed', async () => {
      const input = cases({ case: 'model' }).el.querySelector<HTMLInputElement>('.free input')!
      type(input, 'mine')
      await nextTick()
      expect(input.value).toBe('mine')
    })

    it('trims with .trim, and shows the trimmed text once committed', async () => {
      const m = cases({ case: 'model' })
      const input = m.el.querySelector<HTMLInputElement>('.trim input')!
      type(input, '  ada  ')
      expect(m.exposed.trimmed).toBe('ada')
      input.dispatchEvent(new Event('change', { bubbles: true }))
      expect(input.value).toBe('ada')
    })

    it('gives a number with .number, null when empty, and the text when it is not a number', () => {
      const m = cases({ case: 'model' })
      const input = m.el.querySelector<HTMLInputElement>('.number input')!
      type(input, '42.5')
      expect(m.exposed.amount).toBe(42.5)
      type(input, '')
      expect(m.exposed.amount).toBeNull()
      type(input, 'abc')
      expect(m.exposed.amount).toBe('abc')
    })

    it('updates only on change with .lazy', () => {
      const m = cases({ case: 'model' })
      const input = m.el.querySelector<HTMLInputElement>('.lazy input')!
      type(input, 'later')
      expect(m.exposed.lazy).toBe('')
      input.dispatchEvent(new Event('change', { bubbles: true }))
      expect(m.exposed.lazy).toBe('later')
    })

    it('emits blur, focus and change', () => {
      const onBlur = vi.fn()
      const onFocus = vi.fn()
      const onChange = vi.fn()
      const input = render({ onBlur: () => onBlur, onFocus: () => onFocus, onChange: () => onChange }).querySelector('input')!
      input.focus()
      input.dispatchEvent(new Event('change', { bubbles: true }))
      input.blur()
      expect(onFocus).toHaveBeenCalledTimes(1)
      expect(onChange).toHaveBeenCalledTimes(1)
      expect(onBlur).toHaveBeenCalledTimes(1)
    })
  })

  describe('focus', () => {
    it('exposes focus() to a parent holding a ref', () => {
      const m = cases({ case: 'model' })
      ;(m.exposed as { focusField(): void }).focusField()
      expect(document.activeElement).toBe(m.el.querySelector('.exposed input'))
    })

    it('focuses the text when the box around it is pressed, but leaves a button in it alone', () => {
      const box = cases({ case: 'ends' }).el.querySelector<HTMLElement>('.weft-input')!
      const lead = box.querySelector('.lead')!
      const down = new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0 })
      lead.dispatchEvent(down)
      expect(down.defaultPrevented).toBe(true)
      expect(document.activeElement).toBe(box.querySelector('input'))
      const onButton = new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0 })
      box.querySelector('.clear')!.dispatchEvent(onButton)
      expect(onButton.defaultPrevented).toBe(false)
      const onText = new MouseEvent('mousedown', { bubbles: true, cancelable: true, button: 0 })
      box.querySelector('input')!.dispatchEvent(onText)
      expect(onText.defaultPrevented).toBe(false)
    })
  })
})

describe('FormField', () => {
  it('labels its control, with a description under the label and a hint beside it, both describing it', async () => {
    const el = cases({ case: 'field' }).el
    await settle()
    const field = el.querySelector('.weft-form-field')!
    const label = field.querySelector('label')!
    const input = field.querySelector('input')!
    expect(label.textContent).toContain('Email')
    expect(input.id).not.toBe('')
    expect(label.htmlFor).toBe(input.id)
    const head = field.querySelector('.weft-form-field__head')!
    expect([...head.children].map((c) => c.className)).toEqual(['weft-form-field__label', 'weft-form-field__hint'])
    const hint = field.querySelector('.weft-form-field__hint')!
    expect(hint.textContent).toBe('Optional')
    const description = field.querySelector('.weft-form-field__description')!
    expect(description.textContent).toBe('We never share it.')
    expect(description.previousElementSibling).toBe(head)
    // Its own aria-describedby is kept, then the field's: description, hint.
    expect(input.getAttribute('aria-describedby')).toBe(`extra ${description.id} ${hint.id}`)
    expect(input.hasAttribute('aria-invalid')).toBe(false)
    expect(field.querySelector('.weft-form-field__error')).toBeNull()
  })

  it('mounted in the browser, draws its label once, the right element, its slot set up once', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    const added = new Set<Element>()
    const watch = new MutationObserver((records) => {
      for (const r of records) for (const n of r.addedNodes) {
        if (n instanceof Element) for (const l of [n, ...n.querySelectorAll('.weft-form-field__label')]) if (l.classList.contains('weft-form-field__label')) added.add(l)
      }
    })
    watch.observe(host, { childList: true, subtree: true })
    const m = mount(LabelSlotCases)
    host.appendChild(m.el)
    await settle()
    watch.disconnect()
    // Never a <label> over the radio group, not even for a moment.
    expect(m.el.querySelector('.plan .weft-form-field__label')!.tagName).toBe('SPAN')
    expect(m.el.querySelector('.email .weft-form-field__label')!.tagName).toBe('LABEL')
    // The two labels are the only ones ever drawn.
    expect([...added].map((l) => l.tagName).sort()).toEqual(['LABEL', 'SPAN'])
    for (const l of added) expect(l.isConnected).toBe(true)
    expect(counts).toEqual({ plan: 1, email: 1 })
    m.unmount()
    host.remove()
  })

  it('is a <label> whose control is its labelable control', async () => {
    const el = cases({ case: 'field' }).el
    await settle()
    const label = el.querySelector<HTMLLabelElement>('.weft-form-field__label')!
    expect(label.tagName).toBe('LABEL')
    // A press on it focuses that control (in a browser: site-src's TestFieldLabelsInChrome).
    expect(label.control).toBe(el.querySelector('input'))
  })

  it('marks a required field with an asterisk read only by sight, and the control as aria-required', async () => {
    const el = cases({ case: 'field' }).el
    await settle()
    const mark = el.querySelector('.weft-form-field__required')!
    expect(mark.textContent).toBe('*')
    expect(mark.getAttribute('aria-hidden')).toBe('true')
    expect(el.querySelector('input')!.getAttribute('aria-required')).toBe('true')
  })

  it('shows an error under the control, describing it, and marks the control invalid', () => {
    const el = cases({ case: 'field', error: 'Enter an email address.' }).el
    const input = el.querySelector('input')!
    const error = el.querySelector('.weft-form-field__error')!
    expect(error.textContent!.trim()).toBe('Enter an email address.')
    expect(input.getAttribute('aria-invalid')).toBe('true')
    expect(input.getAttribute('aria-describedby')!.split(' ')).toEqual([
      'extra',
      el.querySelector('.weft-form-field__description')!.id,
      el.querySelector('.weft-form-field__hint')!.id,
      error.id,
    ])
    expect(el.querySelector('.weft-form-field')!.getAttribute('data-invalid')).toBe('true')
    expect(el.querySelector('.weft-input')!.getAttribute('data-invalid')).toBe('true')
  })

  it('is invalid without a sentence when error is true, or shows the error slot', () => {
    const plain = cases({ case: 'field', error: true }).el
    expect(plain.querySelector('input')!.getAttribute('aria-invalid')).toBe('true')
    expect(plain.querySelector('.weft-form-field__error')).toBeNull()
    const slotted = cases({ case: 'error-slot' }).el
    expect(slotted.querySelector('.weft-form-field__error .why')!.textContent).toBe('Ask your admin.')
    expect(slotted.querySelector('input')!.getAttribute('aria-describedby')).toBe(slotted.querySelector('.weft-form-field__error')!.id)
  })

  it('gives its default slot the error', () => {
    expect(cases({ case: 'slot-error' }).el.querySelector('input')!.getAttribute('placeholder')).toBe('Too short.')
  })

  it('describes nothing when it has neither description, hint nor error, and gives the control its name', () => {
    const el = cases({ case: 'bare-field' }).el
    expect(el.querySelector('input')!.hasAttribute('aria-describedby')).toBe(false)
    expect(cases({ case: 'field' }).el.querySelector('input')!.getAttribute('name')).toBe('email')
  })

  it('renders no empty style attribute on the box when none is given', () => {
    expect(render().hasAttribute('style')).toBe(false)
    expect(render({ style: 'color: red' }).getAttribute('style')).toContain('color: red')
  })
})

describe('touch', () => {
  let unload: () => void
  beforeEach(() => { unload = loadStyles() })
  afterEach(() => unload())
  const heights = (input: string, size: string) => {
    const el = cases({ case: 'adaptive', input, size }).el
    const h = (sel: string) => px(getComputedStyle(el.querySelector(sel)!).blockSize)
    return { input: h('.plain'), button: h('.button') }
  }

  it('is 44 px tall at md on touch, as Button is, and its control height on pointer', () => {
    expect(heights('touch', 'md')).toEqual({ input: 44, button: 44 })
    expect(heights('pointer', 'md')).toEqual({ input: 36, button: 36 })
  })

  it('keeps sm and lg at their control sizes, with a hit area of at least --weft-hit-min', () => {
    expect(heights('touch', 'sm')).toEqual({ input: 28, button: 28 })
    expect(heights('touch', 'lg')).toEqual({ input: 44, button: 44 })
    const hit = ruleFor('.weft-input::before')!
    expect(hit.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    // Behind the field, so a press on the text still places the caret.
    expect(hit.style.getPropertyValue('z-index')).toBe('-1')
  })

  it('takes its heights from the theme: a theme that changes md moves Input and Button together', () => {
    unload()
    unload = loadStyles({ ...neutral, control: { sm: 30, md: 40, lg: 48 } })
    expect(heights('pointer', 'md')).toEqual({ input: 40, button: 40 })
    expect(heights('touch', 'md')).toEqual({ input: 44, button: 44 })
    expect(heights('pointer', 'sm')).toEqual({ input: 30, button: 30 })
    expect(heights('pointer', 'lg')).toEqual({ input: 48, button: 48 })
  })

  it('shows focus on the box', () => {
    const ring = ruleFor('.weft-input:focus-within')!
    expect(ring.style.getPropertyValue('outline')).toContain('var(--weft-color-focus')
  })
})
