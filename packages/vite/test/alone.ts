// Build one module alone, as an app's Vite build would: Vapor (every Weft
// SFC is `<script setup vapor>`), production mode, minified, weftIcons() on.
// tree-shaking.test.ts builds with it, and scripts/sizes.ts measures every
// example with it for docs/sizes.md.
import { dirname, resolve } from 'node:path'
import { createRequire } from 'node:module'
import { brotliCompressSync, constants } from 'node:zlib'
import vue from '@vitejs/plugin-vue'
import { build, type Rolldown } from 'vite'
import { weftIcons } from '../src/icons.ts'

/** @weft/ui's package folder; its main entry is src/index.ts. */
export const ui = dirname(dirname(createRequire(import.meta.url).resolve('@weft/ui')))

/** Brotli at quality 11, as the page build compresses. */
export const brotli = (text: string) =>
  brotliCompressSync(text, { params: { [constants.BROTLI_PARAM_QUALITY]: 11 } }).length

export interface AloneBuild {
  /**
   * Each JavaScript chunk: its file name and code. `vue` is Vue's runtime
   * (with Vite's preload helper, as a page build's shared runtime has it);
   * `lazy`, a chunk only a dynamic `import()` reaches, loaded on demand
   * rather than with the page.
   */
  chunks: { name: string; code: string; vue: boolean; lazy: boolean }[]
  /** The CSS, base.css left out. */
  css: string
  /** The modules the output holds code of, by id, tree-shaken ones left out. */
  modules: string[]
}

const ENTRY = 'virtual:weft-alone'
const VUE = /[\\/]node_modules[\\/](?:\.pnpm[\\/][^\\/]+[\\/]node_modules[\\/])?(?:vue|@vue[\\/][^\\/]+)[\\/]/
/** Vite's helper for a dynamic `import()`, which a page build keeps in the runtime every island shares, beside Vue's. */
const PRELOAD = /vite\/preload-helper/
const shared = (id: string) => VUE.test(id) || PRELOAD.test(id)

/**
 * Build `entry` (a module's source, imports resolved from packages/ui, so
 * `@weft/ui` is the package itself) alone: Vapor, production, minified, the
 * icons plugin on, Vue's runtime (and Vite's preload helper) in a chunk of
 * its own, base.css left out.
 */
export async function buildAlone(entry: string): Promise<AloneBuild> {
  const base = resolve(ui, 'src/styles/base.css')
  const result = (await build({
    root: ui,
    configFile: false,
    logLevel: 'error',
    mode: 'production',
    plugins: [
      {
        name: 'weft:alone',
        enforce: 'pre',
        resolveId: (id) => (id === ENTRY ? '\0' + ENTRY : undefined),
        load(id) {
          if (id === '\0' + ENTRY) return entry
          if (id.split('?')[0] === base) return ''
        },
      },
      vue(),
      weftIcons(),
    ],
    build: {
      write: false,
      minify: true,
      cssCodeSplit: false,
      modulePreload: { polyfill: false },
      rolldownOptions: {
        input: { alone: ENTRY },
        output: { codeSplitting: { groups: [{ name: 'vue', test: shared }] } },
      },
    },
  })) as Rolldown.RolldownOutput | Rolldown.RolldownOutput[]
  const output = (Array.isArray(result) ? result : [result]).flatMap((r) => r.output)
  const built = output.filter((o): o is Rolldown.OutputChunk => o.type === 'chunk')
  // What loads with the page: the entry and what it imports statically.
  const byName = new Map(built.map((c) => [c.fileName, c]))
  const eager = new Set<string>()
  const walk = (name: string) => {
    if (eager.has(name)) return
    eager.add(name)
    for (const i of byName.get(name)?.imports ?? []) walk(i)
  }
  for (const c of built) if (c.isEntry) walk(c.fileName)
  const chunks = built.map((c) => ({
    name: c.fileName,
    code: c.code,
    vue: c.moduleIds.length > 0 && c.moduleIds.every(shared),
    lazy: !eager.has(c.fileName),
  }))
  const modules = built.flatMap((c) => c.moduleIds.filter((m) => (c.modules[m]?.renderedLength ?? 0) > 0))
  const css = output
    .filter((o): o is Rolldown.OutputAsset => o.type === 'asset' && o.fileName.endsWith('.css'))
    .map((a) => String(a.source))
    .join('\n')
  return { chunks, css, modules }
}

/** The module that mounts one example, as an app mounts its root. */
export const mountEntry = (file: string) =>
  `import { createVaporApp } from 'vue'\nimport Example from ${JSON.stringify(file)}\ncreateVaporApp(Example).mount('#app')\n`
