/**
 * weftIcons(): bundle exactly the Lucide icons a build uses, inlined as SVG.
 *
 * Weft names icons by string (`<Icon name="trash">`, `<Button icon="plus">`).
 * This plugin finds those names in each module, checks each against Lucide
 * (`lucide-static`), and gives the module one import per icon:
 * `import "virtual:weft/icon/trash"`. That virtual module registers the
 * drawing (the inner markup of Lucide's 24 × 24 svg) with `@weft/ui/icons`,
 * which `Icon` reads. So each chunk carries the icons its own modules use and
 * no others, and a page rendered on the server draws the same icons an
 * island hydrates with.
 *
 * Where names are found:
 * - In `.vue` templates: a static `name="…"` on `<Icon>`; a static `icon="…"`,
 *   or any attribute ending in `-icon` / `Icon`, on any component; and the
 *   string literals inside those attributes when bound
 *   (`:name="open ? 'chevron-up' : 'chevron-down'"`).
 * - In scripts (`.vue` script blocks and `.js`/`.ts` modules, comments
 *   aside): a property `icon: '…'` or `…Icon: '…'` whose string literal looks
 *   like an icon name (lowercase words joined by `-`), as in item lists
 *   (`{ label: 'Home', icon: 'house' }`) and `withDefaults`.
 * - Only the literal tag `Icon` is recognised for `name`: an import under
 *   another name (`import { Icon as Glyph }`), `<icon>` or
 *   `<component :is="Icon">` is not seen.
 * - Names built at run time cannot be seen either; list them in `icons`.
 *
 * A static name that Lucide does not have fails the build with one sentence
 * naming the file. A literal inside a bound expression that is not an icon
 * name is ignored, since the expression may hold other strings.
 * Modules under `node_modules` are not scanned, except `@weft/` packages.
 *
 * `siteConfig` adds this plugin to every page build and render server; an
 * app built with plain Vite (an SPA) adds `weftIcons()` to its plugins.
 */
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join, relative } from 'node:path'
import type { Plugin } from 'vite'
import { parse } from 'vue/compiler-sfc'

export interface IconsOptions {
  /** Icon names the scan cannot find because they are made at run time; bundled with `Icon`. */
  icons?: string[]
}

