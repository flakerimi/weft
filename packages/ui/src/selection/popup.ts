/**
 * usePopup(props, emit): the open state of a Select's or Combobox's list,
 * which opens in a Popover (floating on pointer, a modal bottom sheet on
 * touch) against the component's own field. What they share:
 *
 * - `open` follows `v-model:open` (or `defaultOpen`); nothing opens while
 *   `disabled()`.
 * - `sheet`: whether the list is a touch sheet now (the input decides, and
 *   may change while it is open).
 * - `press()`: the field was pressed: it toggles. (The field is the
 *   Popover's anchor, so the overlay stack does not take the press as one
 *   outside the list.)
 * - `labelText()`: the text of the field's label, for naming the sheet.
 * - `searchField(list, opts)`: the attributes and handlers of the search
 *   field at the top of a list (Select's touch sheet, a Combobox's popup):
 *   a text input with `role="combobox"` that controls the list and carries
 *   its active option; the list's keys work from it, Enter choosing.
 * - `whenFocusLeaves(open, listId, trigger, left)`: the list closed while
 *   focus was in it (its search field) and focus did not come back to the
 *   trigger (Tab left the popup): `left()`, so the field hears a blur.
 */
import { computed, watch, type ComputedRef, type Ref } from 'vue'
import type { Listbox } from './listbox.ts'
import { useWeft } from '../adaptive/context.ts'
import { useControllable } from '../headless/useControllable.ts'
import type { FieldContext } from '../forms/field.ts'

export interface PopupProps {
  open?: boolean
  defaultOpen?: boolean
}

export function usePopup(props: PopupProps, emit: (event: 'update:open', open: boolean) => void, disabled: () => boolean) {
  const open = useControllable(props, 'open', emit, () => props.defaultOpen ?? false) as Ref<boolean>
  const weft = useWeft()
  const sheet = computed(() => weft.input.value === 'touch')
  return {
    open,
    sheet,
    show() {
      if (!disabled()) open.value = true
    },
    hide() {
      open.value = false
    },
    press() {
      if (disabled()) return
      open.value = !open.value
    },
  }
}

/** The text of a FormField's label, if it has one on the page. */
export function labelText(field: FieldContext | null): string | undefined {
  if (!field || typeof document === 'undefined') return undefined
  return document.getElementById(field.labelId)?.textContent?.replace(/\*\s*$/, '').trim() || undefined
}

export function whenFocusLeaves(open: Readonly<Ref<boolean>>, listId: string, trigger: () => Element | null | undefined, left: () => void): void {
  watch(
    open,
    (on) => {
      if (on || typeof document === 'undefined') return
      const panel = document.getElementById(listId)?.closest('.weft-popover')
      if (!panel?.contains(document.activeElement)) return
      // After the overlay stack has returned focus, if it does.
      setTimeout(() => {
        if (!open.value && document.activeElement !== trigger()) left()
      }, 0)
    },
    { flush: 'sync' },
  )
}

export function searchField(
  list: Listbox,
  opts: { value: () => string; onInput: (text: string) => void; placeholder: () => string | undefined; onKeydown?: (event: KeyboardEvent) => void },
): ComputedRef<Record<string, unknown>> {
  const keydown =
    opts.onKeydown ??
    ((event: KeyboardEvent) => {
      if (list.keydown(event, { edges: false, typeahead: false, space: false })) event.preventDefault()
    })
  const input = (event: Event) => opts.onInput((event.target as HTMLInputElement).value)
  return computed(() => ({
    id: `${list.id}-search`,
    class: 'weft-input__field',
    type: 'text',
    role: 'combobox',
    autocomplete: 'off',
    'aria-expanded': 'true',
    'aria-autocomplete': 'list',
    'aria-label': opts.placeholder() ?? 'Search',
    placeholder: opts.placeholder() ?? 'Search…',
    value: opts.value(),
    'aria-controls': list.id,
    'aria-activedescendant': list.activeId.value,
    onInput: input,
    onKeydown: keydown,
  }))
}
