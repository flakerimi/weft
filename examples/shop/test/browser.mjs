// Drives the running shop in a real browser (headless Chrome or Chromium,
// over the DevTools protocol), as a person would, and fails on the first step
// that does not show what it should. cmd/weft/shop_test.go runs it against the
// built binary when a browser is installed.
//
// Usage: node test/browser.mjs <browser> <base url> <setup link> <out folder>
// The out folder gets the browser's profile and a screenshot of each step.
import { spawn } from 'node:child_process'
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const [browserPath, base, setupLink, out] = process.argv.slice(2)
if (!out) {
  console.error('usage: node test/browser.mjs <browser> <base url> <setup link> <out folder>')
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
  try { page = (await (await fetch(`http://127.0.0.1:${port}/json/list`)).json()).find((t) => t.type === 'page') } catch {}
  if (!page && i >= 20 && i % 20 === 0) {
    try { await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: 'PUT' }) } catch {}
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
      return [...root.querySelectorAll('button')].find((b) => b.textContent.trim() === label || (b.getAttribute('aria-label') || '').trim() === label)
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

step('the setup link: the token leaves the address once read')
await go(setupLink)
await waitFor('the setup form, live', `__t.live('SetupForm') && __t.byLabel('Password')`)
if (await evaluate(`location.hash !== '' || location.href.includes('token')`)) await fail('the setup token is still in the address')
await shot('setup')
// The link pasted again into the same tab changes only the fragment: it is
// read again, and taken out of the address again.
await evaluate(`(location.hash = new URL(${JSON.stringify(setupLink)}).hash, true)`)
await waitFor('the token read again and gone from the address', `location.hash === '' && !location.href.includes('token') && !!__t.byLabel('Password')`)
await type(`__t.byLabel('Password')`, password)
await click(`__t.button('Set up the owner')`)
await waitFor('/products after setup', `location.pathname === '/products' && __t.text().includes('No products')`)
await waitFor('the session bar', `__t.text().includes('owner@example.com')`)

step('a next that leads off the site goes to the front page, signed in or not')
for (const bad of ['/.//evil.example', '/a/..//evil.example', '/%2e//evil.example', '//evil.example', 'https://evil.example/']) {
  await go(`${base}/login?next=${encodeURIComponent(bad)}`)
  await waitFor(`the front page after next=${bad}`, `location.origin === ${JSON.stringify(origin)} && location.pathname === '/'`)
  await sleep(300)
  if ((await evaluate('location.origin')) !== origin) await fail(`next=${bad} left the site`)
}

async function create(name, price, status) {
  await click(`__t.button('New product')`)
  await waitFor('the dialog', `__t.dialog() && __t.byLabel('Name', __t.dialog())`)
  await sleep(200)
  await type(`__t.byLabel('Name', __t.dialog())`, name)
  await type(`__t.byLabel('Price', __t.dialog())`, String(price))
  await click(`[...__t.dialog().querySelectorAll('label')].find((l) => l.textContent.trim() === '${status}')`)
}
step('create Lamp, 30, live')
await go(`${base}/products`)
await waitFor('the table', `__t.live('ProductsTable') && __t.text().includes('No products')`)
await create('Lamp', 30, 'Live')
await click(`__t.button('Create', __t.dialog())`)
await waitFor('Lamp in the table', `!__t.dialog() && [...document.querySelectorAll('a')].some((a) => a.textContent.trim() === 'Lamp') && __t.text().includes('30.00')`)
await shot('products-lamp')

step('a second Lamp is refused on the name field')
await create('Lamp', 1, 'Draft')
await click(`__t.button('Create', __t.dialog())`)
await waitFor('the field error', `__t.dialog() && __t.dialog().innerText.includes('This is already taken.')`)
await click(`__t.button('Cancel', __t.dialog())`)
await waitFor('the dialog closed', `!__t.dialog()`)

step('the browser checks the schema before sending')
await click(`__t.button('New product')`)
await waitFor('the dialog', `__t.dialog() && __t.byLabel('Name', __t.dialog())`)
await sleep(200)
await click(`__t.button('Create', __t.dialog())`)
await waitFor('required errors', `__t.dialog().innerText.includes('This is required.')`)
await click(`__t.button('Cancel', __t.dialog())`)
await waitFor('the dialog closed', `!__t.dialog()`)

step('edit Lamp: price 42')
await click(`__t.button('Edit')`)
await waitFor('the edit dialog', `__t.dialog() && __t.dialog().innerText.includes('Edit Lamp')`)
await sleep(200)
await type(`__t.byLabel('Price', __t.dialog())`, '42')
await click(`__t.button('Save', __t.dialog())`)
await waitFor('the new price', `!__t.dialog() && __t.text().includes('42.00') && !__t.text().includes('30.00')`)

