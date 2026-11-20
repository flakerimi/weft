/**
 * The component catalogue: `catalog/components.json`, generated from
 * @weft/ui's components, their docs files and their examples. It is the one
 * source the `/components` gallery, `weft catalog` and the MCP server read.
 *
 *     node scripts/catalog.ts           write catalog/components.json
 *     node scripts/catalog.ts --check   exit 1 if it is out of date
 *
 * (`pnpm catalog` runs the first.) `pnpm --filter @weft/ui test` fails when
 * the file on disk is not what this script would write.
 *
 * ## What makes a component
 *
 * Every `.vue` file under `packages/ui/src/<group>/` is a component, named by
 * its file (`Button.vue` is `Button`). Each needs, or the script stops with
 * one sentence saying what is missing:
 *
 * 1. `<Name>.docs.md` beside it, starting with this frontmatter (every key
 *    required, no others; a value is one line, and a line indented deeper
 *    than its key continues it):
 *
 *        ---
 *        group: layout                 the folder under src/ the component is in
 *        purpose: What it is, in one sentence.
 *        use: When to reach for it.
 *        avoid: When to use something else, and what.
 *        adapts:
 *          input: What changes between pointer and touch.
 *          surface: What changes between landing and app.
 *          width: What changes between narrow, regular and wide.
 *        ---
 *        Optional Markdown after it: notes, keyboard, accessibility.
 *
 * 2. Examples in `packages/ui/examples/<Name>/`: each a `<script setup vapor
 *    lang="ts">` SFC that imports from `@weft/ui` as an app would. File names
 *    are kebab-case and name the example (`with-actions.vue` is "With
 *    actions"); `basic.vue`, the minimal example, is required and comes first.
 *    A PascalCase `.vue` file in the folder is a helper component the examples
 *    import, not an example.
 *
 * Props, defaults, emits and slots come from the SFC itself, through
 * `compileScript` from `vue/compiler-sfc` (the pinned `vue` package's own):
 * prop names, `required` and defaults from the compiled runtime declaration;
 * TypeScript types and descriptions from `defineProps<…>()` (a `/** … *\/`
 * comment on a prop is its description; a type alias from the same file or a
 * relative import is written out); emits from `defineEmits<…>()` or
 * `defineModel`; slots from `defineSlots<…>()`, else from the template's
 * `<slot>` elements.
 *
 * ## The JSON
 *
 *     {
 *       "$comment": string,                  generated, do not edit
 *       "components": [                      sorted by name
 *         {
 *           "name": "Card",
 *           "group": "layout",
 *           "file": "packages/ui/src/layout/Card.vue",
 *           "purpose": string, "use": string, "avoid": string,
 *           "adapts": { "input": string, "surface": string, "width": string },
 *           "props": [                       in declaration order
 *             { "name": "padding", "type": "0 | 2 | 3 | 4 | 6",
 *               "required": false, "default": "4" | null,   default as source text
 *               "description": string | null }
 *           ],
 *           "emits": [ { "name": "update:modelValue", "payload": "[value: string]" | null,
 *                        "description": string | null } ],
 *           "slots": [ { "name": "header", "props": ["item"] } ],   template order
 *           "notes": string,                 the docs file's Markdown body, trimmed
 *           "examples": [                    basic first, then by name
 *             { "name": "basic", "title": "Basic",
 *               "file": "packages/ui/examples/Card/basic.vue", "source": string }
 *           ]
 *         }
 *       ]
 *     }
 */
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { basename, dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

/** The part of vue/compiler-sfc this script uses (its own types do not resolve from scripts/). */
interface CompilerSFC {
  parse(
    source: string,
    opts: { filename: string },
  ): {
    descriptor: { script: { content: string } | null; scriptSetup: { content: string } | null; template: { ast?: unknown } | null }
    errors: { message: string }[]
  }
  compileScript(
    descriptor: unknown,
    opts: { id: string; fs: { fileExists(file: string): boolean; readFile(file: string): string | undefined } },
  ): { content: string; scriptAst?: unknown[]; scriptSetupAst?: unknown[] }
  babelParse(code: string, opts: { sourceType: 'module'; plugins: string[] }): unknown
}

// The compiler of the vue that @weft/ui pins, whatever root is being read.
const sfc = createRequire(fileURLToPath(new URL('../packages/ui/package.json', import.meta.url)))(
  'vue/compiler-sfc',
) as CompilerSFC

export interface CatalogProp {
  name: string
  type: string
  required: boolean
  default: string | null
  description: string | null
}
export interface CatalogEmit {
  name: string
  payload: string | null
  description: string | null
}
export interface CatalogSlot {
  name: string
  props: string[]
}
export interface CatalogExample {
  name: string
  title: string
  file: string
  source: string
}
export interface CatalogComponent {
  name: string
  group: string
  file: string
  purpose: string
  use: string
  avoid: string
  adapts: { input: string; surface: string; width: string }
  props: CatalogProp[]
  emits: CatalogEmit[]
  slots: CatalogSlot[]
  notes: string
  examples: CatalogExample[]
}
export interface Catalog {
  $comment: string
  components: CatalogComponent[]
}

const COMMENT = 'Generated by scripts/catalog.ts from packages/ui; do not edit. Run pnpm catalog.'
const KEYS = ['group', 'purpose', 'use', 'avoid', 'adapts'] as const
const AXES = ['input', 'surface', 'width'] as const
const AXIS_WORDS = { input: 'pointer and touch', surface: 'landing and app', width: 'narrow, regular and wide' }

/** The catalogue's path, from the repository root. */
export const CATALOG_FILE = 'catalog/components.json'

const posix = (p: string) => p.split('\\').join('/')

// ---------------------------------------------------------------- docs files

interface Docs {
  group: string
  purpose: string
  use: string
  avoid: string
  adapts: { input: string; surface: string; width: string }
  notes: string
}

/** Read a `<Name>.docs.md`; `file` is its repository-relative path, for the sentences. */
export function parseDocs(text: string, file: string): Docs {
  const m = /^---\r?\n([\s\S]*?)\r?\n---[ \t]*(?:\r?\n|$)([\s\S]*)$/.exec(text)
  if (!m) throw new Error(`${file} has no frontmatter; start it with a --- block holding ${KEYS.join(', ')}.`)
  const top: Record<string, string> = {}
  const adapts: Record<string, string> = {}
  // The entry the next deeper-indented line continues, and the indent of adapts' axes.
  let last: { into: Record<string, string>; key: string; indent: number } | null = null
  let axisIndent: number | null = null
  m[1]!.split(/\r?\n/).forEach((line, i) => {
    if (!line.trim()) return
    const where = `${file} line ${i + 2}`
    const indent = /^ */.exec(line)![0].length
    const kv = /^([A-Za-z]+):(?:\s+(.*))?$/.exec(line.trim())
    if (indent === 0) {
      if (!kv) throw new Error(`${where} is not "key: value"; write each frontmatter entry on its own line.`)
      const key = kv[1]!
      const value = (kv[2] ?? '').trim()
      if (!(KEYS as readonly string[]).includes(key)) {
        throw new Error(`${file} has the unknown key "${key}" in its frontmatter; the keys are ${KEYS.join(', ')}.`)
      }
      if (key in top) throw new Error(`${file} gives ${key} twice in its frontmatter.`)
      if (key === 'adapts' && value) throw new Error(`${file} gives adapts a value; list input, surface and width under it, indented.`)
      top[key] = value
      last = { into: top, key, indent }
      axisIndent = null
      return
    }
    const underAdapts = last !== null && (last.into === adapts || last.key === 'adapts')
    if (underAdapts && kv && (axisIndent === null || indent === axisIndent)) {
      const axis = kv[1]!
      if (!(AXES as readonly string[]).includes(axis)) {
        throw new Error(`${file} has the unknown axis "${axis}" under adapts; the axes are input, surface and width.`)
      }
      axisIndent = indent
      adapts[axis] = (kv[2] ?? '').trim()
      last = { into: adapts, key: axis, indent }
      return
    }
    if (!last || indent <= last.indent || (last.key === 'adapts' && last.into === top)) {
      throw new Error(`${where} is indented but continues no entry; indent a long value under its key, deeper than the key.`)
    }
    last.into[last.key] = `${last.into[last.key]} ${line.trim()}`.trim()
  })
  for (const key of KEYS) {
    if (!(key in top)) throw new Error(`${file} is missing ${key} in its frontmatter.`)
    if (key !== 'adapts' && !top[key]) throw new Error(`${file} gives ${key} no text.`)
  }
  for (const axis of AXES) {
    if (!adapts[axis]) {
      throw new Error(`${file} is missing adapts.${axis}; say what the component does between ${AXIS_WORDS[axis]}, even if nothing changes.`)
    }
  }
  return {
    group: top.group!,
    purpose: top.purpose!,
    use: top.use!,
    avoid: top.avoid!,
    adapts: { input: adapts.input!, surface: adapts.surface!, width: adapts.width! },
    notes: m[2]!.trim(),
  }
}

// ---------------------------------------------------------------- SFC reading

// Babel AST nodes, as far as this script looks at them.
interface Node {
  type: string
  start?: number | null
  end?: number | null
  [key: string]: unknown
}

const children = (n: Node, key: string) => (n[key] as Node[] | undefined) ?? []
const child = (n: Node, key: string) => n[key] as Node | undefined

/** Where a node's source text lives: a script's content and where type aliases come from. */
interface Scope {
  source: string
  file: string
  aliases: Map<string, Node>
  imports: Map<string, { from: string; name: string }>
}

const squash = (s: string) => s.replace(/\s+/g, ' ').trim()
const text = (n: Node, scope: Scope) => scope.source.slice(n.start!, n.end!)

function scopeOf(source: string, file: string, body: Node[]): Scope {
  const aliases = new Map<string, Node>()
  const imports = new Map<string, { from: string; name: string }>()
  for (let st of body) {
    if ((st.type === 'ExportNamedDeclaration' || st.type === 'ExportDefaultDeclaration') && child(st, 'declaration')) {
      st = child(st, 'declaration')!
    }
    if (st.type === 'TSTypeAliasDeclaration' || st.type === 'TSInterfaceDeclaration') {
      aliases.set((child(st, 'id') as Node & { name: string }).name, st)
    } else if (st.type === 'ImportDeclaration') {
      const from = (child(st, 'source') as Node & { value: string }).value
      if (!from.startsWith('.')) continue
      for (const spec of children(st, 'specifiers')) {
        if (spec.type !== 'ImportSpecifier') continue
        const imported = child(spec, 'imported') as Node & { name?: string; value?: string }
        imports.set((child(spec, 'local') as Node & { name: string }).name, { from, name: imported.name ?? imported.value! })
      }
    }
  }
  return { source, file, aliases, imports }
}

const scopes = new Map<string, Scope | null>()

/** The scope of a module a type is imported from: a `.ts` file, or a `.vue` file's scripts. */
function moduleScope(file: string): Scope | null {
  if (scopes.has(file)) return scopes.get(file) ?? null
  let scope: Scope | null = null
  if (existsSync(file)) {
    let source = readFileSync(file, 'utf8')
    if (file.endsWith('.vue')) {
      const { descriptor } = sfc.parse(source, { filename: file })
      source = [descriptor.script?.content, descriptor.scriptSetup?.content].filter(Boolean).join('\n')
    }
    const ast = sfc.babelParse(source, { sourceType: 'module', plugins: ['typescript'] }) as unknown as { program: Node }
    scope = scopeOf(source, file, children(ast.program, 'body'))
  }
  scopes.set(file, scope)
  return scope
}

/** A type alias named `name` visible in `scope`, and the scope its text is in. */
function findAlias(name: string, scope: Scope): { node: Node; scope: Scope } | null {
  const local = scope.aliases.get(name)
  if (local) return { node: local, scope }
  const imp = scope.imports.get(name)
  if (!imp) return null
  const target = moduleScope(resolve(dirname(scope.file), imp.from))
  return target ? findAlias(imp.name, target) : null
}

/** A type as source text, with aliases from this file or relative imports written out. */
function typeText(n: Node, scope: Scope, depth = 0): string {
  if (depth > 5) return squash(text(n, scope))
  if (n.type === 'TSUnionType') return children(n, 'types').map((t) => typeText(t, scope, depth + 1)).join(' | ')
  if (n.type === 'TSTypeReference' && !child(n, 'typeParameters') && child(n, 'typeName')?.type === 'Identifier') {
    const found = findAlias((child(n, 'typeName') as Node & { name: string }).name, scope)
    if (found?.node.type === 'TSTypeAliasDeclaration') return typeText(child(found.node, 'typeAnnotation')!, found.scope, depth + 1)
    if (found?.node.type === 'TSInterfaceDeclaration') return squash(text(child(found.node, 'body')!, found.scope))
  }
  return squash(text(n, scope))
}

/** The members of an object type: a literal, or an interface or alias naming one. */
function members(n: Node, scope: Scope): { nodes: Node[]; scope: Scope } {
  if (n.type === 'TSTypeLiteral') return { nodes: children(n, 'members'), scope }
  if (n.type === 'TSTypeReference' && child(n, 'typeName')?.type === 'Identifier') {
    const found = findAlias((child(n, 'typeName') as Node & { name: string }).name, scope)
    if (found?.node.type === 'TSInterfaceDeclaration') return { nodes: children(child(found.node, 'body')!, 'body'), scope: found.scope }
    if (found?.node.type === 'TSTypeAliasDeclaration') return members(child(found.node, 'typeAnnotation')!, found.scope)
  }
  return { nodes: [], scope }
}

const keyName = (m: Node) => {
  const key = child(m, 'key') as (Node & { name?: string; value?: string }) | undefined
  return key ? (key.name ?? key.value ?? null) : null
}

/** The text of a member's `/** … *\/` comment, or null. */
function jsdoc(m: Node): string | null {
  const comments = (m.leadingComments as { type: string; value: string }[] | undefined) ?? []
  const doc = [...comments].reverse().find((c) => c.type === 'CommentBlock' && c.value.startsWith('*'))
  if (!doc) return null
  return squash(doc.value.replace(/^\*/, '').replace(/^\s*\* ?/gm, '')) || null
}

/** Every call expression in `body` whose callee is the identifier `name`. */
function calls(body: Node[], name: string): Node[] {
  const out: Node[] = []
  // compileScript annotates nodes with links back up the tree (`_ownerScope`
  // and the like); only the AST's own fields are walked, each node once.
  const seen = new WeakSet<object>()
  const visit = (n: unknown) => {
    if (!n || typeof n !== 'object' || seen.has(n)) return
    seen.add(n)
    if (Array.isArray(n)) return n.forEach(visit)
    const node = n as Node
    if (node.type === 'CallExpression' && (child(node, 'callee') as Node & { name?: string })?.name === name) out.push(node)
    for (const [k, v] of Object.entries(node)) {
      if (!k.startsWith('_') && !['leadingComments', 'trailingComments', 'innerComments', 'loc', 'extra'].includes(k)) visit(v)
    }
  }
  visit(body)
  return out
}

const typeArg = (call: Node) => children(child(call, 'typeParameters') ?? ({} as Node), 'params')[0]

/** The runtime props and emits Vue compiled the SFC to: names, required, defaults. */
function compiled(content: string) {
  const ast = sfc.babelParse(content, { sourceType: 'module', plugins: ['typescript'] }) as unknown as { program: Node }
  const exp = children(ast.program, 'body').find((s) => s.type === 'ExportDefaultDeclaration')
  let decl = exp && child(exp, 'declaration')
  if (decl?.type === 'CallExpression') decl = children(decl, 'arguments')[0]
  const props: { name: string; required: boolean; default: string | null; runtimeType: string | null }[] = []
  const emits: string[] = []
  if (decl?.type !== 'ObjectExpression') return { props, emits }
  const src = (n: Node) => content.slice(n.start!, n.end!)
  // With defineModel, Vue merges the model's props and emits in: mergeModels(a, b).
  const parts = (n: Node | undefined): Node[] =>
    n?.type === 'CallExpression' && /mergeModels$/.test((child(n, 'callee') as Node & { name?: string }).name ?? '')
      ? children(n, 'arguments').flatMap(parts)
      : n ? [n] : []
  /** An object expression's properties, with spread object literals (`...{ default: false }`) opened. */
  const fields = (n: Node): Node[] =>
    children(n, 'properties').flatMap((f) =>
      f.type === 'SpreadElement' && child(f, 'argument')?.type === 'ObjectExpression' ? fields(child(f, 'argument')!) : [f],
    )
  for (const p of children(decl, 'properties')) {
    const key = keyName(p)
    for (const value of parts(child(p, 'value'))) {
      if (key === 'props' && value.type === 'ObjectExpression') {
        for (const prop of children(value, 'properties')) {
          const name = keyName(prop)
          if (!name || name.endsWith('Modifiers')) continue
          const opts = child(prop, 'value')
          const field = (k: string) => (opts?.type === 'ObjectExpression' ? fields(opts).find((f) => keyName(f) === k) : undefined)
          const def = field('default')
          const type = field('type')
          const required = field('required')
          props.push({
            name,
            required: required ? src(child(required, 'value')!) === 'true' : false,
            default: def ? squash(src(child(def, 'value')!)) : null,
            runtimeType: type ? squash(src(child(type, 'value')!)) : null,
          })
        }
      } else if (key === 'emits' && value.type === 'ArrayExpression') {
        for (const e of children(value, 'elements')) if (e.type === 'StringLiteral') emits.push((e as Node & { value: string }).value)
      }
    }
  }
  return { props, emits }
}

const RUNTIME_TYPES: Record<string, string> = {
  String: 'string', Number: 'number', Boolean: 'boolean', Object: 'object', Array: 'unknown[]', Function: 'Function', null: 'unknown',
}
const fromRuntime = (t: string | null) =>
  !t ? 'unknown' : t.replace(/^\[|\]$/g, '').split(/\s*,\s*/).map((x) => RUNTIME_TYPES[x] ?? x).join(' | ')

interface TemplateNode {
  type: number
  tag?: string
  props?: { type: number; name: string; value?: { content: string }; arg?: { content?: string } }[]
  children?: TemplateNode[]
}

function templateSlots(node: TemplateNode | undefined, out: CatalogSlot[] = []): CatalogSlot[] {
  if (!node) return out
  if (node.type === 1 && node.tag === 'slot') {
    let name = 'default'
    const props: string[] = []
    for (const p of node.props ?? []) {
      if (p.type === 6 && p.name === 'name') name = p.value?.content ?? 'default'
      else if (p.type === 6) props.push(p.name)
      else if (p.type === 7 && p.name === 'bind' && p.arg?.content) props.push(p.arg.content)
    }
    if (!out.some((s) => s.name === name)) out.push({ name, props })
  }
  for (const c of node.children ?? []) templateSlots(c, out)
  return out
}

/** Props, emits and slots of one SFC. */
export function readComponent(file: string): Pick<CatalogComponent, 'props' | 'emits' | 'slots'> {
  const source = readFileSync(file, 'utf8')
  const { descriptor, errors } = sfc.parse(source, { filename: file })
  if (errors.length) throw new Error(`${file} does not parse: ${errors[0]!.message}`)
  if (!descriptor.script && !descriptor.scriptSetup) {
    return { props: [], emits: [], slots: templateSlots(descriptor.template?.ast as unknown as TemplateNode | undefined) }
  }
  const block = sfc.compileScript(descriptor, {
    id: basename(file),
    fs: { fileExists: existsSync, readFile: (p: string) => readFileSync(p, 'utf8') },
  })
  const runtime = compiled(block.content)
  const setup = (block.scriptSetupAst ?? []) as unknown as Node[]
  const scope = descriptor.scriptSetup
    ? scopeOf(descriptor.scriptSetup.content, file, [...((block.scriptAst ?? []) as unknown as Node[]), ...setup])
    : null

  // TypeScript types and descriptions from defineProps<…>().
  const typed = new Map<string, { type: string; description: string | null }>()
  const propsType = scope && calls(setup, 'defineProps').map(typeArg).find(Boolean)
  if (scope && propsType) {
    const { nodes, scope: s } = members(propsType, scope)
    for (const m of nodes) {
      const name = keyName(m)
      const ann = child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
      if (name) typed.set(name, { type: ann ? typeText(ann, s) : 'unknown', description: jsdoc(m) })
    }
  }
  if (scope) {
    for (const call of calls(setup, 'defineModel')) {
      const first = children(call, 'arguments')[0] as (Node & { value?: string }) | undefined
      const name = first?.type === 'StringLiteral' ? first.value! : 'modelValue'
      const t = typeArg(call)
      typed.set(name, { type: t ? typeText(t, scope) : 'unknown', description: null })
    }
  }
  const props: CatalogProp[] = runtime.props.map((p) => ({
    name: p.name,
    type: typed.get(p.name)?.type ?? fromRuntime(p.runtimeType),
    required: p.required,
    default: p.default,
    description: typed.get(p.name)?.description ?? null,
  }))

  // Payloads and descriptions from defineEmits<…>().
  const payloads = new Map<string, { payload: string | null; description: string | null }>()
  const emitsType = scope && calls(setup, 'defineEmits').map(typeArg).find(Boolean)
  if (scope && emitsType) {
    const { nodes, scope: s } = members(emitsType, scope)
    for (const m of nodes) {
      if (m.type === 'TSPropertySignature') {
        const ann = child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
        payloads.set(keyName(m)!, { payload: ann ? typeText(ann, s) : null, description: jsdoc(m) })
      } else if (m.type === 'TSCallSignatureDeclaration') {
        const [event, ...rest] = children(m, 'parameters')
        const lit = child(child(child(event ?? ({} as Node), 'typeAnnotation') ?? ({} as Node), 'typeAnnotation') ?? ({} as Node), 'literal') as
          | (Node & { value?: string })
          | undefined
        if (lit?.value) payloads.set(lit.value, { payload: rest.length ? `[${rest.map((r) => squash(text(r, s))).join(', ')}]` : null, description: jsdoc(m) })
      }
    }
  }
  const emits: CatalogEmit[] = runtime.emits.map((name) => ({
    name,
    payload: payloads.get(name)?.payload ?? (typed.has(name.replace(/^update:/, '')) ? `[value: ${typed.get(name.replace(/^update:/, ''))!.type}]` : null),
    description: payloads.get(name)?.description ?? null,
  }))

  // Slots: defineSlots<…>() when the SFC declares them, else the template's <slot>s.
  let slots: CatalogSlot[] = []
  const slotsType = scope && calls(setup, 'defineSlots').map(typeArg).find(Boolean)
  if (scope && slotsType) {
    const { nodes, scope: s } = members(slotsType, scope)
    slots = nodes.flatMap((m) => {
      const name = keyName(m)
      if (!name) return []
      const param = children(m, 'parameters')[0] ?? children(child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation') ?? ({} as Node), 'parameters')[0]
      const ann = param && child(child(param, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
      const names = ann ? members(ann, s).nodes.map(keyName).filter((k): k is string => !!k) : []
      return [{ name, props: names }]
    })
  } else {
    slots = templateSlots(descriptor.template?.ast as unknown as TemplateNode | undefined)
  }
  return { props, emits, slots }
}

// ---------------------------------------------------------------- examples

const EXAMPLE = /^[a-z0-9]+(?:-[a-z0-9]+)*\.vue$/
const HELPER = /^[A-Z][A-Za-z0-9]*\.vue$/

function readExamples(root: string, name: string): CatalogExample[] {
  const dir = join(root, 'packages/ui/examples', name)
  const rel = (f: string) => posix(relative(root, join(dir, f)))
  const files = existsSync(dir) ? readdirSync(dir).filter((f) => f.endsWith('.vue')).sort() : []
  for (const f of files) {
    if (!EXAMPLE.test(f) && !HELPER.test(f)) {
      throw new Error(`${rel(f)} is named like neither an example (kebab-case, as basic.vue) nor a helper component (PascalCase).`)
    }
  }
  const examples = files.filter((f) => EXAMPLE.test(f))
  if (!examples.includes('basic.vue')) {
    throw new Error(`${name} has no basic example; add ${posix(relative(root, join(dir, 'basic.vue')))}, its minimal example.`)
  }
  examples.sort((a, b) => (a === 'basic.vue' ? -1 : b === 'basic.vue' ? 1 : a.localeCompare(b)))
  return examples.map((f) => {
    const stem = f.slice(0, -4)
    const words = stem.replace(/-/g, ' ')
    return { name: stem, title: words[0]!.toUpperCase() + words.slice(1), file: rel(f), source: readFileSync(join(dir, f), 'utf8') }
  })
}

// ---------------------------------------------------------------- the whole

function walk(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
    e.isDirectory() ? walk(join(dir, e.name)) : [join(dir, e.name)],
  )
}

/** Build the catalogue from the repository at `root`, or throw the sentence that says what is missing. */
export function buildCatalog(root: string): Catalog {
  const src = join(root, 'packages/ui/src')
  const rel = (f: string) => posix(relative(root, f))
  const files = walk(src)
  const byName = new Map<string, string>()
  for (const file of files.filter((f) => f.endsWith('.vue')).sort()) {
    const name = basename(file, '.vue')
    const other = byName.get(name)
    if (other) throw new Error(`${rel(other)} and ${rel(file)} are both named ${name}; component names must be unique.`)
    byName.set(name, file)
  }
  for (const doc of files.filter((f) => f.endsWith('.docs.md'))) {
    const name = basename(doc, '.docs.md')
    if (byName.get(name) !== join(dirname(doc), `${name}.vue`)) {
      throw new Error(`${rel(doc)} has no ${name}.vue beside it; name it after the component it documents.`)
    }
  }
  const examplesDir = join(root, 'packages/ui/examples')
  if (existsSync(examplesDir)) {
    for (const e of readdirSync(examplesDir, { withFileTypes: true })) {
      if (e.isDirectory() && !byName.has(e.name)) {
        throw new Error(`${rel(join(examplesDir, e.name))} holds examples for ${e.name}, which is not a component; name the folder after the component it shows.`)
      }
    }
  }

  const components: CatalogComponent[] = []
  for (const [name, file] of [...byName].sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))) {
    const docsFile = join(dirname(file), `${name}.docs.md`)
    if (!existsSync(docsFile)) {
      throw new Error(`${rel(file)} has no ${name}.docs.md beside it; add one with ${KEYS.join(', ')} in its frontmatter.`)
    }
    const docs = parseDocs(readFileSync(docsFile, 'utf8'), rel(docsFile))
    const folder = posix(relative(src, dirname(file))).split('/')[0]!
    if (docs.group !== folder) {
      throw new Error(`${rel(docsFile)} says group ${docs.group}, but ${name} is in src/${folder}; make them match.`)
    }
    components.push({
      name,
      group: docs.group,
      file: rel(file),
      purpose: docs.purpose,
      use: docs.use,
      avoid: docs.avoid,
      adapts: docs.adapts,
      ...readComponent(file),
      notes: docs.notes,
      examples: readExamples(root, name),
    })
  }
  return { $comment: COMMENT, components }
}

/** The catalogue as the file holds it. */
export const serialise = (c: Catalog) => JSON.stringify(c, null, 2) + '\n'

// Run as a script (`node scripts/catalog.ts`), not imported by the test.
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = fileURLToPath(new URL('..', import.meta.url))
  const out = join(root, CATALOG_FILE)
  const fresh = serialise(buildCatalog(root))
  if (process.argv.includes('--check')) {
    const current = existsSync(out) ? readFileSync(out, 'utf8') : ''
    if (current !== fresh) {
      console.error(`${CATALOG_FILE} is out of date; run pnpm catalog.`)
      process.exit(1)
    }
  } else {
    writeFileSync(out, fresh)
    console.log(`wrote ${CATALOG_FILE}`)
  }
}
