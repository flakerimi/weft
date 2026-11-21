// Shared by the date tests.
import { $, $$, active, settle } from '../overlays/helpers.ts'

/** Vapor reads a root component's function props as getters: wrap the functions given as values. */
export const wrapFunctions = (props: Record<string, unknown>) =>
  Object.fromEntries(Object.entries(props).map(([k, v]) => [k, typeof v === 'function' ? () => v : v]))

/** The button of a date of the month on show. */
export const day = (iso: string) => $<HTMLButtonElement>(`.weft-calendar__day[data-date="${iso}"]:not([data-outside])`)
/** The calendar's heading, its spaces (thin ones in ranges) made plain. */
export const heading = () => $('.weft-calendar__heading').textContent!.replace(/\s+/g, ' ').trim()
/** The date buttons that are the tab stop: one. */
export const tabStop = () => $$<HTMLButtonElement>('.weft-calendar__day').filter((b) => b.tabIndex === 0)
/** The date of the focused element. */
export const focusedIso = () => (active() as HTMLElement | null)?.dataset.date

/** A finger dragged `dx` px across `el`, lifted there, and the click the browser then sends. */
export async function swipe(el: Element, dx: number) {
  const at = (type: string, x: number) =>
    el.dispatchEvent(new PointerEvent(type, { bubbles: true, cancelable: true, pointerType: 'touch', pointerId: 7, clientX: x, clientY: 100 }))
  at('pointerdown', 200)
  at('pointermove', 200 + dx / 2)
  at('pointermove', 200 + dx)
  at('pointerup', 200 + dx)
  el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, detail: 1 }))
  await settle()
}
