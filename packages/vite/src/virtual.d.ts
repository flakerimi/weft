declare module 'virtual:weft/islands' {
  /** Island name (the component's SFC `__name`) → a loader for its chunk; served by `weft()`. */
  const islands: Record<string, () => Promise<import('vue').Component>>
  export default islands
}

declare module 'virtual:weft/content' {
  type ContentPage = import('./content.ts').ContentPage
  type ContentLink = import('./content.ts').ContentLink
  /** One page of a collection by its slug, rendered and checked at build time; for pages on the server, served by `weftContent()`. */
  export function useContent(collection: string, slug: string): ContentPage
  /** Every page of a collection, in its order (by `order`, then slug). */
  export function useCollection(collection: string): ContentLink[]
  /** A collection's slugs for a dynamic page's `getPaths()`, each under `param` (default `slug`). */
  export function contentPaths(collection: string, param?: string): Record<string, string>[]
}
