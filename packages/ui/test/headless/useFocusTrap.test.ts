import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useFocusTrap } from '../../src/headless/useFocusTrap.ts'
import { $, active, html, key, reset } from './dom.ts'

afterEach(reset)

const markup = `
  <button class="opener">open</button>
  <div class="trap">
    <button class="first">first</button>
    <button disabled>off</button>
    <input class="middle">
    <a class="last" href="#x">last</a>
    <span tabindex="-1" class="skip">not tabbable</span>
  </div>
  <button class="outside">outside</button>`

function setup(opts: { initial?: () => HTMLElement | null; returnTo?: HTMLElement | null } = {}) {
  const root = html(markup)
  $('.opener', root).focus()
  const el = ref<HTMLElement | null>($('.trap', root))
  const on = ref(false)
  const scope = effectScope()
  scope.run(() => useFocusTrap(el, { active: on, ...opts }))
  return { root, el, on, scope, q: (s: string) => $(s, root) }
}

/** Press Tab; when the trap does not take it, move focus as the browser would (here: a no-op). */
const tab = (shift = false) => key('Tab', { shiftKey: shift })

describe('useFocusTrap', () => {
  it('moves focus to the first tabbable element when it activates', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    expect(active()).toBe(q('.first'))
    scope.stop()
  })

  it('moves focus to the initial element when one is given', async () => {
    const { on, q, scope } = setup({ initial: () => document.querySelector<HTMLElement>('.middle') })
    on.value = true
    await nextTick()
    expect(active()).toBe(q('.middle'))
    scope.stop()
  })

  it('wraps Tab from the last element to the first, and Shift+Tab back', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    q('.last').focus()
    const forward = tab()
    expect(forward.defaultPrevented).toBe(true)
    expect(active()).toBe(q('.first'))
    const back = tab(true)
    expect(back.defaultPrevented).toBe(true)
    expect(active()).toBe(q('.last'))
    scope.stop()
  })

  it('leaves Tab between inner elements to the browser', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    q('.first').focus()
    expect(tab().defaultPrevented).toBe(false)
    scope.stop()
  })

  it('keeps focus on the container when nothing inside can take it', async () => {
    const root = html('<button class="o">o</button><div class="empty"><p>text</p></div>')
    $('.o', root).focus()
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useFocusTrap(ref($('.empty', root)), { active: on }))
    await nextTick()
    expect(active()).toBe($('.empty', root))
    expect(key('Tab').defaultPrevented).toBe(true)
    expect(active()).toBe($('.empty', root))
    scope.stop()
  })

  it('brings focus back when it moves outside', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    q('.middle').focus()
    q('.outside').focus()
    expect(active()).toBe(q('.middle'))
    scope.stop()
  })

  it('returns focus to the opener when it deactivates', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    on.value = false
    await nextTick()
    expect(active()).toBe(q('.opener'))
    q('.outside').focus()
    expect(active()).toBe(q('.outside'))
    scope.stop()
  })

  it('returns focus to returnTo when given', async () => {
    const root = html('<button class="target">t</button>')
    const { on, scope } = setup({ returnTo: $('.target', root) })
    on.value = true
    await nextTick()
    on.value = false
    await nextTick()
    expect(active()).toBe($('.target', root))
    scope.stop()
  })

  it('lets only the innermost of two active traps act', async () => {
    const outer = setup()
    outer.on.value = true
    await nextTick()
    const inner = html('<div class="inner"><button class="a">a</button><button class="b">b</button></div>')
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useFocusTrap(ref($('.inner', inner)), { active: on }))
    await nextTick()
    expect(active()).toBe($('.a', inner))
    // Focus may not go back to the outer trap while the inner one is active.
    outer.q('.middle').focus()
    expect(active()).toBe($('.a', inner))
    scope.stop()
    // The outer trap takes over again; focus returned to what it had.
    expect(active()).toBe(outer.q('.first'))
    outer.scope.stop()
  })

  it('stops trapping and returns focus when its scope ends', async () => {
    const { on, q, scope } = setup()
    on.value = true
    await nextTick()
    scope.stop()
    expect(active()).toBe(q('.opener'))
    q('.outside').focus()
    expect(active()).toBe(q('.outside'))
  })

  it('moves to a new element that replaces its own while active, keeping the opener to return to', async () => {
    const { el, on, q, scope, root } = setup()
    on.value = true
    await nextTick()
    const sheet = document.createElement('div')
    sheet.innerHTML = '<button class="sheet-first">a</button>'
    root.appendChild(sheet)
    let openerFocused = 0
    q('.opener').addEventListener('focus', () => openerFocused++)
    q('.trap').remove()
    el.value = sheet // the same overlay, re-rendered in another form
    await nextTick()
    expect(active()).toBe($('.sheet-first', sheet))
    expect(openerFocused).toBe(0) // focus went straight across, not back to the opener and in again
    on.value = false
    await nextTick()
    expect(active()).toBe(q('.opener'))
    scope.stop()
  })

  it('waits for its element when it activates before the element exists', async () => {
    const root = html(markup)
    $('.opener', root).focus()
    const el = ref<HTMLElement | null>(null)
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useFocusTrap(el, { active: on }))
    await nextTick()
    expect(active()).toBe($('.opener', root))
    el.value = $('.trap', root)
    await nextTick()
    expect(active()).toBe($('.first', root))
    scope.stop()
  })
})
