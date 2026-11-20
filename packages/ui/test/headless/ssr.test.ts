// @vitest-environment node
// Server rendering runs component setup under Node, where there is no window
// or document. Every composable must load and run there and leave nothing behind.
import { describe, expect, it } from 'vitest'
import { effectScope, ref } from 'vue'

describe('the headless layer under Node', () => {
  it('has no DOM here', () => {
    expect(typeof window).toBe('undefined')
    expect(typeof document).toBe('undefined')
  })

  it('imports and runs every composable inside an effect scope', async () => {
    const h = await import('../../src/headless/index.ts')
    const el = ref<HTMLElement | null>(null)
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => {
      expect(h.useId()).toMatch(/^weft-/)
      const model = h.useControllable({ modelValue: undefined as string | undefined }, 'modelValue', () => {}, 'a')
      model.value = 'b'
      h.useFocusTrap(el, { active: on })
      h.useDismiss(el, { onDismiss: () => {} })
      h.useRovingFocus(el, { orientation: 'horizontal' })
      h.useFloating(el, el, { placement: 'bottom', offset: 4, flip: true, shift: true, size: true })
      const t = h.useTypeahead(() => ['a'], () => {})
      t.reset()
      h.useScrollLock(on)
      expect(h.overlayStack.top()).toBeUndefined()
      expect(h.overlayStack.isTop('x')).toBe(false)
    })
    scope.stop()
    expect(h.overlayStack.ids()).toEqual([])
  })
})
