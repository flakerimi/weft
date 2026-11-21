// How a CheckboxGroup and the Checkboxes in it talk: the group holds the list
// of chosen values; each Checkbox shows whether its value is in it and asks
// the group to add or remove it.
import { inject, provide, type InjectionKey } from 'vue'
import type { Size, Tone } from '../types.ts'

export interface CheckboxGroupContext {
  has(value: unknown): boolean
  toggle(value: unknown, on: boolean): void
  name: () => string | undefined
  size: () => Size
  tone: () => Tone
  disabled: () => boolean
}

const key: InjectionKey<CheckboxGroupContext> = Symbol('weft-checkbox-group')

export const provideCheckboxGroup = (group: CheckboxGroupContext) => provide(key, group)
export const useCheckboxGroup = () => inject(key, null)

/** An option of a CheckboxGroup or RadioGroup: a value alone, or with its label and description. */
export type ChoiceItem<V = string | number> =
  | V
  | { value: V; label?: string; description?: string; disabled?: boolean }

/** An item as an object, whichever way it was given. */
export function choice<V>(item: ChoiceItem<V>): { value: V; label: string; description?: string; disabled?: boolean } {
  if (item !== null && typeof item === 'object' && 'value' in (item as object)) {
    const o = item as { value: V; label?: string; description?: string; disabled?: boolean }
    return { ...o, label: o.label ?? String(o.value) }
  }
  return { value: item as V, label: String(item) }
}
