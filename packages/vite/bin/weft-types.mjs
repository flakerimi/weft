#!/usr/bin/env node
// Write a site's .weft-types/components.d.ts, which tells TypeScript what each
// auto-imported Weft component is, without a build: `weft-types --root web`.
// `weft create` and `weft build` run it in an app, so `pnpm typecheck` checks
// the components' props before the first build. It prints nothing when it
// has written the file, or found it current.
// It imports the package's TypeScript source, so Node must strip types,
// which it does by default from Node 22.18.
import { parseArgs } from 'node:util'

const usage = 'usage is weft-types --root <site>.'

function fail(message, code = 1) {
  process.stderr.write(`weft-types: ${message}\n`)
  process.exit(code)
}

if (!process.features.typescript) fail(`Node ${process.version} does not strip TypeScript types; use Node 22.18 or later.`)

let values
try {
  ;({ values } = parseArgs({ options: { root: { type: 'string' } } }))
} catch (e) {
  fail(`${e.message.split(/\.\s/)[0].replace(/\.$/, '')}; ${usage}`, 2)
}
if (!values.root) fail(`--root is required; ${usage}`, 2)

const { writeComponentTypes } = await import('../src/auto-import.ts')
if (!writeComponentTypes(values.root)) fail(`@weft/ui cannot be found from ${values.root}; run pnpm install in the app and run the command again.`)
