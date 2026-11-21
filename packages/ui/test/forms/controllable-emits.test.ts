// useControllable with a component's real defineEmits overloads, in Vapor:
// the probe type-checks under vue-tsc (see ControllableProbe.vue) and two
// v-models round-trip through it at run time.
import { expect, it } from 'vitest'
import { nextTick } from 'vue'
import ControllableParent from '../fixtures/forms/ControllableParent.vue'
import { mount } from '../mount.ts'

it('round-trips two v-models through useControllable and a defineEmits overload set', async () => {
  const m = mount(ControllableParent)
  const out = m.el.querySelector('output')!
  expect(out.textContent).toBe('5 false')
  ;(m.exposed.bump as () => void)()
  ;(m.exposed.toggle as () => void)()
  await nextTick()
  expect(m.exposed.count).toBe(6)
  expect(m.exposed.open).toBe(true)
  expect(out.textContent).toBe('6 true')
  m.exposed.count = 10
  await nextTick()
  expect(out.textContent).toBe('10 true')
  // Unbound, it keeps its own value from the default.
  expect(m.el.querySelectorAll('output')[1]!.textContent).toBe('0 false')
  m.unmount()
})
