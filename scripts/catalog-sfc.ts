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
 *   a `<slot :name="…">` is a dynamic slot, its name the expression. In
 *   `defineSlots`, an index signature is a dynamic slot named by its key's
 *   pattern, `<key>` standing for the part named at run time
 *   (`` [cell: `${string}-cell`] `` is `<key>-cell`). A slot's props are the
 *   members of its function's first parameter, through aliases and unions.
 * - Options from a plain `<script>` beside `<script setup>` are read too,
 *   when they are object literals in that script. Options it cannot read (an
 *   imported object, a call, defaults that are not an object literal) stop
 *   the build with one sentence, never an empty list.
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

/** A prop as the compiled runtime declaration gives it. */
export interface RuntimeProp {
  name: string
  required: boolean
  /** The default's source text. */
  default: string | null
  /** The runtime type's source text (`String`, `[Number, Object]`). */
  runtimeType: string | null
}

/**
 * The runtime props and emits in the code Vue compiled an SFC to (`content`):
 * names, required, defaults. The component options may be an object, wrapped
 * in `defineVaporComponent(…)`, merged with `Object.assign(__default__, …)`
 * (a plain `<script>` beside `<script setup>`, in JavaScript) or with
 * `...__default__` (the same, in TypeScript); a name is followed to the
 * object literal it is declared with in the same code. Anything else (an
 * imported value, a call, defaults that are not an object literal) stops with
 * one sentence naming `file`.
 */
