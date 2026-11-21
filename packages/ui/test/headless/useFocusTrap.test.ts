import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { overlayStack } from '../../src/headless/overlayStack.ts'
import { useFocusTrap, type FocusTrapOptions } from '../../src/headless/useFocusTrap.ts'
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

function setup(opts: Partial<Omit<FocusTrapOptions, 'active'>> = {}) {
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

  it('skips elements hidden by CSS: focus never lands on one it cannot show', async () => {
    const root = html(markup)
    $('.first', root).style.display = 'none'
    $('.opener', root).focus()
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useFocusTrap(ref($('.trap', root)), { active: on }))
    await nextTick()
    expect(active()).toBe($('.middle', root))
    $('.last', root).focus()
    key('Tab')
    expect(active()).toBe($('.middle', root))
    scope.stop()
  })

  it('takes initial and returnTo as refs, read when they are used', async () => {
    const target = ref<HTMLElement | null>(null)
    const first = ref<HTMLElement | null>(null)
    const { on, q, scope } = setup({ initial: first, returnTo: target })
    first.value = q('.middle')
    target.value = q('.outside')
    on.value = true
    await nextTick()
    expect(active()).toBe(q('.middle'))
    on.value = false
    await nextTick()
    expect(active()).toBe(q('.outside'))
    scope.stop()
  })

  it('lets focus into an overlay stacked above it, not into one below it', async () => {
    const root = html('<div class="below"><button class="b">b</button></div>' + markup + '<div class="above"><button class="a">a</button></div>')
    overlayStack.push('below', { el: () => $('.below', root) })
    overlayStack.push('trap', { el: () => $('.trap', root) })
    overlayStack.push('above', { el: () => $('.above', root) })
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useFocusTrap(ref($('.trap', root)), { active: on }))
    await nextTick()
    $('.a', root).focus()
    expect(active()).toBe($('.a', root))
    $('.b', root).focus()
    expect(active()).toBe($('.first', root))
    scope.stop()
    for (const id of ['above', 'trap', 'below']) overlayStack.pop(id)
  })

  it('returns focus by the overlay stack’s rules when its stacked element stays shown (v-show)', async () => {
    const { on, q, scope } = setup()
    overlayStack.push('dialog', { el: () => q('.trap') }) // pushed while the opener had focus
    on.value = true
    await nextTick()
    expect(active()).toBe(q('.first'))
    on.value = false // hidden by v-show: the element stays mounted and stacked
    await nextTick()
    expect(active()).toBe(q('.opener'))
    overlayStack.pop('dialog') // unmounted later: nothing more to do
    expect(active()).toBe(q('.opener'))
    scope.stop()
  })

  it('returns no focus when a press outside closed its stacked element (v-show)', async () => {
    const { on, q, scope } = setup()
    overlayStack.push('dialog', { el: () => q('.trap'), onOutside: () => void (on.value = false) })
    on.value = true
    await nextTick()
    q('.outside').dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, pointerType: 'mouse' }))
    await nextTick()
    expect(active()).not.toBe(q('.opener'))
    overlayStack.pop('dialog')
    scope.stop()
  })

  it('leaves focus where it is when it ends while returnFocus says no (an overlay that stays open in another form)', async () => {
    const open = ref(true)
    const { on, q, scope } = setup({ returnFocus: () => !open.value })
    overlayStack.push('popover', { el: () => q('.trap') })
    on.value = true
    await nextTick()
    q('.middle').focus()
    on.value = false // a sheet that became a popover: no trap now, still open
    await nextTick()
    expect(active()).toBe(q('.middle'))
    on.value = true
    await nextTick()
    expect(active()).toBe(q('.middle')) // focus was inside already: it stays
    open.value = false
    on.value = false // now it closes: focus goes home
    await nextTick()
    expect(active()).toBe(q('.opener'))
    overlayStack.pop('popover')
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
