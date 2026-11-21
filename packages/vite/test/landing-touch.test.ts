// A static page (never hydrated) with <WeftProvider surface="landing"> sizes
// its hit targets from the device: the provider leaves the input to the
// theme's `(pointer: coarse)` at the root, so a phone gets 44 px targets and a
// mouse 24 px, with no script. A provider writing the detected input on its
// element would pin the server's guess (`pointer`) on every phone.
import { createRequire } from 'node:module'
import { dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { Window } from 'happy-dom'
import { expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

const ui = createRequire(import.meta.url).resolve('@weft/ui')
const themes = createRequire(ui).resolve('@weft/themes')

it.each([
  [5, '44px'],
  [0, '24px'],
])('with %i touch points, a landing page resolves --weft-hit-min to %s', async (maxTouchPoints, hit) => {
  const root = fileURLToPath(new URL('./fixtures/landing-touch', import.meta.url))
  const { html } = await renderPage({ root, page: 'pages/index.vue' })
  expect(html).toContain('data-weft-surface="landing"')
  expect(html).not.toContain('data-weft-input')

  // (Not a dependency of this package: reached through @weft/ui's, typed by hand.)
  const { neutral, themeToCss } = (await import(themes)) as { neutral: object; themeToCss: (theme: object) => string }
  const window = new Window({ settings: { navigator: { maxTouchPoints } } })
  // happy-dom reports a custom property only where it is declared, so it is read through a property that uses it.
  const probe = 'button { min-block-size: var(--weft-hit-min, 0px); }'
  window.document.write(`<!doctype html><html><head><style>${themeToCss(neutral)}\n${probe}</style></head><body>${html}</body></html>`)
  const button = window.document.querySelector('button')!
  expect(window.getComputedStyle(button).minBlockSize).toBe(hit)
  await window.happyDOM.close()
})
