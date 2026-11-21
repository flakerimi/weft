// The buttons an Alert, a Banner, an EmptyState or an ErrorState offers.
import type { Tone, Variant } from '../types.ts'

/** A button given as data: `{ label, onClick }`, or a link with `to`. */
export interface ActionItem {
  /** The button's text. */
  label: string
  /** A Lucide icon before the label. */
  icon?: string
  /** Its colour; default the component's choice. */
  tone?: Tone
  /** Its variant; default the component's choice. */
  variant?: Variant
  /** An address: the button is a link there. */
  to?: string
  /** Unavailable. */
  disabled?: boolean
  /** Work in progress: a spinner, and no presses. */
  loading?: boolean
  /** Run on press. */
  onClick?: (event: MouseEvent) => void
}

/** The text of an error: an Error's message, a string, or a wire-format error's `{ error: { code, message } }`. */
export function errorText(error: unknown): { message?: string; code?: string } {
  if (error == null || error === '') return {}
  if (typeof error === 'string') return { message: error }
  if (error instanceof Error) return { message: error.message }
  if (typeof error === 'object') {
    const body = (error as { error?: unknown }).error
    const inner = (body && typeof body === 'object' ? body : error) as { code?: unknown; message?: unknown }
    return {
      message: typeof inner.message === 'string' ? inner.message : undefined,
      code: typeof inner.code === 'string' ? inner.code : undefined,
    }
  }
  return { message: String(error) }
}
