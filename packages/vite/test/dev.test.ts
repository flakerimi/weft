// The page dev server: pages rendered when they are asked for, through one
// Vite server that listens. Each test runs a copy of a fixture in a temporary
// app laid out as `weft create` writes one: the site in `web/`, with no
// package.json of its own, and `node_modules` one folder up. The folder's
// name has a space and a non-ASCII letter (Review Focus 1).
import { spawn, type ChildProcess } from 'node:child_process'
import { closeSync, cpSync, mkdirSync, mkdtempSync, openSync, rmSync, symlinkSync, writeFileSync, writeSync } from 'node:fs'
import { createServer as createHTTPServer, request, type Server } from 'node:http'
import { createServer as createNetServer, type AddressInfo } from 'node:net'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { devSite } from '../src/dev.ts'

const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))

const temps: string[] = []
const closers: (() => Promise<unknown> | void)[] = []

afterEach(async () => {
  for (const close of closers.splice(0).reverse()) await close()
  for (const dir of temps.splice(0)) rmSync(dir, { recursive: true, force: true, maxRetries: 10 })
})

/** A copy of the fixture as `<temp>/app/web`, with the package's node_modules as `<temp>/app/node_modules`. */
function site(name: string): string {
  const temp = mkdtempSync(join(tmpdir(), 'weft dev café-'))
  temps.push(temp)
  const app = join(temp, 'app')
  const web = join(app, 'web')
  mkdirSync(app)
  cpSync(here(`./fixtures/${name}`), web, { recursive: true, filter: (src) => !src.includes('.weft-types') })
  symlinkSync(here('../node_modules'), join(app, 'node_modules'))
  return web
}

async function start(root: string, opts: Partial<Parameters<typeof devSite>[0]> = {}) {
  const dev = await devSite({ root, port: 0, ...opts })
  closers.push(() => dev.close())
  return dev
}

async function get(url: string) {
  const res = await fetch(url)
  return { status: res.status, type: res.headers.get('content-type'), body: await res.text() }
}

/** A GET of `path` exactly as written, dot segments kept, as a tool sends it: fetch would clean it. */
function getRaw(base: string, path: string): Promise<{ status: number; body: string }> {
  return new Promise((done, fail) => {
    const req = request(base, { path, method: 'GET' }, (res) => {
      let body = ''
      res.setEncoding('utf8')
      res.on('data', (c: string) => (body += c))
      res.on('end', () => done({ status: res.statusCode ?? 0, body }))
    })
    req.on('error', fail)
    req.end()
  })
}

/** The body's text, without its tags. */
/**
 * Waits until `messages` has had no new entry for 600 ms (at most 5 s). The
 * file watcher can report the fixture's own files, copied just before the
 * server started, some time later (macOS's FSEvents does): a test that counts
 * reloads waits for those to pass first.
 */
async function quiet(messages: unknown[]) {
  for (let seen = -1, i = 0; i < 20 && seen !== messages.length; i++) {
    seen = messages.length
    await new Promise((r) => setTimeout(r, 600))
  }
}

const text = (html: string) => html.replace(/<style>[\s\S]*?<\/style>|<script[\s\S]*?<\/script>/g, '').replace(/<[^>]+>/g, '')

