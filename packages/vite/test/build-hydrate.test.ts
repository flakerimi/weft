import { execFile } from 'node:child_process'
import { mkdtempSync, readdirSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

const run = promisify(execFile)
const here = (p: string) => fileURLToPath(new URL(p, import.meta.url))
let temp: string
let out: string

beforeAll(async () => {
  temp = mkdtempSync(join(tmpdir(), 'weft-hydrate-'))
  out = join(temp, 'dist')
  // Built as a site builds it: the bin, with no NODE_ENV.
  const bin = here('../bin/weft-build-pages.mjs')
  await run(process.execPath, [bin, '--root', here('./fixtures/modules-site'), '--out', out], {
    env: { ...process.env, NODE_ENV: undefined },
  })
}, 60_000)

afterAll(() => {
  rmSync(temp, { recursive: true, force: true })
})

describe('an island with CSS modules, built and hydrated', () => {
  it('keeps the class names the server rendered, which the inlined CSS defines', async () => {
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    const css = /<style>([\s\S]*?)<\/style>/.exec(html)![1]!
    const { stdout } = await run(process.execPath, [here('./hydrate-page.mjs'), out, 'pages/index/index.html'])
    const island = JSON.parse(stdout) as { before: string; after: string; hydrated: boolean; text: string; sameNode: boolean }

    // `<style module>` and an imported `*.module.css`, both hashed, both in the page's CSS.
    const classes = island.before.split(' ')
    expect(classes).toHaveLength(2)
    for (const name of classes) expect(css).toMatch(new RegExp(`\\.${name}\\s*\\{`))
    // Hydrated, not re-mounted: the server's button reacts, and the class the
    // click adds comes from the client's own mapping, the same as the server's.
    expect(island).toMatchObject({ hydrated: true, text: 'n is 1', sameNode: true })
    const after = island.after.split(' ')
    expect(after.slice(0, 2)).toEqual(classes)
    expect(after).toHaveLength(3)
    expect(css).toMatch(new RegExp(`\\.${after[2]}\\s*\\{\\s*font-weight:\\s*700`))
  })

  it('preloads the runtime beside the loader, since the island starts on load', () => {
    const html = readFileSync(join(out, 'pages/index/index.html'), 'utf8')
    const entry = /<script type="module" src="\/([^"]+)"/.exec(html)![1]!
    const runtime = /data-weft-runtime="([^"]+)"/.exec(html)![1]!
    expect(html).toContain(`<link rel="modulepreload" href="/assets/${runtime}">`)
    const manifest = JSON.parse(readFileSync(join(out, 'manifest.json'), 'utf8')) as { hints: Record<string, string[]> }
    expect(manifest.hints['/']).toEqual([`</${entry}>; rel=modulepreload`, `</assets/${runtime}>; rel=modulepreload`])
  })

  it('ships no stylesheet for them: their CSS is inlined in the page', () => {
    const all = readdirSync(out, { recursive: true }) as string[]
    expect(all.filter((f) => /\.css(\.br|\.gz)?$/.test(f))).toEqual([])
    // No chunk names a stylesheet to load or preload.
    for (const f of all.filter((f) => f.endsWith('.js'))) {
      expect(readFileSync(join(out, f), 'utf8'), f).not.toMatch(/["'`/][\w-]+\.css["'`]/)
    }
  })
})
