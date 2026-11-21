// Shared by the overlay tests.
import { nextTick } from 'vue'

/** Wait until Vue has flushed, the overlay stack has settled (it leaves a microtask late) and positions resolved. */
export async function settle(): Promise<void> {
  await nextTick()
  await new Promise((r) => setTimeout(r, 0))
  await nextTick()
}

export const $ = <T extends Element = HTMLElement>(sel: string, from: ParentNode = document) => from.querySelector(sel) as T
export const $$ = <T extends Element = HTMLElement>(sel: string, from: ParentNode = document) => [...from.querySelectorAll(sel)] as T[]
export const active = () => document.activeElement

/** Press a key on the focused element (or `target`), as a bubbling, cancelable keydown. Returns the event. */
export function key(name: string, init: KeyboardEventInit = {}, target: Element | null = document.activeElement) {
  const event = new KeyboardEvent('keydown', { key: name, bubbles: true, cancelable: true, ...init })
  ;(target ?? document.body).dispatchEvent(event)
  return event
}

/** A pointer press on `target`: pointerdown, then (for touch) the click that follows, as the overlay stack hears it. */
export function press(target: Element, pointerType: 'mouse' | 'touch' = 'mouse') {
  target.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true, pointerType }))
  target.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
}

/** Move focus from where it is to `to`, as a browser does: focusout with relatedTarget, then focus. */
export function moveFocus(to: HTMLElement) {
  const from = document.activeElement
  from?.dispatchEvent(new FocusEvent('focusout', { bubbles: true, relatedTarget: to }))
  to.focus()
}

export const locked = () => document.documentElement.style.overflow === 'hidden'
export const overlays = () => document.getElementById('weft-overlays')

export function reset() {
  document.body.innerHTML = ''
  document.documentElement.removeAttribute('style')
  document.body.removeAttribute('style')
}
