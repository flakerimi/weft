/** Small DOM helpers for the headless tests (happy-dom). */

/** Build elements from HTML, attach them to the body and return the wrapper. */
export function html(markup: string): HTMLElement {
  const root = document.createElement('div')
  root.innerHTML = markup
  document.body.appendChild(root)
  return root
}

export const $ = <T extends Element = HTMLElement>(sel: string, from: ParentNode = document) => from.querySelector(sel) as T

/** Press a key on the focused element (or `target`), as a bubbling keydown. Returns the event. */
export function key(name: string, init: KeyboardEventInit = {}, target: Element | null = document.activeElement) {
  const event = new KeyboardEvent('keydown', { key: name, bubbles: true, cancelable: true, ...init })
  ;(target ?? document.body).dispatchEvent(event)
  return event
}

/** A pointer press on `target`: pointerdown, then (for touch) the click that follows. */
export function press(target: Element, pointerType: 'mouse' | 'touch' = 'mouse') {
  target.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, cancelable: true, pointerType }))
  target.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
}

export const active = () => document.activeElement

export function reset() {
  document.body.innerHTML = ''
  document.documentElement.removeAttribute('style')
  document.body.removeAttribute('style')
}
