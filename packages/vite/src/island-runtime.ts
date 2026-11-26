import { createVaporSSRApp, type Component } from 'vue'
import { scheduleIslands, type IslandOptions } from './island-schedule.ts'
import { takeTouched } from './touched.ts'

export type { IslandOptions } from './island-schedule.ts'

/** Island name (the component's SFC `__name`) → a loader for its chunk. */
export type IslandRegistry = Record<string, () => Promise<Component>>

/**
 * The key @weft/ui's useId reads an island's id scope under (its ID_SCOPE).
 * Each island app gets `{ prefix: 'i<n>', next: 0 }`, n its position among
 * the page's islands, which is what Island provides on the server.
 */
export const ISLAND_ID_SCOPE = 'weft:id-scope'

// Its own copy, not the scheduler's: the runtime and the loader share no module.
const reason = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Hydrate every `<weft-island>` on the page, each when its `data-when` says:
 * `load` at once, `idle` when the browser is idle, `visible` on its first
 * intersection, `media` when `data-media` matches. An island that cannot start
 * warns once and keeps its server HTML; the others are unaffected.
 */
export function startIslands(registry: IslandRegistry, opts: IslandOptions = {}): void {
  scheduleIslands((el, index, warn) => void hydrateIsland(el, index, loaderOf(registry, el), warn), opts)
}

/** The registry's loader for the island `el`, if it has one of its own (not, say, `constructor`). */
export const loaderOf = (registry: IslandRegistry, el: HTMLElement) => {
  const id = el.dataset.id ?? ''
  return Object.hasOwn(registry, id) ? registry[id] : undefined
}

/** Hydrate one island from its chunk's loader; what stops it is reported once with `warn`, and its server HTML stays. */
export async function hydrateIsland(el: HTMLElement, index: number, load: (() => Promise<Component>) | undefined, warn: (why: string) => void) {
  try {
    if (!load) throw new Error('no chunk is registered for it.')
    const component = await load()
    // The props sit in the island's first child, ahead of the server HTML; take
    // them out so the mount sees exactly what the component rendered.
    const script = el.firstElementChild
    let props = {}
    if (script?.matches('script[type="application/json"]')) {
      props = JSON.parse(script.textContent || '{}')
      script.remove()
    }
    let failed: unknown
    let mounting = true
    const app = createVaporSSRApp(component as never, props)
    app.provide(ISLAND_ID_SCOPE, { prefix: `i${index}`, next: 0 })
    // Errors during the mount mean the island did not start; later ones are the
    // component's own and are reported as Vue would.
    app.config.errorHandler = (err) => {
      if (mounting) failed ??= err
      else console.error(err)
    }
    // What a person typed or chose while the chunk loaded survives the mount.
    const putBack = takeTouched(el)
    try {
      app.mount(el)
    } finally {
      mounting = false
      putBack()
    }
    if (failed) throw failed
  } catch (e) {
    warn(reason(e))
  }
}
