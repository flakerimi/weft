// A runtime page as the pages guide shows it: an island that reads the
// address with useParams() once it is mounted in the browser.
import { execFile } from 'node:child_process'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { afterAll, beforeAll, expect, it } from 'vitest'

const run = promisify(execFile)
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))
let temp: string
let out: string

beforeAll(async () => {
  temp = mkdtempSync(join(tmpdir(), 'weft-runtime-hydrate-'))
  out = join(temp, 'dist')
  await run(process.execPath, [here('../bin/weft-build-pages.mjs'), '--root', here('./fixtures/runtime-pages'), '--out', out], {
    env: { ...process.env, NODE_ENV: undefined },
  })
}, 60_000)

afterAll(() => {
  rmSync(temp, { recursive: true, force: true })
})

it('an island on a runtime page reads the params of the address it was opened at', async () => {
  for (const [address, want] of [
    ['/products/42', 'Product 42'],
    ['/products/caf%C3%A9', 'Product café'],
  ]) {
    const { stdout } = await run(process.execPath, [here('./hydrate-params.mjs'), out, 'pages/products/[id]/index.html', address!, 'Product'])
    // The shell has no id; the browser fills it in.
    expect(JSON.parse(stdout), address).toEqual({ before: 'Product ', after: want, hydrated: true })
  }
})
