import { hasInjectionContext, inject, onMounted, onScopeDispose, readonly, ref, watch, type Ref } from 'vue'
import { usePrePaint } from '../head/usePrePaint.ts'
import { useId } from '../headless/useId.ts'
import { weftKey } from './context.ts'
import { currentMatchMedia, NARROW_MAX, REGULAR_MAX, widthOf, type Width } from './detect.ts'

/**
 * The pre-paint script (usePrePaint) for components that measure their own
 * box (`useWidth`): one `ResizeObserver` for the page, attached to every
 * `[data-width]` element (`[inert]` included, since inert is about input, not
 * layout) as the parser adds it (through a `MutationObserver` on the document,
 * disconnected at `DOMContentLoaded` after one final sweep), measuring each
 * unseen element's `getBoundingClientRect().width` in a read pass before
 * writing `data-width` in a write pass so `data-width` is already set for the
 * first layout with zero layout shift (CLS 0) and one layout per batch, and
 * keeping it observed (its border box, the box `getBoundingClientRect`
 * measures, as `useWidth` does once mounted) for resizes without hydration.
 * When a component mounts
 * in an island, `useWidth` marks the element `data-weft-width` and unobserves
 * it from the shared observer (`window.__weftWidthRo`) so the pre-paint
 * observer never fights the component's own `ResizeObserver`.
 */
export const widthPrePaint =
  `if(typeof ResizeObserver<"u"){` +
  `var o=new Set,d=function(t,w){if(w>0)t.setAttribute("data-width",w<${NARROW_MAX}?"narrow":w<${REGULAR_MAX}?"regular":"wide")},` +
  `r=window.__weftWidthRo=new ResizeObserver(function(e){for(var i=0;i<e.length;i++){var t=e[i].target;if(t.hasAttribute("data-weft-width"))r.unobserve(t);else d(t,b(e[i]))}}),b=function(x){var z=x.borderBoxSize;return z?(z[0]||z).inlineSize:x.target.getBoundingClientRect().width},` +
  `s=function(){var l=document.querySelectorAll("[data-width]:not([data-weft-width])"),n=[],w=[],i;for(i=0;i<l.length;i++)if(!o.has(l[i])){o.add(l[i]);n.push(l[i]);w.push(l[i].getBoundingClientRect().width)}for(i=0;i<n.length;i++){d(n[i],w[i]);r.observe(n[i],{box:"border-box"})}};` +
  `s();if(document.readyState==="loading"){var m=new MutationObserver(s);m.observe(document,{childList:true,subtree:true});` +
  `addEventListener("DOMContentLoaded",function(){m.disconnect();s()},{once:true})}}`

/** An observed element's border-box width (`borderBoxSize`, else its bounding box where a browser lacks it). */
function borderWidth(entry: ResizeObserverEntry): number {
  const box = entry.borderBoxSize as ResizeObserverSize[] | ResizeObserverSize | undefined
  if (!box) return entry.target.getBoundingClientRect().width
  return (Array.isArray(box) ? box[0] : box)?.inlineSize ?? entry.target.getBoundingClientRect().width
}

function preHydratedWidth(id: string): Width | undefined {
  if (typeof document === 'undefined') return undefined
  const attr = document.querySelector(`[data-weft-w="${id}"]`)?.getAttribute('data-width')
  if (attr === 'narrow' || attr === 'regular' || attr === 'wide') return attr
  return undefined
}

export interface UseWidthOptions {
  /**
   * Register the page-wide pre-paint observer (`weft:width`) on the server so
   * `[data-width]` is set from the element's width before the first paint and
   * on resize without hydration. Default `true` (a component that can sit in
   * any box and styles from `[data-width]` in CSS must keep the pre-paint
   * measuring); set `false` only on a component that spans the page and
   * adapts before mounting with a viewport `@media` fallback (`Footer`), on
   * one whose layout does not depend on its width (a vertical `Stepper` or
   * `Timeline`, a horizontal `Tabs`), or on one that switches its markup in
   * JavaScript after mounting (`Table`). A component that follows the
   * viewport uses `useViewportWidth` instead.
   */
  prepaint?: boolean
}

