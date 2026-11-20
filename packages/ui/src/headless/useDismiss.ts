/**
 * useDismiss(el, { onDismiss, escape?, outside?, ignore?, returnTo? })
 *
 * Close an overlay on Escape or on a press outside it, the way every Weft
 * overlay closes. While `el` holds an element, the overlay is on the
 * `overlayStack`; Escape and outside presses go to the top overlay only, so
 * with a dialog, a menu from it and a tooltip from the menu open, one Escape
 * closes the tooltip and nothing else.
 *
 * - `onDismiss(reason, event)`: the overlay should close; `reason` is
 *   `'escape'` or `'outside'`. The composable never changes state itself.
 * - `escape` (default true) and `outside` (default true) turn either off.
 *   Each may be a ref or getter (`escape: () => props.dismissible`), read at
 *   each press, so it can change while the overlay is open. A top overlay with
 *   Escape off does not pass it down to the overlay below.
 * - `ignore`: elements whose presses are not outside. Give the trigger, so a
 *   press on it toggles the overlay instead of closing and reopening it.
 * - `returnTo`: where focus goes when the overlay closes, an element, ref or
 *   getter read at close (a template ref works); by default the element that
 *   had focus when it opened. The overlay stack returns focus, and none after
 *   a press outside; see `overlayStack`.
 *
 * The overlay joins the stack when `el` gets an element and leaves a
 * microtask after `el` empties, so swapping the element in one render (a
 * pointer popover becoming a touch sheet) keeps its place and its focus. It
 * leaves at once when its scope ends. Nothing happens on the server.
 *
 * Returns `{ id, isTop }`: the overlay's stack id and whether it is on top.
 */
import { computed, onScopeDispose, toValue, watch, type ComputedRef, type MaybeRefOrGetter, type Ref } from 'vue'
import { overlayStack } from './overlayStack.ts'

export type DismissReason = 'escape' | 'outside'

export interface DismissOptions {
  onDismiss: (reason: DismissReason, event: Event) => void
  escape?: MaybeRefOrGetter<boolean | undefined>
  outside?: MaybeRefOrGetter<boolean | undefined>
  ignore?: () => (Element | null | undefined)[]
  returnTo?: MaybeRefOrGetter<HTMLElement | null | undefined>
}

let count = 0

export function useDismiss(
  el: Readonly<Ref<HTMLElement | null | undefined>>,
  opts: DismissOptions,
): { id: string; isTop: ComputedRef<boolean> } {
  const id = `weft-overlay-${++count}`
  let pushed = false
  let leaving = false

  const enter = () => {
    pushed = true
    overlayStack.push(id, {
      el: () => el.value,
      // Read at each press: false means this overlay does not take it now.
      onEscape: (e) => toValue(opts.escape) !== false && (opts.onDismiss('escape', e), true),
      onOutside: (e) => toValue(opts.outside) !== false && (opts.onDismiss('outside', e), true),
      ignore: opts.ignore,
      returnTo: () => toValue(opts.returnTo),
    })
  }
  const leave = () => {
    if (!pushed) return
    pushed = false
    overlayStack.pop(id)
  }

  watch(
    el,
    (node) => {
      if (node && !pushed) enter()
      else if (!node && pushed && !leaving) {
        // Leave only if the element is still gone after this render.
        leaving = true
        queueMicrotask(() => {
          leaving = false
          if (!el.value) leave()
        })
      }
    },
    { immediate: true, flush: 'sync' },
  )
  onScopeDispose(leave)

  return { id, isTop: computed(() => overlayStack.isTop(id)) }
}
