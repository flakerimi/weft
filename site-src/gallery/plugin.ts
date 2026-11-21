// The gallery's Vite plugin: the modules the /components pages are made
// from, generated from catalog/components.json and the example files, so a
// component added to @weft/ui appears in the gallery with no page code.
//
// - `virtual:weft-gallery` (for pages, on the server): the catalogue's text,
//   every example file's text and component, and each component's islands.
// - An island file per component, `Stage_<Name>.vue`, its first example in
//   the preview stage, and one per other example, `Example_<Name>_<Example>.vue`,
//   the example alone. The file's name is the island's id, so every island
//   on the site has its own (every component has a basic.vue).
// - `stage-themes.css`: the neutral and lisa themes, scoped to the stage's
//   canvas, so the stage can show either in light or dark inside the site's
//   own theme.
//
// The generated files live under <site>/.gallery/, a folder that does not
// exist: this plugin serves them.
import { readdirSync, readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join, resolve } from 'node:path'
import type { Plugin } from 'vite'
import { exampleIsland, stageIsland, type Catalog } from './model.ts'

const PUBLIC_ID = 'virtual:weft-gallery'
const DIR = '.gallery'

/**
 * The scoped CSS of a theme as @weft/themes writes it: its light block (with
 * the tokens every scheme shares) on `.gallery-canvas[data-theme=<name>]`,
 * its dark block where the canvas says `data-weft-scheme="dark"`, and its
 * dark block again under a dark system when the canvas says neither. The
 * input and surface blocks are the same in every theme and stay the site's.
 */
export function scopeTheme(css: string, name: string): string {
  const canvas = `.gallery-canvas[data-theme="${name}"]`
  const body = (selector: string) => {
    const at = css.indexOf(`${selector} {`)
    if (at === -1) throw new Error(`gallery: the ${name} theme has no ${selector} block; rebuild @weft/themes.`)
    const start = css.indexOf('{', at) + 1
    return css.slice(start, css.indexOf('}', start)).trim()
  }
  const light = body(':where(:root, [data-weft-scheme="light"])')
  const dark = body(':where([data-weft-scheme="dark"])')
  return [
    `${canvas} {\n  ${light}\n}`,
    `${canvas}[data-weft-scheme="dark"] {\n  ${dark}\n}`,
    `@media (prefers-color-scheme: dark) {\n  ${canvas}:not([data-weft-scheme="light"]) {\n  ${dark}\n  }\n}`,
  ].join('\n')
}

export interface GalleryOptions {
  /** catalog/components.json, from the site root; its example paths are from the folder above catalog/. */
  catalog: string
}

