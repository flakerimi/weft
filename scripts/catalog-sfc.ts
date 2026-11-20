/**
 * The SFC reader of the component catalogue (`scripts/catalog.ts`): the
 * props, emits and slots of one component, read with `compileScript` from
 * `vue/compiler-sfc` (the pinned `vue` package's own, loaded from
 * packages/ui).
 *
 * - Prop names, `required` and defaults (as source text) come from the
 *   runtime declaration Vue compiles the SFC to, through `mergeModels`
 *   (defineModel) and `mergeDefaults` (defaults destructured from runtime
 *   props).
 * - TypeScript types and descriptions (a JSDoc comment on a prop) come from
 *   `defineProps<…>()`: a type literal, an interface (with what it extends),
 *   an alias, or an intersection of those. A type alias from the same file
 *   or a relative import is written out. A prop whose type the reader cannot
 *   find keeps the runtime type (`string | unknown[]`).
 * - Emits come from the compiled list, with payloads from `defineEmits<…>()`
 *   or implied by `defineModel`.
 * - Slots come from `defineSlots<…>()`, else from the template's `<slot>`s;
 *   a `<slot :name="…">` is a dynamic slot, its name the expression.
 * - Options set in a plain `<script>` (`export default { props }`) cannot be
 *   read and stop the build with one sentence.
 */
