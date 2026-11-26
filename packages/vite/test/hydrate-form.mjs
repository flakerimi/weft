// Open one built page with a Form island in happy-dom, run its pre-paint
// script as the browser would before the body, and report, as JSON on stdout,
// what a submit does before the island hydrates and after.
// usage: node hydrate-form.mjs <dist> <page html, relative to dist>
//
// It runs in a process of its own, as hydrate-page.mjs does: the built chunks
// are plain browser modules, loaded with Node's own import.
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { Window } from 'happy-dom'

const [dist, page] = process.argv.slice(2)
const html = readFileSync(join(dist, page), 'utf8')
const window = new Window({ url: 'https://weft.test/login' })
for (const key of Object.getOwnPropertyNames(window)) {
  if (!(key in globalThis)) globalThis[key] = window[key]
}
globalThis.window = window
globalThis.document = window.document

// The <head>'s inline pre-paint script runs before the body is parsed.
const head = html.slice(0, html.indexOf('</head>'))
for (const [, code] of head.matchAll(/<script data-weft-prepaint>([\s\S]*?)<\/script>/g)) new Function(code)()
document.body.innerHTML = /<body>([\s\S]*)<\/body>/.exec(html)[1]

const island = document.querySelector('weft-island')
const form = island.querySelector('form')
const password = form.querySelector('input[type="password"]')
const button = form.querySelector('button[type="submit"]')
const output = form.querySelector('output')

/** A submit as a press on the button makes it, and whether it was stopped (as a later page listener sees it). */
const submit = () => {
  let seen
  const look = (e) => (seen = e.defaultPrevented)
  window.addEventListener('submit', look)
  const event = new window.SubmitEvent('submit', { bubbles: true, cancelable: true, submitter: button })
  form.dispatchEvent(event)
  window.removeEventListener('submit', look)
  return { prevented: event.defaultPrevented, seen }
}

// A password manager fills the field while the script loads.
password.value = 'hunter2'
const before = {
  method: form.getAttribute('method'),
  ...submit(),
  disabled: password.matches(':disabled') || form.querySelector('fieldset').disabled || button.disabled,
  password: password.value,
}

const settle = async (done) => {
  for (let i = 0; i < 200 && !done(); i++) await new Promise((r) => setTimeout(r, 10))
}
const [script, entry] = /<script type="module" src="\/([^"]+)"[^>]*>/.exec(html)
const named = document.createElement('template')
named.innerHTML = script.replace('type="module"', 'type="text/plain"') + '</script>'
document.head.append(named.content)
await import(pathToFileURL(join(dist, entry)).href)
await settle(() => !island.querySelector('script'))
const hydrated = !island.querySelector('script')

const after = submit()
await settle(() => output.textContent !== '0')
// Long enough for a second run of the handler to show.
await new Promise((r) => setTimeout(r, 50))

process.stdout.write(
  JSON.stringify({ before, hydrated, after: { ...after, sent: output.textContent, sameForm: island.querySelector('form') === form } }) + '\n',
)
await window.happyDOM.close()
