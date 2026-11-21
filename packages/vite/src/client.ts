/// <reference path="./virtual.d.ts" />
// The runtime of a built site's islands: Vue, the registry weft() serves, and
// the hydration, an entry of the client build of its own. The page's loader
// (boot.ts) imports it when the first island is due, once for the page; a
// page with an island that starts on load preloads it.
import islands from 'virtual:weft/islands'
import { hydrateIsland, loaderOf } from './island-runtime.ts'

export function start(el: HTMLElement, index: number, warn: (why: string) => void): Promise<void> {
  return hydrateIsland(el, index, loaderOf(islands, el), warn)
}
