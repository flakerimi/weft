// What a Form gives the components in it, apart from the field wiring in
// field.ts (which re-exports all of this). Button and Link import only this
// file, so a page whose only form-aware component is a Button does not load
// the form controls' helpers with it.
import { inject, type InjectionKey } from 'vue'

/** What a Form gives each FormField in it. */
export interface FormContext {
  error(name: string): string | undefined
  required(name: string): boolean
  /** Disabled, or busy submitting: every control in the form is disabled. */
  disabled: () => boolean
  /** Busy submitting (`loading`, or an async submit handler running). */
  busy: () => boolean
  blur(name: string): void
  input(name: string): void
}

export const formKey: InjectionKey<FormContext> = Symbol('weft-form')

export const useForm = () => inject(formKey, null)

/**
 * For a control that is not a form control (Button, Link): the Form it is
 * in, as `disabled` (disabled or busy) and `busy`. A disabled fieldset
 * disables native buttons but not links, and a busy Form keeps its buttons
 * focusable (see Form.vue), so these read the Form themselves.
 */
export function useFormState(): { disabled: () => boolean; busy: () => boolean } | null {
  const form = useForm()
  return form ? { disabled: form.disabled, busy: form.busy } : null
}
