// Hydrate one built page in happy-dom and report what its first island's
// button looks like before and after a click, as JSON on stdout.
// usage: node hydrate-page.mjs <dist> <page html, relative to dist>
//
// It runs in a process of its own: the built chunks are plain browser modules,
// loaded with Node's own import, which a test file cannot do from inside Vitest.
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { Window } from 'happy-dom'

const [dist, page] = process.argv.slice(2)
const html = readFileSync(join(dist, page), 'utf8')
const window = new Window({ url: 'https://weft.test/' })
// What a browser module sees as globals: the window's, where Node has none.
for (const key of Object.getOwnPropertyNames(window)) {
  if (!(key in globalThis)) globalThis[key] = window[key]
}
globalThis.window = window
globalThis.document = window.document

document.body.innerHTML = /<body>([\s\S]*)<\/body>/.exec(html)[1]
const island = document.querySelector('weft-island')
const button = island.querySelector('button')
const before = button.className

const settle = async (done) => {
  for (let i = 0; i < 200 && !done(); i++) await new Promise((r) => setTimeout(r, 10))
}
const [script, entry] = /<script type="module" src="\/([^"]+)"[^>]*>/.exec(html)
// The page's module script names the runtime the loader imports; a copy that
// does not run stands in for it, and the loader is imported as the browser would.
const named = document.createElement('template')
named.innerHTML = script.replace('type="module"', 'type="text/plain"') + '</script>'
document.head.append(named.content)
await import(pathToFileURL(join(dist, entry)).href)
await settle(() => !island.querySelector('script'))
const hydrated = !island.querySelector('script')
button.click()
await settle(() => button.textContent === 'n is 1')

process.stdout.write(
  JSON.stringify({
    before,
    after: button.className,
    hydrated,
    text: button.textContent,
    sameNode: island.querySelector('button') === button,
  }) + '\n',
)
await window.happyDOM.close()
