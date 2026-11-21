// @vitest-environment happy-dom
// An island of every form component hydrates the server's markup in place:
// the same nodes, no warning (a node or text mismatch warns), and it reacts:
// typing, choosing, stepping and a submit that shows the Form's errors. The
// server half is forms-island-ssr.test.ts.
import { afterEach, expect, it, vi } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Forms from './fixtures/forms-island/components/Forms.vue'
import { formsIsland } from './fixtures/forms-island/markup.ts'

const settle = () => new Promise((r) => setTimeout(r, 0))
const key = (el: Element, k: string) => el.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true }))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

it('hydrates every form component inside an island', async () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const error = vi.spyOn(console, 'error').mockImplementation(() => {})
  document.body.innerHTML = `<main>${formsIsland}</main>`
  const island = document.querySelector('weft-island')!
  const before = Array.from(island.querySelectorAll('*')).filter((e) => e.tagName !== 'SCRIPT')
  startIslands({ Forms: () => Promise.resolve(Forms as Component) })
  await settle()
  for (const el of before) expect(el.isConnected, el.outerHTML).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()

  // Labels of controls that name themselves by the label's id (a group, a
  // slider) drop `for` once hydrated; the server could not know yet.
  const forOf = (text: string) =>
    [...island.querySelectorAll('label.weft-form-field__label')].find((l) => l.textContent!.includes(text))!.getAttribute('for')
  expect([forOf('Fruit'), forOf('Plan'), forOf('Volume')]).toEqual([null, null, null])
  expect(forOf('Email')).toBe(island.querySelector('input[type="email"]')!.id)

  const out = island.querySelector('output')!
  // Typing reaches the Form's state.
  const email = island.querySelector<HTMLInputElement>('input[type="email"]')!
  email.value = 'ada'
  email.dispatchEvent(new Event('input', { bubbles: true }))
  await settle()
  expect(out.textContent).toBe('ada|pro|30')
  // The radio group took its tab stop on the chosen radio, and an arrow chooses the next.
  const radios = island.querySelectorAll<HTMLInputElement>('.weft-radio__input')
  expect(radios[1]!.getAttribute('tabindex')).toBe('0')
  radios[1]!.focus()
  key(radios[1]!, 'ArrowDown')
  await settle()
  expect(out.textContent).toBe('ada|free|30')
  // The slider steps.
  key(island.querySelector('[role="slider"]')!, 'ArrowRight')
  await settle()
  expect(out.textContent).toBe('ada|free|31')
  // A submit shows the schema's errors and focuses the first invalid control.
  island.querySelector<HTMLButtonElement>('button[type="submit"]')!.click()
  await settle()
  const errors = [...island.querySelectorAll('.weft-form-field__error')].map((e) => e.textContent!.trim())
  expect(errors).toEqual(['Enter an email address.', 'This field is required.'])
  expect(document.activeElement).toBe(email)
  expect(email.getAttribute('aria-invalid')).toBe('true')
  expect(warn).not.toHaveBeenCalled()
  expect(error).not.toHaveBeenCalled()
})
