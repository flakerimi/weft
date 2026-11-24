// The page dev server, in a browser: an island drawn on the server hydrates
// from Vite's own modules (unbundled source, `vue` pre-bundled once), in
// place, and responds; the console has no error and no warning, so no
// hydration mismatch; one copy of Vue is loaded; and the island's styles,
// already in the page's <style>, are not applied a second time.
//
// It needs headless Chrome, driven over the DevTools protocol on a pipe
// (--remote-debugging-pipe), as site-src/browser_test.go drives it: Chrome
// is $CHROME_PATH, or the first found where it is usually installed. Without
// one the test is skipped and says so, except in CI ($CI set), where it
// fails: a browser test that never runs proves nothing.
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, mkdtempSync, rmSync, symlinkSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import type { Readable, Writable } from 'node:stream'
import { fileURLToPath } from 'node:url'
import { afterEach, describe, expect, it } from 'vitest'
import { devSite } from '../src/dev.ts'

const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))

function chromePath(): string {
  if (process.env.CHROME_PATH) return process.env.CHROME_PATH
  for (const p of ['/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', '/Applications/Chromium.app/Contents/MacOS/Chromium']) {
    if (existsSync(p)) return p
  }
  for (const name of ['google-chrome', 'google-chrome-stable', 'chromium', 'chromium-browser']) {
    const found = spawnSync('which', [name], { encoding: 'utf8' })
    if (found.status === 0 && found.stdout.trim()) return found.stdout.trim()
  }
  return ''
}

const chrome = chromePath()

interface Message {
  id?: number
  method?: string
  sessionId?: string
  params?: Record<string, unknown>
  result?: Record<string, unknown>
  error?: { message: string }
}

/** One headless Chrome with one page. */
class Browser {
  private next = 0
  private buffer = ''
  private waiting = new Map<number, { done: (r: Record<string, unknown>) => void; fail: (e: Error) => void }>()
  readonly events: Message[] = []
  session = ''

  constructor(private child: ChildProcess) {
    const out = child.stdio[4] as Readable
    out.setEncoding('utf8')
    out.on('data', (chunk: string) => {
      this.buffer += chunk
      for (let end = this.buffer.indexOf('\0'); end !== -1; end = this.buffer.indexOf('\0')) {
        const m = JSON.parse(this.buffer.slice(0, end)) as Message
        this.buffer = this.buffer.slice(end + 1)
        const call = m.id === undefined ? undefined : this.waiting.get(m.id)
        if (call) {
          this.waiting.delete(m.id!)
          if (m.error) call.fail(new Error(m.error.message))
          else call.done(m.result ?? {})
        } else if (m.method) this.events.push(m)
      }
    })
  }

  static async start(userDir: string): Promise<Browser> {
    const args = ['--headless=new', '--remote-debugging-pipe', '--no-first-run', '--no-default-browser-check', '--hide-scrollbars', '--disable-gpu', '--disable-extensions', `--user-data-dir=${userDir}`]
    if (process.getuid?.() === 0) args.push('--no-sandbox')
    // Chrome reads the protocol from fd 3 and writes it to fd 4.
    const child = spawn(chrome, args, { stdio: ['ignore', 'ignore', 'ignore', 'pipe', 'pipe'] })
    const b = new Browser(child)
    const { targetId } = (await b.call('', 'Target.createTarget', { url: 'about:blank' })) as { targetId: string }
    const { sessionId } = (await b.call('', 'Target.attachToTarget', { targetId, flatten: true })) as { sessionId: string }
    b.session = sessionId
    for (const domain of ['Page', 'Runtime', 'Log', 'Network']) await b.call(b.session, `${domain}.enable`)
    return b
  }

  call(session: string, method: string, params?: Record<string, unknown>): Promise<Record<string, unknown>> {
    const id = ++this.next
    ;(this.child.stdio[3] as Writable).write(JSON.stringify({ id, method, params, ...(session ? { sessionId: session } : {}) }) + '\0')
    return new Promise((done, fail) => this.waiting.set(id, { done, fail }))
  }

