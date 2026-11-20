import { onScopeDispose, readonly, ref, watch, type Ref } from 'vue'
import { widthOf, type Width } from './detect.ts'

/** The width class of an element, measured from the element itself, not the window. */
export function useWidth(el: Readonly<Ref<HTMLElement | null>>): Readonly<Ref<Width>> {
  const width = ref<Width>('wide')
  let observer: ResizeObserver | undefined

  const stop = watch(
    el,
    (node) => {
      observer?.disconnect()
      observer = undefined
      if (!node || typeof ResizeObserver === 'undefined') return
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
    observer?.disconnect()
  })
  return readonly(width)
}