import { existsSync, readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { basename, dirname, resolve } from 'node:path'
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
  /** The slot's name; for a dynamic slot, the expression that names it. */
  name: string
  props: string[]
  /** Named at run time (`<slot :name="…">`). */
  dynamic: boolean
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

/** A member of an object type, with the scope its source text is in. */
interface Member {
  node: Node
  scope: Scope
}

/**
 * The members of an object type: a literal, an interface (its own members,
 * then those it extends), an alias of one, or an intersection of those. A
 * part the reader cannot resolve (a type from a package) adds no members.
 */
function members(n: Node, scope: Scope, depth = 0): Member[] {
  if (depth > 8) return []
  const own = (nodes: Node[], s: Scope) => nodes.map((node) => ({ node, scope: s }))
  if (n.type === 'TSTypeLiteral') return own(children(n, 'members'), scope)
  if (n.type === 'TSIntersectionType') return children(n, 'types').flatMap((t) => members(t, scope, depth + 1))
  if (n.type === 'TSParenthesizedType') return members(child(n, 'typeAnnotation')!, scope, depth + 1)
  const name =
    n.type === 'TSTypeReference' ? (child(n, 'typeName') as Node & { name?: string }).name
    : n.type === 'TSExpressionWithTypeArguments' || n.type === 'TSInterfaceHeritage' ? (child(n, 'expression') as Node & { name?: string }).name
    : undefined
  const found = name ? findAlias(name, scope) : null
  if (found?.node.type === 'TSInterfaceDeclaration') {
    const body = own(children(child(found.node, 'body')!, 'body'), found.scope)
    const inherited = children(found.node, 'extends').flatMap((h) => members(h, found.scope, depth + 1))
    return [...body, ...inherited]
  }
  if (found?.node.type === 'TSTypeAliasDeclaration') return members(child(found.node, 'typeAnnotation')!, found.scope, depth + 1)
  return []
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
function compiled(content: string, file: string) {
  const ast = sfc.babelParse(content, { sourceType: 'module', plugins: ['typescript'] }) as unknown as { program: Node }
  const exp = children(ast.program, 'body').find((s) => s.type === 'ExportDefaultDeclaration')
  let decl = exp && child(exp, 'declaration')
  if (decl?.type === 'CallExpression') decl = children(decl, 'arguments')[0]
  const props: { name: string; required: boolean; default: string | null; runtimeType: string | null }[] = []
  const emits: string[] = []
  if (decl?.type !== 'ObjectExpression') return { props, emits }
  const src = (n: Node) => content.slice(n.start!, n.end!)
  const unreadable = () =>
    new Error(`${file} sets component options in a plain <script> the catalogue cannot read; declare props, emits and slots in <script setup>.`)
  const callee = (n: Node) => (child(n, 'callee') as Node & { name?: string } | undefined)?.name ?? ''
  /** An object expression's properties, with spread object literals (`...{ default: false }`) opened. */
  const fields = (n: Node): Node[] =>
    children(n, 'properties').flatMap((f) =>
      f.type === 'SpreadElement' && child(f, 'argument')?.type === 'ObjectExpression' ? fields(child(f, 'argument')!) : [f],
    )
  const byKey = (n: Node | undefined, k: string) => (n?.type === 'ObjectExpression' ? fields(n).find((f) => keyName(f) === k) : undefined)

  /** Props from a runtime declaration; `defaults` from mergeDefaults' second argument. */
  function readProps(value: Node, defaults?: Node) {
    if (value.type === 'CallExpression' && /mergeModels$/.test(callee(value))) {
      for (const part of children(value, 'arguments')) readProps(part)
    } else if (value.type === 'CallExpression' && /mergeDefaults$/.test(callee(value))) {
      const [declared, given] = children(value, 'arguments')
      if (declared) readProps(declared, given)
    } else if (value.type === 'ArrayExpression') {
      for (const e of children(value, 'elements')) {
        if (e.type === 'StringLiteral') props.push({ name: (e as Node & { value: string }).value, required: false, default: null, runtimeType: null })
      }
    } else if (value.type === 'ObjectExpression') {
      for (const prop of fields(value)) {
        const name = keyName(prop)
        if (!name) throw unreadable()
        if (name.endsWith('Modifiers')) continue
        const opts = child(prop, 'value')!
        // `size: String` is a type alone; `{ type, required, default }` the full form.
        const full = opts.type === 'ObjectExpression'
        const type = full ? byKey(opts, 'type') : undefined
        const required = full ? byKey(opts, 'required') : undefined
        const def = byKey(defaults, name) ?? (full ? byKey(opts, 'default') : undefined)
        props.push({
          name,
          required: required ? src(child(required, 'value')!) === 'true' : false,
          default: def ? squash(src(child(def, 'value')!)) : null,
          runtimeType: full ? (type ? squash(src(child(type, 'value')!)) : null) : squash(src(opts)),
        })
      }
    } else throw unreadable()
  }

  function readEmits(value: Node) {
    if (value.type === 'CallExpression' && /mergeModels$/.test(callee(value))) {
      for (const part of children(value, 'arguments')) readEmits(part)
    } else if (value.type === 'ArrayExpression') {
      for (const e of children(value, 'elements')) if (e.type === 'StringLiteral') emits.push((e as Node & { value: string }).value)
    } else if (value.type === 'ObjectExpression') {
      for (const f of fields(value)) {
        const name = keyName(f)
        if (name) emits.push(name)
      }
    } else throw unreadable()
  }

  for (const p of children(decl, 'properties')) {
    // Options spread in from a plain <script>'s export default.
    if (p.type === 'SpreadElement') throw unreadable()
    const key = keyName(p)
    const value = child(p, 'value')
    if (key === 'props' && value) readProps(value)
    else if (key === 'emits' && value) readEmits(value)
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
  props?: { type: number; name: string; value?: { content: string }; arg?: { content?: string; isStatic?: boolean }; exp?: { content?: string } }[]
  children?: TemplateNode[]
}

function templateSlots(node: TemplateNode | undefined, out: CatalogSlot[] = []): CatalogSlot[] {
  if (!node) return out
  if (node.type === 1 && node.tag === 'slot') {
    let name = 'default'
    let dynamic = false
    const props: string[] = []
    for (const p of node.props ?? []) {
      if (p.type === 6 && p.name === 'name') name = p.value?.content ?? 'default'
      else if (p.type === 6) props.push(p.name)
      else if (p.type === 7 && p.name === 'bind' && p.arg?.content === 'name' && p.arg.isStatic) {
        // `<slot :name="…">`: the slot's name is made at run time.
        name = squash(p.exp?.content ?? '')
        dynamic = true
      } else if (p.type === 7 && p.name === 'bind' && p.arg?.content) props.push(p.arg.content)
    }
    if (!out.some((s) => s.name === name)) out.push({ name, props, dynamic })
  }
  for (const c of node.children ?? []) templateSlots(c, out)
  return out
}

/** What a component takes and gives: its props, emits and slots. */
export interface ComponentApi {
  props: CatalogProp[]
  emits: CatalogEmit[]
  slots: CatalogSlot[]
}

/** Props, emits and slots of the SFC at `file`; `rel` names it in the sentences it throws. */
export function readComponent(file: string, rel = file): ComponentApi {
  const source = readFileSync(file, 'utf8')
  const { descriptor, errors } = sfc.parse(source, { filename: file })
  if (errors.length) throw new Error(`${rel} does not parse: ${errors[0]!.message}`)
  if (!descriptor.script && !descriptor.scriptSetup) {
    return { props: [], emits: [], slots: templateSlots(descriptor.template?.ast as unknown as TemplateNode | undefined) }
  }
  const block = sfc.compileScript(descriptor, {
    id: basename(file),
    fs: { fileExists: existsSync, readFile: (p: string) => readFileSync(p, 'utf8') },
  })
  const runtime = compiled(block.content, rel)
  const setup = (block.scriptSetupAst ?? []) as unknown as Node[]
  const scope = descriptor.scriptSetup
    ? scopeOf(descriptor.scriptSetup.content, file, [...((block.scriptAst ?? []) as unknown as Node[]), ...setup])
    : null

  // TypeScript types and descriptions from defineProps<…>().
  const typed = new Map<string, { type: string; description: string | null }>()
  const propsType = scope && calls(setup, 'defineProps').map(typeArg).find(Boolean)
  if (scope && propsType) {
    for (const { node: m, scope: s } of members(propsType, scope)) {
      const name = keyName(m)
      const ann = child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
      // The first declaration wins: an interface's own member over the one it extends.
      if (name && !typed.has(name)) typed.set(name, { type: ann ? typeText(ann, s) : 'unknown', description: jsdoc(m) })
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
    for (const { node: m, scope: s } of members(emitsType, scope)) {
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
    slots = members(slotsType, scope).flatMap(({ node: m, scope: s }) => {
      const name = keyName(m)
      if (!name) return []
      const param = children(m, 'parameters')[0] ?? children(child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation') ?? ({} as Node), 'parameters')[0]
      const ann = param && child(child(param, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
      const names = ann ? members(ann, s).map((x) => keyName(x.node)).filter((k): k is string => !!k) : []
      return [{ name, props: names, dynamic: false }]
    })
  } else {
    slots = templateSlots(descriptor.template?.ast as unknown as TemplateNode | undefined)
  }
  return { props, emits, slots }
}

