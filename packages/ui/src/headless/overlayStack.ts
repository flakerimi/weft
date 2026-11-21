/**
 * overlayStack: the one ordered list of open overlays on the page (dialogs,
 * sheets, popovers, menus, tooltips), most recent last.
 *
 * - `push(id, entry?)` puts an overlay on top (an id pushed again moves up);
 *   `pop(id)` takes it out, wherever it is.
 * - `top()`, `isTop(id)` and `ids()` are reactive: read them in a computed,
 *   a watcher or a template and they update.
 * - Escape goes to the top overlay only, once per press: one keydown listener
 *   on the document hands it to the top entry's `onEscape`. A widget inside
 *   that handles Escape itself calls `preventDefault()` and the stack leaves
 *   that press alone. If the top overlay has no `onEscape`, or it returns
 *   false (it does not close on Escape now), nothing closes and the press is
 *   left unhandled.
 * - A press outside goes to the top overlay only, the same way, through its
 *   `onOutside`. Outside means not inside its `el` and not inside any of its
 *   `ignore` elements (its trigger, so a press on the trigger toggles instead
 *   of closing and reopening). A touch press counts on the click that follows
 *   it, so a scroll that starts outside does not close anything. A
 *   `passThrough` entry (a tooltip) takes a press outside it and passes it
 *   on to the entry below, so one press closes a tooltip and the menu it
 *   shows over; Escape still stops at it. A press on one of its `ignore`
 *   elements (the tooltip's trigger) is not outside it, but goes on down
 *   just the same. An entry that declines a press (`onOutside` returning
 *   false, as a modal does) passes nothing on.
 * - Restoring focus: the stack is the one owner of returning focus when an
 *   overlay closes. `pop` returns focus to the entry's `returnTo` (an element,
 *   ref or getter, read at pop), by default the element that had focus when
 *   it was pushed (its opener), but only when focus is now inside the closing
 *   overlay or an overlay above it, or on nothing (it was in an element that
 *   went away). A user who clicked into another field keeps their focus, and
 *   an overlay closed by a press outside it returns no focus at all: the
 *   press put the user's attention elsewhere. An overlay that never takes
 *   focus (a tooltip) sets `returnFocus: false` and returns none. An
 *   overlay that declines the press (it stays open past the end of that
 *   task) returns focus as usual when it closes later. `returnFocus(el)`
 *   applies the same rules without popping, for an overlay hidden but still
 *   mounted (`v-show`); `useFocusTrap` calls it, and returns focus itself
 *   only when its element is not a stacked overlay.
 *
 * Components normally do not call this directly: `useDismiss` pushes and pops
 * while the overlay's element exists. Document listeners exist only while the
 * stack is not empty, and nothing is pushed on the server.
 */
import { shallowRef, toValue, type MaybeRefOrGetter } from 'vue'
import { focused, restoreFocus } from './dom.ts'

export interface OverlayEntry {
  /** The overlay's element: presses inside it are not outside, and focus inside it is restored on pop. */
  el?: () => HTMLElement | null | undefined
  /** Called with the keydown when Escape is pressed while this overlay is on top; false means not taken. */
  onEscape?: (event: KeyboardEvent) => void | boolean
  /** Called with the event when a press lands outside this overlay while it is on top; false means not taken. */
  onOutside?: (event: PointerEvent | MouseEvent) => void | boolean
  /** Elements whose presses do not count as outside, such as the overlay's trigger. */
  ignore?: () => (Element | null | undefined)[]
  /** Where focus returns on pop, read then; defaults to the element focused at push, when it was outside the overlay. */
  returnTo?: MaybeRefOrGetter<HTMLElement | null | undefined>
  /** Presses go on to the entry below as well (a tooltip); Escape does not. */
  passThrough?: boolean
  /** False: this overlay returns no focus (it never takes any, as a tooltip). Read at pop. */
  returnFocus?: MaybeRefOrGetter<boolean | undefined>
}

interface Entry extends OverlayEntry {
  id: string
  /** The element of the overlay as it was last seen, kept for pop after it is gone. */
  last: HTMLElement | null
  opener: HTMLElement | null
  /** A press outside was taken by this overlay, so its closing returns no focus. */
  closedOutside: boolean
}

const entries: Entry[] = []
const order = shallowRef<readonly string[]>([])
let listening = false
/** The touch press waiting for its click: the entries it was outside of. */
let pendingTouch: Entry[] = []

const elOf = (e: Entry) => {
  const el = e.el?.() ?? null
  if (el) e.last = el
  return el ?? e.last
}
const topEntry = () => entries[entries.length - 1]

/** Where a press landed for `e`: in its element, on one of its ignored elements (its trigger), or outside. */
function placeOf(e: Entry, target: EventTarget | null): 'inside' | 'ignored' | 'outside' {
  if (!(target instanceof Node) || !target.isConnected) return 'inside'
  if (elOf(e)?.contains(target)) return 'inside'
  return (e.ignore?.() ?? []).some((el) => el?.contains(target)) ? 'ignored' : 'outside'
}
const isOutside = (e: Entry, target: EventTarget | null) => placeOf(e, target) === 'outside'

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || event.defaultPrevented || event.isComposing) return
  const top = topEntry()
  if (!top?.onEscape) return
  if (top.onEscape(event) === false) return
  top.closedOutside = false
  event.preventDefault()
}

