/** Whether the Weft repository is public, so the header links it: WEFT_REPO_PUBLIC=1 at build (vite.config.ts). */
declare const __WEFT_REPO_PUBLIC__: boolean

/** The gallery plugin's module for the /components pages (gallery/plugin.ts). */
declare module 'virtual:weft-gallery' {
  import type { Component } from 'vue'
  /** catalog/components.json as it is on disk. */
  export const catalogText: string
  /** Every example file's text, by its path from the repository root (helpers too). */
  export const exampleSources: Record<string, string>
  /** Each catalogued example's component, by its file. */
  export const exampleComponents: Record<string, Component>
  /** Each catalogued example's island, by its file: the first example's preview stage, any other example alone. */
  export const exampleIslands: Record<string, Component>
}