step('edit Lamp: a sale of 35, then emptied, which clears it, and an emptied price, which the form asks for')
async function editLamp() {
  await click(`__t.button('Edit')`)
  await waitFor('the edit dialog', `__t.dialog() && __t.dialog().innerText.includes('Edit Lamp')`)
  await sleep(200)
}
async function empty(label) {
  await evaluate(`(() => { const el = __t.byLabel('${label}', __t.dialog()); el.focus(); el.value = ''; el.dispatchEvent(new Event('input', { bubbles: true })); el.dispatchEvent(new Event('change', { bubbles: true })); el.blur(); return true })()`)
}
await editLamp()
await type(`__t.byLabel('Sale', __t.dialog())`, '35')
await click(`__t.button('Save', __t.dialog())`)
await waitFor('the sale', `!__t.dialog() && __t.text().includes('35.00')`)
await editLamp()
await empty('Sale')
await click(`__t.button('Save', __t.dialog())`)
await waitFor('the sale gone', `!__t.dialog() && __t.text().includes('42.00') && !__t.text().includes('35.00')`)
await go(`${base}/products`)
await waitFor('the sale gone after a reload', `__t.live('ProductsTable') && __t.text().includes('42.00') && !__t.text().includes('35.00')`)
const lamp = await evaluate(`fetch('/api/products?q=Lamp').then((r) => r.json()).then((l) => l.items[0])`)
if (lamp.sale !== null || lamp.price !== 42) await fail(`Lamp is ${JSON.stringify(lamp)}, want price 42 and sale null`)
await editLamp()
await empty('Price')
await click(`__t.button('Save', __t.dialog())`)
await waitFor("the form's own message", `__t.dialog() && __t.dialog().innerText.includes('This is required.')`)
await click(`__t.button('Cancel', __t.dialog())`)
await waitFor('the dialog closed, 42 as it was', `!__t.dialog() && __t.text().includes('42.00')`)

step('search and filter')
await type(`__t.byLabel('Search')`, 'nothing like it')
await waitFor('no rows for the search', `__t.text().includes('No products')`)
await type(`__t.byLabel('Search')`, 'lam')
await waitFor('Lamp again', `__t.text().includes('42.00')`)
await click(`[...document.querySelectorAll('label')].find((l) => l.textContent.trim() === 'Draft')`)
await waitFor('no draft products', `__t.text().includes('No products')`)
await click(`[...document.querySelectorAll('label')].find((l) => l.textContent.trim() === 'Live')`)
await waitFor('Lamp is live', `__t.text().includes('42.00')`)

step('its page, and pages for ids that are not')
const href = await evaluate(`[...document.querySelectorAll('a')].find((a) => a.textContent.trim() === 'Lamp').getAttribute('href')`)
const id = href.split('/').pop()
await go(base + href)
await waitFor('the product', `__t.text().includes('Lamp') && __t.text().includes('42.00') && __t.text().includes('live')`)
await shot('product-page')
for (const path of ['/products/999', `/products/0x${Number(id).toString(16)}`, `/products/${id}.0`, `/products/00${id}`]) {
  await go(base + path)
  await waitFor(`not found at ${path}`, `__t.text().includes('No product has that id.')`)
}

step('delete Lamp, after confirming')
await go(`${base}/products`)
await waitFor('the table', `__t.live('ProductsTable') && __t.text().includes('42.00')`)
await click(`__t.button('Delete')`)
await waitFor('the confirmation', `__t.dialog() && __t.dialog().innerText.includes('Delete Lamp?')`)
await click(`__t.button('Delete', __t.dialog())`)
await waitFor('the empty table', `!__t.dialog() && __t.text().includes('No products')`)
await waitFor('focus on New product after deleting last row', `document.activeElement === __t.button('New product')`)

step('a session that ends while the list is open sends the visitor to sign in, and back')
for (let i = 1; i <= 11; i++) {
  const made = await evaluate(`fetch('/api/products', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: 'P${String(i).padStart(2, '0')}', price: ${i}, status: 'live' }) }).then((r) => r.status)`)
  if (made !== 201) await fail(`making P${i} answered ${made}`)
}
await go(`${base}/products`)
await waitFor('the first page', `__t.live('ProductsTable') && __t.text().includes('P10') && !__t.text().includes('P11')`)
await send('Network.clearBrowserCookies')
await click(`document.querySelector('.weft-table [aria-label="Page 2"]')`)
await waitFor('sent to /login with next', `location.pathname === '/login' && location.search === '?next=%2Fproducts'`)
await waitFor('the login form, live', `__t.live('LoginForm')`)
await type(`__t.byLabel('Email')`, 'owner@example.com')
await type(`__t.byLabel('Password')`, password)
await click(`__t.button('Sign in')`)
await waitFor('back on /products', `location.pathname === '/products' && __t.live('ProductsTable') && __t.text().includes('P10')`)

