// Drives an app weft create made, with a products resource weft scaffold
// wrote (weft scaffold product name:string:required:unique price:float:min=0
// sale:float:nullable status:enum=draft,live:required), in a real browser (headless Chrome or
// Chromium, over the DevTools protocol), as a person would: the setup link,
// the table, New product, the record's page, Edit and Delete. It fails on the
// first step that does not show what it should. cmd/weft/scaffold_test.go
// (TestScaffoldedPagesInBrowser) runs it against the built binary. The
// driver is examples/shop/test/browser.mjs's.
//
// Usage: node scaffold-browser.mjs <browser> <base url> <setup link> <out folder>
// The out folder gets the browser's profile and a screenshot of each step.
import { spawn } from 'node:child_process'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const [browserPath, base, setupLink, out] = process.argv.slice(2)
if (!out) {
  console.error('usage: node scaffold-browser.mjs <browser> <base url> <setup link> <out folder>')
  process.exit(2)
}
const origin = new URL(base).origin
const profile = join(out, 'profile')
mkdirSync(profile, { recursive: true })
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

// Port 0: the browser picks a free port and writes it to DevToolsActivePort.
const browser = spawn(browserPath, [
  '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`,
  '--no-first-run', '--no-default-browser-check', '--window-size=1200,900', 'about:blank',
], { stdio: 'ignore' })
const quit = () => browser.kill()
process.on('exit', quit)
// The test that runs this stops its process group on a timeout; the browser
// is in the group too, and goes with it.
process.on('SIGTERM', () => process.exit(1))

let port
for (let i = 0; i < 200 && !port; i++) {
  try { port = readFileSync(join(profile, 'DevToolsActivePort'), 'utf8').split('\n')[0] } catch { await sleep(50) }
}
// A slow machine (a CI runner) may not have opened its first tab by the time
// the debugging port answers, or open none, or drop the first ask: from two
// seconds on, ask for one every two seconds, for up to a minute.
let page
for (let i = 0; i < 600 && !page; i++) {
  try { page = (await (await fetch(`http://127.0.0.1:${port}/json/list`, { signal: AbortSignal.timeout(2000) })).json()).find((t) => t.type === 'page') } catch {}
  if (!page && i >= 20 && i % 20 === 0) {
    try { await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: 'PUT', signal: AbortSignal.timeout(2000) }) } catch {}
  }
  if (!page) await sleep(100)
}
if (!page) throw new Error(`the browser at ${browserPath} gave no page to drive within 60 s; check that it starts headless on this machine`)
const ws = new WebSocket(page.webSocketDebuggerUrl)
await new Promise((resolve, reject) => { ws.addEventListener('open', resolve); ws.addEventListener('error', reject) })

let seq = 0
const waiting = new Map()
const logs = []
const requests = []
const held = []
// Requests to answer as a server out of reach would: their URLs contain one of these.
const unreachableUrls = new Set()
ws.addEventListener('message', (ev) => {
  const m = JSON.parse(ev.data)
  if (m.id && waiting.has(m.id)) { waiting.get(m.id)(m); waiting.delete(m.id) }
  switch (m.method) {
    case 'Runtime.consoleAPICalled':
      if (m.params.type === 'error' || m.params.type === 'warning') logs.push(`console.${m.params.type}: ${m.params.args.map((a) => a.value ?? a.description).join(' ')}`)
      break
    case 'Runtime.exceptionThrown':
      logs.push(`exception: ${m.params.exceptionDetails.exception?.description ?? m.params.exceptionDetails.text}`)
      break
    case 'Network.requestWillBeSent':
      requests.push({ url: m.params.request.url, method: m.params.request.method, type: m.params.type, body: m.params.request.postData ?? '' })
      break
    case 'Fetch.requestPaused':
      if ([...unreachableUrls].some((u) => m.params.request.url.includes(u))) {
        ws.send(JSON.stringify({ id: ++seq, method: 'Fetch.failRequest', params: { requestId: m.params.requestId, errorReason: 'ConnectionRefused' } }))
      } else held.push(m.params.requestId)
      break
  }
})
const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++seq
  waiting.set(id, (m) => (m.error ? reject(new Error(`${method}: ${m.error.message}`)) : resolve(m.result)))
  ws.send(JSON.stringify({ id, method, params }))
})
await send('Runtime.enable')
await send('Page.enable')
await send('Network.enable')

const helpers = `
  window.__t = {
    text: () => document.body.innerText,
    byLabel(label, root = document) {
      const l = [...root.querySelectorAll('label')].find((x) => x.textContent.trim().replace(/\\s*\\*$/, '') === label)
      return l && (l.htmlFor ? document.getElementById(l.htmlFor) : l.querySelector('input'))
    },
    button(label, root = document) {
      return [...root.querySelectorAll('button')].find((b) => (b.getAttribute('aria-label') || b.textContent).trim() === label)
    },
    dialog: () => document.querySelector('[role=dialog][aria-modal=true], [role=alertdialog]'),
    // Hydration takes the island's props script out.
    live(id) {
      const el = document.querySelector('weft-island[data-id="' + id + '"]')
      return !!el && !el.querySelector(':scope > script[type="application/json"]')
    },
  }`
