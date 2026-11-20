import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope, ref } from 'vue'
import { overlayStack } from '../../src/headless/overlayStack.ts'
import { useDismiss } from '../../src/headless/useDismiss.ts'
import { $, active, html, key, press, reset } from './dom.ts'

afterEach(() => {
  for (const id of [...overlayStack.ids()]) overlayStack.pop(id)
  reset()
})

const tick = () => new Promise<void>((r) => queueMicrotask(r))

function setup(opts: Partial<Parameters<typeof useDismiss>[1]> = {}) {
  const root = html('<button class="trigger">t</button><div class="panel"><button class="in">i</button></div><p class="out">o</p>')
  const el = ref<HTMLElement | null>(null)
  const onDismiss = vi.fn()
  const scope = effectScope()
  const api = scope.run(() => useDismiss(el, { onDismiss, ...opts }))!
  return { root, el, onDismiss, scope, api, panel: $('.panel', root) }
}

describe('useDismiss', () => {
  it('joins the overlay stack while its element exists and leaves when it goes', async () => {
    const { el, panel, scope, api } = setup()
    expect(overlayStack.ids()).toEqual([])
    el.value = panel
    expect(overlayStack.ids()).toEqual([api.id])
    expect(api.isTop.value).toBe(true)
    el.value = null
    await tick()
    expect(overlayStack.ids()).toEqual([])
    scope.stop()
  })

  it('stays in place when its element is swapped for another in the same tick', async () => {
    const { el, panel, scope, api, root } = setup()
    el.value = panel
    overlayStack.push('above')
    el.value = null
    el.value = $('.in', root)
    await tick()
    expect(overlayStack.ids()).toEqual([api.id, 'above'])
    scope.stop()
  })

  it('dismisses on Escape with the reason and the event', () => {
    const { el, panel, onDismiss, scope } = setup()
    el.value = panel
    const e = key('Escape', {}, document.body)
    expect(onDismiss).toHaveBeenCalledWith('escape', e)
    expect(e.defaultPrevented).toBe(true)
    scope.stop()
  })

  it('dismisses on a press outside, not on one inside or on an ignored trigger', () => {
    const { el, panel, onDismiss, scope, root } = setup({ ignore: () => [$('.trigger', root)] })
    el.value = panel
    press($('.in', root))
    press($('.trigger', root))
    expect(onDismiss).not.toHaveBeenCalled()
    press($('.out', root))
    expect(onDismiss).toHaveBeenCalledWith('outside', expect.anything())
    scope.stop()
  })

  it('can turn off Escape or outside presses', () => {
    const { el, panel, onDismiss, scope, root } = setup({ escape: false, outside: false })
    el.value = panel
    key('Escape', {}, document.body)
    press($('.out', root))
    expect(onDismiss).not.toHaveBeenCalled()
    scope.stop()
  })

  it('only the top of two dismisses on Escape', () => {
    const one = setup()
    const two = setup()
    one.el.value = one.panel
    two.el.value = two.panel
    key('Escape', {}, document.body)
    expect(two.onDismiss).toHaveBeenCalledTimes(1)
    expect(one.onDismiss).not.toHaveBeenCalled()
    one.scope.stop()
    two.scope.stop()
  })

  it('leaves the stack and returns focus when its scope is disposed', () => {
    const { el, panel, scope, root } = setup()
    $('.trigger', root).focus()
    el.value = panel
    $('.in', root).focus()
    scope.stop()
    expect(overlayStack.ids()).toEqual([])
    expect(active()).toBe($('.trigger', root))
  })
})
