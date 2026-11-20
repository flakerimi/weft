// What the built Weft landing's HTML is made of: its inline CSS and Vue's
// hydration markers, raw and in brotli. Writes weft_html_breakdown into the
// results JSON. Brotli is quality 11, text mode, as the Weft build stores it.
// Usage (after the page is built): node bench/tools/html-breakdown.mjs <results.json>
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import zlib from 'node:zlib'

const bench = join(dirname(fileURLToPath(import.meta.url)), '..')
const [resultsFile] = process.argv.slice(2)
if (!resultsFile) {
  process.stderr.write('usage: node html-breakdown.mjs <results.json>\n')
  process.exit(2)
}
const file = join(bench, 'weft', 'dist', 'pages', 'index', 'index.html')
const html = readFileSync(file, 'utf8')
const bytes = (s) => Buffer.byteLength(s)
const br = (s) =>
  zlib.brotliCompressSync(Buffer.from(s), {
    params: {
      [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
      [zlib.constants.BROTLI_PARAM_MODE]: zlib.constants.BROTLI_MODE_TEXT,
      [zlib.constants.BROTLI_PARAM_SIZE_HINT]: bytes(s),
    },
  }).length

const stored = readFileSync(file + '.br').length
const whole = br(html)
if (whole !== stored) throw new Error(`brotli here is ${whole} B, the build stored ${stored} B`)

// The inline stylesheet: the content of the page's <style> element.
const css = /<style>([\s\S]*?)<\/style>/.exec(html)?.[1] ?? ''

// Vue's SSR markers: <!--[--> <!--]--> (fragments), <!--{--> <!--}--> (Vapor
// fragments) and <!----> (an empty v-if). Inside <weft-island> they are what
// hydration reads; outside, nothing hydrates.
const MARKER = /<!--(\[|\]|\{|\}|)-->/g
const islandAt = html.indexOf('<weft-island')
const islandEnd = html.indexOf('</weft-island>') + '</weft-island>'.length
const kinds = {}
let outside = { count: 0, bytes: 0 }
for (const m of html.matchAll(MARKER)) {
  const k = (kinds[m[0]] ??= { count: 0, bytes: 0 })
  k.count++
  k.bytes += bytes(m[0])
  if (m.index < islandAt || m.index >= islandEnd) {
    outside.count++
    outside.bytes += bytes(m[0])
  }
}
const all = Object.values(kinds).reduce((t, k) => ({ count: t.count + k.count, bytes: t.bytes + k.bytes }), { count: 0, bytes: 0 })
const outsideRemoved = html.slice(0, islandAt).replace(MARKER, '') + html.slice(islandAt, islandEnd) + html.slice(islandEnd).replace(MARKER, '')

const results = JSON.parse(readFileSync(resultsFile, 'utf8'))
results.weft_html_breakdown = {
  file: 'bench/weft/dist/pages/index/index.html',
  html: { raw: bytes(html), br: whole },
  inline_css: {
    raw: bytes(css),
    br_alone: br(css),
    br_share: whole - br(html.replace(css, '')),
    method: 'raw: UTF-8 bytes of the <style> element\'s content. br_alone: that CSS compressed on its own. br_share: the page\'s brotli size minus the brotli size of the page with the CSS removed.',
  },
  hydration_markers: {
    kinds,
    all,
    outside_islands: { ...outside, br_share: whole - br(outsideRemoved) },
    method:
      'Every match of /<!--(\\[|\\]|\\{|\\}|)-->/ in the page: <!--[--> <!--]--> <!--{--> <!--}--> <!---->. outside_islands leaves out those inside <weft-island>...</weft-island>; br_share is the page\'s brotli size minus that of the page with those removed.',
  },
}
writeFileSync(resultsFile, JSON.stringify(results, null, 2) + '\n')
process.stdout.write(JSON.stringify(results.weft_html_breakdown, null, 2) + '\n')
