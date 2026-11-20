import { afterEach, describe, expect, it, vi } from 'vitest'
import { createVaporApp, effectScope } from 'vue'
import { ID_SCOPE, useId } from '../../src/headless/useId.ts'
import IdCase from '../fixtures/headless/IdCase.vue'
import { reset } from './dom.ts'

afterEach(reset)

function mountWith(provide?: (app: ReturnType<typeof createVaporApp>) => void) {
  const el = document.createElement('div')
  document.body.appendChild(el)
  const app = createVaporApp(IdCase as never)
  provide?.(app)
  app.mount(el)
  const ids = [...el.querySelectorAll('.id')].map((e) => e.id)
  app.unmount()
  return ids
}

describe('useId', () => {
  it("gives each component its own id from Vue's per-app counter, with the prefix in front", () => {
    const ids = mountWith()
    expect(ids).toHaveLength(2)
    expect(ids[0]).toMatch(/^weft-v-\w+$/)
    expect(ids[1]).toMatch(/^menu-v-\w+$/)
    expect(new Set(ids.map((i) => i.split('-').slice(1).join('-'))).size).toBe(2)
  })

  it('is the same on every render of the same tree, so server and browser agree', () => {
    expect(mountWith()).toEqual(mountWith())
  })

  it("inside an island counts in the island's own scope, the same on the server and in the browser", () => {
    const ids = mountWith((app) => app.provide(ID_SCOPE, { prefix: 'i3', next: 0 }))
    expect(ids).toEqual(['weft-i3-0', 'menu-i3-1'])
  })

  it('outside a component still gives unique ids, without a warning', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const scope = effectScope()
    const [a, b] = scope.run(() => [useId(), useId('x')])!
    scope.stop()
    expect(a).toMatch(/^weft-/)
    expect(b).toMatch(/^x-/)
    expect(a.slice(5)).not.toBe(b.slice(2))
    expect(warn).not.toHaveBeenCalled()
    warn.mockRestore()
  })
})
