import { expect, it, vi } from 'vitest'
import WithHead from './fixtures/WithHead.vue'
import { mount } from './mount.ts'

it('useHead does nothing in the browser: the build writes the head', () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  document.title = 'Before'
  const before = document.head.innerHTML
  const { el, unmount } = mount(WithHead)
  expect(el.textContent).toBe('with head')
  expect(document.title).toBe('Before')
  expect(document.head.innerHTML).toBe(before)
  expect(warn).not.toHaveBeenCalled()
  unmount()
  warn.mockRestore()
})
