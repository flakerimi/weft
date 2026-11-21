import { useSSRContext } from 'vue'

/**
 * Register a pre-paint snippet: a few lines of script the page build writes
 * inline at the top of the `<head>` of every page that renders the calling
 * component, and of no other page, so they run before the first paint (a
 * blocking inline script, before the CSS). Use it only to copy a little
 * state the server cannot know onto `document.documentElement` (a stored
 * preference) for CSS to read until the component mounts; the server's
 * markup must not depend on it, so hydration still matches.
 *
 * `id` names the snippet: a page carries each id once, whatever renders it
 * how often (the first registration wins). The code must be a script on its
 * own: wrap what can throw in `try`, and keep it short. It runs at build,
 * recording into the server render's context; in the browser it does
 * nothing.
 */
export function usePrePaint(id: string, code: string): void {
  if (!import.meta.env.SSR) return
  const ctx = useSSRContext<{ prePaint?: Map<string, string> }>()
  if (!ctx) return
  ctx.prePaint ??= new Map()
  if (!ctx.prePaint.has(id)) ctx.prePaint.set(id, code)
}
