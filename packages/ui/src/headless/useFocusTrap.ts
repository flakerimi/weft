/**
 * useFocusTrap(el, { active, initial?, returnTo?, returnFocus? })
 *
 * Keep keyboard focus inside `el` while `active` is true, as a modal dialog
 * or sheet must (WAI-ARIA APG, Dialog (Modal) pattern).
 *
 * - On activation (once `active` is true and `el` has an element, after the
 *   render that showed it) focus moves to `initial` if given (an element,
 *   ref or getter, read then), else the tabbable element inside marked
 *   `data-weft-autofocus`, else the first tabbable element inside, else
 *   `el` itself (made focusable with `tabindex="-1"`). Tabbable means
 *   focusable, not `tabindex="-1"`, and visible: not hidden by CSS, `hidden`
 *   or `inert`.
 * - Tab from the last tabbable element goes to the first, Shift+Tab from the
 *   first goes to the last. With nothing tabbable inside, Tab stays put.
 * - Focus that lands outside `el` (a click, a script) is brought back to the
 *   element inside that had it last, unless it landed in an overlay stacked
 *   above this one on the `overlayStack` (a menu or tooltip opened from
 *   inside the trap). An overlay below, such as the popover the dialog was
 *   opened from, does not count.
 * - Focus may also go into an element marked `data-weft-trap-exempt` (the
 *   toaster, reached with F8 while a dialog is open), and stays there.
 * - Traps nest: only the most recently activated active trap acts; the one
 *   below resumes when it ends.
 * - Returning focus: when `el` is a stacked overlay (the same element has
 *   `useDismiss`, as every Weft overlay does), the overlay stack is the one
 *   owner of returning focus: on deactivation the trap asks it to return
 *   focus by its rules (`overlayStack.returnFocus`), so an overlay hidden with
 *   `v-show` returns focus too; give `returnTo` to `useDismiss` there. Otherwise, on deactivation or when the scope ends,
 *   focus returns to `returnTo` (an element, ref or getter, read then), by
 *   default the element focused before activation, unless the user has
 *   moved focus out on purpose (see `restoreFocus` in `dom.ts`).
 * - `returnFocus` (default true), a ref or getter read at deactivation:
 *   false leaves focus where it is. For an overlay that stops trapping but
 *   stays open, re-rendered in a form that does not trap (a touch sheet
 *   becoming a pointer popover): `returnFocus: () => !open.value`.
 * - When `el` changes to another element while active (the overlay
 *   re-rendered in another form, a popover becoming a sheet), the trap moves
 *   to it directly: focus goes into the new element without passing through
 *   the opener, and the opener stays the place to return to.
 *
 * Nothing happens on the server.
 */
import { onScopeDispose, toValue, watch, type MaybeRefOrGetter, type Ref } from 'vue'
import { firstFocus, focus, focused, restoreFocus, tabbables } from './dom.ts'
import { overlayStack } from './overlayStack.ts'

export interface FocusTrapOptions {
  active: Readonly<Ref<boolean>>
  initial?: MaybeRefOrGetter<HTMLElement | null | undefined>
  returnTo?: MaybeRefOrGetter<HTMLElement | null | undefined>
  returnFocus?: MaybeRefOrGetter<boolean>
}

interface Trap {
  container: HTMLElement
}

/** Active traps, innermost last. */
const traps: Trap[] = []

export function useFocusTrap(el: Readonly<Ref<HTMLElement | null | undefined>>, opts: FocusTrapOptions): void {
  let trap: Trap | null = null
  let before: HTMLElement | null = null
  let lastInside: HTMLElement | null = null
  let addedTabindex = false

  const isInnermost = () => trap !== null && traps[traps.length - 1] === trap

  function onKeydown(event: KeyboardEvent) {
    if (event.key !== 'Tab' || event.defaultPrevented || !isInnermost()) return
    const container = trap!.container
    const list = tabbables(container)
    const now = focused(container.ownerDocument)
    if (!list.length) {
      event.preventDefault()
      focus(container)
      return
    }
    const first = list[0]!
    const last = list[list.length - 1]!
    if (event.shiftKey && (now === first || now === container)) {
      event.preventDefault()
      focus(last)
    } else if (!event.shiftKey && now === last) {
      event.preventDefault()
      focus(first)
    }
  }

  function onFocusin(event: FocusEvent) {
    if (!isInnermost()) return
    const container = trap!.container
    const target = event.target as HTMLElement | null
    if (target && container.contains(target)) {
      lastInside = target
      return
    }
    if (overlayStack.isAbove(target, container) || target?.closest?.('[data-weft-trap-exempt]')) return
    focus(lastInside?.isConnected && container.contains(lastInside) ? lastInside : (firstFocus(container) ?? container))
  }

  function attach(container: HTMLElement) {
    container.addEventListener('keydown', onKeydown)
    container.ownerDocument.addEventListener('focusin', onFocusin, true)
    if (!container.hasAttribute('tabindex')) {
      container.setAttribute('tabindex', '-1')
      addedTabindex = true
    }
  }

  function detach(container: HTMLElement) {
    container.removeEventListener('keydown', onKeydown)
    container.ownerDocument.removeEventListener('focusin', onFocusin, true)
    if (addedTabindex) container.removeAttribute('tabindex')
    addedTabindex = false
  }

  /** Focus the place to start in `container`, unless focus is in it already. */
  function enter(container: HTMLElement) {
    const now = focused(container.ownerDocument)
    const start = toValue(opts.initial) ?? firstFocus(container) ?? container
    lastInside = start
    if (!(now && container.contains(now))) focus(start)
  }

  function activate(container: HTMLElement) {
    const now = focused(container.ownerDocument)
    before = now && !container.contains(now) ? now : null
    trap = { container }
    traps.push(trap)
    attach(container)
    enter(container)
  }

  /** The overlay re-rendered in another form: trap there, in the same place among nested traps. */
  function swap(container: HTMLElement) {
    detach(trap!.container)
    trap!.container = container
    attach(container)
    enter(container)
  }

  function deactivate() {
    if (!trap) return
    const { container } = trap
    traps.splice(traps.indexOf(trap), 1)
    trap = null
    detach(container)
    lastInside = null
    // A stacked overlay's focus is returned by the overlay stack's rules, now,
    // since the element may stay mounted (v-show) and so never pop.
    if (toValue(opts.returnFocus) !== false && !overlayStack.returnFocus(container)) {
      restoreFocus(toValue(opts.returnTo) ?? before, [container])
    }
    before = null
  }

  watch(
    [el, opts.active],
    ([node, on]) => {
      if (trap && on && node && node !== trap.container) swap(node)
      else if (trap && (!on || !node)) deactivate()
      else if (!trap && on && node && typeof document !== 'undefined') activate(node)
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(deactivate)
}
