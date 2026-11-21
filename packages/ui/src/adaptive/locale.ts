/**
 * useLocale(own): the locale a date or file component speaks: its own
 * `locale` prop, else the nearest WeftProvider's, else `en-US`. Never the
 * browser's: the server renders with the same locale the browser hydrates
 * with, so the markup matches.
 */
import { computed, type ComputedRef } from 'vue'
import { useWeft } from './context.ts'

export const DEFAULT_LOCALE = 'en-US'

export function useLocale(own: () => string | undefined): ComputedRef<string> {
  const weft = useWeft()
  return computed(() => own() ?? weft.locale.value ?? DEFAULT_LOCALE)
}
