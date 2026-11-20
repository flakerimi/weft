// Measure every benchmark target and write the results as JSON.
// bench/run.sh starts the servers and runs this; see bench/README.md.
//
// For each target:
//   bytes       what the landing loads (the URLs a Lighthouse run fetched,
//               status 200), each fetched again here. Text is counted as the
//               brotli the Weft build makes (quality 11), whatever the server
//               sends; images as they are. Wire bytes are what this server
//               really sent with Accept-Encoding: br, gzip.
//   ttfb        `load -ttfb N`: N requests on new connections, to the final
//               response's headers.
//   lighthouse  N runs of Lighthouse's mobile preset, the targets interleaved
//               run by run; the median score and LCP, with min and max.
//   load        `load`: 64 connections, 2 s warm-up, 10 s measured; three
//               rounds, the targets interleaved; the median and range.
//
// Usage: node measure.mjs --load <load binary> --out <results.json>
import { execFile } from 'node:child_process'
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import http from 'node:http'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { parseArgs, promisify } from 'node:util'
import zlib from 'node:zlib'

const run = promisify(execFile)
const tools = dirname(fileURLToPath(import.meta.url))
const bench = join(tools, '..')

const { values: args } = parseArgs({
  options: {
    load: { type: 'string' },
    out: { type: 'string' },
    runs: { type: 'string', default: '5' },
    ttfb: { type: 'string', default: '50' },
    conns: { type: 'string', default: '64' },
    duration: { type: 'string', default: '10s' },
    warmup: { type: 'string', default: '2s' },
    'load-runs': { type: 'string', default: '3' },
  },
})
if (!args.load || !args.out) {
  process.stderr.write('usage: node measure.mjs --load <load binary> --out <results.json>\n')
  process.exit(2)
}

const weft = process.env.WEFT_URL ?? 'http://127.0.0.1:4301'
const preview = process.env.ASTRO_PREVIEW_URL ?? 'http://127.0.0.1:4321'
const plain = process.env.ASTRO_STATIC_URL ?? 'http://127.0.0.1:4302'

const targets = [
  { id: 'weft', label: 'Weft', server: 'weft binary (weft.Serve), GOMAXPROCS=1', url: `${weft}/` },
  { id: 'astro-vue-preview', label: 'Astro, Vue island', server: 'astro preview', url: `${preview}/` },
  { id: 'astro-vue-static', label: 'Astro, Vue island', server: 'Go static + gzip, GOMAXPROCS=1', url: `${plain}/` },
  { id: 'astro-plain-preview', label: 'Astro, no framework island', server: 'astro preview', url: `${preview}/no-framework/` },
  { id: 'astro-plain-static', label: 'Astro, no framework island', server: 'Go static + gzip, GOMAXPROCS=1', url: `${plain}/no-framework/` },
]

const log = (m) => process.stderr.write(`measure: ${m}\n`)

function median(xs) {
  const s = [...xs].sort((a, b) => a - b)
  const m = s.length >> 1
  return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2
}

function spread(xs) {
  return { median: median(xs), min: Math.min(...xs), max: Math.max(...xs), values: xs }
}

// get fetches a URL with node:http, which does not decompress, so the body is
// the bytes on the wire. Any 1xx (Early Hints) before the response is skipped.
function get(url, encoding) {
  return new Promise((resolve, reject) => {
    const req = http.get(url, { headers: encoding ? { 'accept-encoding': encoding } : {} }, (res) => {
      const chunks = []
      res.on('data', (c) => chunks.push(c))
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks) }))
      res.on('error', reject)
    })
    req.on('error', reject)
  })
}

// Inline <script> elements that run JavaScript: no src, and not a JSON type.
const INLINE_SCRIPT = /<script\b([^>]*)>([\s\S]*?)<\/script>/gi
function inlineJS(html) {
  let n = 0
  for (const [, attrs, body] of html.matchAll(INLINE_SCRIPT)) {
    if (/\bsrc=/.test(attrs)) continue
    const type = /\btype="([^"]*)"/.exec(attrs)?.[1] ?? ''
    if (type === '' || type === 'module' || /javascript/.test(type)) n += Buffer.byteLength(body)
  }
  return n
}

