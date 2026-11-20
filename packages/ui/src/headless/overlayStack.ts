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
 *   that press alone. If the top overlay has no `onEscape`, nothing closes.
 * - A press outside goes to the top overlay only, the same way, through its
 *   `onOutside`. Outside means not inside its `el` and not inside any of its
 *   `ignore` elements (its trigger, so a press on the trigger toggles instead
 *   of closing and reopening). A touch press counts on the click that follows
 *   it, so a scroll that starts outside does not close anything.
 * - Restoring focus: `pop` returns focus to the entry's `returnTo`, by default
 *   the element that had focus when it was pushed (its opener), but only when
 *   focus is now inside the closing overlay or an overlay above it, or on
 *   nothing (it was in an element that went away). A user who clicked into
 *   another field keeps their focus.
 *
 * Components normally do not call this directly: `useDismiss` pushes and pops
 * while the overlay's element exists. Document listeners exist only while the
 * stack is not empty, and nothing is pushed on the server.
 */
import { shallowRef } from 'vue'
import { focused, restoreFocus } from './dom.ts'

export interface OverlayEntry {
  /** The overlay's element: presses inside it are not outside, and focus inside it is restored on pop. */
  el?: () => HTMLElement | null | undefined
  /** Called with the keydown when Escape is pressed while this overlay is on top. */
  onEscape?: (event: KeyboardEvent) => void
  /** Called with the event when a press lands outside this overlay while it is on top. */
  onOutside?: (event: PointerEvent | MouseEvent) => void
  /** Elements whose presses do not count as outside, such as the overlay's trigger. */
  ignore?: () => (Element | null | undefined)[]
  /** Where focus returns on pop; defaults to the element focused at push, when it was outside the overlay. */
  returnTo?: HTMLElement | null
}

interface Entry extends OverlayEntry {
  id: string
  /** The element of the overlay as it was last seen, kept for pop after it is gone. */
  last: HTMLElement | null
  opener: HTMLElement | null
}

const entries: Entry[] = []
const order = shallowRef<readonly string[]>([])
let listening = false
/** The touch press waiting for its click: the entry it was outside of. */
let pendingTouch: Entry | null = null

const elOf = (e: Entry) => {
  const el = e.el?.() ?? null
  if (el) e.last = el
  return el ?? e.last
}
const topEntry = () => entries[entries.length - 1]

function isOutside(e: Entry, target: EventTarget | null): boolean {
  if (!(target instanceof Node) || !target.isConnected) return false
  if (elOf(e)?.contains(target)) return false
  return !(e.ignore?.() ?? []).some((el) => el?.contains(target))
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape' || event.defaultPrevented || event.isComposing) return
  const top = topEntry()
  if (!top?.onEscape) return
  event.preventDefault()
  top.onEscape(event)
}

function onPointerdown(event: PointerEvent) {
  pendingTouch = null
  const top = topEntry()
  if (!top?.onOutside || !isOutside(top, event.target)) return
  if (event.pointerType === 'touch') pendingTouch = top
  else top.onOutside(event)
}

function onClick(event: MouseEvent) {
  const waiting = pendingTouch
  pendingTouch = null
  if (waiting && waiting === topEntry() && isOutside(waiting, event.target)) waiting.onOutside?.(event)
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
    const e: Entry = { ...entry, id, last: null, opener: null }
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
    const [e] = entries.splice(at, 1) as [Entry]
    // Focus may be in this overlay or in one stacked above it, which closes with it.
    const within = [e, ...entries.slice(at)].map(elOf)
    if (pendingTouch === e) pendingTouch = null
    publish()
    if (!entries.length && typeof document !== 'undefined') listen(false)
    restoreFocus(e.returnTo ?? e.opener, within)
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
}
