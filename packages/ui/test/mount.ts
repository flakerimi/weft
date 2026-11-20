import { createVaporApp, type Component } from 'vue'

/** Mount a Vapor component into a fresh element attached to document.body. */
export function mount(component: Component, props?: Record<string, unknown>) {
  const el = document.createElement('div')
  document.body.appendChild(el)
  const app = createVaporApp(component as never, props as never)
  app.mount(el)
  return {
    el,
    unmount() {
      app.unmount()
      el.remove()
    },
  }
}
