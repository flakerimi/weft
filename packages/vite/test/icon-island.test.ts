// @vitest-environment happy-dom
// An island drawing an Icon hydrates the server's svg in place: the same node,
// the same drawing, no warning, and it reacts. The server half is in
// ssr-headless.test.ts; this package's test config runs weftIcons().
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { lucideBody } from '../src/icons.ts'
import { startIslands } from '../src/island-runtime.ts'
import Rated from './fixtures/icon-island/components/Rated.vue'
import { ratedIsland } from './fixtures/icon-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates an Icon inside an island', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const star = lucideBody('star')!
  document.body.innerHTML = `<main>${ratedIsland(star)}</main>`
  const svg = document.querySelector('svg')!
  startIslands({ Rated: () => Promise.resolve(Rated as Component) })
  await settle()
  expect(document.querySelector('weft-island script')).toBeNull()
  document.querySelector('button')!.click()
  await settle()
  expect(document.querySelector('svg')).toBe(svg)
  expect(svg.getAttribute('aria-label')).toBe('2 stars')
  expect(svg.innerHTML.replace(/><\/path>/g, '/>')).toBe(star)
  expect(warn).not.toHaveBeenCalled()
})
