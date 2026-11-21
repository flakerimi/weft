// When each island starts, with no Vue in sight: the page's loader runs this
// at once, and fetches the runtime (Vue, the registry, the hydration) only
// when the first island is due.

export interface IslandOptions {
  observe?: typeof IntersectionObserver
  idle?: (cb: () => void) => void
  matchMedia?: (q: string) => MediaQueryList
}

/** Start one island: `index` is its position among the page's islands; `warn` reports once why it did not start. */
export type IslandStart = (el: HTMLElement, index: number, warn: (why: string) => void) => void

export const reason = (e: unknown) => (e instanceof Error ? e.message : String(e))

/**
 * Call `start` for every `<weft-island>` on the page, each when its `data-when`
 * says: `load` at once, `idle` when the browser is idle, `visible` on its first
 * intersection, `media` when `data-media` matches. A trigger that cannot be set
 * up warns once and costs that island only.
 */
export function scheduleIslands(start: IslandStart, opts: IslandOptions = {}): void {
  for (const [index, el] of [...document.querySelectorAll<HTMLElement>('weft-island')].entries()) {
    const { id = '', when, media } = el.dataset
    const warn = (why: string) => console.warn(`weft: island ${id} did not start: ${why}`)
    const go = () => start(el, index, warn)

    try {
      if (when === 'visible') {
        const io = new (opts.observe ?? IntersectionObserver)((entries) => {
          if (entries.some((e) => e.isIntersecting)) {
            io.disconnect()
            go()
          }
        })
        io.observe(el)
      } else if (when === 'idle') {
        ;(opts.idle ?? ((cb) => (typeof requestIdleCallback == 'function' ? requestIdleCallback(cb) : setTimeout(cb, 200))))(go)
      } else if (when === 'media') {
        if (!media) throw new Error('data-when="media" has no data-media.')
        const query = (opts.matchMedia ?? matchMedia)(media)
        const check = () => {
          if (!query.matches) return
          query.removeEventListener('change', check)
          go()
        }
        query.addEventListener('change', check)
        check()
      } else {
        // No data-when means load; anything else unknown is reported once.
        if (when !== undefined && when !== 'load') console.warn(`weft: island ${id} has unknown when "${when}"; starting on load`)
        go()
      }
    } catch (e) {
      warn(reason(e))
    }
  }
}
