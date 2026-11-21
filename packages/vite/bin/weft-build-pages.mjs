#!/usr/bin/env node
// Build a Weft site's pages into a dist directory. The weft CLI has no build
// command yet: a site runs this from its package.json (`weft-build-pages --root . --out dist`).
// It imports the package's TypeScript source, so Node must strip types,
// which it does by default from Node 22.18.
import { parseArgs } from 'node:util'

if (!process.features.typescript) {
  process.stderr.write(`weft-build-pages: Node ${process.version} does not strip TypeScript types; use Node 22.18 or later.\n`)
  process.exit(1)
}
const { buildSite } = await import('../src/build.ts')

const usage =
  'usage: weft-build-pages --root <site> --out <dist> [--app-prefix /app [--app-dir <app build>]] [--theme neutral|lisa|weft]'

function fail(message) {
  process.stderr.write(`weft-build-pages: ${message}\n${usage}\n`)
  process.exit(1)
}

let values
try {
  ;({ values } = parseArgs({
    options: {
      root: { type: 'string' },
      out: { type: 'string' },
      'app-prefix': { type: 'string' },
      'app-dir': { type: 'string' },
      theme: { type: 'string' },
    },
  }))
} catch (e) {
  fail(e.message)
}
if (!values.root) fail('--root is required.')
if (!values.out) fail('--out is required.')

try {
  const manifest = await buildSite({
    root: values.root,
    outDir: values.out,
    appPrefix: values['app-prefix'],
    appDir: values['app-dir'],
    theme: values.theme,
  })
  const pages = Object.keys(manifest.files).filter((f) => /^\/pages\/.*\/index\.html$/.test(f)).length
  process.stdout.write(`weft: built ${pages} ${pages === 1 ? 'page' : 'pages'} into ${values.out}\n`)
} catch (e) {
  process.stderr.write(`weft-build-pages: ${e instanceof Error ? e.message : String(e)}\n`)
  process.exit(1)
}