describe('devSite', { timeout: 30_000 }, () => {
  it('serves a page', async () => {
    const dev = await start(site('dev-site'))
    expect(dev.url).toMatch(/^http:\/\/localhost:\d+\/$/)
    const page = await get(dev.url)
    expect(page.status).toBe(200)
    expect(page.type).toBe('text/html; charset=utf-8')
    expect(text(page.body)).toContain('Hello from dev')
    // The theme and base CSS, then the CSS of what the page renders: Badge, which it uses without an import.
    const style = /<style>([\s\S]*?)<\/style>/.exec(page.body)![1]!
    expect(style).toMatch(/--weft-[\w-]+\s*:/)
    expect(style).toContain('.weft-badge')
    expect(page.body).toContain('weft-badge')
    expect(page.body).toContain('<script type="module" src="/@vite/client"></script>')
    // No island, so no island script.
    expect(page.body).not.toContain('dev-islands')
  })

  it('writes one module script that starts the islands of a page that has them', async () => {
    const dev = await start(site('dev-site'))
    const page = await get(dev.url + 'counter')
    expect(page.status).toBe(200)
    expect(page.body).toContain('<weft-island data-id="Counter" data-when="load">')
    const scripts = [...page.body.matchAll(/<script type="module" src="([^"]+)"><\/script>/g)].map((m) => m[1]!)
    expect(scripts).toHaveLength(2)
    expect(scripts[0]).toBe('/@vite/client')
    const entry = await get(new URL(scripts[1]!, dev.url).href)
    expect(entry.status).toBe(200)
    expect(entry.body).toContain('startIslands')
    expect(entry.body).toMatch(/"Counter": \(\) => import\("[^"]*Counter\.vue[^"]*"\)/)
    // The island's own CSS is in the page; in the browser its styles load empty.
    expect(/<style>([\s\S]*?)<\/style>/.exec(page.body)![1]).toMatch(/\.counter \{\s*min-inline-size: 12ch;/)
  })

  it('refuses an island script for files no render has used', async () => {
    const dev = await start(site('dev-site'))
    const forged = Buffer.from(JSON.stringify({ Secret: '/etc/hosts' })).toString('base64url')
    const res = await get(`${dev.url}@id/__x00__virtual:weft/dev-islands?i=${forged}`)
    expect(res.status).toBe(500)
    expect(res.body).toContain('weft dev: Secret is not an island a page rendered; reload the page.')
    expect(res.body).not.toContain('import("/etc/hosts")')
  })

  it('sees an edit', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    expect(text((await get(dev.url)).body)).toContain('Hello from dev')
    writeFileSync(join(root, 'pages/index.vue'), '<template><main><h1>Edited at once</h1></main></template>\n')
    expect(text((await get(dev.url)).body)).toContain('Edited at once')

    // A component the page imports.
    expect(text((await get(dev.url + 'counter')).body)).toContain('n is 2')
    const counter = join(root, 'components/Counter.vue')
    writeFileSync(counter, '<script setup vapor lang="ts">\ndefineProps<{ start: number }>()\n</script>\n\n<template><b>{{ start }} edited</b></template>\n')
    expect(text((await get(dev.url + 'counter')).body)).toContain('2 edited')
  })

  it('calls getPaths() on each request', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    expect((await get(dev.url + 'blog/first')).status).toBe(200)
    expect(text((await get(dev.url + 'blog/first')).body)).toContain('Post first')
    expect((await get(dev.url + 'blog/second')).status).toBe(404)
    writeFileSync(join(root, 'content/posts.json'), '["first", "second"]\n')
    const second = await get(dev.url + 'blog/second')
    expect(second.status).toBe(200)
    expect(text(second.body)).toContain('Post second')
    // A trailing slash is the same page, as the Go server has it.
    expect(text((await get(dev.url + 'blog/second/')).body)).toContain('Post second')
  })

  it('dev serves a runtime page for any id', async () => {
    const dev = await start(site('dev-site'))
    for (const path of ['products/42', 'products/999', 'products/42/']) {
      const page = await get(dev.url + path)
      expect(page.status, path).toBe(200)
      // The shell, as the build writes it: the pattern on <html>, and every param empty.
      expect(page.body, path).toContain('<html lang="en" data-weft-route="/products/:id">')
      expect(text(page.body), path).toContain('Product []')
    }
    for (const path of ['products', 'products/1/extra', 'product/1']) {
      const page = await get(dev.url + path)
      expect(page.status, path).toBe(404)
      expect(text(page.body), path).toContain('Nothing here')
    }
    // No other page carries the pattern.
    expect((await get(dev.url)).body).toContain('<html lang="en">')
    // As warp.Serve answers them: an escape that is not UTF-8 is a param like
    // any other, and a path that starts with // is a path, not a host.
    const odd = await get(dev.url + 'products/%FF')
    expect(odd.status).toBe(200)
    expect(odd.body).toContain('data-weft-route="/products/:id"')
    const doubled = await get(dev.url.replace(/\/$/, '') + '//api/products')
    expect(doubled.status).toBe(404)
    expect(text(doubled.body)).toContain('Nothing here')
  })

  it('cleans dot segments as warp.Serve does', async () => {
    const dev = await start(site('dev-site'))
    // The rows of TestDotSegments in warp/manifest_test.go.
    for (const [path, status, has] of [
      ['/products/../counter', 200, 'n is 2'],
      ['/products/%2e%2e/counter', 200, 'n is 2'],
      ['/products/./42', 200, 'Product []'],
      ['/blog/../products/42/', 200, 'Product []'],
      ['/products/42/..', 404, 'Nothing here'],
      ['/..', 200, 'Hello from dev'],
    ] as const) {
      const page = await getRaw(dev.url, path)
      expect(page.status, path).toBe(status)
      expect(text(page.body), path).toContain(has)
    }
  })

  it('refuses two runtime pages of one shape, as the build does', async () => {
    const root = site('dev-site')
    mkdirSync(join(root, 'pages/a'))
    for (const name of ['x', 'y']) writeFileSync(join(root, `pages/a/[${name}].vue`), '<script lang="ts">\nexport const runtime = true\n</script>\n\n<template><p>a</p></template>\n')
    const dev = await start(root)
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})
    const page = await get(dev.url + 'a/1')
    expect(page.status).toBe(500)
    expect(page.body).toContain('pages &quot;a/[x].vue&quot; and &quot;a/[y].vue&quot; both match paths like /a/1; keep one.')
    // As the build fails, so does every path that reaches the patterns.
    expect((await get(dev.url + 'products/42')).status).toBe(500)
    // A path an exact page has does not.
    expect((await get(dev.url)).status).toBe(200)
    error.mockRestore()
  })

  it('serves the files a page wrote with useFile, once it has rendered', async () => {
    const dev = await start(site('dev-site'))
    expect((await get(dev.url + 'feed.json')).status).toBe(404)
    expect(text((await get(dev.url + 'feed')).body)).toContain('Feed')
    expect(await get(dev.url + 'feed.json')).toEqual({ status: 200, type: 'application/json', body: '{"posts":["first"]}\n' })
  })

  it("answers 404 with the site's page", async () => {
    const dev = await start(site('dev-site'))
    for (const path of ['no-such-page', '404', 'blog/none']) {
      const page = await get(dev.url + path)
      expect(page.status, path).toBe(404)
      expect(text(page.body), path).toContain('Nothing here')
      expect(page.body, path).toMatch(/<style>[\s\S]*--weft-/)
    }
  })

  it('writes the icon tags of the files in public/, as the build does, on a page and the 404 page', async () => {
    const root = site('dev-site')
    const icons = (html: string) => [...html.matchAll(/<link rel="(?:icon|apple-touch-icon)"[^>]*>/g)].map((m) => m[0])
    const dev = await start(root)
    // No public/ folder: no icon tag.
    expect(icons((await get(dev.url)).body)).toEqual([])
    mkdirSync(join(root, 'public'))
    for (const name of ['favicon.svg', 'favicon.ico', 'apple-touch-icon.png']) writeFileSync(join(root, 'public', name), `bytes of ${name}`)
    const want = [
      '<link rel="icon" href="/favicon.ico" sizes="32x32">',
      '<link rel="icon" href="/favicon.svg" type="image/svg+xml">',
      '<link rel="apple-touch-icon" href="/apple-touch-icon.png">',
    ]
    // Read on each request: a file dropped in shows on the next one.
    expect(icons((await get(dev.url)).body)).toEqual(want)
    expect(icons((await get(dev.url + 'no-such-page')).body)).toEqual(want)
  })

  it('answers 404 plainly without one', async () => {
    const dev = await start(site('static-site'))
    // The site's own vite.config is used: its define reaches the page.
    expect(text((await get(dev.url)).body)).toContain('Static from vite.config')
    const page = await get(dev.url + 'no-such-page')
    expect(page.status).toBe(404)
    expect(page.type).toBe('text/plain; charset=utf-8')
    expect(page.body).toBe('404 page not found\n')
  })

  it('proxies /api', async () => {
    const api = createHTTPServer((req, res) => res.end(`api got ${req.method} ${req.url}`))
    await new Promise<void>((done) => api.listen(0, '127.0.0.1', done))
    closers.push(() => new Promise((done) => api.close(done)))
    const dev = await start(site('dev-site'), { api: `http://127.0.0.1:${(api.address() as AddressInfo).port}` })
    expect(await get(dev.url + 'api/x?y=1')).toMatchObject({ status: 200, body: 'api got GET /api/x?y=1' })
    expect(await get(dev.url + 'api')).toMatchObject({ status: 200, body: 'api got GET /api' })
    const posted = await fetch(dev.url + 'api/items', { method: 'POST', body: 'x' })
    expect(await posted.text()).toBe('api got POST /api/items')
    // Only /api and what is under it, as warp.Serve sends to the API.
    const other = await get(dev.url + 'apiary')
    expect(other.status).toBe(404)
    expect(text(other.body)).toContain('Nothing here')
  })

  it('sends to /api what warp.Serve sends there, and refuses an escape net/http refuses', async () => {
    const api = createHTTPServer((req, res) => res.end(`api got ${req.method} ${req.url}`))
    await new Promise<void>((done) => api.listen(0, '127.0.0.1', done))
    closers.push(() => new Promise((done) => api.close(done)))
    const dev = await start(site('dev-site'), { api: `http://127.0.0.1:${(api.address() as AddressInfo).port}` })
    // The path as decoded, before any cleaning, is under /api: the API gets it as sent.
    for (const path of ['/api/../login', '/api/./auth/me', '/api/auth/../auth/me', '/%61pi/auth/me']) {
      expect(await getRaw(dev.url, path), path).toEqual({ status: 200, body: `api got GET ${path}` })
    }
    // A % that starts no escape is net/http's 400, as built.
    for (const path of ['/login%', '/a%2', '/%zz']) {
      expect((await getRaw(dev.url, path)).status, path).toBe(400)
    }
  })

  it('a render error is a 500 that names the file, and the next request works', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    // The terminal gets a line that names the page (and Vite's and Vue's own).
    const logged: string[] = []
    for (const level of ['error', 'warn'] as const) vi.spyOn(console, level).mockImplementation((m: unknown) => void logged.push(String(m)))
    closers.push(() => void vi.restoreAllMocks())
    writeFileSync(
      join(root, 'pages/broken.vue'),
      '<script setup vapor lang="ts">\nthrow new Error("broken <on> purpose")\n</script>\n\n<template><main /></template>\n',
    )
    const thrown = await get(dev.url + 'broken')
    expect(thrown.status).toBe(500)
    expect(thrown.type).toBe('text/html; charset=utf-8')
    expect(thrown.body).toContain('pages/broken.vue')
    expect(thrown.body).toContain('broken &lt;on&gt; purpose')
    // It reloads when the file is saved again.
    expect(thrown.body).toContain('<script type="module" src="/@vite/client"></script>')
    expect(logged).toContain('weft dev: pages/broken.vue did not render: broken <on> purpose')
    expect((await get(dev.url)).status).toBe(200)

    // A template that does not compile, in a component the page imports: both files are named.
    writeFileSync(join(root, 'components/Counter.vue'), '<template><main><b>unclosed</main></template>\n')
    const failed = await get(dev.url + 'counter')
    expect(failed.status).toBe(500)
    expect(failed.body).toContain('pages/counter.vue')
    expect(failed.body).toMatch(/<h1>pages\/counter\.vue did not render in components\/Counter\.vue:1:\d+<\/h1>/)
    writeFileSync(join(root, 'components/Counter.vue'), '<script setup vapor lang="ts">\nconst word = "fixed"\n</script>\n\n<template><b>{{ word }}</b></template>\n')
    const fixed = await get(dev.url + 'counter')
    expect(fixed.status).toBe(200)
    expect(text(fixed.body)).toContain('fixed')
  })

  it('sends a full reload when a file under the root changes', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    // The HMR socket's token, as the browser's client gets it.
    const client = await get(dev.url + '@vite/client')
    const token = /const wsToken = "([^"]+)"/.exec(client.body)![1]!
    const socket = new WebSocket(`${dev.url.replace('http', 'ws')}?token=${token}`, 'vite-hmr')
    closers.push(() => socket.close())
    const messages: { type: string; path?: string }[] = []
    socket.addEventListener('message', (e) => messages.push(JSON.parse(String(e.data))))
    const until = async (ok: () => boolean) => {
      for (let i = 0; i < 200 && !ok(); i++) await new Promise((r) => setTimeout(r, 25))
      return ok()
    }
    expect(await until(() => messages.some((m) => m.type === 'connected'))).toBe(true)
    // The island's module, loaded as the browser loads it, which Vite would otherwise replace in place.
    const page = await get(dev.url + 'counter')
    const entry = await get(new URL(/src="(\/@id\/[^"]+)"/.exec(page.body)![1]!, dev.url).href)
    const island = /import\("([^"]+Counter\.vue[^"]*)"\)/.exec(entry.body)![1]!
    expect((await get(new URL(island, dev.url).href)).status).toBe(200)
    await quiet(messages)
    for (const file of ['components/Counter.vue', 'content/posts.json']) {
      messages.length = 0
      writeFileSync(join(root, file), file.endsWith('.json') ? '["first", "third"]\n' : '<template><b>reloaded</b></template>\n')
      expect(await until(() => messages.some((m) => m.type === 'full-reload')), file).toBe(true)
      // And no update in place, before or after it (a custom event, such as plugin-vue's file-changed, does nothing).
      await new Promise((r) => setTimeout(r, 300))
      expect(messages.filter((m) => m.type !== 'custom').map((m) => m.type), file).toEqual(['full-reload'])
    }
  })

  it('sends one full reload for a save written in two steps, as an editor or a busy machine writes one', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    const client = await get(dev.url + '@vite/client')
    const token = /const wsToken = "([^"]+)"/.exec(client.body)![1]!
    const socket = new WebSocket(`${dev.url.replace('http', 'ws')}?token=${token}`, 'vite-hmr')
    closers.push(() => socket.close())
    const messages: { type: string }[] = []
    socket.addEventListener('message', (e) => messages.push(JSON.parse(String(e.data))))
    for (let i = 0; i < 200 && !messages.some((m) => m.type === 'connected'); i++) await new Promise((r) => setTimeout(r, 25))
    expect((await get(dev.url + 'counter')).status).toBe(200)
    await quiet(messages)
    for (const file of ['components/Counter.vue', 'content/posts.json']) {
      for (let round = 0; round < 3; round++) {
        messages.length = 0
        // Emptied first, written a moment later: the watcher sees two changes.
        const fd = openSync(join(root, file), 'w')
        await new Promise((r) => setTimeout(r, 30))
        writeSync(fd, file.endsWith('.json') ? `["first", "round ${round}"]\n` : `<template><b>round ${round}</b></template>\n`)
        closeSync(fd)
        await new Promise((r) => setTimeout(r, 500))
        expect(messages.filter((m) => m.type === 'full-reload'), `${file}, round ${round}`).toHaveLength(1)
      }
    }
  })

  it('reloads at most about a second after the first change, however long files keep changing', async () => {
    const root = site('dev-site')
    const dev = await start(root)
    const client = await get(dev.url + '@vite/client')
    const token = /const wsToken = "([^"]+)"/.exec(client.body)![1]!
    const socket = new WebSocket(`${dev.url.replace('http', 'ws')}?token=${token}`, 'vite-hmr')
    closers.push(() => socket.close())
    const reloads: number[] = []
    socket.addEventListener('message', (e) => JSON.parse(String(e.data)).type === 'full-reload' && reloads.push(Date.now()))
    const opened = await new Promise<boolean>((done) => socket.addEventListener('open', () => done(true)))
    expect(opened).toBe(true)
    expect((await get(dev.url + 'counter')).status).toBe(200)
    await quiet(reloads)
    reloads.length = 0
    // A writer that never pauses for the quiet period: every 60 ms for 2.4 s.
    const start0 = Date.now()
    for (let i = 0; i < 40; i++) {
      writeFileSync(join(root, 'content/posts.json'), `["first", "write ${i}"]\n`)
      await new Promise((r) => setTimeout(r, 60))
    }
    await new Promise((r) => setTimeout(r, 600))
    expect(reloads.length, 'a reload while the writes go on, and one after').toBeGreaterThanOrEqual(2)
    // The watcher adds its own latency (up to about 120 ms) to the second.
    expect(reloads[0]! - start0, 'the first reload').toBeLessThan(1500)
  })

  it('says when the port is taken', async () => {
    const taken = createNetServer()
    await new Promise<void>((done) => taken.listen(0, 'localhost', done))
    closers.push(() => new Promise((done) => taken.close(done)))
    const port = (taken.address() as AddressInfo).port
    await expect(devSite({ root: site('dev-site'), port })).rejects.toThrow(
      new Error(`weft dev: port ${port} is in use; stop what is using it or pass --port.`),
    )
  })

  it('refuses an unknown theme and a site without pages, in a sentence', async () => {
    const root = site('dev-site')
    await expect(devSite({ root, port: 0, theme: 'plain' as never })).rejects.toThrow(
      'weft dev: unknown theme "plain"; use neutral, lisa or weft.',
    )
    await expect(devSite({ root: join(root, 'content'), port: 0 })).rejects.toThrow(
      `weft dev: ${join(root, 'content', 'pages')} does not exist; a site's pages are .vue files in it, like pages/index.vue.`,
    )
    await expect(devSite({ root, port: 0, api: 'localhost:8080' })).rejects.toThrow(
      'weft dev: --api "localhost:8080" is not an http:// or https:// address; give one like http://127.0.0.1:8081.',
    )
  })
})

