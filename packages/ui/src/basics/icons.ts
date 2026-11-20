/**
 * The icons a build draws: Lucide icon name → the inner markup of its 24 × 24
 * `<svg>`. `Icon` reads it; nothing here knows Lucide.
 *
 * The build fills it: `weftIcons()` in `@weft/vite` finds the icon names a
 * module uses (`<Icon name="trash">`, `icon="plus"` on any component,
 * `icon: 'house'` in a script) and gives that module one import per icon,
 * `virtual:weft/icon/<name>`, which calls `registerIcon`. So a chunk carries
 * exactly the icons its own components use, and an island hydrates with the
 * same drawings the server rendered.
 */
const bodies = new Map<string, string>()

/** Record the drawing for `name`. The first registration wins; a later one is ignored. */
export function registerIcon(name: string, body: string): void {
  if (!bodies.has(name)) bodies.set(name, body)
}

/** The drawing registered for `name`, or undefined when the build did not include it. */
export function iconBody(name: string): string | undefined {
  return bodies.get(name)
}

const warned = new Set<string>()

/**
 * The drawing for `name`, for `Icon` to render. A name the build did not
 * include throws on the server, so a page rendered at build never ships a
 * blank icon; in the browser it warns once per name and draws nothing.
 */
export function drawIcon(name: string): string {
  const found = bodies.get(name)
  if (found !== undefined) return found
  const why = `Icon: "${name}" is not in this build; write it where the build can find it (name="…" on Icon, icon="…" on a component, icon: '…' in a script) or list it in the icons option of weftIcons().`
  if (import.meta.env.SSR) throw new Error(why)
  if (!warned.has(name)) {
    warned.add(name)
    console.warn(why)
  }
  return ''
}
