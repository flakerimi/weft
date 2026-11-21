// Small DOM helpers the form controls share. Each runs only in event handlers.

/** Ids joined with spaces for an aria-describedby, skipping empty ones; undefined when none. */
export function joinIds(...ids: unknown[]): string | undefined {
  const all = ids.filter((id): id is string => typeof id === 'string' && id.trim() !== '')
  return all.length ? all.join(' ') : undefined
}

const INTERACTIVE = 'input, textarea, select, button, a[href], [tabindex]'

/**
 * For a control's box: a press on the box itself (its padding, an icon, its
 * touch hit area) focuses `target` instead of doing nothing, while a press on
 * the field or on a button inside the box keeps its own behaviour.
 */
export function pressFocuses(event: MouseEvent, target: HTMLElement | null | undefined): void {
  if (!target || event.button !== 0) return
  const hit = event.target as Element | null
  if (hit?.closest(INTERACTIVE) && hit.closest(INTERACTIVE) !== event.currentTarget) return
  event.preventDefault()
  target.focus()
}

/** For a focusout on a group: whether focus is leaving the group altogether, not moving inside it. */
export function leavesGroup(event: FocusEvent): boolean {
  const to = event.relatedTarget as Node | null
  return !to || !(event.currentTarget as Element).contains(to)
}
