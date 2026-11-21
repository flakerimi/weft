<script setup vapor lang="ts">
import { provide, useSSRContext, type Component } from 'vue'
import { ID_SCOPE, type IdScope } from '../headless/useId.ts'
import '../styles/base.css'

export type IslandWhen = 'load' | 'idle' | 'visible' | 'media'

const p = defineProps<{
  /** When the browser hydrates it: at `load`, when `idle`, when `visible`, or when `media` matches. */
  when: IslandWhen
  /** The media query for `when="media"`, such as `(min-width: 1024px)`. */
  media?: string
  /** The `<script setup>` SFC to render and hydrate; its file name is the island's id. */
  component: Component
  /** Props for the component, sent to the browser as JSON. */
  props?: Record<string, unknown>
}>()

// The page is rendered on the server only; in the browser the island runtime
// hydrates <weft-island> on its own, so this component renders nothing there.
const server = import.meta.env.SSR

/**
 * What `value` is when JSON would drop or change it on the way to the browser,
 * at `path` (or inside it, with the path extended); null when it is plain data.
 */
function unserialisable(value: unknown, path: string, parents: object[] = []): { path: string; what: string } | null {
  switch (typeof value) {
    case 'string':
    case 'boolean':
      return null
    case 'number':
      // JSON writes -0 as 0.
      if (Object.is(value, -0)) return { path, what: '-0' }
      return Number.isFinite(value) ? null : { path, what: String(value) }
    case 'undefined':
      return { path, what: 'undefined' }
    case 'function':
    case 'symbol':
    case 'bigint':
      return { path, what: `a ${typeof value}` }
  }
  if (value === null) return null
  const obj = value as object
  if (parents.includes(obj)) return { path, what: 'an object that contains itself' }
  const inner = [...parents, obj]
  // JSON leaves a symbol-keyed member out, of an array or an object.
  const symbol = Object.getOwnPropertySymbols(obj).find((k) => Object.prototype.propertyIsEnumerable.call(obj, k))
  if (Array.isArray(obj)) {
    if (symbol) return { path, what: `an array with the symbol key ${String(symbol)}` }
    // And an array's members other than its items.
    const extra = Object.keys(obj).find((k) => !/^(0|[1-9]\d*)$/.test(k) || Number(k) >= obj.length)
    if (extra !== undefined) return { path, what: `an array with the member ${extra}` }
    for (let i = 0; i < obj.length; i++) {
      if (!(i in obj)) return { path: `${path}[${i}]`, what: 'an empty array slot' }
      const bad = unserialisable(obj[i], `${path}[${i}]`, inner)
      if (bad) return bad
    }
    return null
  }
  const proto = Object.getPrototypeOf(obj)
  if (proto !== Object.prototype && proto !== null) {
    const name = (proto as { constructor?: { name?: string } }).constructor?.name || 'class instance'
    return { path, what: `${/^[aeiou]/i.test(name) ? 'an' : 'a'} ${name}` }
  }
  if (symbol) return { path, what: `an object with the symbol key ${String(symbol)}` }
  for (const [key, v] of Object.entries(obj)) {
    const bad = unserialisable(v, path ? `${path}.${key}` : key, inner)
    if (bad) return bad
  }
  return null
}

// The island's id is the component's SFC `__name` (its file name), the key the
// weft() plugin gives its chunk in the registry. A `name` option could differ
// from the file name, so it is not used.
const id = (p.component as { __name?: string }).__name
let json = ''
if (server) {
  if (!id) throw new Error('Island: the component has no SFC __name; an island must be a <script setup> single-file component.')
  if (p.when === 'media' && !p.media) throw new Error('Island: when="media" needs a media query in the media prop.')
  // The browser gets the props as JSON: what JSON would lose is refused, not sent changed.
  const bad = unserialisable(p.props ?? {}, '')
  if (bad) {
    throw new Error(
      `Island: ${bad.path ? `prop ${bad.path}` : 'the props object'} of island ${id} is ${bad.what}, which JSON cannot carry to the browser; pass only strings, finite numbers, booleans, null, arrays and plain objects.`,
    )
  }
  // Escape what could end the <script> element or open a comment inside it.
  json = JSON.stringify(p.props ?? {})
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e')
    .replace(/&/g, '\\u0026')
  const ctx = useSSRContext<{ islands?: string[] }>()
  // The island's ids count from its position on the page, as the island
  // runtime counts them when it hydrates the island as an app of its own.
  provide<IdScope>(ID_SCOPE, { prefix: `i${ctx?.islands?.length ?? 0}`, next: 0 })
  if (ctx) (ctx.islands ??= []).push(id)
}
</script>

<template>
  <component is="weft-island" v-if="server" :data-id="id" :data-when="when" :data-media="media">
    <component is="script" type="application/json" :innerHTML="json" />
    <component :is="component" v-bind="p.props" />
  </component>
</template>
