// A site changes a token with a plain `:root` rule after the theme. That rule
// must win in light and in dark, whether dark comes from the system or from
// data-weft-scheme on <html>: the theme's scheme selectors add no specificity.
import { Window } from 'happy-dom'
import { describe, expect, it } from 'vitest'
import { neutral } from '../src/neutral.ts'
import { themeToCss } from '../src/to-css.ts'

const override = ':root { --weft-color-primary: #123456; }'

function primary(scheme: 'light' | 'dark', attribute?: 'light' | 'dark'): string {
  const window = new Window({ settings: { device: { prefersColorScheme: scheme } } })
  const { document } = window
  if (attribute) document.documentElement.setAttribute('data-weft-scheme', attribute)
  document.head.innerHTML = `<style>${themeToCss(neutral)}\n${override}</style>`
  return window.getComputedStyle(document.documentElement).getPropertyValue('--weft-color-primary').trim()
}

describe('a :root override after the theme', () => {
  it('wins in the light scheme', () => expect(primary('light')).toBe('#123456'))
  it('wins when the system prefers dark', () => expect(primary('dark')).toBe('#123456'))
  it('wins on <html data-weft-scheme="dark">', () => expect(primary('light', 'dark')).toBe('#123456'))
  it('wins on <html data-weft-scheme="light"> under a dark system', () => expect(primary('dark', 'light')).toBe('#123456'))
})

it('keeps the theme dark without an override', () => {
  const window = new Window({ settings: { device: { prefersColorScheme: 'dark' } } })
  window.document.head.innerHTML = `<style>${themeToCss(neutral)}</style>`
  expect(window.getComputedStyle(window.document.documentElement).getPropertyValue('--weft-color-primary').trim()).toBe(neutral.dark.primary)
})
