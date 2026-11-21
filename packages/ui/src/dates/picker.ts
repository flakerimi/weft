/**
 * What DatePicker and DateRangePicker share.
 *
 * - `usePicker(props, emit, field)`: whether the calendar is open
 *   (`v-model:open`), whether the picker is in its touch form now (the
 *   input decides, live), its locale, whether it is blocked (disabled or
 *   read-only: nothing opens), the title of its dialog (the field's label,
 *   else `fallback`), and the box's focus bookkeeping: `focusout(…)` tells
 *   when focus has left the whole picker (its box and its calendar), for
 *   the field's blur.
 * - `useTypedDate(opts)`: one text field holding a date typed in the
 *   locale. It shows the value as the locale writes it; on `commit()` (a
 *   change, Enter) it reads the text, and binds the date when it is one the
 *   calendar allows, null when the text is empty, and null with the text
 *   kept and marked `invalid` otherwise, so the person can correct it.
 *   `typed()` is the date the text says now, for the calendar to open on.
 * - `keepFocusOnSwitch(touch, from, to)`: the field leaving with the
 *   switch to touch, focus moves to the button that stays.
 * - `cycleTab(event, panel)`: Tab inside the floating calendar goes round
 *   its buttons, as the APG's date picker dialog asks (the popover is
 *   teleported to the end of the page, so leaving it would land past
 *   everything); the touch sheet traps Tab itself.
 */
import { computed, nextTick, ref, watch, type ComputedRef, type Ref } from 'vue'
import { getLocalTimeZone, today } from '@internationalized/date'
import { useWeft } from '../adaptive/context.ts'
import { focus, tabbables } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import type { Field } from '../forms/field.ts'
import { format, parseLocalDate, type CalendarDate, type CalendarLabels } from './dates.ts'
import { useLocale } from '../adaptive/locale.ts'

export interface PickerProps {
  open?: boolean
  defaultOpen?: boolean
  locale?: string
  readonly?: boolean
}

/** The text of a FormField's label, if it has one on the page. */
function labelText(f: Field): string | undefined {
  if (!f.field || typeof document === 'undefined') return undefined
  return document.getElementById(f.field.labelId)?.textContent?.replace(/\*\s*$/, '').trim() || undefined
}

export function usePicker(props: PickerProps, emit: (event: 'update:open', open: boolean) => void, f: Field) {
  const open = useControllable(props, 'open', emit, () => props.defaultOpen ?? false) as Ref<boolean>
  const weft = useWeft()
  const touch = computed(() => weft.input.value === 'touch')
  const locale = useLocale(() => props.locale)
  const blocked = computed(() => f.disabled.value || !!props.readonly)
  return {
    open,
    touch,
    locale,
    blocked,
    show() {
      if (!blocked.value) open.value = true
    },
    hide() {
      open.value = false
    },
    title: (fallback: string) => labelText(f) ?? fallback,
    /**
     * A focusout in the box or the calendar: `left()` once focus has landed
     * outside both. Judged a task later when focus goes nowhere (the
     * calendar closing takes the focused date away, then returns focus).
     */
    focusout(event: FocusEvent, box: () => Element | null | undefined, panel: () => Element | null | undefined, left: () => void): void {
      const inside = (node: Node | null) => !!node && !!(box()?.contains(node) || panel()?.contains(node))
      if (inside(event.relatedTarget as Node | null)) return
      if (event.relatedTarget) return left()
      setTimeout(() => {
        if (!inside(document.activeElement)) left()
      }, 0)
    },
  }
}

export interface TypedDateOptions {
  locale: ComputedRef<string>
  /** The bound date. */
  value: () => CalendarDate | null
  /** How the field writes a date. */
  format: () => Intl.DateTimeFormatOptions
  /** Whether the calendar would let this date be chosen (min, max, disabled dates). */
  allowed: (date: CalendarDate) => boolean
  /** The text was committed: the date it says, or null. */
  onCommit: (date: CalendarDate | null) => void
}

export function useTypedDate(opts: TypedDateOptions) {
  const show = (d: CalendarDate | null) => (d ? format(d, opts.locale.value, opts.format()) : '')
  const text = ref(show(opts.value()))
  const invalid = ref(false)
  /** What the last commit bound, so the value coming back does not wipe the text it came from. */
  let sent: string | null | undefined
  watch([() => opts.value()?.toString() ?? null, opts.locale, opts.format], ([iso]) => {
    if (sent !== undefined && iso === sent) {
      sent = undefined
      if (invalid.value) return
    }
    sent = undefined
    invalid.value = false
    text.value = show(opts.value())
  })
  const read = () => parseLocalDate(text.value, opts.locale.value, today(getLocalTimeZone()))
  return {
    text,
    invalid,
    input(value: string) {
      text.value = value
    },
    commit() {
      if (!text.value.trim()) {
        invalid.value = false
        text.value = ''
        sent = null
        opts.onCommit(null)
        return
      }
      const d = read()
      if (d && opts.allowed(d)) {
        invalid.value = false
        text.value = show(d)
        sent = d.toString()
        opts.onCommit(d)
      } else {
        invalid.value = true
        sent = null
        opts.onCommit(null)
      }
    },
    /** The date the text says now, else the bound one. */
    typed: () => read() ?? opts.value(),
  }
}

export function keepFocusOnSwitch(touch: Readonly<Ref<boolean>>, fields: () => (HTMLElement | null | undefined)[], to: () => HTMLElement | null | undefined): void {
  watch(
    touch,
    (on) => {
      if (!on || typeof document === 'undefined') return
      if (!fields().some((el) => el && el === document.activeElement)) return
      void nextTick(() => to()?.focus({ preventScroll: true }))
    },
    { flush: 'pre' },
  )
}

/** The words a DatePicker says: its Calendar's, and its button's name. */
export interface DatePickerLabels extends CalendarLabels {
  choose: string
}

/** The words a DateRangePicker says: its Calendar's, its button's and its fields' names. */
export interface DateRangePickerLabels extends CalendarLabels {
  choose: string
  start: string
  end: string
}

export function cycleTab(event: KeyboardEvent, panel: HTMLElement | null | undefined): void {
  if (event.key !== 'Tab' || event.defaultPrevented || !panel) return
  const all = tabbables(panel)
  if (!all.length) return
  const at = all.indexOf(document.activeElement as HTMLElement)
  const edge = event.shiftKey ? at <= 0 : at === all.length - 1
  if (!edge) return
  event.preventDefault()
  focus(event.shiftKey ? all[all.length - 1] : all[0])
}
