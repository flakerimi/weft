/**
 * useFloating(reference, floating, { placement, offset, flip, shift, size?, strategy? })
 *
 * Position a floating element (popover, menu, listbox, tooltip) against its
 * reference, with `@floating-ui/dom`: `computePosition` for the numbers and
 * `autoUpdate` to follow scrolling, resizing and layout shifts while both
 * elements exist. Nothing runs until both refs hold an element, it stops when
 * either empties or the scope ends, and nothing happens on the server.
 *
 * - `placement`: a floating-ui placement (`'bottom-start'`, `'top'`, ...),
 *   or a ref or getter for one.
 * - `offset`: the gap to the reference, in px.
 * - `flip`: move to the opposite side when there is no room.
 * - `shift`: slide along the side to stay in view (8 px from the edge).
 * - `size`: set `--weft-floating-reference-width`,
 *   `--weft-floating-available-width` and `--weft-floating-available-height`
 *   on the floating element, for CSS to match the trigger's width or cap the
 *   height to the room left.
 * - `strategy`: `'absolute'` (default) or `'fixed'`.
 *
 * Right to left: `start` and `end` placements mirror when the reference is
 * in a right-to-left context (`dir="rtl"` on an ancestor), even when the
 * floating element is rendered elsewhere, such as at the end of the body.
 *
 * Returns `{ x, y, placement, styles, update }`: the coordinates, the
 * placement actually used after flipping (for `data-side` styling), the
 * style to bind (`position`, `left`, `top`) and a function to recompute now.
 */
import {
  autoUpdate,
  computePosition,
  flip as flipMiddleware,
  offset as offsetMiddleware,
  platform,
  shift as shiftMiddleware,
  size as sizeMiddleware,
  type Middleware,
  type Placement,
  type ReferenceElement,
  type Strategy,
} from '@floating-ui/dom'
import { computed, onScopeDispose, ref, toValue, watch, type ComputedRef, type MaybeRefOrGetter, type Ref } from 'vue'
import { isRtl } from './dom.ts'

export type { Placement } from '@floating-ui/dom'

export interface FloatingOptions {
  placement: MaybeRefOrGetter<Placement>
  offset: number
  flip: boolean
  shift: boolean
  size?: boolean
  strategy?: Strategy
}

export interface Floating {
  x: Readonly<Ref<number>>
  y: Readonly<Ref<number>>
  placement: Readonly<Ref<Placement>>
  styles: ComputedRef<{ position: Strategy; left: string; top: string }>
  update: () => Promise<void>
}

const EDGE = 8

export function useFloating(
  reference: Readonly<Ref<ReferenceElement | null | undefined>>,
  floating: Readonly<Ref<HTMLElement | null | undefined>>,
  opts: FloatingOptions,
): Floating {
  const strategy = opts.strategy ?? 'absolute'
  const x = ref(0)
  const y = ref(0)
  const placement = ref<Placement>(toValue(opts.placement))
  let stop: (() => void) | undefined
  let run = 0

  async function update() {
    const ref_ = reference.value
    const float = floating.value
    if (!ref_ || !float) return
    const mine = ++run
    const middleware: Middleware[] = [offsetMiddleware(opts.offset)]
    if (opts.flip) middleware.push(flipMiddleware({ padding: EDGE }))
    if (opts.shift) middleware.push(shiftMiddleware({ padding: EDGE }))
    if (opts.size) {
      middleware.push(
        sizeMiddleware({
          padding: EDGE,
          apply({ availableWidth, availableHeight, rects, elements }) {
            const s = elements.floating.style
            s.setProperty('--weft-floating-reference-width', `${rects.reference.width}px`)
            s.setProperty('--weft-floating-available-width', `${Math.max(0, availableWidth)}px`)
            s.setProperty('--weft-floating-available-height', `${Math.max(0, availableHeight)}px`)
          },
        }),
      )
    }
    // The reference's direction decides, not the floating element's: overlays
    // often render at the end of the body, outside the rtl subtree.
    const rtl = isRtl(ref_ instanceof Element ? ref_ : ref_.contextElement ?? float)
    const result = await computePosition(ref_, float, {
      placement: toValue(opts.placement),
      strategy,
      middleware,
      platform: { ...platform, isRTL: () => rtl },
    })
    if (mine !== run) return // a newer update started meanwhile
    x.value = result.x
    y.value = result.y
    placement.value = result.placement
  }

  watch(
    [reference, floating],
    ([r, f]) => {
      stop?.()
      stop = undefined
      if (r && f) stop = autoUpdate(r, f, () => void update())
    },
    { immediate: true, flush: 'post' },
  )
  watch(() => toValue(opts.placement), () => void update())
  onScopeDispose(() => {
    stop?.()
    stop = undefined
    run++ // drop a result still on its way
  })

  return {
    x,
    y,
    placement,
    styles: computed(() => ({ position: strategy, left: `${x.value}px`, top: `${y.value}px` })),
    update,
  }
}