/** The top overlay takes a press outside it. */
function outside(e: Entry, event: PointerEvent | MouseEvent) {
  // Set first: the handler may close the overlay, and pop it, at once. A
  // component closes in the microtasks that follow (its render, then the pop),
  // so the flag lasts until the end of this task. If the overlay is still open
  // then, it declined (an unsaved-changes guard), and a later close returns
  // focus as usual.
  e.closedOutside = true
  if (e.onOutside?.(event) === false) e.closedOutside = false
  else setTimeout(() => (e.closedOutside = false), 0)
}

/** Return focus for `e` by the rules above; `at` is its index, for the overlays above it. */
function returnFocusFor(e: Entry, at: number) {
  if (e.closedOutside || toValue(e.returnFocus) === false) return
  const within = [e, ...entries.slice(at + 1)].map(elOf)
  restoreFocus(toValue(e.returnTo) ?? e.opener, within)
}

/** The entries a press goes to, top first: the pass-through ones on top, then the first that is not. */
function reached(): Entry[] {
  const out: Entry[] = []
  for (let i = entries.length - 1; i >= 0; i--) {
    out.push(entries[i]!)
    if (!entries[i]!.passThrough) break
  }
  return out
}

function onPointerdown(event: PointerEvent) {
  pendingTouch = []
  for (const e of reached()) {
    if (!e.onOutside) {
      if (e.passThrough) continue
      return
    }
    const place = placeOf(e, event.target)
    if (place === 'inside') {
      // A press inside this overlay is outside none below it that it covers.
      e.closedOutside = false
      return
    }
    if (place === 'ignored') {
      e.closedOutside = false
      // A pass-through overlay's trigger (a toolbar button with a tooltip)
      // is not part of the overlay below: the press goes on down.
      if (e.passThrough) continue
      return
    }
    if (event.pointerType === 'touch') pendingTouch.push(e)
    else outside(e, event)
  }
}

function onClick(event: MouseEvent) {
  const waiting = pendingTouch
  pendingTouch = []
  for (const e of waiting) if (entries.includes(e) && isOutside(e, event.target)) outside(e, event)
}

function listen(on: boolean) {
  if (on === listening) return
  listening = on
  if (on) {
    document.addEventListener('keydown', onKeydown)
    document.addEventListener('pointerdown', onPointerdown, true)
    document.addEventListener('click', onClick, true)
  } else {
    document.removeEventListener('keydown', onKeydown)
    document.removeEventListener('pointerdown', onPointerdown, true)
    document.removeEventListener('click', onClick, true)
  }
}

const publish = () => {
  order.value = entries.map((e) => e.id)
}

export const overlayStack = {
  /** Put the overlay `id` on top of the stack. */
  push(id: string, entry: OverlayEntry = {}): void {
    const at = entries.findIndex((e) => e.id === id)
    if (at >= 0) entries.splice(at, 1)
    const e: Entry = { ...entry, id, last: null, opener: null, closedOutside: false }
    const el = elOf(e)
    const now = typeof document === 'undefined' ? null : focused()
    e.opener = now && !el?.contains(now) ? now : null
    entries.push(e)
    publish()
    if (typeof document !== 'undefined') listen(true)
  },

  /** Take the overlay `id` out of the stack and return focus as described above. */
  pop(id: string): void {
    const at = entries.findIndex((e) => e.id === id)
    if (at < 0) return
    const e = entries[at]!
    // Focus may be in this overlay or in one stacked above it, which closes with it.
    returnFocusFor(e, at)
    entries.splice(at, 1)
    pendingTouch = pendingTouch.filter((p) => p !== e)
    publish()
    if (!entries.length && typeof document !== 'undefined') listen(false)
  },

  /**
   * Return focus for the overlay holding `el` as `pop` would, without taking
   * it off the stack: for an overlay hidden but still mounted (`v-show`).
   * Returns whether an overlay holds `el`.
   */
  returnFocus(el: Element): boolean {
    let at = -1
    entries.forEach((e, i) => {
      if (elOf(e)?.contains(el)) at = i
    })
    if (at < 0) return false
    returnFocusFor(entries[at]!, at)
    return true
  },

  /** The id of the top overlay, or undefined when none is open. Reactive. */
  top(): string | undefined {
    const ids = order.value
    return ids[ids.length - 1]
  },

  /** Whether `id` is the top overlay. Reactive. */
  isTop(id: string): boolean {
    return this.top() === id
  },

  /** Every open overlay's id, bottom first. Reactive. */
  ids(): readonly string[] {
    return order.value
  },

  /** Whether `node` is inside any open overlay's element. */
  contains(node: Node | null | undefined): boolean {
    return !!node && entries.some((e) => elOf(e)?.contains(node))
  },

  /**
   * Whether `node` is inside an overlay stacked above the one holding `el`
   * (a menu opened from a dialog, seen from the dialog). When no overlay holds
   * `el`, any open overlay counts.
   */
  isAbove(node: Node | null | undefined, el: Element): boolean {
    if (!node) return false
    let at = -1
    entries.forEach((e, i) => {
      if (elOf(e)?.contains(el)) at = i
    })
    return entries.slice(at + 1).some((e) => elOf(e)?.contains(node))
  },
}
