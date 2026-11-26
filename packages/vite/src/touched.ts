/**
 * What a person did to an island's controls before it hydrated: typed into
 * a field (or let a password manager fill it), ticked a box, chose a radio
 * or an option. Hydration writes each control's model value into its
 * element, which would wipe that, so the island runtime calls this just
 * before it mounts: each control whose value differs from the one the server
 * drew (its default) is set back to the server's, so hydration matches, and
 * the returned function, called once it is mounted, puts the person's value
 * back and fires `input` and `change` on it, as the person's own change
 * would, so the component's handlers give it to the model (`v-model`, a
 * Form's state). A control nobody touched is left alone, so the value the
 * app gave its model stands. Hidden, file and button inputs are skipped.
 */
export function takeTouched(root: Element): () => void {
  const back: (() => void)[] = []
  for (const el of root.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>('input, textarea, select')) {
    if (el instanceof HTMLSelectElement) {
      const chosen = [...el.options].map((o) => o.selected)
      if (chosen.every((on, i) => on === el.options[i]!.defaultSelected)) continue
      for (const o of el.options) o.selected = o.defaultSelected
      back.push(() => {
        chosen.forEach((on, i) => (el.options[i]!.selected = on))
        fire(el)
      })
    } else if (el instanceof HTMLInputElement && (el.type === 'checkbox' || el.type === 'radio')) {
      if (el.checked === el.defaultChecked) continue
      const checked = el.checked
      el.checked = el.defaultChecked
      // A radio is chosen by checking it; the one it unchecked follows.
      if (el.type === 'radio' && !checked) continue
      back.push(() => {
        el.checked = checked
        fire(el)
      })
    } else if (el instanceof HTMLTextAreaElement || !SKIP.has(el.type)) {
      if (el.value === el.defaultValue) continue
      const value = el.value
      const focused = document.activeElement === el
      const [start, end] = [el.selectionStart, el.selectionEnd]
      el.value = el.defaultValue
      back.push(() => {
        el.value = value
        if (focused && start !== null && end !== null) el.setSelectionRange(start, end)
        fire(el)
      })
    }
  }
  return () => {
    for (const put of back) put()
  }
}

/** Inputs a person does not type into, or whose value cannot be set back. */
const SKIP = new Set(['hidden', 'file', 'submit', 'reset', 'button', 'image'])

/** The events a person's change fires, for an element the hydration left in the page. */
function fire(el: Element) {
  if (!el.isConnected) return
  el.dispatchEvent(new Event('input', { bubbles: true }))
  el.dispatchEvent(new Event('change', { bubbles: true }))
}
