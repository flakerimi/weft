#!/usr/bin/env node
// The page dev server, which `weft dev` starts beside the app's Go API:
// `weft-dev --root web --port 5173 --api http://127.0.0.1:8081`. It prints
// `weft: dev server at <url>` once it listens, and stops on SIGTERM or SIGINT.
// With --exit-with-parent it also stops when its standard input ends: `weft
// dev` gives it a pipe it holds open for its whole life, which ends when it
// dies, however it dies.
// It imports the package's TypeScript source, so Node must strip types,
// which it does by default from Node 22.18.
import { parseArgs } from 'node:util'

const usage = 'usage: weft-dev --root <site> [--port n] [--host h] [--api <url>] [--theme neutral|lisa|weft|synthwave|editor|sandstorm] [--exit-with-parent]'

function fail(message, code = 1) {
  process.stderr.write(`weft dev: ${message}\n${code === 2 ? usage + '\n' : ''}`)
  process.exit(code)
}

if (!process.features.typescript) fail(`Node ${process.version} does not strip TypeScript types; use Node 22.18 or later.`)

let values
try {
  ;({ values } = parseArgs({
    options: {
      root: { type: 'string' },
      port: { type: 'string' },
      host: { type: 'string' },
      api: { type: 'string' },
      theme: { type: 'string' },
      'exit-with-parent': { type: 'boolean' },
    },
  }))
} catch (e) {
  // Node's own message, to its first sentence: what is wrong.
  fail(`${e.message.split(/\.\s/)[0].replace(/\.$/, '')}.`, 2)
}
if (!values.root) fail('--root is required.', 2)
let port
if (values.port !== undefined) {
  port = /^\d{1,5}$/.test(values.port) ? Number(values.port) : NaN
  if (!(port <= 65535)) fail(`--port "${values.port}" is not a port number; give one from 0 to 65535.`, 2)
}

// Before anything starts, so a signal while it starts, such as a Ctrl-C to
// `weft dev`, is a clean stop too.
let server
let stopping = false
const stop = () => {
  if (stopping) return
  stopping = true
  // Stopped as asked: success, also for Vite's own SIGTERM handler, which exits once it has closed.
  process.exitCode = 0
  // Nothing listens yet: there is nothing to close.
  if (!server) process.exit(0)
  // However the close goes, the process ends soon.
  setTimeout(() => process.exit(0), 2000).unref()
  server.close().then(
    () => process.exit(0),
    () => process.exit(0),
  )
}
process.on('SIGTERM', stop)
process.on('SIGINT', stop)
// Reading standard input is what lets its end be seen; without the flag
// nothing reads it, so neither does Vite's own handler of its end fire.
if (values['exit-with-parent']) {
  process.stdin.on('end', stop)
  process.stdin.on('error', stop)
  process.stdin.resume()
}

const { devSite } = await import('../src/dev.ts')
try {
  server = await devSite({ root: values.root, port, host: values.host, api: values.api, theme: values.theme })
} catch (e) {
  const message = e instanceof Error ? e.message : String(e)
  // The server's sentences already say who they are from.
  process.stderr.write(`${message.startsWith('weft dev: ') ? message : `weft dev: ${message}`}\n`)
  process.exit(1)
}
process.stdout.write(`weft: dev server at ${server.url}\n`)
