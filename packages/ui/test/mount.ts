import { createVaporApp, type Component } from 'vue'

/**
 * Mount a Vapor component into a fresh element attached to document.body.
 * `exposed` is what the component gave `defineExpose`, with refs unwrapped
 * (read and assign `exposed.text`, as a parent's template ref would).
 */
export function mount(component: Component, props?: Record<string, unknown>) {
  const el = document.createElement('div')
  document.body.appendChild(el)
  const app = createVaporApp(component as never, props as never)
  const exposed = app.mount(el) as unknown as Record<string, unknown>
  return {
    el,
    exposed,
    unmount() {
      app.unmount()
      el.remove()
    },
  }
}
