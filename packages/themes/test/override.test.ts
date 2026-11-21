// A site changes a token with a rule after the theme. On `:root` it wins in
// light and in dark, whether dark comes from the system or from
// data-weft-scheme on <html>: the theme's scheme selectors add no specificity.
// An element with a data-weft-scheme of its own (a dark band, and the
// overlays opened from inside one) sets the scheme's tokens again on itself,
// so the override names those elements too: `:root, [data-weft-scheme]`.
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

describe('an override on :root, [data-weft-scheme] after the theme', () => {
  const both = ':root, [data-weft-scheme] { --weft-color-primary: #123456; }'
  function inside(scheme: 'light' | 'dark', css: string): string {
    const window = new Window({ settings: { device: { prefersColorScheme: scheme } } })
    const { document } = window
    document.head.innerHTML = `<style>${themeToCss(neutral)}\n${css}</style>`
    document.body.innerHTML = '<section data-weft-scheme="dark">text</section>'
    // The element itself: what it declares is what everything inside it inherits.
    return window.getComputedStyle(document.querySelector('section')!).getPropertyValue('--weft-color-primary').trim()
  }

  it('wins inside an element with data-weft-scheme="dark"', () => {
    expect(inside('light', both)).toBe('#123456')
    expect(inside('dark', both)).toBe('#123456')
  })

  it('where a :root rule alone does not: the element sets the dark tokens again', () => {
    expect(inside('light', override)).toBe(neutral.dark.primary)
  })

  it('still wins on the page itself', () => {
    const window = new Window({ settings: { device: { prefersColorScheme: 'dark' } } })
    window.document.head.innerHTML = `<style>${themeToCss(neutral)}\n${both}</style>`
    expect(window.getComputedStyle(window.document.documentElement).getPropertyValue('--weft-color-primary').trim()).toBe('#123456')
  })
})
