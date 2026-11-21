import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { registerIcon } from '../src/basics/icons.ts'
import Button from '../src/basics/Button.vue'
import ButtonCases from './fixtures/basics/ButtonCases.vue'
import { mount } from './mount.ts'
import { loadStyles, ruleFor } from './styles.ts'

registerIcon('plus', '<path d="M5 12h14"></path>')
registerIcon('arrow-right', '<path d="M12 5l7 7"></path>')

let mounted: { unmount(): void }[] = []
afterEach(() => {
  mounted.forEach((m) => m.unmount())
  mounted = []
})
/** Mount a Button. A function prop of a root component is read as a getter, so pass handlers as `() => handler`. */
function render(props: Record<string, unknown>) {
  const m = mount(Button, props)
  mounted.push(m)
  return m.el.firstElementChild as HTMLElement
}
function cases(props: Record<string, unknown>) {
  const m = mount(ButtonCases, props)
  mounted.push(m)
  return m.el
}

describe('Button', () => {
  it('is a native button of type button, solid primary md, by default', () => {
    const b = render({ label: 'Save' })
    expect(b.tagName).toBe('BUTTON')
    expect(b.getAttribute('type')).toBe('button')
    expect(b.className).toBe('weft-button')
    expect(b.dataset).toMatchObject({ variant: 'solid', tone: 'primary', size: 'md' })
    expect(b.textContent).toBe('Save')
    expect(b.hasAttribute('disabled')).toBe(false)
    expect(b.hasAttribute('aria-busy')).toBe(false)
  })

  it('takes variant, tone, size, block and type', () => {
    const b = render({ label: 'Delete', variant: 'outline', tone: 'danger', size: 'lg', block: true, type: 'submit' })
    expect(b.dataset).toMatchObject({ variant: 'outline', tone: 'danger', size: 'lg', block: 'true' })
    expect(b.getAttribute('type')).toBe('submit')
  })

  it('draws a leading icon and a trailing icon around its label, hidden from assistive technology', () => {
    const b = render({ label: 'Next', icon: 'plus', trailingIcon: 'arrow-right' })
    const icons = b.querySelectorAll('svg.weft-icon')
    expect(icons).toHaveLength(2)
    expect(icons[0]!.innerHTML).toBe('<path d="M5 12h14"></path>')
    expect(icons[1]!.innerHTML).toBe('<path d="M12 5l7 7"></path>')
    for (const i of icons) expect(i.getAttribute('aria-hidden')).toBe('true')
    const parts = Array.from(b.querySelector('.weft-button__content')!.children)
    expect(parts.map((p) => p.getAttribute('class'))).toEqual(['weft-icon', 'weft-button__label', 'weft-icon'])
    expect(b.hasAttribute('data-square')).toBe(false)
  })

  it('is square with an icon and no label', () => {
    const b = render({ icon: 'plus', 'aria-label': 'Add' })
    expect(b.dataset.square).toBe('true')
    expect(b.getAttribute('aria-label')).toBe('Add')
    expect(b.querySelector('.weft-button__label')).toBeNull()
  })

  it('puts its default slot in place of the label', () => {
    const b = cases({ case: 'slot' }).querySelector('button')!
    expect(b.querySelector('.mine')?.textContent).toBe('Add')
    expect(b.textContent).toBe('Add')
    expect(b.hasAttribute('data-square')).toBe(false)
  })

  it('passes attributes such as aria-label and class to its element', () => {
    const b = render({ label: 'x', class: 'extra', 'data-test': 'go' })
    expect(b.classList.contains('weft-button')).toBe(true)
    expect(b.classList.contains('extra')).toBe(true)
    expect(b.dataset.test).toBe('go')
  })

  it('emits click when pressed', () => {
    const onClick = vi.fn()
    const b = render({ label: 'Go', onClick: () => onClick })
    b.click()
    expect(onClick).toHaveBeenCalledTimes(1)
    expect(onClick.mock.calls[0]![0]).toBeInstanceOf(MouseEvent)
  })

  describe('disabled', () => {
    it('is natively disabled, so mouse, keyboard and form submit cannot activate it', () => {
      const onClick = vi.fn()
      const b = render({ label: 'Go', disabled: true, onClick: () => onClick })
      expect((b as HTMLButtonElement).disabled).toBe(true)
      b.click()
      expect(onClick).not.toHaveBeenCalled()
    })
  })

  describe('loading', () => {
    it('is busy and unavailable but keeps focus, its content and so its width', () => {
      const b = render({ label: 'Save', icon: 'plus', loading: true })
      expect(b.getAttribute('aria-busy')).toBe('true')
      expect(b.getAttribute('aria-disabled')).toBe('true')
      // Not natively disabled, which would drop focus from a button pressed by keyboard.
      expect((b as HTMLButtonElement).disabled).toBe(false)
      expect(b.dataset.loading).toBe('true')
      // The content stays in place (hidden by CSS) so the button keeps its width.
      expect(b.querySelector('.weft-button__label')?.textContent).toBe('Save')
      const spinner = b.querySelector('.weft-spinner')!
      expect(spinner.getAttribute('aria-hidden')).toBe('true')
    })

    it('blocks a click, from the mouse or from a keyboard press, and the form submit it would cause', () => {
      const onPress = vi.fn()
      const onSubmit = vi.fn()
      const el = cases({ case: 'form', loading: true, onPress: () => onPress, onSubmit: () => onSubmit })
      const b = el.querySelector('button')!
      // Enter and Space on a native button reach it as a click (detail 0).
      const keyboard = new MouseEvent('click', { bubbles: true, cancelable: true, detail: 0 })
      b.dispatchEvent(keyboard)
      expect(keyboard.defaultPrevented).toBe(true)
      b.click()
      expect(onPress).not.toHaveBeenCalled()
      expect(onSubmit).not.toHaveBeenCalled()
    })

    it('submits its form once it is not loading', () => {
      const onPress = vi.fn()
      const onSubmit = vi.fn()
      const el = cases({ case: 'form', onPress: () => onPress, onSubmit: () => onSubmit })
      el.querySelector('button')!.click()
      expect(onPress).toHaveBeenCalledTimes(1)
      expect(onSubmit).toHaveBeenCalledTimes(1)
    })

    it('follows the loading prop live', async () => {
      const loading = ref(false)
      const onClick = vi.fn()
      const b = render({ label: 'Save', loading: () => loading.value, onClick: () => onClick })
      expect(b.querySelector('.weft-spinner')).toBeNull()
      loading.value = true
      await nextTick()
      expect(b.getAttribute('aria-busy')).toBe('true')
      b.click()
      expect(onClick).not.toHaveBeenCalled()
      loading.value = false
      await nextTick()
      expect(b.hasAttribute('aria-busy')).toBe(false)
      expect(b.hasAttribute('aria-disabled')).toBe(false)
      b.click()
      expect(onClick).toHaveBeenCalledTimes(1)
    })
  })

  describe('with to', () => {
    it('is a link to that address, with no type', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', onClick: () => onClick })
      expect(a.tagName).toBe('A')
      expect(a.getAttribute('href')).toBe('/docs')
      expect(a.hasAttribute('type')).toBe(false)
      expect(a.className).toBe('weft-button')
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(onClick).toHaveBeenCalledTimes(1)
      expect(press.defaultPrevented).toBe(false)
    })

    it('to another site gets rel="noopener", as a Link does', () => {
      expect(render({ label: 'Out', to: 'https://example.com' }).getAttribute('rel')).toBe('noopener')
      expect(render({ label: 'In', to: '/docs' }).hasAttribute('rel')).toBe(false)
    })

    it('when disabled, drops its address and leaves the tab order, announced as a disabled link', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', disabled: true, onClick: () => onClick })
      expect(a.hasAttribute('href')).toBe(false)
      expect(a.getAttribute('role')).toBe('link')
      expect(a.getAttribute('aria-disabled')).toBe('true')
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(press.defaultPrevented).toBe(true)
      expect(onClick).not.toHaveBeenCalled()
    })

    it('when loading, keeps focus but does not navigate', () => {
      const onClick = vi.fn()
      const a = render({ label: 'Docs', to: '/docs', loading: true, onClick: () => onClick })
      expect(a.getAttribute('href')).toBe('/docs')
      expect(a.getAttribute('aria-busy')).toBe('true')
      const press = new MouseEvent('click', { bubbles: true, cancelable: true })
      a.dispatchEvent(press)
      expect(press.defaultPrevented).toBe(true)
      expect(onClick).not.toHaveBeenCalled()
    })
  })

  describe('keyboard and focus', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('is in the tab order and shows a focus ring when focused', () => {
      const b = render({ label: 'Go' })
      expect(b.tabIndex).toBe(0)
      b.focus()
      expect(document.activeElement).toBe(b)
      expect(getComputedStyle(b).outlineStyle).toBe('solid')
      expect(getComputedStyle(b).outlineWidth).toBe('2px')
    })

    it('stays in the tab order while loading, so keyboard focus is not lost', () => {
      const b = render({ label: 'Go', loading: true })
      b.focus()
      expect(document.activeElement).toBe(b)
    })
  })

  describe('touch', () => {
    let unload: () => void
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => unload())

    it('grows its hit area to --weft-hit-min with a pseudo-element, centred on the button', () => {
      const hit = ruleFor('.weft-button::before')!
      expect(hit).toBeDefined()
      expect(hit.style.getPropertyValue('content')).toMatch(/^(""|'')$/)
      expect(hit.style.getPropertyValue('position')).toBe('absolute')
      expect(hit.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(hit.style.getPropertyValue('block-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
      expect(ruleFor('.weft-button')!.style.getPropertyValue('position')).toBe('relative')
    })

    it('has a hit area of 44 px under touch and keeps its drawn size', () => {
      const probe = (input: string) => {
        const el = cases({ case: 'adaptive', input })
        const button = el.querySelector('button')!
        // The probe, inside the button, resolves --weft-hit-min where the pseudo-element does.
        const hit = getComputedStyle(el.querySelector('.hit-probe')!).minBlockSize
        return { button: getComputedStyle(button).blockSize, hit }
      }
      const touch = probe('touch')
      const pointer = probe('pointer')
      expect(touch.button).toBe('28px')
      expect(pointer.button).toBe('28px')
      expect(parseFloat(touch.hit)).toBeGreaterThanOrEqual(44)
      expect(parseFloat(pointer.hit)).toBe(24)
    })
  })
})
