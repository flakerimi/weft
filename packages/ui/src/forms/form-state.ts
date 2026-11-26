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
  /**
   * For a number control, during setup: its field, by the name `name` gives,
   * is left out of what the Form submits while it is empty (null), so the
   * server reads it as not sent; Go refuses null for a number that is not a
   * pointer. It returns the function that forgets the control.
   */
  absentWhenEmpty?(name: () => string | undefined): () => void
}

/**
 * What a Form's template ref gives, for `useTemplateRef<FormApi>('form')`:
 * the methods and state Form exposes, its refs read as values.
 */
export interface FormApi {
  /** Show these errors (a server's answer), in place of the ones shown now; `{}` clears them. */
  setErrors(errors: Record<string, string>): void
  /** Check every field and show the errors, as a submit would, without submitting; true when all pass. */
  validate(): boolean
  /** Clear every error, and forget the submit (fields are checked on leaving only after the next one). */
  clear(): void
  /** The errors shown now, by field name. */
  readonly errors: Record<string, string>
  /** Busy: `loading`, or an async submit handler still running. */
  readonly loading: boolean
  /** Submit as the submit button would: check, then run the handler or emit `error`. */
  submit(): void
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
