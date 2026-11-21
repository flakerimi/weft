import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useInert } from '../../src/headless/useInert.ts'
import { $, html, reset } from './dom.ts'

afterEach(reset)

describe('useInert', () => {
  function page() {
    const app = html('<button class="in-app">a</button>')
    const already = html('<p>already</p>')
    already.inert = true
    const root = document.createElement('div')
    root.id = 'weft-overlays'
    document.body.appendChild(root)
    return { app, already, root }
  }

  it('makes everything beside #weft-overlays inert while any holder is active, counted', async () => {
    const { app, already, root } = page()
    const a = ref(false)
    const b = ref(false)
    const scope = effectScope()
    scope.run(() => {
      useInert(a)
      useInert(b)
    })
    a.value = true
    expect(app.inert).toBe(false) // after the render, once focus has moved in
    await nextTick()
    expect(app.inert).toBe(true)
    expect(root.inert).toBe(false)
    b.value = true
    await nextTick()
    a.value = false // at once, so focus can go back to the page
    expect(app.inert).toBe(true)
    b.value = false
    expect(app.inert).toBe(false)
    expect(already.inert).toBe(true) // it was inert before: left so
    scope.stop()
  })

  it('makes an element added beside the root while held inert too, and lets go when its scope ends', async () => {
    page()
    const on = ref(true)
    const scope = effectScope()
    scope.run(() => useInert(on))
    await nextTick()
    const late = html('<p>late</p>')
    await new Promise((r) => setTimeout(r, 0))
    expect(late.inert).toBe(true)
    scope.stop()
    expect(late.inert).toBe(false)
    expect($('.in-app').closest('div')!.inert).toBe(false)
  })
})
