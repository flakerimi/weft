import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import Icon from '../src/basics/Icon.vue'
import { iconBody, registerIcon } from '../src/basics/icons.ts'
import { mount } from './mount.ts'

const TRASH = '<path d="M3 6h18"></path><path d="M8 6V4h8v2"></path>'
registerIcon('trash', TRASH)

afterEach(() => vi.restoreAllMocks())

const svg = (el: HTMLElement) => el.querySelector('svg') as SVGSVGElement

describe('Icon', () => {
  it('draws the bundled Lucide icon as an inline svg that follows the text colour', () => {
    const { el, unmount } = mount(Icon, { name: 'trash' })
    const s = svg(el)
    expect(s.getAttribute('class')).toBe('weft-icon')
    expect(s.getAttribute('viewBox')).toBe('0 0 24 24')
    expect(s.getAttribute('stroke')).toBe('currentColor')
    expect(s.getAttribute('fill')).toBe('none')
    expect(s.innerHTML).toBe(TRASH)
    unmount()
  })

  it('is decorative, hidden from assistive technology, without a label', () => {
    const { el, unmount } = mount(Icon, { name: 'trash' })
    expect(svg(el).getAttribute('aria-hidden')).toBe('true')
    expect(svg(el).hasAttribute('role')).toBe(false)
    expect(svg(el).hasAttribute('aria-label')).toBe(false)
    unmount()
  })

  it('is an image named by its label when given one', () => {
    const { el, unmount } = mount(Icon, { name: 'trash', label: 'Delete' })
    expect(svg(el).getAttribute('role')).toBe('img')
    expect(svg(el).getAttribute('aria-label')).toBe('Delete')
    expect(svg(el).hasAttribute('aria-hidden')).toBe(false)
    unmount()
  })

  it('takes the shared size scale, and otherwise sizes with the text', () => {
    const plain = mount(Icon, { name: 'trash' })
    expect(svg(plain.el).hasAttribute('data-size')).toBe(false)
    plain.unmount()
    const sized = mount(Icon, { name: 'trash', size: 'lg' })
    expect(svg(sized.el).getAttribute('data-size')).toBe('lg')
    sized.unmount()
  })

  it('switches drawing when its name changes', async () => {
    registerIcon('plus', '<path d="M5 12h14"></path>')
    const name = ref('trash')
    const { el, unmount } = mount(Icon, { name: () => name.value } as never)
    expect(svg(el).innerHTML).toBe(TRASH)
    name.value = 'plus'
    await nextTick()
    expect(svg(el).innerHTML).toBe('<path d="M5 12h14"></path>')
    unmount()
  })

  it('warns once, naming the icon, when a name is not in the build', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const a = mount(Icon, { name: 'not-bundled' })
    const b = mount(Icon, { name: 'not-bundled' })
    expect(svg(a.el).innerHTML).toBe('')
    expect(warn).toHaveBeenCalledTimes(1)
    expect(String(warn.mock.calls[0]![0])).toContain('"not-bundled"')
    a.unmount()
    b.unmount()
  })
})

describe('the icon registry', () => {
  it('keeps the first drawing registered under a name', () => {
    registerIcon('once', '<circle r="1"></circle>')
    registerIcon('once', '<circle r="2"></circle>')
    expect(iconBody('once')).toBe('<circle r="1"></circle>')
    expect(iconBody('never')).toBeUndefined()
  })
})
