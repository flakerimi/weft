/**
 * What each @weft/ui component costs a page: `docs/sizes.md`, from a Vite
 * build of each component's first example (`examples/<Name>/basic.vue`)
 * alone, mounted with `createVaporApp` as an app mounts it.
 *
 *     node scripts/sizes.ts     write docs/sizes.md
 *
 * Each build is what an app's would be: Vapor (every SFC is `<script setup
 * vapor>`), production mode, minified, with `weftIcons()` from @weft/vite so
 * only the icons the example draws ship. Sizes are brotli at quality 11, the
 * way the page build compresses (`zlib.brotliCompressSync`).
 *
 * Two things every page with any component carries are measured once rather
 * than in each row: Vue's runtime (split into a chunk of its own in every
 * build, with Vite's preload helper where the build uses `import()`, as the
 * page build's shared runtime has both; it varies a little with the Vue
 * features a component uses, so the smallest and largest are given) and
 * `styles/base.css` (left out of each build's CSS, since the page build
 * inlines it on every page anyway).
 *
 * The builds are `buildAlone` from packages/vite/test/alone.ts, which
 * packages/vite/test/tree-shaking.test.ts builds with too.
 */
import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { brotli, buildAlone, mountEntry, ui } from '../packages/vite/test/alone.ts'

const repo = fileURLToPath(new URL('..', import.meta.url))
const fromVite = createRequire(join(repo, 'packages/vite/package.json'))
const load = async <T>(spec: string) => (await import(pathToFileURL(fromVite.resolve(spec)).href)) as T

interface Row {
  name: string
  js: number
  /** JavaScript loaded on demand (a dynamic import), not with the page. */
  later: number
  css: number
  vue: number
}

async function measure(): Promise<{ rows: Row[]; base: number }> {
  const names = readdirSync(join(ui, 'examples'), { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name)
    .sort()
  const rows: Row[] = []
  for (const name of names) {
    const { chunks, css } = await buildAlone(mountEntry(join(ui, 'examples', name, 'basic.vue')))
    const own = chunks.filter((c) => !c.vue && !c.lazy)
    const later = chunks.filter((c) => !c.vue && c.lazy)
    const runtime = chunks.filter((c) => c.vue)
    rows.push({
      name,
      js: own.reduce((n, c) => n + brotli(c.code), 0),
      later: later.reduce((n, c) => n + brotli(c.code), 0),
      css: css.trim() ? brotli(css) : 0,
      vue: runtime.reduce((n, c) => n + brotli(c.code), 0),
    })
    process.stdout.write('.')
  }
  process.stdout.write('\n')
  const { transform } = await load<typeof import('lightningcss')>('lightningcss')
  const baseCSS = transform({ filename: 'base.css', code: readFileSync(join(ui, 'src/styles/base.css')), minify: true }).code.toString()
  return { rows, base: brotli(baseCSS) }
}

const kb = (n: number) => (n / 1024).toFixed(2)

function markdown({ rows, base }: { rows: Row[]; base: number }): string {
  const vue = rows.map((r) => r.vue)
  const byTotal = [...rows].sort((a, b) => b.js + b.css - (a.js + a.css))
  const median = (xs: number[]) => {
    const s = [...xs].sort((a, b) => a - b)
    return s[Math.floor(s.length / 2)]!
  }
  return `# Component sizes

What each \`@weft/ui\` component costs a page, measured by \`node scripts/sizes.ts\`
(which wrote this file). Each row is a Vite build of the component's first
example, \`packages/ui/examples/<Name>/basic.vue\`, alone: mounted with
\`createVaporApp\`, Vapor, production mode, minified, with \`weftIcons()\`, so
only the icons the example draws ship. Sizes are brotli at quality 11, as the
page build sends them, in KB (1024 bytes).

An example may draw more than its component: a Dialog example has a Button
to open it, a Form example its fields. The numbers are what that example
costs, which is what a page using the component in that way pays.

JS is what loads with the page; Later, what a component loads on demand
(a dynamic \`import()\`), such as the DatePicker's popover and calendar, fetched
when the pointer or focus first comes to it. Total is JS and CSS: what the
page pays before anyone touches it.

## Shared by every page

| What | Brotli |
| --- | ---: |
| Vue runtime (Vapor), its own chunk in each build, with Vite's preload helper where the build has one, as a page build's shared runtime has it | ${kb(Math.min(...vue))} to ${kb(Math.max(...vue))} KB |
| \`styles/base.css\`, inlined on every page by the page build | ${kb(base)} KB |

Vue's runtime is tree-shaken per build, so it varies with the Vue features a
component uses. Neither is counted in the rows below.

## Per component

${rows.length} components. Median: JS ${kb(median(rows.map((r) => r.js)))} KB, CSS ${kb(median(rows.map((r) => r.css)))} KB.

| Component | JS | Later | CSS | Total |
| --- | ---: | ---: | ---: | ---: |
${rows.map((r) => `| ${r.name} | ${kb(r.js)} | ${r.later ? kb(r.later) : '—'} | ${kb(r.css)} | ${kb(r.js + r.css)} |`).join('\n')}

The five heaviest: ${byTotal
    .slice(0, 5)
    .map((r) => `${r.name} (${kb(r.js + r.css)} KB)`)
    .join(', ')}.

## Tree-shaking

\`packages/vite/test/tree-shaking.test.ts\` builds \`import { Button } from
'@weft/ui'\` the same way and checks that the output holds no trace of
\`Dialog\`, \`Table\`, the overlay stack or the overlays' default icons: a page
pays only for the components it imports, through the index or by path.
\`@weft/ui\`'s package.json declares its scripts free of side effects and its
stylesheets as having them (\`"sideEffects": ["**/*.css"]\`), so a bundler
drops every module a page does not use, and the page build, following the
same declaration, inlines only the stylesheets of the components an island
takes from the index.
`
}

// Run as a script (`node scripts/sizes.ts`), not when imported by a test.
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.env.NODE_ENV = 'production'
  const out = join(repo, 'docs/sizes.md')
  writeFileSync(out, markdown(await measure()))
  console.log('wrote docs/sizes.md')
}