const TEXT = /^(text\/|application\/(javascript|json|manifest\+json)|image\/svg\+xml)/

function brotli(bytes) {
  return zlib.brotliCompressSync(bytes, {
    params: {
      [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
      [zlib.constants.BROTLI_PARAM_MODE]: zlib.constants.BROTLI_MODE_TEXT,
      [zlib.constants.BROTLI_PARAM_SIZE_HINT]: bytes.length,
    },
  }).length
}

async function lighthouse(url) {
  const bin = join(tools, 'node_modules', '.bin', 'lighthouse')
  const { stdout } = await run(
    bin,
    [url, '--quiet', '--only-categories=performance', '--output=json', '--output-path=stdout', '--chrome-flags=--headless=new --no-first-run'],
    { maxBuffer: 256 * 1024 * 1024, env: process.env },
  )
  const r = JSON.parse(stdout)
  if (r.runtimeError) throw new Error(`lighthouse ${url}: ${r.runtimeError.code} ${r.runtimeError.message}`)
  const a = r.audits
  return {
    score: Math.round(r.categories.performance.score * 100),
    lcp_ms: a['largest-contentful-paint'].numericValue,
    fcp_ms: a['first-contentful-paint'].numericValue,
    tbt_ms: a['total-blocking-time'].numericValue,
    cls: a['cumulative-layout-shift'].numericValue,
    lcp_element: a['lcp-breakdown-insight']?.details?.items?.find((i) => i.type === 'node')?.selector ?? null,
    requests: a['network-requests'].details.items.map((i) => ({
      url: i.url,
      type: i.resourceType,
      status: i.statusCode,
      transfer: i.transferSize,
    })),
    lighthouse: r.lighthouseVersion,
  }
}

async function bytes(requests) {
  const resources = []
  for (const r of requests) {
    if (r.status !== 200 || r.url.startsWith('data:')) continue
    const raw = await get(r.url, '')
    const wire = await get(r.url, 'br, gzip')
    const type = raw.headers['content-type'] ?? ''
    const text = TEXT.test(type)
    resources.push({
      url: r.url,
      type: r.type,
      content_type: type,
      raw: raw.body.length,
      br: text ? brotli(raw.body) : raw.body.length,
      wire: wire.body.length,
      wire_encoding: wire.headers['content-encoding'] ?? 'identity',
      chrome_transfer: r.transfer,
      inline_js: r.type === 'Document' ? inlineJS(raw.body.toString('utf8')) : 0,
    })
  }
  const sum = (pick, f) => resources.filter(pick).reduce((n, x) => n + x[f], 0)
  const all = () => true
  return {
    requests: resources.length,
    js_br: sum((x) => x.type === 'Script', 'br'),
    js_inline_raw: sum(all, 'inline_js'),
    css_br: sum((x) => x.type === 'Stylesheet', 'br'),
    html_br: sum((x) => x.type === 'Document', 'br'),
    image: sum((x) => x.type === 'Image', 'br'),
    total_br: sum(all, 'br'),
    js_wire: sum((x) => x.type === 'Script', 'wire'),
    total_wire: sum(all, 'wire'),
    resources,
  }
}

async function load(url, extra) {
  const { stdout } = await run(args.load, ['-url', url, ...extra], { maxBuffer: 16 * 1024 * 1024 })
  return JSON.parse(stdout)
}

async function sh(cmd, argv) {
  try {
    return (await run(cmd, argv)).stdout.trim()
  } catch {
    return null
  }
}

const pkg = (p) => JSON.parse(readFileSync(p, 'utf8')).version
const results = {
  date: new Date().toISOString().slice(0, 10),
  machine: {
    chip: await sh('sysctl', ['-n', 'machdep.cpu.brand_string']),
    cores: Number(await sh('sysctl', ['-n', 'hw.ncpu'])),
    performance_cores: Number(await sh('sysctl', ['-n', 'hw.perflevel0.physicalcpu'])),
    memory_gb: Number(await sh('sysctl', ['-n', 'hw.memsize'])) / 2 ** 30,
    macos: await sh('sw_vers', ['-productVersion']),
    macos_build: await sh('sw_vers', ['-buildVersion']),
  },
  versions: {
    weft_commit: await sh('git', ['-C', bench, 'rev-parse', '--short', 'HEAD']),
    vue_weft: pkg(join(bench, 'weft', 'node_modules', 'vue', 'package.json')),
    astro: pkg(join(bench, 'astro', 'node_modules', 'astro', 'package.json')),
    astrojs_vue: pkg(join(bench, 'astro', 'node_modules', '@astrojs', 'vue', 'package.json')),
    vue_astro: pkg(join(bench, 'astro', 'node_modules', 'vue', 'package.json')),
    lighthouse: pkg(join(tools, 'node_modules', 'lighthouse', 'package.json')),
    sharp: pkg(join(tools, 'node_modules', 'sharp', 'package.json')),
    node: process.version,
    go: await sh('go', ['env', 'GOVERSION']),
    chrome: await sh(process.env.CHROME_PATH ?? 'google-chrome', ['--version']),
  },
  method: {
    lighthouse_runs: Number(args.runs),
    lighthouse_preset: 'mobile (default), simulated throttling, headless Chrome, a fresh profile each run',
    ttfb_requests: Number(args.ttfb),
    load: { runs: Number(args['load-runs']), connections: Number(args.conns), duration: args.duration, warmup: args.warmup, accept_encoding: 'br, gzip' },
  },
  targets: [],
}

for (const t of targets) results.targets.push({ ...t, lighthouse: { runs: [] } })

log(`lighthouse, ${args.runs} runs of each target, interleaved`)
for (let i = 0; i < Number(args.runs); i++) {
  for (const t of results.targets) {
    const r = await lighthouse(t.url)
    t.lighthouse.runs.push(r)
    log(`  run ${i + 1} ${t.id}: score ${r.score}, LCP ${Math.round(r.lcp_ms)} ms`)
  }
}

for (const t of results.targets) {
  const runs = t.lighthouse.runs
  t.bytes = await bytes(runs[0].requests)
  t.lighthouse = {
    score: spread(runs.map((r) => r.score)),
    lcp_ms: spread(runs.map((r) => Math.round(r.lcp_ms))),
    fcp_ms: spread(runs.map((r) => Math.round(r.fcp_ms))),
    tbt_ms: spread(runs.map((r) => Math.round(r.tbt_ms))),
    cls: spread(runs.map((r) => r.cls)),
    lcp_element: runs[0].lcp_element,
    requests: runs[0].requests,
  }
  log(`${t.id}: ${t.bytes.requests} requests, JS ${t.bytes.js_br} B br + ${t.bytes.js_inline_raw} B inline, total ${t.bytes.total_br} B br`)

  t.ttfb = await load(t.url, ['-ttfb', args.ttfb])
  log(`${t.id}: TTFB median ${t.ttfb.median_ms.toFixed(3)} ms`)
}

const rounds = new Map(results.targets.map((t) => [t, []]))
for (let i = 0; i < Number(args['load-runs']); i++) {
  for (const t of results.targets) {
    const r = await load(t.url, ['-c', args.conns, '-d', args.duration, '-warmup', args.warmup])
    rounds.get(t).push(r)
    log(`  load ${i + 1} ${t.id}: ${Math.round(r.rps)} req/s, p50 ${r.p50_ms.toFixed(2)} ms, p99 ${r.p99_ms.toFixed(2)} ms, errors ${r.errors}`)
  }
}
for (const [t, runs] of rounds) {
  t.load = {
    rps: spread(runs.map((r) => Math.round(r.rps))),
    p50_ms: spread(runs.map((r) => r.p50_ms)),
    p99_ms: spread(runs.map((r) => r.p99_ms)),
    errors: runs.reduce((n, r) => n + r.errors, 0),
    runs,
  }
}

mkdirSync(dirname(args.out), { recursive: true })
writeFileSync(args.out, JSON.stringify(results, null, 2) + '\n')
log(`wrote ${args.out}`)