export function gallery(opts: GalleryOptions): Plugin {
  let root = process.cwd()
  let isBuild = false
  const dir = () => join(root, DIR)
  const catalogFile = () => resolve(root, opts.catalog)
  const repo = () => dirname(dirname(catalogFile()))
  const read = () => JSON.parse(readFileSync(catalogFile(), 'utf8')) as Catalog

  /** Every .vue file under each component's examples folder, from the repository root. */
  function exampleFiles(): string[] {
    const examples = join(repo(), 'packages/ui/examples')
    return readdirSync(examples, { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .flatMap((d) => readdirSync(join(examples, d.name)).filter((f) => f.endsWith('.vue')).map((f) => `packages/ui/examples/${d.name}/${f}`))
      .sort()
  }

  /** The source of each generated file, by its name in .gallery/. */
  function load(name: string): string | undefined {
    const catalog = read()
    const abs = (file: string) => JSON.stringify(join(repo(), file))
    if (name === 'index.ts') {
      const files = exampleFiles()
      const lines = [`import catalogText from ${JSON.stringify(catalogFile() + '?raw')}`]
      const sources: string[] = []
      const components: string[] = []
      files.forEach((file, i) => {
        lines.push(`import source${i} from ${JSON.stringify(join(repo(), file) + '?raw')}`)
        sources.push(`  ${JSON.stringify(file)}: source${i},`)
      })
      let n = 0
      const islands: string[] = []
      for (const c of catalog.components) {
        c.examples.forEach((e, i) => {
          lines.push(`import example${n} from ${abs(e.file)}`)
          components.push(`  ${JSON.stringify(e.file)}: example${n},`)
          const island = i === 0 ? stageIsland(c.name) : exampleIsland(c.name, e.name)
          lines.push(`import island${n} from ${JSON.stringify(join(dir(), island + '.vue'))}`)
          islands.push(`  ${JSON.stringify(e.file)}: island${n},`)
          n++
        })
      }
      return [
        ...lines,
        '/** catalog/components.json as it is on disk. */',
        'export { catalogText }',
        '/** Every example file\'s text, by its path from the repository root (helpers too). */',
        `export const exampleSources = {\n${sources.join('\n')}\n}`,
        '/** Each catalogued example\'s component. */',
        `export const exampleComponents = {\n${components.join('\n')}\n}`,
        "/** Each catalogued example's island: the first example's preview stage, any other example alone. */",
        `export const exampleIslands = {\n${islands.join('\n')}\n}`,
        '',
      ].join('\n')
    }
    if (name === 'stage-themes.css') {
      const require = createRequire(join(root, 'package.json'))
      const fromUI = createRequire(require.resolve('@weft/ui/styles/base.css'))
      return ['neutral', 'lisa'].map((t) => scopeTheme(readFileSync(fromUI.resolve(`@weft/themes/${t}.css`), 'utf8'), t)).join('\n') + '\n'
    }
    for (const c of catalog.components) {
      const [first, ...rest] = c.examples
      if (first && name === `${stageIsland(c.name)}.vue`) {
        return [
          '<script setup vapor lang="ts">',
          `import PreviewStage from ${JSON.stringify(join(root, 'components/gallery/PreviewStage.vue'))}`,
          `import Example from ${abs(first.file)}`,
          '</script>',
          '',
          '<template>',
          `  <PreviewStage component="${c.name}"><Example /></PreviewStage>`,
          '</template>',
          '',
        ].join('\n')
      }
      for (const e of rest) {
        if (name !== `${exampleIsland(c.name, e.name)}.vue`) continue
        // inset: a Page or an AppShell in the example is a preview, not the page's main or h1.
        return [
          '<script setup vapor lang="ts">',
          "import { WeftProvider } from '@weft/ui'",
          `import Example from ${abs(e.file)}`,
          '</script>',
          '',
          '<template>',
          '  <WeftProvider inset><Example /></WeftProvider>',
          '</template>',
          '',
        ].join('\n')
      }
    }
    return undefined
  }

  /** The name in .gallery/ an import means, and its query, or null for any other import. */
  const generated = (id: string): { name: string; query: string } | null => {
    const q = id.indexOf('?')
    const path = q === -1 ? id : id.slice(0, q)
    const query = q === -1 ? '' : id.slice(q)
    const prefixes = [`${dir()}/`, `/${DIR}/`]
    const prefix = prefixes.find((p) => path.startsWith(p))
    return prefix ? { name: path.slice(prefix.length), query } : null
  }

  return {
    name: 'weft-site:gallery',
    enforce: 'pre',
    configResolved(config) {
      root = config.root
      isBuild = config.command === 'build'
    },
    resolveId(id) {
      if (id === PUBLIC_ID) return join(dir(), 'index.ts')
      const g = generated(id)
      if (g) return join(dir(), g.name) + g.query
    },
    load(id, options) {
      const g = generated(id)
      // A block of a generated SFC (?vue&type=…) is @vitejs/plugin-vue's to load, from the SFC it compiled.
      if (!g || /[?&]vue\b/.test(g.query)) return
      if (g.name === 'index.ts' && !options?.ssr) {
        throw new Error(`${PUBLIC_ID} is read by pages on the server; an island gets what it needs in its own file.`)
      }
      // The client build ships no CSS: the pages inline it (see @weft/vite's build).
      if (g.name === 'stage-themes.css' && isBuild && !options?.ssr) return ''
      const code = load(g.name)
      if (code === undefined) throw new Error(`gallery: ${DIR}/${g.name} is no file the catalogue makes.`)
      return code
    },
  }
}
