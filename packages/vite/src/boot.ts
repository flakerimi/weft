// The browser entry of a built site's pages with islands, and all of their
// JavaScript before an island is due: it sets each island's trigger, and the
// first that fires imports the runtime (client.ts: Vue, the registry, the
// hydration), which every island then shares. The page names the runtime's
// file, beside this one, in the script's data-weft-runtime; so the loader is
// the same file for every page and every section, cached once.
import { reason, scheduleIslands } from './island-schedule.ts'

let runtime: Promise<typeof import('./client.ts')> | undefined

const load = () => {
  const file = document.querySelector<HTMLScriptElement>('script[data-weft-runtime]')?.dataset.weftRuntime
  if (!file) return Promise.reject(new Error('the page names no runtime in data-weft-runtime.'))
  // The same URL the island chunks import it by, so it loads once.
  return import(/* @vite-ignore */ new URL(file, import.meta.url).href) as Promise<typeof import('./client.ts')>
}

scheduleIslands((el, index, warn) => {
  ;(runtime ??= load()).then(
    (r) => r.start(el, index, warn),
    (e) => warn(reason(e)),
  )
})
