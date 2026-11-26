// @vitest-environment happy-dom
// The pre-paint guard a Form puts in the head: it cancels a Weft form's
// submit until its island is live; in development it also says, once, that
// a form in no island does nothing.
import { afterEach, expect, it, vi } from 'vitest'
import { formGuard } from '../../src/forms/guard.ts'

afterEach(() => {
  document.body.innerHTML = ''
  delete (window as { __weftFormAlone?: unknown }).__weftFormAlone
  vi.restoreAllMocks()
})

function submit(form: HTMLFormElement): boolean {
  const e = new Event('submit', { bubbles: true, cancelable: true })
  form.dispatchEvent(e)
  return e.defaultPrevented
}

it('in development, warns once for a form in no island, and never for one in an island', () => {
  const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
  const off = new AbortController()
  new Function('window', 'signal', formGuard(true).replace('true)', '{capture:true,signal})'))(window, off.signal)
  document.body.innerHTML = '<weft-island><form class="weft-form"></form></weft-island><form class="weft-form alone"></form><form class="other"></form>'
  const [inIsland, alone, other] = [...document.querySelectorAll('form')] as HTMLFormElement[]
  expect(submit(inIsland!)).toBe(true)
  expect(warn).not.toHaveBeenCalled()
  expect(submit(alone!)).toBe(true)
  expect(submit(alone!)).toBe(true)
  expect(warn).toHaveBeenCalledTimes(1)
  expect(String(warn.mock.calls[0]![0])).toContain('island')
  expect(submit(other!)).toBe(false)
  off.abort()
})

it('in production is the guard alone, with no warning in it', () => {
  expect(formGuard(false)).toBe(
    'window.addEventListener("submit",function(e){var t=e.target;if(t&&t.classList&&t.classList.contains("weft-form"))e.preventDefault()},true)',
  )
})
