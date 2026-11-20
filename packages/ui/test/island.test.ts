import { expect, it } from 'vitest'
import { Island } from '../src/index.ts'
import Hello from './fixtures/Hello.vue'
import { mount } from './mount.ts'

it('renders no element in the browser; the island runtime hydrates', () => {
  expect(Island).toMatchObject({ __name: 'Island' })
  const { el, unmount } = mount(Island, { when: 'load', component: Hello, props: { name: 'Weft' } })
  expect(el.querySelector('*')).toBeNull()
  expect(el.textContent).toBe('')
  unmount()
})
