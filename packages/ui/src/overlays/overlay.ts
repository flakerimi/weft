/**
 * What every Weft overlay shares, beside the headless layer it is built on
 * (useDismiss, useFocusTrap, useScrollLock, useFloating):
 *
 * - `overlayRoot()`: the element overlays are teleported into,
 *   `<div id="weft-overlays">` at the end of the body, created the first time
 *   an overlay opens. Overlays inside it stack in the order they opened.
 * - `useMounted()`: true once the component is mounted in the browser.
 *   Overlay content renders only then, so the server renders none and an
 *   island hydrates the markup the server sent, then opens overlays into the
 *   root. (Its CSS reaches the page through the island's module graph.)
 * - `useTrigger(wrap, attrs?)`: the element a component's default slot gives
 *   as its trigger: the first element inside `wrap`, a `display: contents`
 *   span around the slot, kept current as the slot re-renders. `attrs()`
 *   (reactive) is set on that element: a string sets an attribute, undefined
 *   takes away one this composable set (never one the element had of its
 *   own). `owns(name)` says whether the composable set `name`. A trigger is
 *   an element of the app's (a Button), so nothing here renders it; its
 *   attributes appear once the component is mounted.
 * - `useOverlayContext(from)`: what teleported content must carry, since it
 *   leaves the subtree it was opened from: `dir` (read from `from()` when
 *   `refresh()` is called, at opening), the scheme of a `data-weft-scheme`
 *   ancestor, and the WeftProvider's input and surface, which set the
 *   tokens (`--weft-hit-min`) inside.
 *
 * Nothing here runs on the server: the DOM is touched only from watchers
 * that run once elements exist, and from `overlayRoot()`, called by a
 * template only after mounting.
 */
import { computed, onMounted, onScopeDispose, ref, shallowRef, watch, watchEffect, type ComputedRef, type Ref } from 'vue'
import { useWeft } from '../adaptive/context.ts'
import { isRtl } from '../headless/dom.ts'

/** An element or a point to position a floating overlay against: anything with `getBoundingClientRect`. */
export interface Anchor {
  getBoundingClientRect(): DOMRect
  /** For a point: the element it belongs to, whose scrolling and direction the overlay follows. */
  contextElement?: Element
}

export const OVERLAY_ROOT_ID = 'weft-overlays'

/** The `#weft-overlays` element at the end of the body, created when first needed. */
export function overlayRoot(): HTMLElement {
  let el = document.getElementById(OVERLAY_ROOT_ID)
  if (!el) {
    el = document.createElement('div')
    el.id = OVERLAY_ROOT_ID
    document.body.appendChild(el)
  }
  return el
}

/** True once mounted in the browser; false on the server and while hydrating. */
export function useMounted(): Readonly<Ref<boolean>> {
  const mounted = ref(false)
  onMounted(() => (mounted.value = true))
  return mounted
}

export interface Trigger {
  el: Readonly<Ref<HTMLElement | null>>
  owns: (name: string) => boolean
}

export function useTrigger(
  wrap: Readonly<Ref<HTMLElement | null | undefined>>,
  attrs?: () => Record<string, string | undefined>,
): Trigger {
  const el = shallowRef<HTMLElement | null>(null)
  let observer: MutationObserver | undefined
  const read = () => {
    const first = wrap.value?.firstElementChild
    el.value = first instanceof HTMLElement ? first : null
  }
  watch(
    wrap,
    (node) => {
      if (typeof document === 'undefined') return
      observer?.disconnect()
      observer = undefined
      read()
      if (node && typeof MutationObserver !== 'undefined') {
        observer = new MutationObserver(read)
        observer.observe(node, { childList: true })
      }
    },
    { immediate: true, flush: 'post' },
  )
  onScopeDispose(() => observer?.disconnect())

  const owned = new Set<string>()
  let on: HTMLElement | null = null
  if (attrs) {
    watchEffect(
      () => {
        const node = el.value
        const next = attrs()
        if (on && on !== node) {
          for (const name of owned) on.removeAttribute(name)
          owned.clear()
        }
        on = node
        if (!node) return
        for (const [name, value] of Object.entries(next)) {
          if (value !== undefined) {
            if (node.getAttribute(name) !== value) node.setAttribute(name, value)
            owned.add(name)
          } else if (owned.delete(name)) node.removeAttribute(name)
        }
      },
      { flush: 'post' },
    )
  }
  return { el, owns: (name) => owned.has(name) }
}

