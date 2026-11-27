import { hasInjectionContext, inject, onScopeDispose, readonly, ref, watch, type Ref } from 'vue'
import { usePrePaint } from '../head/usePrePaint.ts'
import { weftKey } from './context.ts'
import { NARROW_MAX, REGULAR_MAX, widthOf, type Width } from './detect.ts'

/**
 * The pre-paint script (usePrePaint) for components that measure their own
 * box (`useWidth`): one `ResizeObserver` for the page, attached to every
 * `[data-width]` element outside `[inert]` as the parser adds it (through a
 * `MutationObserver` on the document, disconnected at `DOMContentLoaded`
 * after one final sweep), so `data-width` is set from the element's width
 * before the first paint and follows resizes without hydration. When a
 * component mounts in an island, `useWidth` marks the element
 * `data-weft-width` and unobserves it from the shared observer (`window.__weftWidthRo`)
 * so the pre-paint observer never fights the component's own `ResizeObserver`.
 */
export const widthPrePaint =
  `if(typeof ResizeObserver<"u"){` +
  `var r=window.__weftWidthRo=new ResizeObserver(function(e){for(var i=0;i<e.length;i++){var t=e[i].target,w=e[i].contentRect.width;` +
  `if(t.hasAttribute("data-weft-width"))r.unobserve(t);` +
  `else if(w>0)t.setAttribute("data-width",w<${NARROW_MAX}?"narrow":w<${REGULAR_MAX}?"regular":"wide")}}),` +
  `s=function(){var l=document.querySelectorAll("[data-width]:not([data-weft-width])");for(var i=0;i<l.length;i++)if(!l[i].closest("[inert]"))r.observe(l[i])};` +
  `s();if(document.readyState==="loading"){var m=new MutationObserver(s);m.observe(document,{childList:true,subtree:true});` +
  `addEventListener("DOMContentLoaded",function(){m.disconnect();s()},{once:true})}}`

export interface UseWidthOptions {
  /**
   * Register the page-wide pre-paint observer (`weft:width`) on the server so
   * `[data-width]` is set from the element's width before the first paint and
   * on resize without hydration. Default `true` (a component that can sit in
   * any box and styles from `[data-width]` in CSS must keep the pre-paint
   * measuring); set `false` only on components that always span the page or the
   * viewport and adapt before mounting with a viewport `@media` fallback
   * (`Header`, `Footer`, `AppShell`, `NavigationMenu`, `Dialog`, `Pagination`),
   * or that switch their template/orientation in JavaScript after mounting
   * (`Table`, `Timeline`, `Tabs`, `Stepper`).
   */
  prepaint?: boolean
}

/**
 * The width class of an element, measured from the element itself, not the
 * window. Until it is measured (on the server, while hydrating) it is the
 * width the nearest WeftProvider gives, else `wide`. On the server, when no
 * WeftProvider forces a fixed width and `prepaint` is not `false`, it registers
 * one page-wide pre-paint script (`weft:width`) that keeps `[data-width]`
 * elements updated from their own width before paint and on resize even without
 * hydration.
 */
export function useWidth(el: Readonly<Ref<HTMLElement | null>>, options?: UseWidthOptions): Readonly<Ref<Width>> {
  const given = hasInjectionContext() ? inject(weftKey, null)?.width.value : undefined
  if (import.meta.env.SSR && given === undefined && options?.prepaint !== false) usePrePaint('weft:width', widthPrePaint)
  const width = ref<Width>(given ?? 'wide')
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
      node.setAttribute('data-weft-width', '')
      ;(globalThis as { __weftWidthRo?: ResizeObserver }).__weftWidthRo?.unobserve(node)
      const initial = node.getBoundingClientRect().width
      if (initial > 0) width.value = widthOf(initial)
      observer = new ResizeObserver((entries) => {
        const entry = entries[entries.length - 1]
        if (entry) width.value = widthOf(entry.contentRect.width)
      })
      observer.observe(node)
    },
    { immediate: true, flush: 'sync' },
  )

  onScopeDispose(() => {
    stop()
    release()
  })
  return readonly(width)
}
