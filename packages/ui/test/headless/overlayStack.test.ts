import { afterEach, describe, expect, it, vi } from 'vitest'
import { computed, effectScope, ref } from 'vue'
import { overlayStack } from '../../src/headless/overlayStack.ts'
import { $, active, html, key, press, reset } from './dom.ts'

afterEach(() => {
  for (const id of [...overlayStack.ids()]) overlayStack.pop(id)
  reset()
})

describe('overlayStack', () => {
  it('keeps the order overlays were pushed in, with top() and isTop() reactive', () => {
    const scope = effectScope()
    const top = scope.run(() => computed(() => overlayStack.top()))!
    const aOnTop = scope.run(() => computed(() => overlayStack.isTop('a')))!
    expect(top.value).toBeUndefined()
    overlayStack.push('a')
    expect(top.value).toBe('a')
    expect(aOnTop.value).toBe(true)
    overlayStack.push('b')
    expect(top.value).toBe('b')
    expect(aOnTop.value).toBe(false)
    overlayStack.pop('b')
    expect(aOnTop.value).toBe(true)
    overlayStack.pop('a')
    expect(top.value).toBeUndefined()
    scope.stop()
  })

  it('moves an id pushed again to the top instead of listing it twice', () => {
    overlayStack.push('a')
    overlayStack.push('b')
    overlayStack.push('a')
    expect(overlayStack.ids()).toEqual(['b', 'a'])
  })

  it('sends Escape to the top overlay only, once per press', () => {
    const a = vi.fn()
    const b = vi.fn()
    overlayStack.push('a', { onEscape: a })
    overlayStack.push('b', { onEscape: b })
    key('Escape', {}, document.body)
    expect(b).toHaveBeenCalledTimes(1)
    expect(a).not.toHaveBeenCalled()
  })

  it('does not pass Escape down when the top overlay does not close on it', () => {
    const a = vi.fn()
    overlayStack.push('a', { onEscape: a })
    overlayStack.push('b', {})
    key('Escape', {}, document.body)
    expect(a).not.toHaveBeenCalled()
  })

  it('leaves an Escape a widget inside already handled', () => {
    const a = vi.fn()
    overlayStack.push('a', { onEscape: a })
    const root = html('<input class="field">')
    $('.field', root).addEventListener('keydown', (e) => e.preventDefault())
    key('Escape', {}, $('.field', root))
    expect(a).not.toHaveBeenCalled()
  })

  it('reports a press outside the top overlay, not one inside it or on an ignored element', () => {
    const root = html('<button class="trigger">t</button><div class="panel"><button class="in">i</button></div><p class="out">o</p>')
    const outside = vi.fn()
    overlayStack.push('a', { el: () => $('.panel', root), onOutside: outside, ignore: () => [$('.trigger', root)] })
    press($('.in', root))
    press($('.trigger', root))
    expect(outside).not.toHaveBeenCalled()
    press($('.out', root))
    expect(outside).toHaveBeenCalledTimes(1)
  })

  it('waits for the click on touch, so a scroll that starts outside does not dismiss', () => {
    const root = html('<div class="panel"></div><p class="out">o</p>')
    const outside = vi.fn()
    overlayStack.push('a', { el: () => $('.panel', root), onOutside: outside })
    $('.out', root).dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerType: 'touch' }))
    expect(outside).not.toHaveBeenCalled()
    $('.out', root).dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(outside).toHaveBeenCalledTimes(1)
  })

  it('returns focus on pop to the element focused at push, when focus was inside the overlay', () => {
    const root = html('<button class="opener">o</button><div class="panel"><button class="in">i</button></div>')
    $('.opener', root).focus()
    overlayStack.push('a', { el: () => $('.panel', root) })
    $('.in', root).focus()
    overlayStack.pop('a')
    expect(active()).toBe($('.opener', root))
  })

  it('returns focus when the overlay was removed and focus fell to the body', () => {
    const root = html('<button class="opener">o</button><div class="panel"><button class="in">i</button></div>')
    $('.opener', root).focus()
    const panel = $('.panel', root)
    overlayStack.push('a', { el: () => panel })
    $('.in', root).focus()
    panel.remove()
    ;(document.activeElement as HTMLElement | null)?.blur()
    overlayStack.pop('a')
    expect(active()).toBe($('.opener', root))
  })

  it('returns no focus from an overlay whose returnFocus is false (one that never takes focus)', () => {
    const root = html('<button class="opener">o</button><div class="tip">t</div>')
    $('.opener', root).focus()
    const on = ref(false)
    overlayStack.push('tip', { el: () => $('.tip', root), returnFocus: on })
    ;(document.activeElement as HTMLElement | null)?.blur()
    overlayStack.pop('tip')
    expect(active()).toBe(document.body)
  })

  it('does not take focus back from an element the user moved to outside the overlay', () => {
    const root = html('<button class="opener">o</button><div class="panel"></div><input class="elsewhere">')
    $('.opener', root).focus()
    overlayStack.push('a', { el: () => $('.panel', root) })
    $('.elsewhere', root).focus()
    overlayStack.pop('a')
    expect(active()).toBe($('.elsewhere', root))
  })

  it('does not return focus to the opener after a press outside closed the overlay', () => {
    const root = html('<button class="opener">o</button><div class="panel"><button class="in">i</button></div><p class="out">o</p>')
    $('.opener', root).focus()
    overlayStack.push('a', { el: () => $('.panel', root), onOutside: () => overlayStack.pop('a') })
    $('.in', root).focus()
    press($('.out', root))
    expect(overlayStack.ids()).toEqual([])
    expect(active()).not.toBe($('.opener', root))
  })

  it('does not return focus after a press outside that closes the overlay a little later', async () => {
    const root = html('<button class="opener">o</button><div class="panel"><button class="in">i</button></div><p class="out">o</p>')
    $('.opener', root).focus()
    // As components close: state changes now, the element goes in a later microtask.
    overlayStack.push('a', { el: () => $('.panel', root), onOutside: () => queueMicrotask(() => queueMicrotask(() => overlayStack.pop('a'))) })
    $('.in', root).focus()
    press($('.out', root))
    await new Promise((r) => setTimeout(r, 0))
    expect(overlayStack.ids()).toEqual([])
    expect(active()).not.toBe($('.opener', root))
  })

  it('returns focus as usual on a later close when the overlay declined a press outside', async () => {
    const root = html('<button class="opener">o</button><div class="panel"><button class="in">i</button></div><p class="out">o</p>')
    $('.opener', root).focus()
    // An unsaved-changes guard: the press outside is taken, but the overlay stays open.
    overlayStack.push('a', { el: () => $('.panel', root), onOutside: () => {}, onEscape: () => overlayStack.pop('a') })
    $('.in', root).focus()
    press($('.out', root))
    await new Promise((r) => setTimeout(r, 0))
    expect(overlayStack.ids()).toEqual(['a'])
    $('.in', root).focus()
    key('Escape')
    expect(active()).toBe($('.opener', root))
  })

  it('takes returnTo as a ref or getter, read at pop', () => {
    const root = html('<button class="later">l</button><div class="panel" tabindex="-1"></div>')
    let target: HTMLElement | null = null
    overlayStack.push('a', { el: () => $('.panel', root), returnTo: () => target })
    target = $('.later', root)
    $('.panel', root).focus()
    overlayStack.pop('a')
    expect(active()).toBe($('.later', root))
  })

  it('knows whether a node is in an overlay stacked above a given element', () => {
    const root = html('<div class="low"><i class="trap"></i></div><div class="mid"><i class="m"></i></div><div class="high"><i class="h"></i></div><i class="loose"></i>')
    overlayStack.push('low', { el: () => $('.low', root) })
    overlayStack.push('mid', { el: () => $('.mid', root) })
    overlayStack.push('high', { el: () => $('.high', root) })
    expect(overlayStack.isAbove($('.h', root), $('.mid', root))).toBe(true)
    expect(overlayStack.isAbove($('.m', root), $('.mid', root))).toBe(false) // its own
    expect(overlayStack.isAbove($('.m', root), $('.high', root))).toBe(false) // below
    expect(overlayStack.isAbove($('.trap', root), $('.mid', root))).toBe(false)
    expect(overlayStack.isAbove($('.loose', root), $('.low', root))).toBe(false)
  })

  it('returns focus to an explicit returnTo element', () => {
    const root = html('<button class="opener">o</button><button class="other">x</button><div class="panel" tabindex="-1"></div>')
    $('.panel', root).focus()
    overlayStack.push('a', { el: () => $('.panel', root), returnTo: $('.other', root) })
    overlayStack.pop('a')
    expect(active()).toBe($('.other', root))
  })

  it('restores focus when a lower overlay closes while focus is in one above it', () => {
    const root = html(
      '<button class="opener">o</button><div class="dialog"><button class="menu-trigger">m</button></div><div class="menu"><button class="item">i</button></div>',
    )
    $('.opener', root).focus()
    overlayStack.push('dialog', { el: () => $('.dialog', root) })
    $('.menu-trigger', root).focus()
    overlayStack.push('menu', { el: () => $('.menu', root) })
    $('.item', root).focus()
    // The dialog closes first (programmatically); its menu goes with it.
    overlayStack.pop('dialog')
    overlayStack.pop('menu')
    expect(active()).toBe($('.opener', root))
  })

  it('knows whether a node is inside any stacked overlay', () => {
    const root = html('<div class="panel"><i class="in"></i></div><i class="out"></i>')
    overlayStack.push('a', { el: () => $('.panel', root) })
    expect(overlayStack.contains($('.in', root))).toBe(true)
    expect(overlayStack.contains($('.out', root))).toBe(false)
  })

  it('passes presses on from a pass-through entry (a tooltip) to the one below, but not Escape', () => {
    const root = html('<div class="menu"><button class="item">i</button></div><div class="tip">t</div><p class="out">o</p>')
    const menu = { onOutside: vi.fn(), onEscape: vi.fn() }
    const tip = { onOutside: vi.fn(), onEscape: vi.fn() }
    overlayStack.push('menu', { el: () => $('.menu', root), ...menu })
    overlayStack.push('tip', { el: () => $('.tip', root), passThrough: true, ...tip })
    press($('.out', root)) // outside both: each takes it
    expect(tip.onOutside).toHaveBeenCalledOnce()
    expect(menu.onOutside).toHaveBeenCalledOnce()
    press($('.item', root)) // inside the menu: only the tooltip is left behind
    expect(tip.onOutside).toHaveBeenCalledTimes(2)
    expect(menu.onOutside).toHaveBeenCalledOnce()
    press($('.tip', root)) // on the tooltip: no one's outside
    expect(tip.onOutside).toHaveBeenCalledTimes(2)
    expect(menu.onOutside).toHaveBeenCalledOnce()
    press($('.out', root), 'touch') // on touch, at the click
    expect(menu.onOutside).toHaveBeenCalledTimes(2)
    key('Escape')
    expect(tip.onEscape).toHaveBeenCalledOnce()
    expect(menu.onEscape).not.toHaveBeenCalled()
  })

  it('does not pass presses on from an entry that declines them (a modal)', () => {
    const root = html('<div class="menu">m</div><div class="modal">d</div><p class="out">o</p>')
    const menu = vi.fn()
    overlayStack.push('menu', { el: () => $('.menu', root), onOutside: menu })
    overlayStack.push('modal', { el: () => $('.modal', root), onOutside: () => false })
    press($('.out', root))
    expect(menu).not.toHaveBeenCalled()
  })

  it('stops listening to the document once the stack is empty', () => {
    const remove = vi.spyOn(document, 'removeEventListener')
    overlayStack.push('a', { onEscape: () => {} })
    overlayStack.pop('a')
    const types = remove.mock.calls.map((c) => c[0])
    expect(types).toEqual(expect.arrayContaining(['keydown', 'pointerdown', 'click']))
    remove.mockRestore()
  })
})
