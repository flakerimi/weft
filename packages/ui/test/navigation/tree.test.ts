import { afterEach, describe, expect, it } from 'vitest'
import TreeCases from '../fixtures/navigation/TreeCases.vue'
import { mount } from '../mount.ts'
import { loadStyles, ruleFor } from '../styles.ts'
import { px } from '../forms/helpers.ts'
import { $, $$, active, key, reset, settle } from '../overlays/helpers.ts'

let unmount: (() => void) | undefined
afterEach(() => {
  unmount?.()
  unmount = undefined
  reset()
})
async function cases(props: Record<string, unknown> = {}) {
  const m = mount(TreeCases, props)
  unmount = m.unmount
  await settle()
  return m
}
const nodes = () => $$('[role="treeitem"]')
const node = (label: string) => nodes().find((n) => n.querySelector('.weft-tree__label')!.textContent!.trim() === label)!
const labels = () => nodes().map((n) => n.querySelector('.weft-tree__label')!.textContent!.trim())
async function press(k: string, init: KeyboardEventInit = {}) {
  const e = key(k, init)
  await settle()
  return e
}

describe('Tree', () => {
  it('is an APG tree: treeitems with level, set size, position and expanded state', async () => {
    await cases()
    const tree = $('[role="tree"]')
    expect(tree.getAttribute('aria-label')).toBe('Files')
    expect(labels()).toEqual(['src', 'app.ts', 'components', 'locked.ts', 'docs', 'package.json'])
    const attrs = (n: Element) => [n.getAttribute('aria-level'), n.getAttribute('aria-setsize'), n.getAttribute('aria-posinset'), n.getAttribute('aria-expanded')]
    expect(attrs(node('src'))).toEqual(['1', '3', '1', 'true'])
    expect(attrs(node('app.ts'))).toEqual(['2', '3', '1', null])
    expect(attrs(node('components'))).toEqual(['2', '3', '2', 'false'])
    expect(attrs(node('package.json'))).toEqual(['1', '3', '3', null])
    expect(node('locked.ts').getAttribute('aria-disabled')).toBe('true')
    expect(node('app.ts').getAttribute('aria-selected')).toBe('false')
    expect(tree.hasAttribute('aria-multiselectable')).toBe(false)
  })

  it('is one tab stop', async () => {
    await cases()
    expect(nodes().map((n) => n.getAttribute('tabindex'))).toEqual(['0', '-1', '-1', '-1', '-1', '-1'])
  })

  it('moves with Down, Up, Home and End over the visible nodes', async () => {
    await cases()
    node('src').focus()
    await press('ArrowDown')
    expect(active()).toBe(node('app.ts'))
    await press('ArrowDown')
    await press('ArrowDown')
    expect(active()).toBe(node('locked.ts')) // disabled nodes are reached
    await press('ArrowUp')
    expect(active()).toBe(node('components'))
    await press('End')
    expect(active()).toBe(node('package.json'))
    await press('Home')
    expect(active()).toBe(node('src'))
    expect(node('src').getAttribute('tabindex')).toBe('0')
  })

  it('opens with Right, then enters; closes with Left, then goes to the parent', async () => {
    const m = await cases()
    node('src').focus()
    await press('ArrowDown')
    await press('ArrowDown')
    expect(active()).toBe(node('components'))
    await press('ArrowRight')
    expect(node('components').getAttribute('aria-expanded')).toBe('true')
    expect(labels()).toContain('Button.vue')
    expect(active()).toBe(node('components'))
    await press('ArrowRight')
    expect(active()).toBe(node('Button.vue'))
    expect(node('Button.vue').getAttribute('aria-level')).toBe('3')
    await press('ArrowLeft')
    expect(active()).toBe(node('components'))
    await press('ArrowLeft')
    expect(node('components').getAttribute('aria-expanded')).toBe('false')
    expect(labels()).not.toContain('Button.vue')
    await press('ArrowLeft')
    expect(active()).toBe(node('src'))
    expect(m.exposed.expanded).toEqual(['src'])
    expect(m.exposed.toggled).toEqual(['components:true', 'components:false'])
  })

  it('mirrors Right and Left right to left', async () => {
    await cases({ dir: 'rtl' })
    node('src').focus()
    await press('ArrowRight')
    expect(node('src').getAttribute('aria-expanded')).toBe('false')
    await press('ArrowLeft')
    expect(node('src').getAttribute('aria-expanded')).toBe('true')
  })

  it('selects with Enter or a press, opening or closing a parent too', async () => {
    const m = await cases()
    node('src').focus()
    await press('ArrowDown')
    await press('Enter')
    expect(m.exposed.selected).toBe('app')
    expect(node('app.ts').getAttribute('aria-selected')).toBe('true')
    node('docs').click()
    await settle()
    expect(m.exposed.selected).toBe('docs')
    expect(node('docs').getAttribute('aria-expanded')).toBe('true')
    expect(active()).toBe(node('docs'))
    node('locked.ts').click()
    await settle()
    expect(m.exposed.selected).toBe('docs')
  })

  it('jumps to a node by typing the start of its label', async () => {
    await cases()
    node('src').focus()
    await press('d')
    expect(active()).toBe(node('docs'))
  })

  it('opens every sibling of the focused node with *', async () => {
    const m = await cases()
    node('src').focus()
    await press('ArrowDown')
    await press('*')
    expect(node('components').getAttribute('aria-expanded')).toBe('true')
    expect(node('docs').getAttribute('aria-expanded')).toBe('false')
    node('src').focus()
    await press('*')
    expect(node('docs').getAttribute('aria-expanded')).toBe('true')
    expect(m.exposed.expanded).toEqual(expect.arrayContaining(['src', 'components', 'docs']))
  })

  it('moves without choosing on Shift with an arrow when not multiple', async () => {
    const m = await cases()
    node('src').focus()
    await press('ArrowDown', { shiftKey: true })
    expect(active()).toBe(node('app.ts'))
    expect(m.exposed.selected).toBeNull()
    expect(nodes().every((n) => n.getAttribute('aria-selected') === 'false')).toBe(true)
  })

  it('says nothing of selection, and chooses nothing, when not selectable', async () => {
    const m = await cases({ selectable: false })
    expect(nodes().every((n) => !n.hasAttribute('aria-selected'))).toBe(true)
    node('src').focus()
    await press('ArrowDown')
    await press('Enter')
    await press(' ')
    expect(m.exposed.selected).toBeNull()
    node('docs').click()
    await settle()
    expect(node('docs').getAttribute('aria-expanded')).toBe('true') // opening still works
    expect(m.exposed.selected).toBeNull()
  })

  it('gives siblings with the same label and no value keys of their own', async () => {
    const m = await cases({ dupes: true })
    ;(m.exposed as { expanded: string[] }).expanded = []
    await settle()
    const [first, second] = nodes()
    first!.click()
    await settle()
    expect(nodes()).toHaveLength(4) // only the first "a" opened
    expect(second!.getAttribute('aria-expanded')).toBe('false')
    const kids = nodes().slice(1, 3)
    kids[1]!.click()
    await settle()
    expect(kids[1]!.getAttribute('aria-selected')).toBe('true')
    expect(kids[0]!.getAttribute('aria-selected')).toBe('false')
    expect(new Set([m.exposed.selected, ...(m.exposed.expanded as string[])]).size).toBe(2)
  })

  describe('multiple', () => {
    it('says aria-multiselectable; Space toggles a node, Shift+arrows extend, Ctrl+A selects all', async () => {
      const m = await cases({ multiple: true })
      expect($('[role="tree"]').getAttribute('aria-multiselectable')).toBe('true')
      node('src').focus()
      await press('ArrowDown')
      await press(' ')
      expect(m.exposed.selected).toEqual(['app'])
      await press('ArrowDown', { shiftKey: true })
      expect(active()).toBe(node('components'))
      expect(m.exposed.selected).toEqual(['app', 'components'])
      await press(' ')
      expect(m.exposed.selected).toEqual(['app'])
      await press('a', { ctrlKey: true })
      expect(m.exposed.selected).toEqual(['src', 'app', 'components', 'docs', 'package'])
    })
  })

  describe('with more than 200 visible nodes', () => {
    it('renders only a window of them, each still saying its place, and keeps the focused one', async () => {
      await cases({ big: true })
      const tree = $('[role="tree"]')
      expect(tree.dataset.virtual).toBe('true')
      expect(nodes().length).toBeLessThan(60)
      expect(node('Row 1').getAttribute('aria-setsize')).toBe('500')
      expect(node('Row 1').getAttribute('aria-posinset')).toBe('1')
      node('rows').focus()
      await press('End')
      expect(active()!.querySelector('.weft-tree__label')!.textContent).toBe('Row 500')
      expect(active()!.getAttribute('aria-posinset')).toBe('500')
      await press('Home')
      expect(active()).toBe(node('rows'))
    })
  })

  it('gives each node a 44 px row on touch and a focus ring', async () => {
    const unstyle = loadStyles()
    try {
      await cases({ input: 'touch' })
      expect(px(getComputedStyle(node('src')).minBlockSize)).toBe(44)
      expect(ruleFor('.weft-tree__item:focus-visible')!.style.getPropertyValue('outline')).toBe('var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)')
    } finally {
      unstyle()
    }
  })
})
