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
 * - `escape` (default true) and `outside` (default true) turn either off. A
 *   top overlay with Escape off swallows it rather than passing it down.
 * - `ignore`: elements whose presses are not outside. Give the trigger, so a
 *   press on it toggles the overlay instead of closing and reopening it.
 * - `returnTo`: where focus goes when the overlay closes; by default the
 *   element that had focus when it opened (see `overlayStack` for when focus
 *   is left where the user put it).
 *
 * The overlay joins the stack when `el` gets an element and leaves a
 * microtask after `el` empties, so swapping the element in one render (a
 * pointer popover becoming a touch sheet) keeps its place and its focus. It
 * leaves at once when its scope ends. Nothing happens on the server.
 *
 * Returns `{ id, isTop }`: the overlay's stack id and whether it is on top.
 */
import { computed, onScopeDispose, watch, type ComputedRef, type Ref } from 'vue'
import { overlayStack } from './overlayStack.ts'

export type DismissReason = 'escape' | 'outside'

export interface DismissOptions {
  onDismiss: (reason: DismissReason, event: Event) => void
  escape?: boolean
  outside?: boolean
  ignore?: () => (Element | null | undefined)[]
  returnTo?: HTMLElement | null
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
      onEscape: opts.escape === false ? undefined : (e) => opts.onDismiss('escape', e),
      // Without outside presses, the overlay still swallows them while on top.
      onOutside: opts.outside === false ? undefined : (e) => opts.onDismiss('outside', e),
      ignore: opts.ignore,
      returnTo: opts.returnTo,
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
