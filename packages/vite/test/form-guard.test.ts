// A Form sends nothing before its island is live. The server renders it in
// full, button and all, so until the island hydrates a press on submit (or
// Enter in a field) would be the browser's own GET, putting a password in
// the address. Form registers a pre-paint script that stops a submit of any
// Weft form from the first moment, and renders method="post", so that with
// scripts off a submit puts nothing in the address either.
import { execFile } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { renderPage } from '../src/render.ts'

const run = promisify(execFile)
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))
const root = here('./fixtures/form-guard')
let temp: string
let out: string
const page = (name: string) => readFileSync(join(out, 'pages', name, 'index.html'), 'utf8')
const head = (name: string) => page(name).slice(0, page(name).indexOf('</head>'))
const snippets = (html: string) => [...html.matchAll(/<script data-weft-prepaint>([\s\S]*?)<\/script>/g)].map((m) => m[1]!)

beforeAll(async () => {
  temp = mkdtempSync(join(tmpdir(), 'weft-form-guard-'))
  out = join(temp, 'dist')
  await run(process.execPath, [here('../bin/weft-build-pages.mjs'), '--root', root, '--out', out], {
    env: { ...process.env, NODE_ENV: undefined },
  })
}, 60_000)

afterAll(() => {
  rmSync(temp, { recursive: true, force: true })
})

describe('a Form', () => {
  it('renders method="post" on the server and registers its pre-paint script', async () => {
    const { html, prePaint } = await renderPage({ root, page: 'pages/login.vue' })
    expect(html).toContain('<form class="weft-form" method="post" novalidate>')
    expect(prePaint).toHaveLength(1)
    expect(prePaint[0]).toContain('"weft-form"')
  })

  it('in an island puts its script in <head> before any module script', () => {
    const h = head('login')
    expect(snippets(h)).toHaveLength(1)
    expect(h.indexOf('data-weft-prepaint')).toBeGreaterThan(-1)
    expect(h.indexOf('data-weft-prepaint')).toBeLessThan(h.indexOf('<script type="module"'))
    expect(h.indexOf('data-weft-prepaint')).toBeLessThan(h.indexOf('<style>'))
    expect(page('login')).toMatch(/<form class="weft-form" method="post" novalidate>/)
  })

  it('sends nothing before the island hydrates, keeps its fields usable, and submits once when live', async () => {
    const { stdout } = await run(process.execPath, [here('./hydrate-form.mjs'), out, 'pages/login/index.html'])
    expect(JSON.parse(stdout)).toEqual({
      before: { method: 'post', prevented: true, seen: true, disabled: false, password: 'hunter2' },
      hydrated: true,
      after: { prevented: true, seen: true, sent: '1', sameForm: true },
    })
  })

  it('leaves a page with two forms one script, and a page with none no script', () => {
    expect(snippets(head('two'))).toHaveLength(1)
    expect(page('two').match(/<form class="weft-form" method="post"/g)).toHaveLength(2)
    expect(page('plain')).not.toContain('data-weft-prepaint')
  })

  it('stops a submit only of a Weft form', () => {
    const code = snippets(head('login'))[0]!
    const listeners: ((e: unknown) => void)[] = []
    const window = { addEventListener: (type: string, fn: (e: unknown) => void, capture: boolean) => type === 'submit' && capture && listeners.push(fn) }
    new Function('window', code)(window)
    expect(listeners).toHaveLength(1)
    const submit = (className: string | null) => {
      let prevented = false
      const target = className === null ? null : { classList: { contains: (c: string) => className.split(' ').includes(c) } }
      listeners[0]!({ target, preventDefault: () => (prevented = true) })
      return prevented
    }
    expect(submit('weft-form')).toBe(true)
    expect(submit('search weft-form')).toBe(true)
    expect(submit('search')).toBe(false)
    expect(submit('weft-form-field')).toBe(false)
    expect(submit(null)).toBe(false)
  })
})
