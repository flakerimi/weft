/**
 * weftAutoImport(): write the imports of Weft's components for the author.
 *
 * A page or component that writes `<Button>` and does not import it gets
 * `import { Button } from '@weft/ui'` at the top of its `<script setup>`, so
 * it compiles, renders and hydrates exactly as if the line were written by
 * hand: the server render follows the named import through `@weft/ui`'s index
 * to Button's CSS, and the client build tree-shakes the rest. A file with no
 * script gets `<script setup vapor lang="ts">` to hold the line.
 *
 * The names are the components `@weft/ui`'s index exports (`export { default
 * as Button } from './basics/Button.vue'`). A tag gets an import only when:
 * - it is PascalCase and Vue's parser takes it for a component, so `<button>`
 *   and anything inside a `v-pre` element are left as they are;
 * - it is one of those names, and not the file's own name as written
 *   (`<Card>` inside `Card.vue` is the component itself, as Vue resolves it;
 *   inside the page `card.vue` it is Weft's Card, where Vue alone would draw
 *   the page inside itself);
 * - neither the `<script setup>` nor a plain `<script>` beside it imports or
 *   declares the name: the author's own `Card` stays theirs.
 * A component with only a plain `<script>` registers its own components, and
 * is left alone, as is a file the Vue compiler would refuse (it says why).
 * Files under `node_modules` and `@weft/ui`'s own source are not touched.
 *
 * With `dts` (by default `<root>/.weft-types/components.d.ts`), the plugin
 * also writes a declaration that tells TypeScript and the editor what each
 * name is, rewritten only when its content changes. A Weft app's tsconfig
 * includes it. `writeComponentTypes(root)`, which the `weft-types` bin runs,
 * writes the same file with no build, so a fresh app typechecks before its
 * first build or dev run.
 *
 * `siteConfig` adds this plugin to every page build and render server; an
 * app built with plain Vite adds `weftAutoImport()` to its plugins.
 */
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join, resolve } from 'node:path'
import MagicString, { type SourceMap } from 'magic-string'
import { normalizePath, type Plugin } from 'vite'
import { babelParse, parse, type SFCScriptBlock } from 'vue/compiler-sfc'

export interface AutoImportOptions {
  /** Where to write the components' declaration file, from the root; `false` for none. */
  dts?: string | false
}

const SOURCE = '@weft/ui'
const PASCAL = /^[A-Z][A-Za-z0-9]*$/
/** compiler-core's NodeTypes.ELEMENT and ElementTypes.COMPONENT. */
const ELEMENT = 1
const COMPONENT = 1

/**
 * The tag that names an SFC itself: its file name without `.vue`, letter for
 * letter. Vue also takes `<Card>` in `card.vue` for the file; that is left out
 * here, since a page's name is its lowercase route, never a component's.
 */
const ownName = (file: string) => /([^/\\]+)\.vue$/.exec(file)?.[1]

// The parts of Babel's AST read here; `vue/compiler-sfc` ships the parser, not its types.
interface Node {
  type: string
  [key: string]: unknown
}

/**
 * The names `@weft/ui`'s index exports as components: `export { default as
 * Name } from './….vue'`, in the order the index has them.
 */
export function componentNames(indexSource: string): string[] {
  const ast = babelParse(indexSource, { sourceType: 'module', plugins: ['typescript'] }) as unknown as { program: { body: Node[] } }
  const out: string[] = []
  for (const stmt of ast.program.body) {
    if (stmt.type !== 'ExportNamedDeclaration') continue
    const from = (stmt.source as { value?: string } | null)?.value
    if (!from?.endsWith('.vue') || stmt.exportKind === 'type') continue
    for (const spec of stmt.specifiers as Node[]) {
      if (spec.type !== 'ExportSpecifier' || spec.exportKind === 'type') continue
      const local = spec.local as { name?: string; value?: string }
      const exported = spec.exported as { name?: string; value?: string }
      const name = exported.name ?? exported.value
      if ((local.name ?? local.value) === 'default' && name && PASCAL.test(name) && !out.includes(name)) out.push(name)
    }
  }
  return out
}

/** The names a binding pattern declares: `a`, `{ a, b: c }`, `[a, ...b]`, `a = 1`. */
function patternNames(node: Node | null | undefined, out: Set<string>) {
  if (!node) return
  switch (node.type) {
    case 'Identifier':
      out.add(node.name as string)
      break
    case 'ObjectPattern':
      for (const p of node.properties as Node[]) patternNames(p.type === 'RestElement' ? p : (p.value as Node), out)
      break
    case 'ArrayPattern':
      for (const e of node.elements as (Node | null)[]) patternNames(e, out)
      break
    case 'RestElement':
      patternNames(node.argument as Node, out)
      break
    case 'AssignmentPattern':
      patternNames(node.left as Node, out)
      break
  }
}

