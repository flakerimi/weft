import { useSSRContext } from 'vue'

/** What a page puts in its `<head>`: `<title>`, the description, and Open Graph. */
export interface Head {
  title?: string
  description?: string
  /** The Open Graph image URL. */
  image?: string
}

/**
 * Set the page's title, description and Open Graph image. Pages are rendered
 * at build, so this only records them in the server render's context, where
 * the build writes them into the page's `<head>`; in the browser it does
 * nothing. A later call overrides the fields it sets.
 */
export function useHead(head: Head): void {
  if (!import.meta.env.SSR) return
  const ctx = useSSRContext<{ head?: Head }>()
  if (!ctx) return
  const set = Object.fromEntries(Object.entries(head).filter(([, v]) => v !== undefined))
  ctx.head = { ...ctx.head, ...set }
}
