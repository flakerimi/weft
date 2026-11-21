// How a form control learns where it is. A FormField tells the control inside
// it its id, what describes it (the hint and the error), whether it is invalid
// or required, and hears when it is left or changed; a Form tells each
// FormField its error and whether its field is required; a FieldGroup tells
// the controls in it their size. Every form control reads all of this through
// `useField`, so Select, Combobox and DatePicker join forms the same way.
import { computed, inject, provide, useAttrs, type ComputedRef, type InjectionKey } from 'vue'
import { joinIds } from './dom.ts'
import type { Size } from '../types.ts'

/** What a FormField gives the control inside it. */
export interface FieldContext {
  /** The control's id, which the field's label is `for`. */
  id: string
  /** The label's id, for a group control (radios, checkboxes, a slider) to be `aria-labelledby`. */
  labelId: string
  name: () => string | undefined
  /** The ids of the hint and the error shown now, space-separated, or undefined. */
  describedBy: () => string | undefined
  invalid: () => boolean
  required: () => boolean
  disabled: () => boolean
  size: () => Size | undefined
  /** The control lost focus. */
  blur(): void
  /** The control's value changed. */
  input(): void
}

/** What a Form gives each FormField in it. */
export interface FormContext {
  error(name: string): string | undefined
  required(name: string): boolean
  disabled: () => boolean
  blur(name: string): void
  input(name: string): void
}

/** What a FieldGroup gives the controls in it. */
export interface FieldGroupContext {
  size: () => Size | undefined
  orientation: () => 'horizontal' | 'vertical'
}

export const fieldKey: InjectionKey<FieldContext> = Symbol('weft-field')
export const formKey: InjectionKey<FormContext> = Symbol('weft-form')
export const fieldGroupKey: InjectionKey<FieldGroupContext> = Symbol('weft-field-group')

export const provideField = (field: FieldContext) => provide(fieldKey, field)
export const provideForm = (form: FormContext) => provide(formKey, form)
export const provideFieldGroup = (group: FieldGroupContext) => provide(fieldGroupKey, group)
export const useForm = () => inject(formKey, null)
/**
 * For a group control (CheckboxGroup, RadioGroup): the group takes its
 * FormField's wiring itself, so the controls inside it must not, or each would
 * take the field's id.
 */
export const hideField = () => provide(fieldKey, null as unknown as FieldContext)

/** What a control's own props say, which win over what its field and group say. */
export interface ControlProps {
  id?: string
  name?: string
  size?: Size
  disabled?: boolean
}

/** A form control's view of where it is: its field's wiring, merged with its own props. */
export interface Field {
  /** The FormField it is in, if any. */
  field: FieldContext | null
  /** Inside a FormField, the field's id (so its label points here); else the control's own `id`. */
  id: ComputedRef<string | undefined>
  name: ComputedRef<string | undefined>
  /** The control's own size, else its FieldGroup's, else its FormField's, else md. */
  size: ComputedRef<Size>
  disabled: ComputedRef<boolean>
  invalid: ComputedRef<boolean>
  /** The aria attributes for the element that takes focus. */
  aria: ComputedRef<{ 'aria-describedby'?: string; 'aria-invalid'?: 'true'; 'aria-required'?: 'true' }>
  blur(): void
  input(): void
}

/** For a form control, during setup: its FormField and FieldGroup, if any, merged with `props`. */
export function useField(props: ControlProps): Field {
  const field = inject(fieldKey, null)
  const group = inject(fieldGroupKey, null)
  return {
    field,
    id: computed(() => (field ? field.id : props.id)),
    name: computed(() => props.name ?? field?.name()),
    size: computed(() => props.size ?? group?.size() ?? field?.size() ?? 'md'),
    disabled: computed(() => !!props.disabled || !!field?.disabled()),
    invalid: computed(() => !!field?.invalid()),
    aria: computed(() => ({
      'aria-describedby': field?.describedBy(),
      'aria-invalid': field?.invalid() ? 'true' : undefined,
      'aria-required': field?.required() ? 'true' : undefined,
    })),
    blur: () => field?.blur(),
    input: () => field?.input(),
  }
}

/**
 * For a control that sets `inheritAttrs: false`: the attributes for its
 * focusable element (all but class, style and aria-describedby, which the box
 * takes or `describedBy` merges), and its aria-describedby: its own first,
 * then the field's, then any ids the control adds.
 */
export function useControlAttrs(f: Field): {
  own: ComputedRef<Record<string, unknown>>
  describedBy: (...more: unknown[]) => string | undefined
} {
  const attrs = useAttrs()
  return {
    own: computed(() => {
      const { class: _class, style: _style, 'aria-describedby': _describedBy, ...rest } = attrs
      return rest
    }),
    describedBy: (...more) => joinIds(attrs['aria-describedby'], f.aria.value['aria-describedby'], ...more),
  }
}
