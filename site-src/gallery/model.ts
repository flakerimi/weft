// The /components gallery's model: what the pages and the islands plugin read
// from catalog/components.json, as plain functions of it, so a component
// added to @weft/ui (its catalogue entry and examples) appears in the gallery
// with no page code of its own.

/** A component as catalog/components.json describes it (scripts/catalog.ts writes the file). */
export interface CatalogEntry {
  name: string
  group: string
  file: string
  purpose: string
  use: string
  avoid: string
  adapts: { input: string; surface: string; width: string }
  props: { name: string; type: string; required: boolean; default: string | null; description: string }[]
  emits: { name: string; payload: string; description: string }[]
  slots: { name: string; props: { name: string; type: string }[]; dynamic: boolean }[]
  examples: { name: string; title: string; file: string; source: string }[]
  notes: string
}

export interface Catalog {
  components: CatalogEntry[]
}

/** The groups in the order the gallery shows them, with their headings; a group not named here follows, under its own name. */
const GROUPS: [string, string][] = [
  ['layout', 'Layout'],
  ['basics', 'Basics'],
  ['forms', 'Forms'],
  ['selection', 'Selection'],
  ['dates', 'Dates'],
  ['overlays', 'Overlays'],
  ['navigation', 'Navigation'],
  ['disclosure', 'Disclosure'],
  ['feedback', 'Feedback'],
  ['data', 'Data'],
  ['shell', 'Shell'],
  ['adaptive', 'Adaptive'],
  ['islands', 'Islands'],
]

/** A group's heading: its name in GROUPS, or the folder's name capitalised. */
export const groupLabel = (group: string) =>
  GROUPS.find(([g]) => g === group)?.[1] ?? group.charAt(0).toUpperCase() + group.slice(1)

/** The components by group, the groups in GROUPS' order (others after, by name), each group's components by name. */
export function groupsOf<T extends { name: string; group: string }>(components: T[]): { group: string; label: string; components: T[] }[] {
  const rank = (g: string) => {
    const i = GROUPS.findIndex(([name]) => name === g)
    return i === -1 ? GROUPS.length : i
  }
  const names = [...new Set(components.map((c) => c.group))].sort((a, b) => rank(a) - rank(b) || (a < b ? -1 : a > b ? 1 : 0))
  return names.map((group) => ({
    group,
    label: groupLabel(group),
    components: components.filter((c) => c.group === group).sort((a, b) => (a.name < b.name ? -1 : 1)),
  }))
}

/** A component's URL segment: its name in kebab case (DatePicker → date-picker). */
export const slugOf = (name: string) => name.replace(/([a-z0-9])([A-Z])/g, '$1-$2').replace(/([A-Z])([A-Z][a-z])/g, '$1-$2').toLowerCase()

const pascal = (kebab: string) => kebab.split('-').map((w) => w.charAt(0).toUpperCase() + w.slice(1)).join('')

/**
 * The names of the gallery's island files, which are their islands' ids
 * (`data-id`, the SFC `__name`): `Stage_<Name>` for a component's preview
 * stage, `Example_<Name>_<Example>` for one of its other examples. An
 * underscore keeps two apart that letters alone would join (Input's
 * tags-basic and InputTags's basic), and no hyphen, so no id is the start
 * of another's chunk name (`<id>-<hash>.js`).
 */
export const stageIsland = (component: string) => `Stage_${component}`
export const exampleIsland = (component: string, example: string) => `Example_${component}_${pascal(example)}`

/**
 * The components an example may draw and still be shown without JavaScript:
 * each does nothing in the browser that its server render does not already
 * show, when the example gives it no listener. Any other component, a
 * component added later included, makes its examples hydrate.
 */
export const STILL = new Set([
  'AvatarGroup', 'Badge', 'Breadcrumb', 'Button', 'Card', 'Chip', 'Container', 'Divider', 'FieldGroup',
  'Icon', 'Kbd', 'Page', 'Progress', 'Section', 'Separator', 'Skeleton', 'Spinner', 'Stack',
])

/**
 * Whether an example needs JavaScript to be what it shows: its script does
 * more than import, its template listens (`@`, `v-on`, `v-model`), or it
 * draws a component (any PascalCase tag) outside STILL.
 */
export function isInteractive(source: string): boolean {
  const script = /<script\b[^>]*>([\s\S]*?)<\/script>/.exec(source)?.[1] ?? ''
  const code = script
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '')
    .replace(/^\s*import\s[^;]*?from\s*['"][^'"]+['"];?\s*$/gm, '')
    .replace(/^\s*import\s*['"][^'"]+['"];?\s*$/gm, '')
  if (code.trim()) return true
  const template = /<template>([\s\S]*)<\/template>/.exec(source)?.[1] ?? ''
  const markup = template.replace(/<!--[\s\S]*?-->/g, '')
  if (/\s(?:@|v-on[:\s=]|v-model\b)/.test(markup)) return true
  return [...markup.matchAll(/<([A-Z][A-Za-z0-9]*)/g)].some((m) => !STILL.has(m[1]!))
}

/**
 * Refuse a catalogue the examples have moved on from: every example file
 * must be in it, each with the source the file has, and nothing it lists
 * may be gone. `files` maps each example file (from the repository root,
 * `packages/ui/examples/Button/basic.vue`) to its text; a PascalCase file
 * is a helper, not an example.
 */
export function checkFresh(catalog: Catalog, files: Record<string, string>): void {
  const stale = (why: string) =>
    new Error(`gallery: catalog/components.json is out of date (${why}); run pnpm catalog.`)
  const listed = new Map(catalog.components.flatMap((c) => c.examples.map((e) => [e.file, e] as const)))
  for (const [file, source] of Object.entries(files).sort()) {
    if (!/\/[a-z0-9-]+\.vue$/.test(file)) continue
    const entry = listed.get(file)
    if (!entry) throw stale(`it does not list ${file}`)
    if (entry.source !== source) throw stale(`${file} has changed since it was written`)
  }
  for (const file of listed.keys()) if (!(file in files)) throw stale(`it lists ${file}, which is gone`)
  for (const c of catalog.components) {
    if (!c.examples.length) throw stale(`${c.name} has no example`)
  }
}

/** The previous and next components in a component's group, by name. */
export function neighbours<T extends { name: string; group: string }>(components: T[], name: string): { prev?: T; next?: T } {
  const group = groupsOf(components).find((g) => g.components.some((c) => c.name === name))
  if (!group) return {}
  const i = group.components.findIndex((c) => c.name === name)
  return { prev: group.components[i - 1], next: group.components[i + 1] }
}

const escapeHTML = (s: string) => s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]!)

/** A catalogue sentence as HTML: escaped, with its `code` spans as <code>. */
export const inlineHTML = (text: string) => escapeHTML(text).replace(/`([^`]+)`/g, '<code>$1</code>')