step('delete the last row of the last page: the page before it shows')
await click(`document.querySelector('.weft-table [aria-label="Page 2"]')`)
await waitFor('the last page', `__t.text().includes('P11') && !__t.text().includes('P10')`)
await click(`__t.button('Delete')`)
await waitFor('the confirmation', `__t.dialog() && __t.dialog().innerText.includes('Delete P11?')`)
await click(`__t.button('Delete', __t.dialog())`)
await waitFor('the first page again', `!__t.dialog() && __t.text().includes('P01') && __t.text().includes('P10') && !__t.text().includes('P11') && !__t.text().includes('No products')`)
await waitFor('focus on a row Delete button after delete', `document.activeElement?.getAttribute('data-weft-row-action') === 'delete'`)
await shot('last-row-deleted')

step('signing out with the server out of reach says so, and stays signed in')
unreachableUrls.add('/api/auth/logout')
await send('Fetch.enable', { patterns: [{ urlPattern: '*/api/auth/logout*', requestStage: 'Request' }] })
await click(`__t.button('Sign out')`)
await waitFor('the sentence', `__t.text().includes('The server could not be reached; check the connection and try again.')`)
if ((await evaluate('location.pathname')) !== '/products' || !(await evaluate(`__t.text().includes('owner@example.com')`))) await fail('signing out with no server left the page or the session')
await send('Fetch.disable')
unreachableUrls.clear()

step('sign out')
await click(`__t.button('Sign out')`)
await waitFor('/login', `location.pathname === '/login'`)
await go(`${base}/products`)
await waitFor('sent to /login with next', `location.pathname === '/login' && location.search === '?next=%2Fproducts'`)

step('a submit before the sign-in island is live sends nothing')
// Hold every script, so the page is the server's HTML and its inline guard
// only; with the cache off, since a script from the cache is never asked for.
await send('Network.setCacheDisabled', { cacheDisabled: true })
await send('Fetch.enable', { patterns: [{ urlPattern: '*', resourceType: 'Script', requestStage: 'Request' }] })
await go(`${base}/login`, 'interactive')
await waitFor('the server-rendered form', `__t.byLabel('Email') && __t.byLabel('Password') && !__t.live('LoginForm')`)
await type(`__t.byLabel('Email')`, 'owner@example.com')
await type(`__t.byLabel('Password')`, password)
const before = requests.length
await click(`__t.button('Sign in')`)
await sleep(1000)
const sent = requests.slice(before).filter((r) => r.type !== 'Script')
if (sent.length > 0) await fail(`a submit before hydration sent ${JSON.stringify(sent)}`)
if ((await evaluate('location.href')) !== `${base}/login`) await fail('a submit before hydration changed the address')
if (await evaluate('__t.live("LoginForm")')) await fail('the island was live though its scripts were held')
await shot('login-before-hydration')
for (const requestId of held.splice(0)) await send('Fetch.continueRequest', { requestId })
await send('Fetch.disable')
await send('Network.setCacheDisabled', { cacheDisabled: false })
await waitFor('the sign-in island, live', `__t.live('LoginForm')`)
if ((await evaluate(`__t.byLabel('Password').value`)) !== password) await fail('what was typed before hydration was lost')
if (requests.some((r) => r.url.includes('password') || (r.type === 'Document' && r.method !== 'GET'))) await fail('the password went into an address or a page request')
await click(`__t.button('Sign in')`)
await waitFor('signed in, on the front page', `location.pathname === '/' && __t.text().includes('owner@example.com')`)

step('sign in again from /login?next=, a wrong password first')
await click(`__t.button('Sign out')`)
await waitFor('/login', `location.pathname === '/login'`)
await go(`${base}/login?next=%2Fproducts`)
await waitFor('the login form, live', `__t.live('LoginForm')`)
await type(`__t.byLabel('Email')`, 'owner@example.com')
await type(`__t.byLabel('Password')`, 'not the password')
await click(`__t.button('Sign in')`)
await waitFor('the refusal', `__t.text().includes('That email and password do not match.')`)
await type(`__t.byLabel('Password')`, password)
await click(`__t.button('Sign in')`)
await waitFor('back on /products', `location.pathname === '/products' && __t.text().includes('owner@example.com')`)
await shot('signed-in-again')

if (logs.length > 0) await fail(`the pages logged errors or warnings:\n${logs.join('\n')}`)
console.log('done')
quit()
process.exit(0)