export interface WidthRef extends Readonly<Ref<Width>> {
  /** Hydration-stable id for `data-weft-w` (defined when `prepaint` is not `false`) so `useWidth` reads the element's pre-paint `data-width` before Vapor's first render effect. */
  readonly id?: string
}

/**
 * The width class of an element, measured from the element itself, not the
 * window. Until it is measured (on the server, while hydrating) it is the
 * width the nearest WeftProvider gives, else the `data-width` attribute the
 * pre-paint script already set on the element (`[data-weft-w="<width.id>"]`),
 * else `wide`. On the server, when no WeftProvider forces a fixed width and
 * `prepaint` is not `false`, it registers one page-wide pre-paint script
 * (`weft:width`) that keeps `[data-width]` elements updated from their own
 * width before paint and on resize even without hydration.
 */
export function useWidth(el: Readonly<Ref<HTMLElement | null>>, options?: UseWidthOptions): WidthRef {
  const prepaint = options?.prepaint !== false
  const id = prepaint ? useId('w') : undefined
  const given = hasInjectionContext() ? inject(weftKey, null)?.width.value : undefined
  if (import.meta.env.SSR && given === undefined && prepaint) usePrePaint('weft:width', widthPrePaint)
  const initial = given ?? (!import.meta.env.SSR && id ? preHydratedWidth(id) : undefined) ?? 'wide'
  const width = ref<Width>(initial)
  let observer: ResizeObserver | undefined
  let current: HTMLElement | null = null

  const release = () => {
    observer?.disconnect()
    observer = undefined
    current?.removeAttribute('data-weft-width')
    current = null
  }

  const stop = watch(
    el,
    (node) => {
      release()
      if (!node || typeof ResizeObserver === 'undefined') return
      current = node
      const attr = node.getAttribute('data-width')
      if (attr === 'narrow' || attr === 'regular' || attr === 'wide') width.value = attr
      node.setAttribute('data-weft-width', '')
      ;(globalThis as { __weftWidthRo?: ResizeObserver }).__weftWidthRo?.unobserve(node)
      const measured = node.getBoundingClientRect().width
      if (measured > 0) width.value = widthOf(measured)
      // The border box, as getBoundingClientRect above and the pre-paint
      // script measure it, so a padded element has one breakpoint.
      observer = new ResizeObserver((entries) => {
        const entry = entries[entries.length - 1]
        if (entry) width.value = widthOf(borderWidth(entry))
      })
      observer.observe(node, { box: 'border-box' })
    },
    { immediate: true, flush: 'sync' },
  )

  onScopeDispose(() => {
    stop()
    release()
  })
  const out = readonly(width) as WidthRef
  Object.defineProperty(out, 'id', { value: id })
  return out
}

/** The media queries of the width classes, as the components' CSS writes them. */
export const NARROW_QUERY = `(max-width: ${NARROW_MAX - 0.02}px)`
export const REGULAR_QUERY = `(max-width: ${REGULAR_MAX - 0.02}px)`

/**
 * The viewport's width class, as a CSS media query sees it (a classic
 * scrollbar included, which the root element's box leaves out), so a
 * component that follows the viewport (`Header`, `AppShell`,
 * `NavigationMenu`, `Pagination`, `Dialog`) switches where its `@media`
 * fallback before mounting does. Until mounted (on the server, while
 * hydrating) it is the width the nearest WeftProvider gives, else `wide`;
 * then it follows the media queries live.
 */
export function useViewportWidth(): Readonly<Ref<Width>> {
  const given = hasInjectionContext() ? inject(weftKey, null)?.width.value : undefined
  const width = ref<Width>(given ?? 'wide')
  let stop: (() => void) | undefined
  onMounted(() => {
    const mq = currentMatchMedia()
    if (!mq) return
    const narrow = mq(NARROW_QUERY)
    const regular = mq(REGULAR_QUERY)
    const read = () => (width.value = narrow.matches ? 'narrow' : regular.matches ? 'regular' : 'wide')
    read()
    narrow.addEventListener('change', read)
    regular.addEventListener('change', read)
    stop = () => {
      narrow.removeEventListener('change', read)
      regular.removeEventListener('change', read)
    }
  })
  onScopeDispose(() => stop?.())
  return readonly(width)
}