/** The top-level names a script block imports or declares; `null` when it does not parse. */
function scriptNames(block: SFCScriptBlock): Set<string> | null {
  const lang = block.lang ?? 'js'
  const plugins: ('typescript' | 'jsx')[] = []
  if (/^tsx?$/.test(lang)) plugins.push('typescript')
  if (/x$/.test(lang)) plugins.push('jsx')
  let body: Node[]
  try {
    body = (babelParse(block.content, { sourceType: 'module', plugins }) as unknown as { program: { body: Node[] } }).program.body
  } catch {
    return null
  }
  const out = new Set<string>()
  const declare = (decl: Node | null | undefined) => {
    if (!decl) return
    if (decl.type === 'VariableDeclaration') {
      for (const d of decl.declarations as Node[]) patternNames(d.id as Node, out)
    } else if (decl.id && (decl.id as Node).type === 'Identifier') {
      // A function, class, enum or namespace.
      out.add((decl.id as { name: string }).name)
    }
  }
  for (const stmt of body) {
    if (stmt.type === 'ImportDeclaration') {
      for (const spec of stmt.specifiers as Node[]) out.add((spec.local as { name: string }).name)
    } else if (stmt.type === 'ExportNamedDeclaration' || stmt.type === 'ExportDefaultDeclaration') {
      declare(stmt.declaration as Node | null)
    } else declare(stmt)
  }
  return out
}

interface TemplateNode {
  type: number
  tag?: string
  tagType?: number
  children?: TemplateNode[]
}

/** The PascalCase component tags of a template, each once. */
function componentTags(node: TemplateNode, out: Set<string>) {
  // Inside `v-pre`, Vue's parser makes every element a plain one: it compiles none of them.
  if (node.type === ELEMENT && node.tagType === COMPONENT && node.tag && PASCAL.test(node.tag)) out.add(node.tag)
  for (const child of node.children ?? []) componentTags(child, out)
}

/**
 * For an SFC's source, the same source with one `import { … } from
 * '@weft/ui'` for each component tag in `names` it uses and does not import
 * or declare itself, at the top of its `<script setup>` (one it gains when it
 * has no script); `null` when it needs none. `filename`, when given, keeps
 * the file's own name for itself.
 */
export function autoImport(code: string, names: Set<string>, filename?: string): { code: string; map: SourceMap } | null {
  const { descriptor, errors } = parse(code, { filename: filename ?? 'anonymous.vue', sourceMap: false })
  // A file Vue refuses: its plugin says why, in its own words.
  if (errors.length || !descriptor.template?.ast) return null
  const { script, scriptSetup } = descriptor
  // Without `<script setup>`, the component registers its own components.
  if (script && !scriptSetup) return null

  const tags = new Set<string>()
  componentTags(descriptor.template.ast as unknown as TemplateNode, tags)
  const own = new Set<string>()
  const self = filename && ownName(filename)
  if (self) own.add(self)
  for (const block of [script, scriptSetup]) {
    if (!block) continue
    const declared = scriptNames(block)
    if (!declared) return null
    for (const n of declared) own.add(n)
  }
  const wanted = [...tags].filter((t) => names.has(t) && !own.has(t)).sort()
  if (!wanted.length) return null

  const line = `import { ${wanted.join(', ')} } from '${SOURCE}'`
  const s = new MagicString(code)
  // On a line of its own: the block's code may start on its opening tag's line.
  if (scriptSetup) s.appendLeft(scriptSetup.loc.start.offset, `\n${line}${/^\r?\n/.test(scriptSetup.content) ? '' : '\n'}`)
  else s.prepend(`<script setup vapor lang="ts">\n${line}\n</script>\n\n`)
  return { code: s.toString(), map: s.generateMap({ source: filename, includeContent: true, hires: true }) }
}

/** Where the declaration file goes, from the root, unless `dts` says otherwise. */
const DTS = '.weft-types/components.d.ts'

/** `@weft/ui`'s component names and its index's folder, resolved from `root` as a site's own imports are; null when the site has no @weft/ui. */
function loadComponents(root: string): { names: Set<string>; dir: string } | null {
  let index: string
  try {
    index = createRequire(join(root, 'package.json')).resolve(SOURCE)
  } catch {
    return null
  }
  // Vite's module ids are posix paths, on Windows too.
  return { names: new Set(componentNames(readFileSync(index, 'utf8'))), dir: normalizePath(dirname(index)) + '/' }
}

interface RouteParamEntry {
  pattern: string
  params: string[]
}

/**
 * The dynamic page patterns under `<root>/pages` (`/products/:id`, `/docs/:...slug`)
 * and their parameter names, in sorted order, for `WeftRouteParams` in
 * `.weft-types/components.d.ts`.
 */
