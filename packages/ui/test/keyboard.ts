// Pressing a key as a browser does. happy-dom dispatches key events but
// performs no activation, so this adds what the HTML spec says a browser does
// when the key event is not cancelled: Enter on a link with an href, or on a
// button, fires click at keydown; Space on a button fires click at keyup;
// Enter in a text field submits its form implicitly, by clicking the form's
// default button (its first submit button) unless that is disabled, or, with
// no submit button, by submitting the form when that field is its only one.
// Keys go to the focused element, so an element that cannot take focus (a
// disabled button, a link without an href) receives nothing. happy-dom lets
// an <a> without an href take focus, which browsers do not; this follows the
// browsers.

/**
 * Whether a form control is disabled as a browser has it: by its own
 * attribute, or by a disabled fieldset around it (outside that fieldset's
 * first legend). happy-dom reads only the attribute.
 */
export function actuallyDisabled(el: Element): boolean {
  if (!(el instanceof HTMLButtonElement || el instanceof HTMLInputElement || el instanceof HTMLSelectElement || el instanceof HTMLTextAreaElement)) return false
  if (el.disabled) return true
  for (let set = el.parentElement?.closest('fieldset'); set; set = set.parentElement?.closest('fieldset')) {
    if (set.disabled && !set.querySelector(':scope > legend')?.contains(el)) return true
  }
  return false
}

/** Whether a browser would let `el` take focus, for the cases happy-dom gets wrong. */
const focusable = (el: Element) =>
  !(el instanceof HTMLAnchorElement && !el.hasAttribute('href') && !el.hasAttribute('tabindex')) && !actuallyDisabled(el)

/** Input types whose Enter submits the form implicitly. */
const IMPLICIT = new Set(['text', 'search', 'url', 'tel', 'email', 'password', 'number', 'date', 'month', 'week', 'time', 'datetime-local'])

/** Whether Enter in `el` submits its form implicitly. */
const implicit = (el: Element): el is HTMLInputElement => el instanceof HTMLInputElement && IMPLICIT.has(el.type) && !!el.form

/** The default button Enter in `el` would click, for implicit submission. */
function defaultButton(el: HTMLInputElement): HTMLButtonElement | HTMLInputElement | null {
  return el.form!.querySelector<HTMLButtonElement | HTMLInputElement>('button[type="submit"], button:not([type]), input[type="submit"]')
}

/** With no default button: whether `el` is the only field of its form that would submit it, so Enter submits. */
const onlyField = (el: HTMLInputElement) =>
  [...el.form!.elements].filter((f) => implicit(f) && !actuallyDisabled(f)).length === 1

const activatable = (el: Element, key: string) =>
  el instanceof HTMLButtonElement ? !actuallyDisabled(el) : key === 'Enter' && el instanceof HTMLAnchorElement && el.hasAttribute('href')

/** Focus `el`, then press `key` on whatever has focus, as a browser would; true when it activated something. */
export function press(el: HTMLElement, key: 'Enter' | ' '): boolean {
  if (!focusable(el)) return false
  el.focus()
  const target = document.activeElement
  if (!(target instanceof HTMLElement) || target === document.body) return false
  const down = target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
  if (key === 'Enter' && down && activatable(target, key)) {
    target.click()
    return true
  }
  if (key === 'Enter' && down && implicit(target)) {
    const submitter = defaultButton(target)
    if (submitter) {
      if (actuallyDisabled(submitter)) return false
      submitter.click()
      return true
    }
    if (onlyField(target)) {
      target.form!.requestSubmit()
      return true
    }
  }
  const up = target.dispatchEvent(new KeyboardEvent('keyup', { key, bubbles: true, cancelable: true }))
  if (key === ' ' && down && up && target instanceof HTMLButtonElement && activatable(target, key)) {
    target.click()
    return true
  }
  return false
}

/**
 * Record the navigations a link performs. happy-dom follows a link by calling
 * open() on the element's own window, which is not the test's global one; it
 * keeps that window under the element's Symbol(window).
 */
export function navigations(el: Element): { urls: string[]; restore(): void } {
  const key = Object.getOwnPropertySymbols(el).find((s) => s.description === 'window')
  const win = key ? ((el as unknown as Record<symbol, Window>)[key]) : undefined
  if (!win) throw new Error('navigations(): this happy-dom keeps no window on the element; update the helper.')
  const urls: string[] = []
  const open = win.open
  win.open = ((url?: string | URL) => {
    urls.push(String(url))
    return null
  }) as typeof win.open
  return { urls, restore: () => { win.open = open } }
}