export function compiledOptions(content: string, file: string): { props: RuntimeProp[]; emits: string[] } {
  const ast = sfc.babelParse(content, { sourceType: 'module', plugins: ['typescript'] }) as unknown as { program: Node }
  const body = children(ast.program, 'body')
  const props: RuntimeProp[] = []
  const emits: string[] = []
  const src = (n: Node) => content.slice(n.start!, n.end!)
  const unreadable = () =>
    new Error(
      `${file} sets component options the catalogue cannot read; declare props, emits and slots in <script setup>, or as object literals in a plain <script>.`,
    )
  const callee = (n: Node) => {
    const c = child(n, 'callee') as (Node & { name?: string }) | undefined
    if (c?.type === 'MemberExpression') {
      const o = child(c, 'object') as Node & { name?: string }
      const k = child(c, 'property') as Node & { name?: string }
      return `${o.name ?? ''}.${k.name ?? ''}`
    }
    return c?.name ?? ''
  }

  /**
   * What `n` stands for: a name declared in this code is followed to its
   * value. A name declared elsewhere (an import, a global such as `String`)
   * is refused, unless `loose`, which gives the name back as it is.
   */
  function resolveNode(n: Node, loose = false, depth = 0): Node {
    if (n.type === 'TSAsExpression' || n.type === 'TSSatisfiesExpression' || n.type === 'ParenthesizedExpression') {
      return resolveNode(child(n, 'expression')!, loose, depth)
    }
    if (n.type !== 'Identifier' || depth > 5) return n
    const name = (n as Node & { name: string }).name
    for (const st of body) {
      if (st.type !== 'VariableDeclaration') continue
      for (const d of children(st, 'declarations')) {
        if ((child(d, 'id') as Node & { name?: string }).name === name && child(d, 'init')) return resolveNode(child(d, 'init')!, loose, depth + 1)
      }
    }
    if (loose) return n
    throw unreadable()
  }

  /** An object expression's properties, with spreads of resolvable object literals opened. */
  const fields = (n: Node): Node[] =>
    children(n, 'properties').flatMap((f) => {
      if (f.type !== 'SpreadElement') return [f]
      const arg = resolveNode(child(f, 'argument')!)
      if (arg.type !== 'ObjectExpression') throw unreadable()
      return fields(arg)
    })
  const byKey = (n: Node | undefined, k: string) => (n?.type === 'ObjectExpression' ? fields(n).find((f) => keyName(f) === k) : undefined)

  /** The options object's properties, later ones overriding earlier ones. */
  function options(n: Node): Node[] {
    const node = resolveNode(n)
    if (node.type === 'ObjectExpression') return fields(node)
    if (node.type === 'CallExpression') {
      const name = callee(node)
      // Vue's wrappers and the merge with a plain <script>'s options.
      if (/define(Vapor)?Component$/.test(name)) return options(children(node, 'arguments')[0] ?? node)
      if (name === 'Object.assign') return children(node, 'arguments').flatMap(options)
    }
    throw unreadable()
  }

  /** Props from a runtime declaration, with defaults from any mergeDefaults around it. */
  function readProps(value: Node, defaults: Node[] = []) {
    const node = resolveNode(value)
    if (node.type === 'CallExpression' && /mergeModels$/.test(callee(node))) {
      for (const part of children(node, 'arguments')) readProps(part, defaults)
    } else if (node.type === 'CallExpression' && /mergeDefaults$/.test(callee(node))) {
      const [declared, given] = children(node, 'arguments')
      const literal = given && resolveNode(given)
      if (!declared || literal?.type !== 'ObjectExpression') throw unreadable()
      readProps(declared, [literal, ...defaults])
    } else if (node.type === 'ArrayExpression') {
      for (const e of children(node, 'elements')) {
        if (e.type !== 'StringLiteral') throw unreadable()
        props.push({ name: (e as Node & { value: string }).value, required: false, default: null, runtimeType: null })
      }
    } else if (node.type === 'ObjectExpression') {
      for (const prop of fields(node)) {
        const name = keyName(prop)
        if (!name) throw unreadable()
        if (name.endsWith('Modifiers')) continue
        const opts = resolveNode(child(prop, 'value')!, true)
        // `size: String` is a type alone; `{ type, required, default }` the full form.
        const full = opts.type === 'ObjectExpression'
        const type = full ? byKey(opts, 'type') : undefined
        const required = full ? byKey(opts, 'required') : undefined
        const given = defaults.map((d) => byKey(d, name)).find(Boolean)
        const def = given ?? (full ? byKey(opts, 'default') : undefined)
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
    const node = resolveNode(value)
    if (node.type === 'CallExpression' && /mergeModels$/.test(callee(node))) {
      for (const part of children(node, 'arguments')) readEmits(part)
    } else if (node.type === 'ArrayExpression') {
      for (const e of children(node, 'elements')) {
        if (e.type !== 'StringLiteral') throw unreadable()
        emits.push((e as Node & { value: string }).value)
      }
    } else if (node.type === 'ObjectExpression') {
      for (const f of fields(node)) {
        const name = keyName(f)
        if (!name) throw unreadable()
        emits.push(name)
      }
    } else throw unreadable()
  }

  const exp = body.find((st) => st.type === 'ExportDefaultDeclaration')
  if (!exp) return { props, emits }
  const opts = options(child(exp, 'declaration')!)
  const last = (k: string) => [...opts].reverse().find((p) => keyName(p) === k)
  const propsNode = last('props')
  const emitsNode = last('emits')
  if (propsNode) readProps(child(propsNode, 'value') ?? propsNode)
  if (emitsNode) readEmits(child(emitsNode, 'value') ?? emitsNode)
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

/** The names of a slot's props: the members of its first parameter's type. */
function paramProps(param: Node | undefined, scope: Scope): string[] {
  const ann = param && child(child(param, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
  return ann ? members(ann, scope).map((x) => keyName(x.node)).filter((k): k is string => !!k) : []
}

/** The props of a slot whose type is `t`: a function type, through parentheses, a union (`… | undefined`) or an alias. */
function slotProps(t: Node | undefined, scope: Scope, depth = 0): string[] {
  if (!t || depth > 8) return []
  if (t.type === 'TSFunctionType') return paramProps(children(t, 'parameters')[0], scope)
  if (t.type === 'TSParenthesizedType') return slotProps(child(t, 'typeAnnotation'), scope, depth + 1)
  if (t.type === 'TSUnionType') {
    for (const u of children(t, 'types')) {
      const found = slotProps(u, scope, depth + 1)
      if (found.length) return found
    }
    return []
  }
  if (t.type === 'TSTypeReference') {
    const found = findAlias((child(t, 'typeName') as Node & { name?: string }).name ?? '', scope)
    if (found?.node.type === 'TSTypeAliasDeclaration') return slotProps(child(found.node, 'typeAnnotation'), found.scope, depth + 1)
  }
  return []
}

/**
 * The name pattern of an index-signature slot, from its key's type, with
 * `<key>` for the part named at run time: `string` is `<key>`, and
 * `` `${string}-cell` `` is `<key>-cell`.
 */
function slotPattern(key: Node, scope: Scope): string {
  if (key.type === 'TSStringKeyword') return '<key>'
  const literal = key.type === 'TSLiteralType' ? child(key, 'literal') : undefined
  if (literal?.type === 'TemplateLiteral') {
    const quasis = children(literal, 'quasis').map((q) => (q as Node & { value: { cooked: string } }).value.cooked)
    return quasis.map((q, i) => (i ? '<key>' : '') + q).join('')
  }
  return squash(typeText(key, scope))
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
  const runtime = compiledOptions(block.content, rel)
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
    slots = members(slotsType, scope).flatMap(({ node: m, scope: s }): CatalogSlot[] => {
      const value = child(child(m, 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
      // `[cell: \`${string}-cell\`]: …`: slots named at run time, by a pattern.
      if (m.type === 'TSIndexSignature') {
        const key = child(child(children(m, 'parameters')[0] ?? ({} as Node), 'typeAnnotation') ?? ({} as Node), 'typeAnnotation')
        const pattern = key && slotPattern(key, s)
        return pattern ? [{ name: pattern, props: slotProps(value, s), dynamic: true }] : []
      }
      const name = keyName(m)
      if (!name) return []
      // A method (`row?(props: …): unknown`) or a property holding a function type.
      const props = m.type === 'TSMethodSignature' ? paramProps(children(m, 'parameters')[0], s) : slotProps(value, s)
      return [{ name, props, dynamic: false }]
    })
  } else {
    slots = templateSlots(descriptor.template?.ast as unknown as TemplateNode | undefined)
  }
  return { props, emits, slots }
}

