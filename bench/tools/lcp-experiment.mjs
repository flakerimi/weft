// Side experiment: does where the Weft page loads its island script change
// its Lighthouse LCP? It changes no Weft code. It writes hand-edited copies of
// the built Weft page next to the two Astro pages, serves them all with the
// same Go static server (bench/static, GOMAXPROCS=1), and runs Lighthouse's
// mobile preset on each, five runs interleaved, twice: with its default
// simulated throttling (Lantern) and with DevTools throttling (the page really
// loads on a throttled network and CPU).
//
//   weft-as-built     the page as weft-build-pages wrote it: the client chunk
//                     as a module script in the head
//   weft-body-end     the same script element moved to the end of the body
//   weft-idle-import  no script element; an inline module at the end of the
//                     body imports the client chunk in requestIdleCallback
//   weft-no-js        no script element at all (the island never starts)
//   astro-vue         Astro's / (Vue island, loaded by its inline loader)
//   astro-plain       Astro's /no-framework/
//
// Writes side_experiments.lcp_script_placement into the results JSON.
// Usage (after bench/run.sh has built everything):
//   node bench/tools/lcp-experiment.mjs <results.json>
import { execFile, spawn } from 'node:child_process'
import { cpSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

const run = promisify(execFile)
const tools = dirname(fileURLToPath(import.meta.url))
const bench = join(tools, '..')
const [resultsFile] = process.argv.slice(2)
if (!resultsFile) {
  process.stderr.write('usage: node lcp-experiment.mjs <results.json>\n')
  process.exit(2)
}
const RUNS = 5
const addr = '127.0.0.1:4303'
const log = (m) => process.stderr.write(`lcp-experiment: ${m}\n`)

// The served folder, under the git-ignored bench/.bin. Copies overwrite.
const dir = join(bench, '.bin', 'lcp-experiment')
const weftDist = join(bench, 'weft', 'dist')
const astroDist = join(bench, 'astro', 'dist')
mkdirSync(dir, { recursive: true })
cpSync(join(weftDist, 'assets'), join(dir, 'assets'), { recursive: true })
cpSync(join(weftDist, 'hero.webp'), join(dir, 'hero.webp'))
cpSync(join(astroDist, '_astro'), join(dir, '_astro'), { recursive: true })

const built = readFileSync(join(weftDist, 'pages', 'index', 'index.html'), 'utf8')
const tag = /<script type="module" src="([^"]+)"><\/script>/.exec(built)
if (!tag) throw new Error('the built Weft page has no module script')
const [element, src] = tag
const without = built.replace(element, '')
const atBodyEnd = (html, add) => html.replace('</body>', `${add}</body>`)
const pages = {
  'weft-as-built': built,
  'weft-body-end': atBodyEnd(without, element),
  'weft-idle-import': atBodyEnd(without, `<script type="module">requestIdleCallback(() => import(${JSON.stringify(src)}))</script>`),
  'weft-no-js': without,
  'astro-vue': readFileSync(join(astroDist, 'index.html'), 'utf8'),
  'astro-plain': readFileSync(join(astroDist, 'no-framework', 'index.html'), 'utf8'),
}
for (const [name, html] of Object.entries(pages)) {
  mkdirSync(join(dir, name), { recursive: true })
  writeFileSync(join(dir, name, 'index.html'), html)
}

async function lighthouse(url, throttling) {
  const { stdout } = await run(
    join(tools, 'node_modules', '.bin', 'lighthouse'),
    [
      url,
      '--quiet',
      '--only-categories=performance',
      `--throttling-method=${throttling}`,
      '--output=json',
      '--output-path=stdout',
      '--chrome-flags=--headless=new --no-first-run',
    ],
    { maxBuffer: 256 * 1024 * 1024 },
  )
  const r = JSON.parse(stdout)
  if (r.runtimeError) throw new Error(`lighthouse ${url}: ${r.runtimeError.code}`)
  const a = r.audits
  const breakdown = a['lcp-breakdown-insight']?.details?.items?.find((i) => i.type === 'table')?.items ?? []
  return {
    score: Math.round(r.categories.performance.score * 100),
    lcp_ms: Math.round(a['largest-contentful-paint'].numericValue),
    fcp_ms: Math.round(a['first-contentful-paint'].numericValue),
    tbt_ms: Math.round(a['total-blocking-time'].numericValue),
    lcp_breakdown_ms: Object.fromEntries(breakdown.map((s) => [s.subpart, Math.round(s.duration)])),
    scripts: a['network-requests'].details.items.filter((i) => i.resourceType === 'Script').map((i) => i.url.replace(/^https?:\/\/[^/]+/, '')),
  }
}

const median = (xs) => {
  const s = [...xs].sort((a, b) => a - b)
  const m = s.length >> 1
  return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2
}

const server = spawn(join(bench, '.bin', 'static'), ['-dir', dir, '-addr', addr], {
  env: { ...process.env, GOMAXPROCS: '1' },
  stdio: 'ignore',
})
try {
  for (let i = 0; ; i++) {
    try {
      if ((await fetch(`http://${addr}/weft-as-built/`)).ok) break
    } catch {}
    if (i > 100) throw new Error(`the static server on ${addr} did not answer`)
    await new Promise((r) => setTimeout(r, 100))
  }

  const variants = {}
  for (const throttling of ['simulate', 'devtools']) {
    for (let i = 0; i < RUNS; i++) {
      for (const name of Object.keys(pages)) {
        const r = await lighthouse(`http://${addr}/${name}/`, throttling)
        ;((variants[name] ??= {})[throttling] ??= { runs: [] }).runs.push(r)
        log(`${throttling} run ${i + 1} ${name}: LCP ${r.lcp_ms} ms`)
      }
    }
  }
  for (const v of Object.values(variants)) {
    for (const t of Object.values(v)) {
      const lcps = t.runs.map((r) => r.lcp_ms)
      Object.assign(t, { lcp_ms: { median: median(lcps), min: Math.min(...lcps), max: Math.max(...lcps) } })
    }
  }

  const results = JSON.parse(readFileSync(resultsFile, 'utf8'))
  results.side_experiments ??= {}
  results.side_experiments.lcp_script_placement = {
    date: new Date().toISOString().slice(0, 10),
    question: 'Does where the Weft page loads its island script change its Lighthouse LCP?',
    method:
      'bench/tools/lcp-experiment.mjs: hand-edited copies of the built Weft page and the two Astro pages, all served by bench/static ' +
      '(GOMAXPROCS=1, gzip on the fly) on ' + addr + '; Lighthouse ' + RUNS + ' mobile runs each, interleaved, with simulated ' +
      '(Lantern) and with DevTools throttling. lcp_breakdown_ms is Lighthouse\'s LCP breakdown insight, read from the trace Lighthouse recorded: ' +
      'unthrottled under simulate (only lcp_ms is simulated), throttled under devtools.',
    variants,
  }
  writeFileSync(resultsFile, JSON.stringify(results, null, 2) + '\n')
  log(`wrote side_experiments.lcp_script_placement into ${resultsFile}`)
} finally {
  server.kill()
}
