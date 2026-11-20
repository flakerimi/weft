// @vitest-environment happy-dom
import { mkdtempSync, readFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { brotliCompressSync, constants as zlibConstants } from 'node:zlib'
import { build } from 'vite'
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest'
import type { Component } from 'vue'
import { startIslands } from '../src/island-runtime.ts'
import Counter from './fixtures/site/components/Counter.vue'
import Boom from './fixtures/runtime/Boom.vue'

// Server HTML exactly as renderPage writes it for <Island :component="Counter" :props="{ start: 2 }" />.
const counterHTML = (when: string, extra = '') =>
  `<weft-island data-id="Counter" data-when="${when}"${extra}>` +
  `<script type="application/json">{"start":2}</script>` +
  `<button type="button">n is 2</button></weft-island>`

const settle = () => new Promise((r) => setTimeout(r, 0))
const registry = { Counter: () => Promise.resolve(Counter as Component) }

let warn: MockInstance<typeof console.warn>

beforeEach(() => {
  warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
})

afterEach(() => {
  warn.mockRestore()
  vi.useRealTimers()
  vi.unstubAllGlobals()
  document.body.innerHTML = ''
})

function page(html: string) {
  document.body.innerHTML = `<main>${html}</main>`
}

/** Click the island's button; report whether it reacted and whether the node is the server's. */
async function clickCounter(island: Element, serverButton: Element) {
  const button = island.querySelector('button')!
  ;(button as HTMLButtonElement).click()
  await settle()
  return { text: island.querySelector('button')!.textContent, same: island.querySelector('button') === serverButton }
}

describe('startIslands', () => {
  it('hydrates a load island in place: the server button node becomes interactive', async () => {
    page(counterHTML('load'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!

    startIslands(registry)
    await settle()

    // The props script is read and taken out, so the mount sees only the server HTML.
    expect(island.querySelector('script')).toBeNull()
    expect(island.firstElementChild).toBe(button)
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
    expect(warn).not.toHaveBeenCalled()
  })

  it('waits for the first intersection before hydrating a visible island', async () => {
    page(counterHTML('visible'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!
    let fire: (entries: Partial<IntersectionObserverEntry>[]) => void = () => {}
    const observed: Element[] = []
    const disconnect = vi.fn()
    class FakeObserver {
      constructor(cb: (entries: Partial<IntersectionObserverEntry>[]) => void) {
        fire = cb
      }
      observe(el: Element) {
        observed.push(el)
      }
      disconnect = disconnect
    }

    startIslands(registry, { observe: FakeObserver as unknown as typeof IntersectionObserver })
    await settle()
    expect(observed).toEqual([island])
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 2', same: true })

    fire([{ isIntersecting: false, target: island }])
    await settle()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 2', same: true })

    fire([{ isIntersecting: true, target: island }])
    await settle()
    expect(disconnect).toHaveBeenCalled()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('waits for its media query to match before hydrating a media island', async () => {
    page(counterHTML('media', ' data-media="(min-width: 40rem)"'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!
    const listeners = new Set<() => void>()
    const query = {
      matches: false,
      addEventListener: (_: string, cb: () => void) => listeners.add(cb),
      removeEventListener: (_: string, cb: () => void) => listeners.delete(cb),
    }
    const asked: string[] = []
    const matchMedia = (q: string) => {
      asked.push(q)
      return query as unknown as MediaQueryList
    }

    startIslands(registry, { matchMedia })
    await settle()
    expect(asked).toEqual(['(min-width: 40rem)'])
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 2', same: true })

    query.matches = true
    for (const cb of [...listeners]) cb()
    await settle()
    expect(listeners.size).toBe(0)
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('hydrates a media island at once when its query already matches', async () => {
    page(counterHTML('media', ' data-media="(min-width: 40rem)"'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!
    const matchMedia = () => ({ matches: true, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList

    startIslands(registry, { matchMedia })
    await settle()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('warns and skips a media island that has no media query', async () => {
    page(counterHTML('media'))
    const island = document.querySelector('weft-island')!
    const before = island.innerHTML
    const matchMedia = vi.fn()

    startIslands(registry, { matchMedia })
    await settle()
    expect(matchMedia).not.toHaveBeenCalled()
    expect(island.innerHTML).toBe(before)
    expect(warn.mock.calls).toEqual([['weft: island Counter did not start: data-when="media" has no data-media.']])
  })

  it('hydrates an idle island when the idle callback runs', async () => {
    page(counterHTML('idle'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!
    let run = () => {}

    startIslands(registry, { idle: (cb) => (run = cb) })
    await settle()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 2', same: true })

    run()
    await settle()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('falls back to a 200 ms timer for idle islands without requestIdleCallback', async () => {
    vi.stubGlobal('requestIdleCallback', undefined)
    vi.useFakeTimers()
    page(counterHTML('idle'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!

    startIslands(registry)
    await vi.advanceTimersByTimeAsync(199)
    expect(island.querySelector('script')).not.toBeNull()
    await vi.advanceTimersByTimeAsync(1)
    expect(island.querySelector('script')).toBeNull()

    vi.useRealTimers()
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('keeps the server HTML of an island whose chunk fails to load, and starts the others', async () => {
    const broken =
      `<weft-island data-id="Broken" data-when="load"><script type="application/json">{}</script>` +
      `<p>static <a href="/about">content</a></p></weft-island>`
    page(broken + counterHTML('load'))
    const [first, second] = document.querySelectorAll('weft-island')
    const before = first!.innerHTML
    const button = second!.querySelector('button')!

    startIslands({ ...registry, Broken: () => Promise.reject(new Error('Failed to fetch')) })
    await settle()

    expect(first!.innerHTML).toBe(before)
    expect(warn.mock.calls).toEqual([['weft: island Broken did not start: Failed to fetch']])
    expect(await clickCounter(second!, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('warns once for an island the registry does not know', async () => {
    page(counterHTML('load').replace('data-id="Counter"', 'data-id="Missing"'))
    const island = document.querySelector('weft-island')!
    const before = island.innerHTML

    startIslands(registry)
    await settle()
    expect(island.innerHTML).toBe(before)
    expect(warn.mock.calls).toEqual([['weft: island Missing did not start: no chunk is registered for it.']])
  })

  it('warns once and keeps the server HTML when hydration throws', async () => {
    const boom =
      `<weft-island data-id="Boom" data-when="load"><script type="application/json">{}</script>` +
      `<p>server copy</p></weft-island>`
    page(boom + counterHTML('load'))
    const [first, second] = document.querySelectorAll('weft-island')
    const button = second!.querySelector('button')!

    startIslands({ ...registry, Boom: () => Promise.resolve(Boom as Component) })
    await settle()

    expect(first!.textContent).toBe('server copy')
    expect(warn.mock.calls).toEqual([['weft: island Boom did not start: boom in setup']])
    expect(await clickCounter(second!, button)).toEqual({ text: 'n is 3', same: true })
  })

  it('recovers to the client render when the server HTML differs, and keeps the page working', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    // The server wrote a stale count and different markup around it.
    page(
      `<weft-island data-id="Counter" data-when="load"><script type="application/json">{"start":2}</script>` +
        `<p>n is 9</p></weft-island>` +
        counterHTML('load'),
    )
    const [stale, good] = document.querySelectorAll('weft-island')
    const button = good!.querySelector('button')!

    startIslands(registry)
    await settle()

    expect(stale!.innerHTML).toBe('<button type="button">n is 2</button>')
    ;(stale!.querySelector('button') as HTMLButtonElement).click()
    await settle()
    expect(stale!.textContent).toBe('n is 3')
    expect(await clickCounter(good!, button)).toEqual({ text: 'n is 3', same: true })
    error.mockRestore()
  })

  it('takes the client text when only the text differs, as with a server-time value', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    page(counterHTML('load').replace('n is 2</button>', 'n is 9</button>'))
    const island = document.querySelector('weft-island')!
    const button = island.querySelector('button')!

    startIslands(registry)
    await settle()
    expect(island.textContent).toBe('n is 2')
    expect(await clickCounter(island, button)).toEqual({ text: 'n is 3', same: true })
    error.mockRestore()
  })
})

describe('island runtime size', () => {
  it('is at most 1.5 KB minified and brotli-compressed, without Vue', async () => {
    const outDir = mkdtempSync(join(tmpdir(), 'weft-island-runtime-'))
    await build({
      configFile: false,
      logLevel: 'silent',
      build: {
        outDir,
        minify: true,
        // An app build, as a page ships it: library mode keeps whitespace in ES output.
        rolldownOptions: {
          input: join(import.meta.dirname, '../src/island-runtime.ts'),
          external: ['vue'],
          preserveEntrySignatures: 'strict',
          output: { entryFileNames: 'island-runtime.js' },
        },
      },
    })
    const js = readFileSync(join(outDir, 'island-runtime.js'))
    expect(js.toString()).toContain('startIslands')
    expect(js.toString()).toMatch(/from\s*"vue"/) // Vue is imported, not bundled
    const size = brotliCompressSync(js, { params: { [zlibConstants.BROTLI_PARAM_QUALITY]: 11 } }).length
    expect(size).toBeGreaterThan(0)
    expect(size).toBeLessThanOrEqual(1536)
  })
})
