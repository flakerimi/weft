/**
 * createSwipe(direction)
 *
 * Follow a drag towards an edge, for something a finger can swipe away (a
 * Drawer to its edge, a toast off the screen). `direction()` gives the axis
 * (`x` or `y`) and the sign of "towards the edge" (1: towards larger
 * coordinates); it is read when a drag starts, so it can follow the text
 * direction.
 *
 * - `start(event)` begins a drag with that pointer.
 * - `move(event)` gives the distance travelled towards the edge, never less
 *   than 0, or null for another pointer or no drag.
 * - `end(event)` finishes it: `{ distance, speed }`, speed in px per ms over
 *   the last 100 ms of moves (a flick is fast at the end, however slowly the
 *   drag began), or null when there was no drag with that pointer.
 * - `active` says whether a drag is under way.
 *
 * Time is read from `Date.now()`. Nothing touches the DOM.
 */
export interface Swipe {
  start: (event: PointerEvent) => void
  move: (event: PointerEvent) => number | null
  end: (event: PointerEvent) => { distance: number; speed: number } | null
  readonly active: boolean
}

export interface SwipeDirection {
  axis: 'x' | 'y'
  sign: 1 | -1
}

/** ms of moves the speed is measured over. */
const WINDOW = 100

export function createSwipe(direction: () => SwipeDirection): Swipe {
  let drag: { id: number; from: number; dir: SwipeDirection; samples: { t: number; d: number }[] } | null = null
  const along = (event: PointerEvent, dir: SwipeDirection) => (dir.axis === 'y' ? event.clientY : event.clientX)
  const distance = (event: PointerEvent) => Math.max(0, (along(event, drag!.dir) - drag!.from) * drag!.dir.sign)

  return {
    start(event) {
      const dir = direction()
      drag = { id: event.pointerId, from: along(event, dir), dir, samples: [{ t: Date.now(), d: 0 }] }
    },
    move(event) {
      if (!drag || event.pointerId !== drag.id) return null
      const d = distance(event)
      const t = Date.now()
      drag.samples.push({ t, d })
      while (drag.samples.length > 2 && t - drag.samples[1]!.t > WINDOW) drag.samples.shift()
      return d
    },
    end(event) {
      if (!drag || event.pointerId !== drag.id) return null
      const d = distance(event)
      const t = Date.now()
      const recent = drag.samples.filter((s) => t - s.t <= WINDOW)
      const first = recent[0] ?? drag.samples[drag.samples.length - 1]!
      const speed = t > first.t ? Math.max(0, d - first.d) / (t - first.t) : 0
      drag = null
      return { distance: d, speed }
    },
    get active() {
      return drag !== null
    },
  }
}