describe('weft-dev', { timeout: 30_000 }, () => {
  const children: ChildProcess[] = []
  afterEach(() => {
    for (const child of children.splice(0)) if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL')
  })

  /** Run the bin; resolves with its first stdout line, or with how it ended. Its standard input is /dev/null, or a pipe the test holds. */
  function run(args: string[], stdin: 'ignore' | 'pipe' = 'ignore') {
    const child = spawn(process.execPath, [here('../bin/weft-dev.mjs'), ...args], { stdio: [stdin, 'pipe', 'pipe'] })
    children.push(child)
    let stdout = ''
    let stderr = ''
    child.stdout!.on('data', (d) => (stdout += d))
    child.stderr!.on('data', (d) => (stderr += d))
    const exited = new Promise<{ code: number | null; signal: NodeJS.Signals | null }>((done) =>
      child.on('exit', (code, signal) => done({ code, signal })),
    )
    const line = new Promise<string>((done, fail) => {
      child.stdout!.on('data', () => stdout.includes('\n') && done(stdout.slice(0, stdout.indexOf('\n') + 1)))
      void exited.then(({ code }) => fail(new Error(`weft-dev exited with ${code}: ${stderr}`)))
    })
    // Awaited only by the tests that expect it to listen.
    line.catch(() => {})
    return { child, line, exited, out: () => ({ stdout, stderr }) }
  }

  for (const signal of ['SIGTERM', 'SIGINT'] as const) {
    it(`says where it listens, and stops at once on ${signal}`, async () => {
      const root = site('dev-site')
      const dev = run(['--root', root, '--port', '0', '--theme', 'weft'])
      const line = await dev.line
      expect(line).toMatch(/^weft: dev server at http:\/\/localhost:\d+\/\n$/)
      const url = line.slice('weft: dev server at '.length, -1)
      expect(text((await get(url)).body)).toContain('Hello from dev')
      const stopped = Date.now()
      dev.child.kill(signal)
      expect(await dev.exited).toEqual({ code: 0, signal: null })
      expect(Date.now() - stopped).toBeLessThan(3000)
      await expect(fetch(url)).rejects.toThrow()
      expect(dev.out().stderr).toBe('')
    }, 30_000)

    // `weft dev` may stop it before it listens, as on a Ctrl-C while it starts.
    it(`stops cleanly on ${signal} while it starts`, async () => {
      const root = site('dev-site')
      writeFileSync(
        join(root, 'vite.config.mjs'),
        "process.stderr.write('loading config\\n')\nawait new Promise((done) => setTimeout(done, 10_000))\nexport default {}\n",
      )
      const dev = run(['--root', root, '--port', '0'])
      await vi.waitFor(() => expect(dev.out().stderr).toBe('loading config\n'), { timeout: 10_000, interval: 20 })
      dev.child.kill(signal)
      expect(await dev.exited).toEqual({ code: 0, signal: null })
      expect(dev.out().stdout).toBe('')
    }, 30_000)
  }

  const pause = (ms: number) => new Promise((done) => setTimeout(done, ms))

  // `weft dev` holds the other end of the pipe for its whole life, so the
  // pipe ends when it dies, however it dies.
  it('with --exit-with-parent, stays up while its standard input is open and exits 0 when it ends', async () => {
    const dev = run(['--root', site('dev-site'), '--port', '0', '--exit-with-parent'], 'pipe')
    const line = await dev.line
    const url = line.slice('weft: dev server at '.length, -1)
    await pause(2000)
    expect(text((await get(url)).body)).toContain('Hello from dev')
    expect(dev.child.exitCode).toBe(null)
    const ended = Date.now()
    dev.child.stdin!.end()
    expect(await dev.exited).toEqual({ code: 0, signal: null })
    expect(Date.now() - ended).toBeLessThan(3000)
    await expect(fetch(url)).rejects.toThrow()
    expect(dev.out().stderr).toBe('')
  }, 30_000)

  it('with --exit-with-parent, exits 0 when its standard input ends while it starts', async () => {
    const root = site('dev-site')
    writeFileSync(join(root, 'vite.config.mjs'), "process.stderr.write('loading config\\n')\nawait new Promise((done) => setTimeout(done, 10_000))\nexport default {}\n")
    const dev = run(['--root', root, '--port', '0', '--exit-with-parent'], 'pipe')
    await vi.waitFor(() => expect(dev.out().stderr).toBe('loading config\n'), { timeout: 10_000, interval: 20 })
    dev.child.stdin!.end()
    expect(await dev.exited).toEqual({ code: 0, signal: null })
    expect(dev.out().stdout).toBe('')
  }, 30_000)

  // Vite closes its server when standard input ends, but only once something
  // reads it: without the flag nothing does, so neither an open pipe nor its
  // end stops the server.
  it('without the flag, stays up with a pipe as its standard input, open or ended', async () => {
    const dev = run(['--root', site('dev-site'), '--port', '0'], 'pipe')
    const line = await dev.line
    const url = line.slice('weft: dev server at '.length, -1)
    await pause(1500)
    expect(text((await get(url)).body)).toContain('Hello from dev')
    dev.child.stdin!.end()
    await pause(1500)
    expect(dev.child.exitCode).toBe(null)
    expect(text((await get(url)).body)).toContain('Hello from dev')
    dev.child.kill('SIGTERM')
    expect(await dev.exited).toEqual({ code: 0, signal: null })
  }, 30_000)

  it('prints the port sentence alone when the port is taken', async () => {
    const taken: Server = createHTTPServer()
    await new Promise<void>((done) => taken.listen(0, 'localhost', done))
    closers.push(() => new Promise((done) => taken.close(done)))
    const port = (taken.address() as AddressInfo).port
    const dev = run(['--root', site('dev-site'), '--port', String(port)])
    expect(await dev.exited).toEqual({ code: 1, signal: null })
    expect(dev.out()).toEqual({ stdout: '', stderr: `weft dev: port ${port} is in use; stop what is using it or pass --port.\n` })
  }, 30_000)

  it('refuses wrong usage with status 2', async () => {
    const dev = run(['--port', '0'])
    expect(await dev.exited).toEqual({ code: 2, signal: null })
    expect(dev.out().stderr).toMatch(/^weft dev: --root is required\.\nusage: weft-dev --root <site>/)
    const bad = run(['--root', '.', '--port', 'eighty'])
    expect(await bad.exited).toEqual({ code: 2, signal: null })
    expect(bad.out().stderr).toMatch(/^weft dev: --port "eighty" is not a port number; give one from 0 to 65535\.\n/)
    const unknown = run(['--root', '.', '--watch'])
    expect(await unknown.exited).toEqual({ code: 2, signal: null })
    expect(unknown.out().stderr).toMatch(/^weft dev: Unknown option '--watch'\.\nusage: /)
  })
})
