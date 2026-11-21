// The items of a ToggleGroup.
import type { Tone, Variant } from '../types.ts'

/** One button of a ToggleGroup. */
export interface ToggleGroupItem {
  /** The text; with only an icon, give `ariaLabel`. */
  label?: string
  /** What it stands for in the group's value; default its place. */
  value?: string | number
  /** A Lucide icon before the label. */
  icon?: string
  /** A Lucide icon after the label. */
  trailingIcon?: string
  /** Its name for assistive technology, for an icon alone. */
  ariaLabel?: string
  /** Its colour and variant, in place of the group's, while not pressed. */
  tone?: Tone
  variant?: Variant
  /** It cannot be pressed, and the arrow keys pass it by. */
  disabled?: boolean
}

/** An item's value: its own, else its place. */
export const toggleValue = (item: ToggleGroupItem, index: number): string | number => item.value ?? index
