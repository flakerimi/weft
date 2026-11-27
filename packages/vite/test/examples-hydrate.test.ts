// @vitest-environment happy-dom
// Review Focus 5, the browser half: every @weft/ui example, rendered inside
// an <Island> on the server by renderPage (render-example-islands.ts, in a
// Node process of its own, with no window or document), hydrates in place:
// the same nodes stay, Vue reports no mismatch, nothing warns or throws, and
// it unmounts cleanly. The island is hydrated as startIslands hydrates one
// (its props script taken out, its id scope provided); here each app is also
// unmounted, so one example's listeners and timers do not reach the next.
// examples.test.ts is the server half: each example as a page of its own.
// The server cannot know the input, so it draws the pointer form; a phone
// (a coarse pointer) must hydrate that same form and switch once mounted:
// each example hydrates under both a fine and a coarse matchMedia.
import { execFile } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { createVaporSSRApp, type Component } from 'vue'
import { setMatchMedia } from '@weft/ui'
import { ISLAND_ID_SCOPE } from '../src/island-runtime.ts'

const modules = import.meta.glob<{ default: Component }>('../../ui/examples/*/*.vue')
/** Example file (`Button/basic.vue`) → its loader; helper components (PascalCase) are not examples. */
const examples = Object.fromEntries(
  Object.entries(modules)
    .map(([path, load]) => [path.replace('../../ui/examples/', ''), load] as const)
    .filter(([file]) => /^[^/]+\/[a-z0-9-]+\.vue$/.test(file)),
)
let server: Record<string, string>

beforeAll(async () => {
  // (Not new URL(…, import.meta.url): URL is happy-dom's here.)
  const script = join(dirname(fileURLToPath(import.meta.url)), 'render-example-islands.ts')
  // Plain Node, not this happy-dom worker: its window must not be seen by the server render.
  const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith('VITEST')))
  const { stdout, stderr } = await promisify(execFile)(process.execPath, [script], { maxBuffer: 64 * 1024 * 1024, env })
  // The server render warns about nothing either.
  expect(stderr).toBe('')
  server = JSON.parse(stdout) as Record<string, string>
}, 120_000)

// No server answers here: GET /api/auth/me, which an example with `perm` or a
// Guard asks, is answered as the gallery's server answers it (site-src/main.go),
// with the demo user in fixtures/demo-me.json; any other request is a mistake
// of the example's.
const demoMe = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'fixtures', 'demo-me.json'), 'utf8')
let strays: string[] = []
beforeEach(() => {
  strays = []
  vi.stubGlobal('fetch', async (url: string) => {
    if (url === '/api/auth/me') return new Response(demoMe, { status: 200, headers: { 'content-type': 'application/json' } })
    strays.push(String(url))
    throw new TypeError('Failed to fetch')
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  expect(strays).toEqual([])
  setMatchMedia(undefined)
  document.body.innerHTML = ''
  localStorage.clear()
  vi.restoreAllMocks()
})

const settle = () => new Promise((r) => setTimeout(r, 0))
/** A matchMedia that reports a coarse pointer or a fine one, and never changes. */
const media = (coarse: boolean) => (q: string) =>
  ({ matches: coarse && q === '(pointer: coarse)', media: q, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList

describe('every @weft/ui example, server-rendered in an island', () => {
  it('is every example there is', () => {
    expect(Object.keys(server).sort()).toEqual(Object.keys(examples).sort())
    expect(Object.keys(examples).length).toBeGreaterThan(100)
  })

  // Island's own example has an island inside: inside another island it is drawn in place, and
  // the outer island hydrates it, as the gallery's preview stage does.
  const islands = Object.keys(examples).sort()

  const cases = islands.flatMap((file) => [
    [file, 'pointer'],
    [file, 'touch'],
  ] as const)

  it.each(cases)('%s hydrates in place on %s, without a warning', async (file, input) => {
    setMatchMedia(media(input === 'touch'))
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    document.body.innerHTML = `<main>${server[file]}</main>`
    const island = document.querySelector<HTMLElement>('weft-island')!
    expect(island.dataset.id).toBe(/([^/]+)\.vue$/.exec(file)![1])
    // Every element the server drew, bar the props script and the inner parts of its svgs (drawn as markup).
    const before = [...island.querySelectorAll('*')].filter(
      (e) => e.tagName !== 'SCRIPT' && !(e instanceof SVGElement && !(e instanceof SVGSVGElement)),
    )
    expect(before.length).toBeGreaterThan(0)

    const component = (await examples[file]!()).default
    island.querySelector(':scope > script[type="application/json"]')!.remove()
    const app = createVaporSSRApp(component as never)
    app.provide(ISLAND_ID_SCOPE, { prefix: 'i0', next: 0 })
    let failed: unknown
    app.config.errorHandler = (e) => {
      failed ??= e
    }
    app.mount(island)
    // Hydrated: every node the server drew is still there, checked before anything runs after mounting
    // (a measured width, an image that fails to load, a fetch) changes what is drawn, as it may.
    for (const el of before) expect(el.isConnected, el.outerHTML.slice(0, 200)).toBe(true)
    await settle()
    await settle()
    expect(failed).toBeUndefined()
    expect(warn.mock.calls.map((c) => String(c[0]))).toEqual([])
    expect(error.mock.calls.map((c) => String(c[0]))).toEqual([])
    // Then the touch form: a Combobox's text field opens its sheet without raising the keyboard.
    if (file.startsWith('Combobox/')) {
      for (const field of island.querySelectorAll('input[role="combobox"]')) {
        expect(field.hasAttribute('readonly'), file).toBe(input === 'touch')
      }
    }

    // The demo user may edit products, not delete them or read users: each perm example shows both ways.
    if (file === 'Button/by-permission.vue') {
      expect([...island.querySelectorAll('button')].map((b) => b.textContent!.trim())).toEqual(['View', 'Edit'])
    }
    if (file === 'Guard/basic.vue') {
      expect(island.textContent).toContain('Demo may edit products.')
      expect(island.textContent).not.toContain('The users of this app.')
      expect(island.querySelector('.weft-alert[role="status"]')!.textContent).toBe('Your account cannot see this; ask an administrator.')
    }

    app.unmount()
    await settle()
    expect(failed).toBeUndefined()
    expect(error).not.toHaveBeenCalled()
  })
})
