// Open one built runtime page at an address in happy-dom, as warp.Serve sends
// its shell for any path its pattern matches, hydrate it, and report the text
// of its island with the given id, as JSON on stdout.
// usage: node hydrate-params.mjs <dist> <page html, relative to dist> <address path> <island id>
//
// It runs in a process of its own, as hydrate-page.mjs does: the built chunks
// are plain browser modules, loaded with Node's own import.
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { Window } from 'happy-dom'

const [dist, page, address, id] = process.argv.slice(2)
const html = readFileSync(join(dist, page), 'utf8')
const window = new Window({ url: `https://weft.test${address}` })
for (const key of Object.getOwnPropertyNames(window)) {
  if (!(key in globalThis)) globalThis[key] = window[key]
}
globalThis.window = window
globalThis.document = window.document

// The root's attributes, the pattern among them, then the body.
const root = /<html([^>]*)>/.exec(html)[1]
for (const [, name, value] of root.matchAll(/\s([\w-]+)="([^"]*)"/g)) document.documentElement.setAttribute(name, value)
document.body.innerHTML = /<body>([\s\S]*)<\/body>/.exec(html)[1]
const island = document.querySelector(`weft-island[data-id="${id}"]`)
// What the island renders, without the script that holds its props.
const text = () => [...island.children].filter((e) => e.tagName !== 'SCRIPT').map((e) => e.textContent).join('')
const before = text()

const settle = async (done) => {
  for (let i = 0; i < 200 && !done(); i++) await new Promise((r) => setTimeout(r, 10))
}
const [script, entry] = /<script type="module" src="\/([^"]+)"[^>]*>/.exec(html)
const named = document.createElement('template')
named.innerHTML = script.replace('type="module"', 'type="text/plain"') + '</script>'
document.head.append(named.content)
await import(pathToFileURL(join(dist, entry)).href)
await settle(() => !island.querySelector('script') && text() !== before)

process.stdout.write(JSON.stringify({ before, after: text(), hydrated: !island.querySelector('script') }) + '\n')
await window.happyDOM.close()
