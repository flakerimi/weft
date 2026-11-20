/**
 * useId(prefix = 'weft')
 *
 * An id for linking elements (`aria-labelledby`, `aria-controls`, `for`)
 * that is the same on the server and in the browser, so hydration matches.
 * Call it once per id during setup.
 *
 * - In a component, ids come from Vue's own `useId`, counted per app in
 *   render order: `weft-v-0`, `menu-v-1`. Vue's counter works in Vapor.
 * - Inside an island, ids are counted in the island's own scope instead:
 *   `weft-i0-0`. On the server an island renders inside the page's app, in
 *   the browser it hydrates as an app of its own, so Vue's per-app counter
 *   would differ between the two. `Island` provides the scope on the server
 *   and the island runtime (`@weft/vite`) provides the same one, keyed by
 *   `ID_SCOPE`, to each island app, both named by the island's position on
 *   the page.
 * - Outside any component (a plain effect scope) it falls back to a page-wide
 *   counter, unique but not stable across renders.
 */
import { hasInjectionContext, inject, useId as vueUseId } from 'vue'

/** The injection key of an island's id scope; the island runtime provides it under this string. */
export const ID_SCOPE = 'weft:id-scope'

/** An island's id scope: ids are `<prefix>-<scope prefix>-<next++>`. */
export interface IdScope {
  prefix: string
  next: number
}

let loose = 0

export function useId(prefix = 'weft'): string {
  if (hasInjectionContext()) {
    const scope = inject<IdScope | null>(ID_SCOPE, null)
    if (scope) return `${prefix}-${scope.prefix}-${scope.next++}`
    const id = vueUseId()
    if (id) return `${prefix}-${id}`
  }
  return `${prefix}-x${++loose}`
}
