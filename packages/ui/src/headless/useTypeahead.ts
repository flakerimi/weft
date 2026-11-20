/**
 * useTypeahead(items, onMatch)
 *
 * Type to jump to an item by its label, as APG asks of menus, listboxes and
 * trees. `items()` gives the labels in order (give `''` for an item that must
 * never match, such as a disabled one); `onMatch(index)` is called with the
 * item to move to.
 *
 * Call `onKeydown(event, current)` from the widget's keydown handler, with
 * the index of the item that is current now (-1 for none). It returns true
 * when it took the key, and the handler should then `preventDefault()`:
 *
 * - A printable character adds to the search; half a second without typing
 *   starts a new one. Matching is a case-insensitive prefix of the trimmed
 *   label, searched from the item after the current one, wrapping round.
 * - The same letter typed again cycles through the labels starting with it.
 * - Typing more of a word keeps the current item while it still matches.
 * - Space is part of the search only once a search has started; a first
 *   Space is left to the widget (it usually means "activate").
 * - Keys with Ctrl, Meta or Alt, and named keys (arrows, Enter), are not taken.
 *
 * `reset()` forgets the search. The timer is cleared when the scope ends.
 */
import { onScopeDispose } from 'vue'

export interface Typeahead {
  onKeydown: (event: KeyboardEvent, current?: number) => boolean
  reset: () => void
}

const WAIT = 500

export function useTypeahead(items: () => string[], onMatch: (index: number) => void): Typeahead {
  let query = ''
  let timer: ReturnType<typeof setTimeout> | undefined

  function reset() {
    query = ''
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }

  function onKeydown(event: KeyboardEvent, current = -1): boolean {
    if (event.ctrlKey || event.metaKey || event.altKey || event.key.length !== 1) return false
    if (event.key === ' ' && query === '') return false
    query += event.key.toLowerCase()
    if (timer !== undefined) clearTimeout(timer)
    timer = setTimeout(reset, WAIT)

    const labels = items().map((l) => l.trim().toLowerCase())
    const repeated = [...query].every((c) => c === query[0])
    const search = repeated ? query[0]! : query
    // A first or repeated letter moves on; a longer word may stay where it is.
    const from = repeated ? current + 1 : Math.max(current, 0)
    for (let i = 0; i < labels.length; i++) {
      const index = (from + i) % labels.length
      const label = labels[index]!
      if (label && label.startsWith(search)) {
        onMatch(index)
        break
      }
    }
    return true
  }

  onScopeDispose(reset)
  return { onKeydown, reset }
}
