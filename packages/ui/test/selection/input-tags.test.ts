import { afterEach, describe, expect, it } from 'vitest'
import InputTagsCases from '../fixtures/selection/InputTagsCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px, type } from '../forms/helpers.ts'
import { press as pressKey } from '../keyboard.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
type Exposed = { tags: unknown[]; log: string[]; submitted: number }
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(InputTagsCases, props)
  unmount = m.unmount
  await settle()
  return { m, e: m.exposed as unknown as Exposed }
}
const field = () => $<HTMLInputElement>('.weft-input-tags input.weft-input__field')
const tags = () => $$('.weft-tag')
const texts = () => tags().map((t) => t.querySelector('.weft-tag__text')!.textContent)
async function add(text: string, k = 'Enter') {
  field().focus()
  type(field(), text)
  const e = key(k)
  await settle()
  return e
}

describe('InputTags', () => {
  it('is a text field in an Input box, its tags a list before it, labelled by its field', async () => {
    await cases({ initial: ['vue', 'go'] })
    expect($('label').getAttribute('for')).toBe(field().id)
    expect(field().getAttribute('aria-describedby')).toBe($('.weft-form-field__hint').id)
    expect(field().hasAttribute('name')).toBe(false)
    const list = $('[role="list"]')
    expect($$('[role="listitem"]', list)).toHaveLength(2)
    expect(texts()).toEqual(['vue', 'go'])
    expect($$<HTMLInputElement>('input[type="hidden"]').map((i) => [i.name, i.value])).toEqual([['tags', 'vue'], ['tags', 'go']])
  })

  it('adds the trimmed text on Enter without submitting the form; Enter in an empty field submits', async () => {
    const { e } = await cases()
    const enter = await add('  rust  ')
    expect(enter.defaultPrevented).toBe(true)
    expect(e.tags).toEqual(['rust'])
    expect(field().value).toBe('')
    expect(e.log).toEqual(['add:rust'])
    expect(pressKey(field(), 'Enter')).toBe(true)
    await settle()
    expect(e.submitted).toBe(1)
  })

  it('adds on the delimiter as you type, and from a paste with add-on-paste', async () => {
    const { e } = await cases({ addOnPaste: true })
    field().focus()
    type(field(), 'a, b,c')
    await settle()
    expect(e.tags).toEqual(['a', 'b'])
    expect(field().value).toBe('c')
    type(field(), '')
    const paste = new Event('paste', { bubbles: true, cancelable: true }) as Event & { clipboardData: unknown }
    paste.clipboardData = { getData: () => 'x,y, z' }
    field().dispatchEvent(paste)
    await settle()
    expect(paste.defaultPrevented).toBe(true)
    expect(e.tags).toEqual(['a', 'b', 'x', 'y', 'z'])
  })

  it('with add-on-paste, inserts a paste at the caret and makes the whole text tags, draft or none', async () => {
    const { e } = await cases({ addOnPaste: true })
    const paste = (text: string) => {
      const event = new Event('paste', { bubbles: true, cancelable: true }) as Event & { clipboardData: unknown }
      event.clipboardData = { getData: () => text }
      field().dispatchEvent(event)
      return event
    }
    field().focus()
    expect(paste('solo').defaultPrevented).toBe(true)
    await settle()
    expect(e.tags).toEqual(['solo'])
    type(field(), 'ac')
    field().setSelectionRange(1, 1)
    paste('b')
    await settle()
    expect(e.tags).toEqual(['solo', 'abc'])
    type(field(), 'd')
    paste(',e,f')
    await settle()
    expect(e.tags).toEqual(['solo', 'abc', 'd', 'e', 'f'])
    expect(field().value).toBe('')
  })

  it('refuses a duplicate and past max, emitting invalid', async () => {
    const { e } = await cases({ initial: ['a'], max: 2 })
    await add('a')
    expect(e.tags).toEqual(['a'])
    expect(field().value).toBe('a')
    await add('b')
    await add('c')
    expect(e.tags).toEqual(['a', 'b'])
    expect(e.log).toEqual(['invalid:a', 'add:b', 'invalid:c'])
  })

  it('takes duplicates with duplicate, and converts and displays values', async () => {
    const { e } = await cases({ duplicate: true, upper: true })
    await add('a')
    await add('a')
    expect(e.tags).toEqual(['A', 'A'])
    expect(texts()).toEqual(['#A', '#A'])
  })

  it('adds on Tab and on leaving, when asked', async () => {
    const { e } = await cases({ addOnTab: true, addOnBlur: true })
    const tab = await add('one', 'Tab')
    expect(tab.defaultPrevented).toBe(true)
    expect(e.tags).toEqual(['one'])
    expect(key('Tab').defaultPrevented).toBe(false) // nothing typed: Tab moves on
    type(field(), 'two')
    field().dispatchEvent(new FocusEvent('blur', { relatedTarget: $('.after') }))
    await settle()
    expect(e.tags).toEqual(['one', 'two'])
  })

  it('moves focus to the tags with Backspace and the arrows, and takes out the focused tag with Backspace or Delete', async () => {
    const { e } = await cases({ initial: ['a', 'b', 'c'] })
    field().focus()
    key('Backspace')
    await settle()
    expect(active()).toBe(tags()[2])
    expect(e.tags).toEqual(['a', 'b', 'c'])
    key('ArrowLeft')
    expect(active()).toBe(tags()[1])
    key('ArrowRight')
    key('ArrowRight')
    expect(active()).toBe(field())
    key('ArrowLeft')
    key('Backspace')
    await settle()
    expect(e.tags).toEqual(['a', 'b'])
    expect(active()).toBe(tags()[1])
    key('ArrowLeft')
    key('Delete')
    await settle()
    expect(e.tags).toEqual(['b'])
    expect(active()).toBe(tags()[0])
    key('Delete')
    await settle()
    expect(e.tags).toEqual([])
    expect(active()).toBe(field())
    expect(e.log).toEqual(['remove:c', 'remove:a', 'remove:b'])
  })

  it('mirrors the arrows right to left', async () => {
    await cases({ initial: ['a', 'b'], dir: 'rtl' })
    field().focus()
    key('ArrowRight')
    expect(active()).toBe(tags()[1])
    key('ArrowRight')
    expect(active()).toBe(tags()[0])
    key('ArrowLeft')
    key('ArrowLeft')
    expect(active()).toBe(field())
  })

  it('leaves the caret alone when there is text before it', async () => {
    await cases({ initial: ['a'] })
    field().focus()
    type(field(), 'xy')
    field().setSelectionRange(2, 2)
    expect(key('ArrowLeft').defaultPrevented).toBe(false)
    expect(key('Backspace').defaultPrevented).toBe(false)
    expect(active()).toBe(field())
  })

  it('takes out a tag with its × button, named for it, focus going back to the field', async () => {
    const { e } = await cases({ initial: ['a', 'b'] })
    const button = $<HTMLButtonElement>('.weft-tag__remove', tags()[0])
    expect(button.getAttribute('aria-label')).toBe('Remove a')
    expect(button.tabIndex).toBe(-1)
    button.click()
    await settle()
    expect(e.tags).toEqual(['b'])
    expect(active()).toBe(field())
  })

  it('draws each tag through the item-label slot, and through its old name item-text', async () => {
    for (const [custom, text] of [['item-label', ['0:vue', '1:vite']], ['item-text', ['0=vue', '1=vite']], [undefined, ['vue', 'vite']]] as const) {
      const m = mount(InputTagsCases, { initial: ['vue', 'vite'], custom })
      expect([...m.el.querySelectorAll('.weft-tag__text')].map((t) => t.textContent!.trim()), String(custom)).toEqual(text)
      m.unmount()
    }
  })

  it('does nothing while disabled', async () => {
    const { e } = await cases({ initial: ['a'], disabled: true })
    expect(field().disabled).toBe(true)
    expect($<HTMLButtonElement>('.weft-tag__remove').disabled).toBe(true)
    $<HTMLButtonElement>('.weft-tag__remove').click()
    key('Delete', {}, tags()[0])
    await settle()
    expect(e.tags).toEqual(['a'])
  })

  it('shows the focus ring on a focused tag, and on the box while the field has focus', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ initial: ['a'] })
      const tag = ruleFor('.weft-tag:focus-visible')!.style
      expect(tag.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
      expect(ruleFor('.weft-input:focus-within')!.style.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
      // And it applies to a focused tag (happy-dom computes :focus-visible, not :focus-within).
      tags()[0]!.focus()
      expect(tags()[0]!.matches(':focus-visible')).toBe(true)
      const drawn = getComputedStyle(tags()[0]!)
      expect(drawn.outlineStyle).toBe('solid')
      expect(drawn.outlineColor).toBe(getComputedStyle(document.documentElement).getPropertyValue('--weft-color-focus').trim())
    } finally {
      unstyle()
    }
  })

  it('is at least 44 px tall on touch, its × buttons with a hit area of at least --weft-hit-min', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch', initial: ['a'] })
      expect(px(getComputedStyle($('.weft-input-tags')).minBlockSize)).toBe(44)
      expect(ruleFor('.weft-tag__remove::before')!.style.getPropertyValue('inline-size')).toBe('max(100%, var(--weft-hit-min, 24px))')
    } finally {
      unstyle()
    }
  })
})