export interface OverlayContext {
  attrs: ComputedRef<{ dir: 'ltr' | 'rtl'; 'data-weft-input': string; 'data-weft-surface': string; 'data-weft-scheme': string | undefined }>
  /** Read the direction and scheme of where the overlay opens from, now. */
  refresh: () => void
}

export function useOverlayContext(from: () => Element | null | undefined): OverlayContext {
  const weft = useWeft()
  const dir = ref<'ltr' | 'rtl'>('ltr')
  const scheme = ref<string | undefined>()
  return {
    attrs: computed(() => ({
      dir: dir.value,
      'data-weft-input': weft.input.value,
      'data-weft-surface': weft.surface.value,
      'data-weft-scheme': scheme.value,
    })),
    refresh() {
      if (typeof document === 'undefined') return
      const el = from() ?? document.activeElement
      dir.value = isRtl(el) ? 'rtl' : 'ltr'
      scheme.value = el?.closest('[data-weft-scheme]')?.getAttribute('data-weft-scheme') ?? undefined
    },
  }
}

/** `anchor` when it is an element, not a point. */
export const elementOf = (anchor: Anchor | undefined): HTMLElement | null =>
  typeof HTMLElement !== 'undefined' && anchor instanceof HTMLElement ? anchor : null

/** Whether a pointer event came from a finger: hover behaviour (menus, tooltips) ignores those. */
export const isTouch = (event: PointerEvent) => event.pointerType === 'touch'

/** ms of the longest transition `el` runs now (duration plus delay), 0 when none (reduced motion sets them to 0). */
export function leaveTime(el: Element): number {
  const style = getComputedStyle(el)
  const times = (list: string) => list.split(',').map((t) => {
    const v = t.trim()
    return v.endsWith('ms') ? parseFloat(v) : v.endsWith('s') ? parseFloat(v) * 1000 : 0
  })
  const durations = times(style.transitionDuration || '0s')
  const delays = times(style.transitionDelay || '0s')
  return Math.max(0, ...durations.map((d, i) => (d > 0 ? d + (delays[i] ?? delays[0] ?? 0) : 0)))
}

export interface Presence {
  /** Whether the overlay's content is rendered: while open, and while its leave transition runs. */
  present: Readonly<Ref<boolean>>
  /** Whether it is leaving: closed, still on screen. Draw it with `data-state="closed"`. */
  closing: Readonly<Ref<boolean>>
}

/**
 * usePresence(open, el): keep an overlay's content rendered while it plays
 * its leave transition. When `open` turns false, `closing` turns true (the
 * component draws `data-state="closed"`, whose CSS transitions it out) and
 * the content stays until `el`'s transition ends, or its duration and a
 * little more have passed if no transitionend arrives. With no transition
 * (none drawn, or reduced motion, which sets durations to 0) it goes at
 * once. Opening again while it leaves keeps it. Everything else about
 * closing (the overlay stack, focus, the scroll lock) happens at once, as
 * the component gives its leaving element to none of them.
 */
export function usePresence(open: Readonly<Ref<boolean>>, el: () => HTMLElement | null | undefined): Presence {
  const present = ref(open.value)
  const closing = ref(false)
  let stop: (() => void) | undefined
  const finish = () => {
    stop?.()
    stop = undefined
    closing.value = false
    present.value = open.value
  }
  watch(
    open,
    (on) => {
      stop?.()
      stop = undefined
      if (on) {
        closing.value = false
        present.value = true
        return
      }
      const node = el()
      const ms = node && typeof document !== 'undefined' ? leaveTime(node) : 0
      if (!node || ms <= 0) return finish()
      closing.value = true
      const onEnd = (event: Event) => event.target === node && finish()
      node.addEventListener('transitionend', onEnd)
      const timer = setTimeout(finish, ms + 50)
      stop = () => {
        node.removeEventListener('transitionend', onEnd)
        clearTimeout(timer)
      }
    },
    { flush: 'sync' },
  )
  onScopeDispose(() => stop?.())
  return { present, closing }
}
