import { createVaporSSRApp, type Component } from 'vue'

/** Island name (the component's SFC `__name`) → a loader for its chunk. */
export type IslandRegistry = Record<string, () => Promise<Component>>

export interface IslandOptions {
  observe?: typeof IntersectionObserver
  idle?: (cb: () => void) => void
  matchMedia?: (q: string) => MediaQueryList
}

const reason = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Hydrate every `<weft-island>` on the page, each when its `data-when` says:
 * `load` at once, `idle` when the browser is idle, `visible` on its first
 * intersection, `media` when `data-media` matches. An island that cannot start
 * warns once and keeps its server HTML; the others are unaffected.
 */
export function startIslands(registry: IslandRegistry, opts: IslandOptions = {}): void {
  for (const el of document.querySelectorAll<HTMLElement>('weft-island')) {
    const { id = '', when, media } = el.dataset
    const warn = (why: string) => console.warn(`weft: island ${id} did not start: ${why}`)
    const start = () => void hydrate(el, registry[id], warn)

    if (when === 'visible') {
      const io = new (opts.observe ?? IntersectionObserver)((entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          io.disconnect()
          start()
        }
      })
      io.observe(el)
    } else if (when === 'idle') {
      ;(opts.idle ?? ((cb) => (typeof requestIdleCallback == 'function' ? requestIdleCallback(cb) : setTimeout(cb, 200))))(start)
    } else if (when === 'media') {
      if (!media) {
        warn('data-when="media" has no data-media.')
        continue
      }
      const query = (opts.matchMedia ?? matchMedia)(media)
      const check = () => {
        if (!query.matches) return
        query.removeEventListener('change', check)
        start()
      }
      query.addEventListener('change', check)
      check()
    } else {
      start()
    }
  }
}

async function hydrate(el: HTMLElement, load: (() => Promise<Component>) | undefined, warn: (why: string) => void) {
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
    // Errors during the mount mean the island did not start; later ones are the
    // component's own and are reported as Vue would.
    app.config.errorHandler = (err) => {
      if (mounting) failed ??= err
      else console.error(err)
    }
    try {
      app.mount(el)
    } finally {
      mounting = false
    }
    if (failed) throw failed
  } catch (e) {
    warn(reason(e))
  }
}
