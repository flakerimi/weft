/**
 * What a person did to an island's controls before it hydrated: typed into
 * a field (or let a password manager fill it), ticked a box, chose a radio
 * or an option. Hydration writes each control's model value into its
 * element, which would wipe that, so the island runtime calls this just
 * before it mounts. It reads every control's state first, then sets each
 * touched one back to the state the server drew, so hydration matches; the
 * returned function, called once it is mounted, puts the person's state
 * back and fires `input` and `change` on each, as the person's own change
 * would, so the component's handlers give it to the model (`v-model`, a
 * Form's state). A control nobody touched is left alone, so the value the
 * app gave its model stands.
 *
 * Touched means: the control's state differs from what a form reset would
 * give it, which is what the server's HTML set up before anyone acted, the
 * browser's own defaults included.
 * - A text field or textarea: `value` against `defaultValue` (the `value`
 *   attribute, or the textarea's text). An autofill changes `value`, so it
 *   counts.
 * - A checkbox: `checked` against `defaultChecked` (the `checked` attribute).
 * - A radio group (the radios of one name with one form owner, or a radio
 *   with no name alone) is one value: the radio checked now against the one
 *   a reset checks (the last with the `checked` attribute, or none).
 * - A select: its options' selectedness against `defaultSelected` (the
 *   `selected` attribute), where a single drop-down select with no
 *   `selected` option resets to its first option that is not disabled (nor
 *   in a disabled optgroup), as the browser shows it. So a select whose model matches none of its
 *   options is untouched while it shows that first option; a person who
 *   chose that very option cannot be told apart, and the model keeps its
 *   value.
 * Hidden, file and button inputs are skipped, and so is anything that is not
 * an input, textarea or select (a `contenteditable` holds no form value).
 * A radio group is read within the island: radios of one name outside any
 * form, in two islands, are one group to the browser, and hydrating one
 * island checks its model's radio, which unchecks the other's.
 */
export function takeTouched(root: Element): () => void {
  // Read in one pass, then reset, then (after the mount) restore: setting one
  // control (a radio) can change another, so none is written while reading.
  const resets: (() => void)[] = []
  const back: (() => void)[] = []
  const groups = new Map<string, HTMLInputElement[]>()
  let unnamed = 0
  for (const el of root.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>('input, textarea, select')) {
    if (el instanceof HTMLSelectElement) {
      const chosen = [...el.options].map((o) => o.selected)
      const reset = resetOf(el)
      if (chosen.every((on, i) => on === reset[i])) continue
      resets.push(() => reset.forEach((on, i) => (el.options[i]!.selected = on)))
      back.push(() => {
        chosen.forEach((on, i) => (el.options[i]!.selected = on))
        fire(el)
      })
    } else if (el instanceof HTMLInputElement && el.type === 'radio') {
      // A group is read whole, below: setting one radio unchecks the others.
      const key = el.name ? `${el.name}\u0000${formKey(el)}` : `\u0001${unnamed++}`
      groups.set(key, [...(groups.get(key) ?? []), el])
    } else if (el instanceof HTMLInputElement && el.type === 'checkbox') {
      if (el.checked === el.defaultChecked) continue
      const checked = el.checked
      resets.push(() => (el.checked = el.defaultChecked))
      back.push(() => {
        el.checked = checked
        fire(el)
      })
    } else if (el instanceof HTMLTextAreaElement || !SKIP.has(el.type)) {
      if (el.value === el.defaultValue) continue
      const value = el.value
      const focused = document.activeElement === el
      const [start, end] = [el.selectionStart, el.selectionEnd]
      resets.push(() => (el.value = el.defaultValue))
      back.push(() => {
        el.value = value
        if (focused && start !== null && end !== null) el.setSelectionRange(start, end)
        fire(el)
      })
    }
  }
  for (const radios of groups.values()) {
    const now = radios.find((r) => r.checked)
    let reset: HTMLInputElement | undefined
    for (const r of radios) if (r.defaultChecked) reset = r
    // A person can choose a radio, not uncheck one.
    if (now === reset || !now) continue
    resets.push(() => {
      if (reset) reset.checked = true
      else now.checked = false
    })
    back.push(() => {
      now.checked = true
      fire(now)
    })
  }
  // Everything is read: now each touched control goes back to the server's state.
  for (const reset of resets) reset()
  return () => {
    for (const put of back) put()
  }
}

/** Inputs a person does not type into, or whose value cannot be set back. */
const SKIP = new Set(['hidden', 'file', 'submit', 'reset', 'button', 'image'])

/** Radios group by name within one form owner; a form is told apart by identity. */
const forms = new WeakMap<HTMLFormElement, number>()
let formCount = 0
function formKey(el: HTMLInputElement): string {
  const form = el.form
  if (!form) return ''
  if (!forms.has(form)) forms.set(form, ++formCount)
  return String(forms.get(form))
}

/** Each option's selectedness after a form reset. */
function resetOf(el: HTMLSelectElement): boolean[] {
  // defaultSelected and size, read as the attributes they reflect, as every DOM has those.
  const on = [...el.options].map((o) => o.hasAttribute('selected'))
  if (el.multiple) return on
  // A single select keeps one: the last `selected`, else (drawn as a
  // drop-down) the first that is not disabled.
  const last = on.lastIndexOf(true)
  const dropDown = Number(el.getAttribute('size') ?? 0) <= 1
  const pick = last >= 0 ? last : dropDown ? [...el.options].findIndex((o) => !optionDisabled(o)) : -1
  return on.map((_, i) => i === pick)
}

/**
 * Whether an option is disabled as HTML defines it, which is what `:disabled`
 * matches in a browser: its own attribute, or a disabled optgroup around it.
 * Written out, since not every DOM a test runs in matches it so.
 */
function optionDisabled(o: HTMLOptionElement): boolean {
  return o.disabled || (o.parentElement?.tagName === 'OPTGROUP' && o.parentElement.hasAttribute('disabled'))
}

/** The events a person's change fires, for an element the hydration left in the page. */
function fire(el: Element) {
  if (!el.isConnected) return
  el.dispatchEvent(new Event('input', { bubbles: true }))
  el.dispatchEvent(new Event('change', { bubbles: true }))
}
