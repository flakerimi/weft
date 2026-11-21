// What the /components pages show, read once when the page build loads this
// module: the catalogue, checked against the example files (a stale one
// fails the build), and each example's source from its own file,
// highlighted as the docs' code is. Server only.
import type { Component } from 'vue'
import { highlightCode } from 'virtual:weft/content'
import { catalogText, exampleComponents, exampleIslands, exampleSources } from 'virtual:weft-gallery'
import { checkFresh, groupsOf, isInteractive, slugOf, type Catalog, type CatalogEntry } from './model.ts'

export interface GalleryExample {
  name: string
  title: string
  /** From the repository root: packages/ui/examples/<Name>/<name>.vue. */
  file: string
  /** The file's text, as it is on disk. */
  source: string
  /** The source highlighted. */
  html: string
  component: Component
  /** Its island: the preview stage for the first example, the example alone for the others. */
  island: Component
  /** Whether it needs JavaScript; a still example is shown as the server drew it. */
  interactive: boolean
}

export interface GalleryComponent extends Omit<CatalogEntry, 'examples'> {
  /** Its page: /components/<slug>. */
  slug: string
  path: string
  examples: GalleryExample[]
  /** Its catalogue entry as JSON, as /components/<slug>.json serves it. */
  json: string
}

const catalog = JSON.parse(catalogText) as Catalog
checkFresh(catalog, exampleSources)

/** The catalogue as it is on disk, served at /catalog.json. */
export { catalogText }
/** docs/parity.md (against Nuxt UI) and docs/sizes.md (what each component costs), served raw beside the gallery. */
export { default as parityText } from '../../docs/parity.md?raw'
export { default as sizesText } from '../../docs/sizes.md?raw'

export const components: GalleryComponent[] = await Promise.all(
  catalog.components.map(async (entry) => {
    const slug = slugOf(entry.name)
    const json = JSON.stringify(entry, null, 2) + '\n'
    return {
      ...entry,
      slug,
      path: `/components/${slug}`,
      json,
      examples: await Promise.all(
        entry.examples.map(async (e) => {
          const source = exampleSources[e.file]!
          return {
            name: e.name,
            title: e.title,
            file: e.file,
            source,
            html: await highlightCode(source, 'vue'),
            component: exampleComponents[e.file]!,
            island: exampleIslands[e.file]!,
            interactive: isInteractive(source),
          }
        }),
      ),
    }
  }),
)

export const groups = groupsOf(components)

/** A component by its page's slug. */
export function componentBySlug(slug: string): GalleryComponent {
  const c = components.find((x) => x.slug === slug)
  if (!c) throw new Error(`gallery: no component has the page /components/${slug}.`)
  return c
}