  /** The expression's value in the page, a promise awaited. */
  async eval<T>(expression: string): Promise<T> {
    const r = (await this.call(this.session, 'Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })) as {
      result: { value: T }
      exceptionDetails?: { text: string; exception?: { description?: string } }
    }
    if (r.exceptionDetails) throw new Error(`${expression} threw: ${r.exceptionDetails.exception?.description ?? r.exceptionDetails.text}`)
    return r.result.value
  }

  /** Wait for the page to send the event. */
  async wait(method: string, ms = 20_000) {
    const end = Date.now() + ms
    while (!this.events.some((e) => e.method === method && e.sessionId === this.session)) {
      if (Date.now() > end) throw new Error(`no ${method} within ${ms} ms`)
      await new Promise((r) => setTimeout(r, 20))
    }
  }

  /** Wait, in the page, for the expression to be true. */
  async until(expression: string, ms = 10_000) {
    const ok = await this.eval<boolean>(`new Promise((done) => {
      const end = Date.now() + ${ms}
      const check = () => (${expression}) ? done(true) : Date.now() > end ? done(false) : setTimeout(check, 20)
      check()
    })`)
    if (!ok) throw new Error(`never became true: ${expression}`)
  }

  async close() {
    const exited = new Promise((done) => this.child.once('exit', done))
    ;(this.child.stdio[3] as Writable).end()
    const timer = setTimeout(() => this.child.kill('SIGKILL'), 5000)
    if (this.child.exitCode === null && this.child.signalCode === null) await exited
    clearTimeout(timer)
  }

  /** What the console and the browser's log said at a level, and what the page threw. */
  said(levels: string[]): string[] {
    const out: string[] = []
    for (const e of this.events) {
      const p = e.params as Record<string, any>
      if (e.method === 'Runtime.consoleAPICalled' && levels.includes(p.type)) {
        out.push(`console.${p.type}: ${p.args.map((a: any) => a.value ?? a.description ?? '').join(' ')}`)
      } else if (e.method === 'Runtime.exceptionThrown') {
        out.push(`thrown: ${p.exceptionDetails.exception?.description ?? p.exceptionDetails.text}`)
      } else if (e.method === 'Log.entryAdded' && levels.includes(p.entry.level === 'warning' ? 'warning' : p.entry.level)) {
        out.push(`log.${p.entry.level}: ${p.entry.text} ${p.entry.url ?? ''}`)
      }
    }
    return out
  }
}

const temps: string[] = []
const closers: (() => Promise<unknown>)[] = []
afterEach(async () => {
  for (const close of closers.splice(0).reverse()) await close()
  for (const dir of temps.splice(0)) rmSync(dir, { recursive: true, force: true, maxRetries: 10 })
})

describe('an island in the dev server, in a browser', { timeout: 60_000 }, () => {
  it.runIf(!chrome && process.env.CI)('needs Chrome in CI', () => {
    throw new Error('no Chrome found, and CI is set: install Chrome or set CHROME_PATH, so the browser tests run')
  })

  it.skipIf(!chrome)('hydrates in place, responds to a click, and logs no error or warning', async () => {
    // A created app's layout, in a folder with a space and a non-ASCII letter.
    const temp = mkdtempSync(join(tmpdir(), 'weft dev café-'))
    temps.push(temp)
    const web = join(temp, 'app/web')
    mkdirSync(join(temp, 'app'))
    cpSync(here('./fixtures/dev-site'), web, { recursive: true, filter: (src) => !src.includes('.weft-types') })
    symlinkSync(here('../node_modules'), join(temp, 'app/node_modules'))
    const dev = await devSite({ root: web, port: 0 })
    closers.push(() => dev.close())

    const browser = await Browser.start(join(temp, 'chrome'))
    closers.push(() => browser.close())
    // The server's button, kept as the parser inserts it, before any module runs.
    await browser.call(browser.session, 'Page.addScriptToEvaluateOnNewDocument', {
      source: `new MutationObserver((_, o) => {
        const b = document.querySelector('weft-island button')
        if (b) { window.__serverButton = b; o.disconnect() }
      }).observe(document, { childList: true, subtree: true })`,
    })
    await browser.call(browser.session, 'Page.navigate', { url: dev.url + 'counter' })
    await browser.wait('Page.loadEventFired')
    // Hydrated: the island's props script is taken out once it has mounted.
    await browser.until(`document.readyState === 'complete' && !document.querySelector('weft-island script')`)
    const before = await browser.eval<{ text: string; same: boolean; width: number }>(`(() => {
      const b = document.querySelector('weft-island button')
      return { text: b.textContent.trim(), same: b === window.__serverButton, width: b.getBoundingClientRect().width }
    })()`)
    expect(before).toMatchObject({ text: 'n is 2', same: true })

    await browser.eval(`document.querySelector('weft-island button').click()`)
    await browser.until(`document.querySelector('weft-island button').textContent.trim() === 'n is 3'`)
    expect(await browser.eval<boolean>(`document.querySelector('weft-island button') === window.__serverButton`)).toBe(true)

    // The page's CSS styles the island; the browser's own style modules are empty.
    const styles = await browser.eval<{ injected: string[]; minWidth: string; className: string }>(`(() => ({
      injected: [...document.querySelectorAll('style[data-vite-dev-id]')].map((s) => s.textContent.trim()).filter(Boolean),
      minWidth: getComputedStyle(document.querySelector('weft-island button')).minInlineSize,
      className: document.querySelector('weft-island button').className,
    }))()`)
    expect(styles.injected).toEqual([])
    expect(styles.className).toContain('weft-button')
    expect(styles.minWidth).not.toBe('auto')

    // One copy of Vue: the pre-bundled one, and no Vue package loaded from source.
    const loaded = await browser.eval<string[]>(`performance.getEntriesByType('resource').map((r) => r.name)`)
    const vue = loaded.filter((u) => /\/(vue|@vue\/[^/]+)(\/|\.js)/.test(decodeURIComponent(new URL(u).pathname)))
    expect(vue).toHaveLength(1)
    expect(new URL(vue[0]!).pathname).toMatch(/\/\.vite\/deps\/vue\.js$/)

    // Bar the favicon the site has none of, as on any server.
    expect(browser.said(['error', 'warning', 'assert']).filter((m) => !m.endsWith('/favicon.ico'))).toEqual([])
    // Loaded once: no reload, such as one for dependencies Vite found late.
    expect(browser.events.filter((e) => e.method === 'Page.frameNavigated')).toHaveLength(1)
  })
})