async function evaluate(expr) {
  const r = await send('Runtime.evaluate', { expression: `(async () => { ${helpers}; return (${expr}) })()`, awaitPromise: true, returnByValue: true })
  if (r.exceptionDetails) throw new Error(`${expr}: ${r.exceptionDetails.exception?.description}`)
  return r.result.value
}
let shots = 0
async function shot(name) {
  const { data } = await send('Page.captureScreenshot', { format: 'png' })
  writeFileSync(join(out, `${String(++shots).padStart(2, '0')}-${name}.png`), Buffer.from(data, 'base64'))
}
async function fail(why) {
  await shot('FAILED').catch(() => {})
  const where = await evaluate('location.href + "\\n" + document.body.innerText').catch(() => '?')
  throw new Error(`${why}\npage: ${where}\n${logs.join('\n')}`)
}
async function waitFor(what, expr, ms = 10000) {
  for (const end = Date.now() + ms; Date.now() < end; await sleep(100)) {
    try { if (await evaluate(expr)) return } catch {}
  }
  await fail(`waited ${ms} ms for ${what}: ${expr}`)
}
async function go(url, ready = 'complete') {
  await send('Page.navigate', { url })
  await waitFor(`${url} to load`, ready === 'complete' ? `document.readyState === 'complete'` : `document.readyState !== 'loading'`)
}
async function type(expr, text) {
  await evaluate(`(() => { const el = ${expr}; el.focus(); el.select?.(); return true })()`)
  await send('Input.insertText', { text })
  await evaluate(`(() => { const el = ${expr}; el.dispatchEvent(new Event('change', { bubbles: true })); el.blur(); return true })()`)
}
async function click(expr) {
  if (!(await evaluate(`(() => { const el = ${expr}; if (!el) return false; el.click(); return true })()`))) await fail(`nothing to click: ${expr}`)
}
const step = (s) => console.log(`- ${s}`)
const password = 'a password for the owner'
const dialogButton = (label) => `__t.button('${label}', __t.dialog())`
const submitIn = `__t.dialog().querySelector('button[type=submit]')`

step('the setup link sets the owner up and signs them in')
await go(setupLink)
await waitFor('the setup form, live', `__t.live('SetupForm') && __t.byLabel('Password')`)
await type(`__t.byLabel('Password')`, password)
await click(`__t.button('Set up the owner')`)
await waitFor('the front page after setup', `location.pathname === '/'`)

step('the products table, empty')
await go(`${base}/products`)
await waitFor('the table', `__t.live('ProductsTable') && __t.text().includes('No products')`)
await shot('products-empty')

step('New product: Lamp, 30, live')
await click(`__t.button('New product')`)
await waitFor('the dialog', `__t.dialog() && __t.byLabel('Name', __t.dialog())`)
await sleep(200)
await type(`__t.byLabel('Name', __t.dialog())`, 'Lamp')
await type(`__t.byLabel('Price', __t.dialog())`, '30')
await click(`[...__t.dialog().querySelectorAll('label')].find((l) => l.textContent.trim() === 'Live')`)
await click(submitIn)
await waitFor('Lamp in the table', `!__t.dialog() && [...document.querySelectorAll('a')].some((a) => a.textContent.trim() === 'Lamp')`)
await shot('products-lamp')

step('a second Lamp is refused on the name field')
await click(`__t.button('New product')`)
await waitFor('the dialog', `__t.dialog() && __t.byLabel('Name', __t.dialog())`)
await sleep(200)
await type(`__t.byLabel('Name', __t.dialog())`, 'Lamp')
await click(submitIn)
await waitFor('the field error', `__t.dialog() && __t.dialog().innerText.includes('This is already taken.')`)
await click(dialogButton('Cancel'))
await waitFor('the dialog closed', `!__t.dialog()`)

step('search finds Lamp, and not a word it lacks')
await type(`document.querySelector('input[type=search]')`, 'zzz')
await waitFor('nothing found', `__t.text().includes('No products')`)
await type(`document.querySelector('input[type=search]')`, 'lam')
await waitFor('Lamp found', `[...document.querySelectorAll('a')].some((a) => a.textContent.trim() === 'Lamp')`)

step('Edit Lamp: price 42')
// The row's actions are named for their row; their text stays Edit and Delete.
if (!(await evaluate(`__t.button('Edit Lamp')?.textContent.includes('Edit')`))) await fail('the Edit button is not named Edit Lamp, or does not say Edit')
await click(`__t.button('Edit Lamp')`)
await waitFor('the edit dialog', `__t.dialog() && __t.byLabel('Price', __t.dialog())`)
await sleep(200)
await type(`__t.byLabel('Price', __t.dialog())`, '42')
await click(submitIn)
await waitFor('the new price', `!__t.dialog() && __t.text().includes((42).toLocaleString()) && !__t.text().includes('30')`)

step("Lamp's page")
await click(`[...document.querySelectorAll('a')].find((a) => a.textContent.trim() === 'Lamp')`)
await waitFor('the record page', `/^\\/products\\/[1-9][0-9]*$/.test(location.pathname) && __t.live('ProductView') && __t.text().includes('Price: 42')`)
await shot('product-lamp')
await go(`${base}/products/0x2a`)
await waitFor('no record for an id that only looks like one', `__t.text().includes('No product has that id.')`)

step('Delete Lamp')
await go(`${base}/products`)
await waitFor('the table', `__t.live('ProductsTable') && [...document.querySelectorAll('a')].some((a) => a.textContent.trim() === 'Lamp')`)
await click(`__t.button('Delete Lamp')`)
await waitFor('the confirmation', `__t.dialog() && __t.dialog().innerText.includes('Lamp')`)
await click(dialogButton('Delete'))
await waitFor('the table empty again', `!__t.dialog() && __t.text().includes('No products')`)
await shot('products-empty-again')

if (logs.length > 0) await fail(`the console said:\n${logs.join('\n')}`)
console.log('ok')
process.exit(0)
