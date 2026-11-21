/**
 * DOM helpers shared by the headless composables. Each is called only from
 * event handlers or watchers that run once an element exists, never at import
 * or setup, so the composables stay safe to run on the server.
 */

/**
 * Whether `el` lays out right to left: the nearest `dir` attribute decides
 * (`dir="rtl"` on any ancestor), else the computed CSS `direction`.
 */
export function isRtl(el: Element | null | undefined): boolean {
  if (!el) return false
  const dir = el.closest('[dir]')?.getAttribute('dir')?.toLowerCase()
  if (dir === 'rtl') return true
  if (dir === 'ltr') return false
  const view = el.ownerDocument.defaultView
  return view ? view.getComputedStyle(el).direction === 'rtl' : false
}

const FOCUSABLE = [
  'a[href]',
  'area[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'iframe',
  'audio[controls]',
  'video[controls]',
  'summary',
  '[contenteditable]:not([contenteditable="false"])',
  '[tabindex]',
].join(',')

/** Whether `el` is rendered: not `display: none` or `visibility: hidden` itself or through an ancestor. */
const visible = (el: HTMLElement) =>
  el.checkVisibility?.({ visibilityProperty: true }) ?? el.getClientRects().length > 0

/**
 * The elements inside `root` that Tab reaches, in document order: focusable,
 * not `tabindex="-1"`, not inside a `hidden` or `inert` subtree, and visible.
 */
export function tabbables(root: Element): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(
    (el) => el.tabIndex >= 0 && !el.closest('[hidden],[inert]') && visible(el),
  )
}

/**
 * Where focus goes first inside `root`: the tabbable element marked
 * `data-weft-autofocus` (a calendar's focused date, which comes after its
 * month buttons), else the first tabbable one.
 */
export function firstFocus(root: Element): HTMLElement | undefined {
  const all = tabbables(root)
  return all.find((el) => el.hasAttribute('data-weft-autofocus')) ?? all[0]
}

/** Focus `el` without scrolling the page to it. */
export function focus(el: HTMLElement | null | undefined): void {
  el?.focus({ preventScroll: true })
}

/** The element that has focus now, or null when it is on the body or nowhere. */
export function focused(doc: Document = document): HTMLElement | null {
  const el = doc.activeElement as HTMLElement | null
  return el && el !== doc.body ? el : null
}

/**
 * Return focus to `target` after an overlay closes, unless the user has moved
 * it somewhere else on purpose: only when focus is now nowhere (it was in an
 * element that went away), or inside one of `within` (the closing overlay and
 * anything still stacked above it).
 */
export function restoreFocus(target: HTMLElement | null | undefined, within: (Element | null | undefined)[]): void {
  if (!target?.isConnected) return
  const now = focused(target.ownerDocument)
  if (now && !within.some((el) => el?.contains(now))) return
  focus(target)
}
