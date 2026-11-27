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
 * 3. Its own stylesheet, `packages/ui/src/styles/<name>.css` in kebab-case
 *    (`FormField.vue` → `form-field.css`), imported at the end of the SFC's
 *    imports after `'../styles/base.css'` and any shared sheet it draws with
 *    (`tone.css`, `choice.css`), and added to its group's sheet (`forms.css`;
 *    a new group's sheet goes in `weft.css`). A component with none says
 *    "It has no stylesheet of its own" in its docs. A page then ships only
 *    the CSS of the components it renders. The comment at the top of
 *    `base.css` says how in full, with the cascade rule between sheets;
 *    `packages/ui/test/stylesheets.test.ts` checks it. (Not read by this
 *    script.)
 *
 * Props, defaults, emits and slots come from the SFC itself, through
 * `compileScript` from `vue/compiler-sfc` (the pinned `vue` package's own);
 * `scripts/catalog-sfc.ts` reads them and its header says how. A
 * `/** … *\/` comment on a prop is its description.
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
 *           "slots": [ { "name": "header", "props": ["item"], "dynamic": false } ],
 *                                            template order; a dynamic slot's name is its expression
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
import { basename, dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { readComponent, type CatalogEmit, type CatalogProp, type CatalogSlot } from './catalog-sfc.ts'

export { readComponent, type CatalogEmit, type CatalogProp, type CatalogSlot }


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
  const purposeLen = Array.from(top.purpose!).length
  if (purposeLen > 80) {
    throw new Error(`${file} has a purpose of ${purposeLen} characters; shorten purpose to at most 80 characters so weft catalog stays compact.`)
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
      ...readComponent(file, rel(file)),
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
