import { expect, it } from 'vitest'
import { nextTick } from 'vue'
import Hello from './fixtures/Hello.vue'
import { mount } from './mount.ts'

it('mounts a Vapor component and reacts', async () => {
  const { el, unmount } = mount(Hello, { name: 'Weft' })
  const p = el.querySelector('.hello') as HTMLElement
  expect(p.textContent).toBe('Hello Weft 1')
  p.click()
  await nextTick()
  expect(p.textContent).toBe('Hello Weft 2')
  unmount()
})
