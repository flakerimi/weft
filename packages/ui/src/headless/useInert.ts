/**
 * useInert(active)
 *
 * Make the page behind a modal overlay inert while `active` is true: every
 * child of the body except `#weft-overlays` (where overlays render) gets the
 * `inert` attribute, so neither a pointer, the keyboard nor assistive
 * technology reaches it. Holders are counted across the page, like
 * useScrollLock: the page comes back only when the last of them lets go. An
 * element that was inert already is left so; one added beside the root
 * while held becomes inert too. Overlays inside the root stay reachable (a
 * toast's actions); the modal's focus trap decides about them.
 *
 * The timing matters for focus. Holding starts after the render that opened
 * the overlay (a post-flush watcher), once its focus trap has moved focus in,
 * so the trigger is not blurred before the overlay stack has noted it as the
 * opener. Letting go happens at once (synchronously), so focus can return
 * to the trigger in that same flush. Nothing happens on the server.
 */
import { onScopeDispose, watch, type Ref } from 'vue'

const ROOT = 'weft-overlays'
let holders = 0
const made = new Set<Element>()
let observer: MutationObserver | undefined

function cover(el: Element) {
  if (el.id === ROOT || el.hasAttribute('inert') || !(el instanceof HTMLElement)) return
  if (el.tagName === 'SCRIPT' || el.tagName === 'STYLE' || el.tagName === 'TEMPLATE') return
  el.inert = true
  made.add(el)
}

function hold() {
  if (holders++ > 0) return
  for (const el of document.body.children) cover(el)
  if (typeof MutationObserver !== 'undefined') {
    observer = new MutationObserver((records) => {
      for (const r of records) for (const node of r.addedNodes) if (node instanceof Element) cover(node)
    })
    observer.observe(document.body, { childList: true })
  }
}

function release() {
  if (--holders > 0) return
  observer?.disconnect()
  observer = undefined
  for (const el of made) (el as HTMLElement).inert = false
  made.clear()
}

export function useInert(active: Readonly<Ref<boolean>>): void {
  let held = false
  const set = (on: boolean) => {
    if (on === held || typeof document === 'undefined') return
    held = on
    if (on) hold()
    else release()
  }
  watch(active, (on) => on && set(true), { immediate: true, flush: 'post' })
  watch(active, (on) => !on && set(false), { flush: 'sync' })
  onScopeDispose(() => set(false))
}