function routeParamEntries(root: string): RouteParamEntry[] {
  const pagesDir = join(root, 'pages')
  if (!existsSync(pagesDir)) return []
  const files: string[] = []
  const walk = (dir: string, prefix: string) => {
    let entries
    try {
      entries = readdirSync(dir, { withFileTypes: true })
    } catch {
      return
    }
    for (const e of entries) {
      if (e.name.startsWith('_') || e.name.startsWith('.') || e.isSymbolicLink()) continue
      if (e.isDirectory()) walk(join(dir, e.name), `${prefix}${e.name}/`)
      else if (e.isFile() && e.name.endsWith('.vue')) files.push(`${prefix}${e.name}`)
    }
  }
  walk(pagesDir, '')
  const byPattern = new Map<string, string[]>()
  for (const file of files.sort()) {
    const parts = file.slice(0, -'.vue'.length).split('/')
    if (parts[parts.length - 1] === 'index') parts.pop()
    const rest = parts.findIndex((p) => /^\[\.\.\..+\]$/.test(p))
    if (rest !== -1 && rest !== parts.length - 1) continue
    const params: string[] = []
    const segs = parts.map((p) => {
      const m = /^\[(\.\.\.)?(.+)\]$/.exec(p)
      if (!m) return p
      params.push(m[2]!)
      return `:${m[1] ?? ''}${m[2]!}`
    })
    if (!params.length) continue
    const pattern = '/' + segs.join('/')
    if (!byPattern.has(pattern)) byPattern.set(pattern, params)
  }
  return [...byPattern.entries()].sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)).map(([pattern, params]) => ({ pattern, params }))
}

const tsProp = (s: string) => (/^[A-Za-z_$][A-Za-z0-9_$]*$/.test(s) ? s : JSON.stringify(s))

/** The declaration file: every name, as the component `@weft/ui` exports, and the dynamic page route patterns for `useParams()`. */
function declaration(names: string[], routes: RouteParamEntry[] = []): string {
  const lines = [...names].sort().map((n) => `    ${n}: typeof import('${SOURCE}')['${n}']`)
  const routeLines = routes.map(
    ({ pattern, params }) => `    ${JSON.stringify(pattern).replace(/^"|"$/g, "'")}: { ${params.map((p) => `${tsProp(p)}: string`).join('; ')} }`,
  )
  return [
    "// Written by @weft/vite's weftAutoImport() on each build: do not edit.",
    'export {}',
    '',
    "declare module 'vue' {",
    '  interface GlobalComponents {',
    ...lines,
    '  }',
    '}',
    ...(routeLines.length
      ? ['', `declare module '${SOURCE}' {`, '  interface WeftRouteParams {', ...routeLines, '  }', '}']
      : []),
    '',
  ].join('\n')
}

/** Write `text` to `file` unless it holds it already, so an editor or watcher sees no change. */
function writeIfChanged(file: string, text: string) {
  try {
    if (readFileSync(file, 'utf8') === text) return
  } catch {
    // Not there yet.
  }
  mkdirSync(dirname(file), { recursive: true })
  writeFileSync(file, text)
}

/**
 * Write the declaration file for the site at `root`, as the plugin does on
 * each build, and return its path; null, writing nothing, when `@weft/ui`
 * cannot be resolved from the root.
 */
export function writeComponentTypes(root: string, dts: string = DTS): string | null {
  // createRequire needs an absolute path; a root may be given from the working folder.
  root = resolve(root)
  const ui = loadComponents(root)
  if (!ui) return null
  const file = resolve(root, dts)
  writeIfChanged(file, declaration([...ui.names], routeParamEntries(root)))
  return file
}

export function weftAutoImport(opts: AutoImportOptions = {}): Plugin {
  let root = process.cwd()
  /** The component names, and the folder of the index that exports them; read once. */
  let found: { names: Set<string>; dir: string } | null | undefined

  /** `@weft/ui`'s components, read once per configuration. */
  function load(): { names: Set<string>; dir: string } | null {
    if (found === undefined) found = loadComponents(root)
    return found
  }

  function refreshTypes() {
    const ui = load()
    if (ui && opts.dts !== false) writeIfChanged(resolve(root, opts.dts ?? DTS), declaration([...ui.names], routeParamEntries(root)))
  }

  return {
    name: 'weft:auto-import',
    // Before vite:vue compiles the SFC, which is what reads the line.
    enforce: 'pre',
    configResolved(config) {
      root = config.root
      found = undefined
    },
    buildStart() {
      refreshTypes()
    },
    configureServer(server) {
      server.watcher.on('add', (f) => f.endsWith('.vue') && refreshTypes())
      server.watcher.on('unlink', (f) => f.endsWith('.vue') && refreshTypes())
    },
    transform(code, id) {
      if (!id.endsWith('.vue') || id.startsWith('\0') || id.includes('/node_modules/')) return
      const ui = load()
      // @weft/ui's components import one another by file, never through the index.
      if (!ui || id.startsWith(ui.dir)) return
      return autoImport(code, ui.names, id)
    },
  }
}