const PREFIX = 'virtual:weft/icon/'
const NAME = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
const SCRIPT = /\.[cm]?[jt]sx?$/
/** `icon: 'x'` or `trailingIcon: "x"` in a script. */
const SCRIPT_NAME = /(?:\bicon|[a-z0-9]Icon)\s*:\s*(['"`])([^'"`\n]*)\1/g
/** A string literal in a bound expression. */
const LITERAL = /(['"`])([^'"`\n]*)\1/g

let lucideDir: string | undefined
const bodies = new Map<string, string | undefined>()

/**
 * The inner markup of Lucide's svg for `name` (its paths and shapes, on one
 * line), or undefined when Lucide has no icon of that name.
 */
export function lucideBody(name: string): string | undefined {
  if (!NAME.test(name)) return undefined
  if (bodies.has(name)) return bodies.get(name)
  lucideDir ??= join(dirname(createRequire(import.meta.url).resolve('lucide-static/package.json')), 'icons')
  let body: string | undefined
  try {
    const svg = readFileSync(join(lucideDir, `${name}.svg`), 'utf8')
    body = /<svg\b[^>]*>([\s\S]*)<\/svg>/
      .exec(svg.replace(/<!--[\s\S]*?-->/g, ''))?.[1]
      ?.replace(/>\s+</g, '><')
      .replace(/\s*\/>/g, '/>')
      .trim()
  } catch {
    body = undefined
  }
  bodies.set(name, body)
  return body
}

const isComponent = (tag: string) => /^[A-Z]/.test(tag) || tag.includes('-')
const isIconProp = (name: string) => name === 'icon' || name.endsWith('-icon') || /[a-z0-9]Icon$/.test(name)

interface Found {
  /** Names that must exist: a missing one fails the build. */
  sure: Set<string>
  /** Literals that are icons only if Lucide has them. */
  maybe: Set<string>
}

interface TemplateNode {
  type: number
  tag?: string
  props?: { type: number; name: string; value?: { content: string }; arg?: { content?: string; isStatic?: boolean }; exp?: { content?: string } }[]
  children?: TemplateNode[]
}

function scanTemplate(node: TemplateNode, found: Found) {
  if (node.type === 1 && node.tag) {
    const tag = node.tag
    for (const p of node.props ?? []) {
      if (p.type === 6) {
        // A static attribute.
        if ((tag === 'Icon' && p.name === 'name') || (isComponent(tag) && isIconProp(p.name))) {
          if (p.value?.content) found.sure.add(p.value.content)
        }
      } else if (p.type === 7 && p.name === 'bind' && p.arg?.isStatic && p.arg.content && p.exp?.content) {
        const arg = p.arg.content
        if ((tag === 'Icon' && arg === 'name') || (isComponent(tag) && isIconProp(arg))) {
          for (const m of p.exp.content.matchAll(LITERAL)) found.maybe.add(m[2]!)
        }
      }
    }
  }
  for (const child of node.children ?? []) scanTemplate(child, found)
}

/** Comments, which may quote names in prose; a `//` inside a string costs the rest of its line only. */
const COMMENTS = /\/\*[\s\S]*?\*\/|\/\/[^\n]*/g

function scanScript(code: string, found: Found) {
  // A value that does not look like a name ('…', 'mdi:home') is some other string.
  for (const m of code.replace(COMMENTS, '').matchAll(SCRIPT_NAME)) if (NAME.test(m[2]!)) found.sure.add(m[2]!)
}

/** The sentence for a name Lucide lacks; `subject` says who named it. */
const missing = (subject: string, name: string) =>
  `${subject} "${name}", which Lucide does not have; pick a name from lucide.dev/icons.`

export function weftIcons(opts: IconsOptions = {}): Plugin[] {
  let root = process.cwd()
  /** Icon names each `.vue` file uses, from its source, for the post pass. */
  const vueNames = new Map<string, string[]>()
  /** Icon.vue's id, found from the registry's: where the `icons` option's names are imported. */
  let iconVue: string | undefined

  const scanned = (id: string) => {
    if (id.startsWith('\0') || id.includes('?')) return false
    const nm = id.lastIndexOf('/node_modules/')
    return nm < 0 || id.startsWith('@weft/', nm + '/node_modules/'.length)
  }

  /** The names found, checked: unknown sure names fail the build. */
  function check(found: Found, file: string, fail: (msg: string) => never): string[] {
    const names = new Set<string>()
    for (const name of found.sure) {
      if (!lucideBody(name)) fail(missing(`${relative(root, file).split('\\').join('/')} uses the icon`, name))
      names.add(name)
    }
    for (const name of found.maybe) if (lucideBody(name)) names.add(name)
    return [...names].sort()
  }

  const imports = (names: string[]) => names.map((n) => `\nimport ${JSON.stringify(PREFIX + n)};`).join('')

  const scan: Plugin = {
    name: 'weft:icons-scan',
    enforce: 'pre',
    configResolved(config) {
      root = config.root
    },
    buildStart() {
      for (const name of opts.icons ?? []) {
        if (!lucideBody(name)) this.error(missing('weftIcons(): the icons option lists', name))
      }
    },
    transform(code, id) {
      if (!id.endsWith('.vue') || !scanned(id)) return
      const { descriptor } = parse(code, { filename: id })
      const found: Found = { sure: new Set(), maybe: new Set() }
      if (descriptor.template?.ast) scanTemplate(descriptor.template.ast as unknown as TemplateNode, found)
      for (const block of [descriptor.script, descriptor.scriptSetup]) if (block) scanScript(block.content, found)
      vueNames.set(id, check(found, id, (msg) => this.error(msg)))
    },
  }

  const inject: Plugin = {
    name: 'weft:icons',
    enforce: 'post',
    resolveId(id) {
      if (id.startsWith(PREFIX)) return '\0' + id
    },
    load(id) {
      if (!id.startsWith('\0' + PREFIX)) return
      const name = id.slice(PREFIX.length + 1)
      const body = lucideBody(name)
      if (body === undefined) this.error(missing(`${PREFIX}${name} names the icon`, name))
      return `import { registerIcon } from "@weft/ui/icons";\nregisterIcon(${JSON.stringify(name)}, ${JSON.stringify(body)});\n`
    },
    async transform(code, id) {
      if (!scanned(id)) return
      let names: string[]
      if (id.endsWith('.vue')) {
        names = vueNames.get(id) ?? []
        if (opts.icons?.length) {
          // Names made at run time ride along with the component that draws them.
          if (!iconVue) {
            const registry = await this.resolve('@weft/ui/icons', id)
            if (registry) iconVue = join(dirname(registry.id), 'Icon.vue')
          }
          if (id === iconVue) names = [...new Set([...names, ...opts.icons])].sort()
        }
      } else if (SCRIPT.test(id)) {
        const found: Found = { sure: new Set(), maybe: new Set() }
        scanScript(code, found)
        names = check(found, id, (msg) => this.error(msg))
      } else return
      // Appended, not prepended: imports hoist, and the code above keeps its source map.
      if (names.length) return { code: code + imports(names), map: null }
    },
  }

  return [scan, inject]
}
