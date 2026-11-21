// Shared by the form control tests.

/** A length in px; happy-dom resolves the var()s in max() but leaves max() itself to the reader. */
export function px(value: string): number {
  const max = /^max\((.*)\)$/.exec(value)
  return max ? Math.max(...max[1]!.split(',').map((v) => parseFloat(v))) : parseFloat(value)
}

/** Type into a text field as a person would: its value changes, then it fires input. */
export function type(el: HTMLInputElement | HTMLTextAreaElement, value: string): void {
  el.value = value
  el.dispatchEvent(new Event('input', { bubbles: true }))
}

/** Press a key on `el` (it gets the keydown), returning the event so its default can be checked. */
export function key(el: Element, k: string, init: KeyboardEventInit = {}): KeyboardEvent {
  const event = new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true, ...init })
  el.dispatchEvent(event)
  return event
}

/**
 * Press Space on a checkbox, radio or switch as a browser does: focus it,
 * keydown, then at keyup (neither cancelled) a click, which toggles it and
 * fires change unless the click is cancelled. True when it clicked.
 */
export function space(el: HTMLInputElement): boolean {
  el.focus()
  if (document.activeElement !== el) return false
  const down = el.dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true, cancelable: true }))
  const up = el.dispatchEvent(new KeyboardEvent('keyup', { key: ' ', bubbles: true, cancelable: true }))
  if (!down || !up || el.disabled) return false
  el.click()
  return true
}

/** Wait until Vue has flushed everything queued, ticks included. */
export const settle = () => new Promise<void>((resolve) => setTimeout(resolve, 0))
