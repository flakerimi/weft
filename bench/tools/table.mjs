// Write the results table for a results JSON into bench/README.md, between
// its <!-- results:start --> and <!-- results:end --> markers.
// Usage: node table.mjs <results.json> <README.md>
import { readFileSync, writeFileSync } from 'node:fs'
import { basename } from 'node:path'

const [file, readme] = process.argv.slice(2)
if (!file || !readme) {
  process.stderr.write('usage: node table.mjs <results.json> <README.md>\n')
  process.exit(2)
}
const r = JSON.parse(readFileSync(file, 'utf8'))

const kb = (n) => (n === 0 ? '0 B' : `${(n / 1024).toFixed(1)} KB`)
const ms = (n) => (n < 100 ? n.toFixed(2) : Math.round(n).toString())
const count = (n) => Math.round(n).toLocaleString('en-US')
const range = (s, f) => `${f(s.median)} [${f(s.min)}–${f(s.max)}]`
const chrome = /([\d.]+)\s*$/.exec(r.versions.chrome ?? '')?.[1] ?? 'unknown'
const m = r.machine
const v = r.versions

const lines = [
  `Run of ${r.date} ([\`results/${basename(file)}\`](results/${basename(file)})).`,
  '',
  `Machine: ${m.chip}, ${m.cores} cores (${m.performance_cores} performance), ${m.memory_gb} GB, macOS ${m.macos} (${m.macos_build}). ` +
    `Node ${v.node}, ${v.go}, Chrome ${chrome} headless. Weft at \`${v.weft_commit}\` with Vue ${v.vue_weft}; ` +
    `Astro ${v.astro}, @astrojs/vue ${v.astrojs_vue}, Vue ${v.vue_astro}; ` +
    (v.nuxt ? `Nuxt ${v.nuxt}, Vue ${v.vue_nuxt}; Next.js ${v.next}, React ${v.react}; SvelteKit ${v.sveltekit}, Svelte ${v.svelte}; ` : '') +
    `Lighthouse ${v.lighthouse}.`,
  '',
  '| Site | Server | HTML+CSS (br) | JS files (br) | Inline JS (raw) | Total (br) | Requests | TTFB, ms | Lighthouse | LCP, ms | Req/s (Go servers at GOMAXPROCS=1) | p50 / p99, ms |',
  '|---|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|--:|',
]
for (const t of r.targets) {
  const b = t.bytes
  lines.push(
    `| ${t.label} | ${t.server} | ${kb(b.html_br + b.css_br)} | ${kb(b.js_br)} | ${b.js_inline_raw} B | ${kb(b.total_br)} | ${b.requests} | ` +
      `${ms(t.ttfb.median_ms)} [${ms(t.ttfb.p25_ms)}–${ms(t.ttfb.p75_ms)}] | ${range(t.lighthouse.score, String)} | ` +
      `${range(t.lighthouse.lcp_ms, String)} | ${range(t.load.rps, count)} | ${ms(t.load.p50_ms.median)} / ${ms(t.load.p99_ms.median)} |`,
  )
}
lines.push(
  '',
  'Bytes: KB is 1024 B. Text is counted as brotli quality 11 (what the Weft build stores), whatever the server sends; the image as it is. ' +
    'TTFB: median of ' +
    (r.method.ttfb_runs
      ? `all ${r.method.ttfb_runs * r.method.ttfb_requests} requests on new connections, pooled from ${r.method.ttfb_runs} interleaved rounds of ${r.method.ttfb_requests}, [p25–p75]. `
      : `${r.method.ttfb_requests} requests on new connections, [p25–p75]. `) +
    `Lighthouse and LCP: median of ${r.method.lighthouse_runs} mobile runs, [min–max]. ` +
    `Load: ${r.method.load.connections} connections, ${r.method.load.warmup} warm-up, ${r.method.load.duration} measured, ` +
    `median of ${r.method.load.runs} rounds [min–max]; p50 and p99 are the medians of the rounds' own. No errors unless noted. ` +
    'The Go servers (Weft, Go static, Go pre-compressed) run at GOMAXPROCS=1; the frameworks\' own servers are Node and are not held to one core.',
  '',
  'What each server really sent (Accept-Encoding: br, gzip):',
  '',
  '| Site | Server | Page encoding | JS on the wire | Total on the wire | LCP element |',
  '|---|---|---|--:|--:|---|',
)
for (const t of r.targets) {
  const doc = t.bytes.resources.find((x) => x.type === 'Document')
  lines.push(
    `| ${t.label} | ${t.server} | ${doc?.wire_encoding ?? '?'} | ${kb(t.bytes.js_wire)} | ${kb(t.bytes.total_wire)} | \`${t.lighthouse.lcp_element ?? '?'}\` |`,
  )
}
const errors = r.targets.filter((t) => t.load.errors > 0)
for (const t of errors) lines.push('', `${t.label} (${t.server}): ${t.load.errors} errors under load.`)

const start = '<!-- results:start -->'
const end = '<!-- results:end -->'
const text = readFileSync(readme, 'utf8')
const i = text.indexOf(start)
const j = text.indexOf(end)
if (i < 0 || j < i) {
  process.stderr.write(`table: ${readme} has no ${start} ... ${end} section\n`)
  process.exit(1)
}
writeFileSync(readme, text.slice(0, i + start.length) + '\n' + lines.join('\n') + '\n' + text.slice(j))
