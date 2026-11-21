import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useRovingFocus, type RovingFocusOptions } from '../../src/headless/useRovingFocus.ts'
import { $, active, html, key, reset } from './dom.ts'

afterEach(reset)

const items = (n: number, extra: Record<number, string> = {}) =>
  Array.from({ length: n }, (_, i) => `<button data-weft-item class="i${i}" ${extra[i] ?? ''}>${i}</button>`).join('')

function setup(opts: RovingFocusOptions, markup = items(4), wrap = (s: string) => s) {
  const root = html(wrap(`<div class="group">${markup}</div>`))
  const container = ref<HTMLElement | null>($('.group', root))
  const scope = effectScope()
  const api = scope.run(() => useRovingFocus(container, opts))!
  const q = (i: number) => $(`.i${i}`, root)
  return { root, scope, api, q, container }
}

const tabindexes = (root: ParentNode) => [...root.querySelectorAll('[data-weft-item]')].map((e) => e.getAttribute('tabindex'))

describe('useRovingFocus', () => {
  it('makes one item tabbable: the selected one, else the first', () => {
    const plain = setup({ orientation: 'horizontal' })
    expect(tabindexes(plain.root)).toEqual(['0', '-1', '-1', '-1'])
    plain.scope.stop()
    const picked = setup({ orientation: 'horizontal' }, items(3, { 2: 'aria-selected="true"' }))
    expect(tabindexes(picked.root)).toEqual(['-1', '-1', '0'])
    expect(picked.api.active.value).toBe(2)
    picked.scope.stop()
  })

  it('moves with the arrow keys along its orientation and ignores the others', () => {
    const { q, scope } = setup({ orientation: 'horizontal' })
    q(0).focus()
    expect(key('ArrowRight').defaultPrevented).toBe(true)
    expect(active()).toBe(q(1))
    expect(key('ArrowDown').defaultPrevented).toBe(false)
    expect(active()).toBe(q(1))
    key('ArrowLeft')
    expect(active()).toBe(q(0))
    scope.stop()
  })

  it('moves the tab stop with focus', () => {
    const { q, root, scope, api } = setup({ orientation: 'vertical' })
    q(0).focus()
    key('ArrowDown')
    expect(tabindexes(root)).toEqual(['-1', '0', '-1', '-1'])
    expect(api.active.value).toBe(1)
    q(3).focus() // a click focuses an item directly
    expect(tabindexes(root)).toEqual(['-1', '-1', '-1', '0'])
    scope.stop()
  })

  it('wraps at the ends by default and stops there with loop: false', () => {
    const looped = setup({ orientation: 'vertical' })
    looped.q(3).focus()
    key('ArrowDown')
    expect(active()).toBe(looped.q(0))
    key('ArrowUp')
    expect(active()).toBe(looped.q(3))
    looped.scope.stop()
    reset()
    const stopped = setup({ orientation: 'vertical', loop: false })
    stopped.q(3).focus()
    key('ArrowDown')
    expect(active()).toBe(stopped.q(3))
    stopped.scope.stop()
  })

  it('goes to the first and last item with Home and End', () => {
    const { q, scope } = setup({ orientation: 'horizontal' })
    q(1).focus()
    key('End')
    expect(active()).toBe(q(3))
    key('Home')
    expect(active()).toBe(q(0))
    scope.stop()
  })

  it('skips disabled items', () => {
    const { q, scope } = setup({ orientation: 'horizontal' }, items(4, { 1: 'disabled', 2: 'data-disabled' }))
    q(0).focus()
    key('ArrowRight')
    expect(active()).toBe(q(3))
    scope.stop()
  })

  it('mirrors left and right under dir="rtl" on an ancestor', () => {
    const { q, scope } = setup({ orientation: 'horizontal' }, items(3), (s) => `<section dir="rtl"><div>${s}</div></section>`)
    q(0).focus()
    key('ArrowLeft')
    expect(active()).toBe(q(1))
    key('ArrowRight')
    expect(active()).toBe(q(0))
    scope.stop()
  })

  it('follows an rtl ref over the document direction', async () => {
    const rtl = ref(true)
    const { q, scope } = setup({ orientation: 'horizontal', rtl })
    q(0).focus()
    key('ArrowLeft')
    expect(active()).toBe(q(1))
    rtl.value = false
    await nextTick()
    key('ArrowRight')
    expect(active()).toBe(q(2))
    scope.stop()
  })

  it('moves by rows in a grid, and mirrors columns in rtl', () => {
    const grid = setup({ orientation: 'grid', columns: () => 3 }, items(9))
    grid.q(4).focus()
    key('ArrowUp')
    expect(active()).toBe(grid.q(1))
    key('ArrowUp') // no row above: stays
    expect(active()).toBe(grid.q(1))
    key('ArrowDown')
    key('ArrowDown')
    expect(active()).toBe(grid.q(7))
    key('ArrowRight')
    expect(active()).toBe(grid.q(8))
    key('Home')
    expect(active()).toBe(grid.q(6))
    key('End', { ctrlKey: true })
    expect(active()).toBe(grid.q(8))
    key('Home', { ctrlKey: true })
    expect(active()).toBe(grid.q(0))
    grid.scope.stop()
    reset()
    const rtl = setup({ orientation: 'grid', columns: () => 3, rtl: ref(true) }, items(9))
    rtl.q(4).focus()
    key('ArrowLeft')
    expect(active()).toBe(rtl.q(5))
    rtl.scope.stop()
  })

  it('picks up items added later', async () => {
    const { root, scope } = setup({ orientation: 'horizontal' }, items(1))
    const added = document.createElement('button')
    added.setAttribute('data-weft-item', '')
    $('.group', root).appendChild(added)
    await new Promise((r) => setTimeout(r, 0))
    expect(added.getAttribute('tabindex')).toBe('-1')
    scope.stop()
  })

  it('counts indices over every item, disabled ones included, and only skips them when moving', () => {
    const { q, api, scope } = setup({ orientation: 'horizontal' }, items(4, { 1: 'disabled' }))
    api.focus(2)
    expect(active()).toBe(q(2))
    expect(api.active.value).toBe(2)
    key('ArrowLeft')
    expect(active()).toBe(q(0))
    expect(api.active.value).toBe(0)
    api.focus(1) // disabled: nothing happens
    expect(active()).toBe(q(0))
    scope.stop()
  })

  it('moves the tab stop without moving focus, for a selection the parent sets', () => {
    const { q, api, root, scope } = setup({ orientation: 'horizontal' }, items(4, { 1: 'disabled' }))
    const outside = html('<button class="elsewhere">x</button>')
    $('.elsewhere', outside).focus()
    api.setActive(3)
    expect(api.active.value).toBe(3)
    expect(tabindexes(root)).toEqual(['-1', '-1', '-1', '0'])
    expect(active()).toBe($('.elsewhere', outside))
    api.setActive(1) // disabled: the tab stop stays
    expect(api.active.value).toBe(3)
    expect(q(3).getAttribute('tabindex')).toBe('0')
    scope.stop()
  })

  it('leaves the items of a group nested inside it to that group', () => {
    const markup =
      '<button data-weft-item class="i0">0</button>' +
      '<div class="inner"><button data-weft-item class="n0">a</button><button data-weft-item class="n1">b</button></div>' +
      '<button data-weft-item class="i1">1</button>'
    const outer = setup({ orientation: 'horizontal' }, markup)
    const inner = ref<HTMLElement | null>($('.inner', outer.root))
    const innerScope = effectScope()
    const innerApi = innerScope.run(() => useRovingFocus(inner, { orientation: 'vertical' }))!
    expect(tabindexes(outer.root)).toEqual(['0', '0', '-1', '-1'])
    outer.q(0).focus()
    key('ArrowRight')
    expect(active()).toBe(outer.q(1)) // over the nested group, not into it
    $('.n0', outer.root).focus()
    key('ArrowDown')
    expect(active()).toBe($('.n1', outer.root))
    key('ArrowRight') // the inner group's item: not the outer group's to move
    expect(active()).toBe($('.n1', outer.root))
    expect(outer.api.active.value).toBe(1)
    expect(innerApi.active.value).toBe(1)
    innerScope.stop()
    outer.scope.stop()
  })

  it('marks its container while it runs', () => {
    const { container, scope } = setup({ orientation: 'horizontal' })
    expect(container.value!.hasAttribute('data-weft-roving')).toBe(true)
    scope.stop()
    expect(container.value!.hasAttribute('data-weft-roving')).toBe(false)
  })

  it('focuses an item by index', () => {
    const { q, api, scope } = setup({ orientation: 'horizontal' })
    api.focus(2)
    expect(active()).toBe(q(2))
    expect(api.active.value).toBe(2)
    scope.stop()
  })

  it('stops handling keys when its scope ends', () => {
    const { q, scope } = setup({ orientation: 'horizontal' })
    q(0).focus()
    scope.stop()
    expect(key('ArrowRight').defaultPrevented).toBe(false)
    expect(active()).toBe(q(0))
  })
  it('with tabStops "all", moves with the arrows but leaves every item its own tab stop', () => {
    const { q, root, scope, api } = setup({ orientation: 'vertical', tabStops: 'all' }, items(3, { 1: 'disabled' }))
    expect(tabindexes(root)).toEqual([null, null, null])
    q(0).focus()
    key('ArrowDown')
    expect(active()).toBe(q(2)) // the disabled one is skipped
    expect(api.active.value).toBe(2)
    key('Home')
    expect(active()).toBe(q(0))
    expect(tabindexes(root)).toEqual([null, null, null])
    scope.stop()
  })
})
