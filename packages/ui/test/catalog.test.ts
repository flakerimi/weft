// @vitest-environment node
// The catalogue (catalog/components.json) must match what scripts/catalog.ts
// builds from the components, their docs files and their examples.
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildCatalog, CATALOG_FILE, parseDocs, serialise, type Catalog, type CatalogComponent } from '../../../scripts/catalog.ts'
import { compiledOptions } from '../../../scripts/catalog-sfc.ts'

const repo = fileURLToPath(new URL('../../..', import.meta.url))
const temps: string[] = []
afterAll(() => temps.forEach((d) => rmSync(d, { recursive: true, force: true })))

const DOCS = `---
group: widgets
purpose: A thing.
use: When you need a thing.
avoid: When you need another thing.
adapts:
  input: Nothing changes.
  surface: Nothing changes.
  width: Nothing changes.
---
`

/** A minimal repository holding one component, Thing, with what `parts` gives it. */
function repoWith(parts: { docs?: string | null; examples?: Record<string, string>; sfc?: string }) {
  const root = mkdtempSync(join(tmpdir(), 'weft-catalog-'))
  temps.push(root)
  const src = join(root, 'packages/ui/src/widgets')
  mkdirSync(src, { recursive: true })
  writeFileSync(
    join(src, 'Thing.vue'),
    parts.sfc ?? '<script setup vapor lang="ts">\ndefineProps<{ on?: boolean }>()\n</script>\n<template><i><slot /></i></template>\n',
  )
  if (parts.docs !== null) writeFileSync(join(src, 'Thing.docs.md'), parts.docs ?? DOCS)
  const examples = parts.examples ?? { 'basic.vue': '<template><i /></template>\n' }
  for (const [file, source] of Object.entries(examples)) {
    mkdirSync(join(root, 'packages/ui/examples/Thing'), { recursive: true })
    writeFileSync(join(root, 'packages/ui/examples/Thing', file), source)
  }
  return root
}

describe('the catalogue', () => {
  let catalog: Catalog
  beforeAll(() => {
    catalog = buildCatalog(repo)
  })
  const byName = (name: string) => catalog.components.find((c) => c.name === name) as CatalogComponent

  it(`is up to date in ${CATALOG_FILE}; run pnpm catalog when this fails`, () => {
    expect(readFileSync(join(repo, CATALOG_FILE), 'utf8')).toBe(serialise(catalog))
  })

  it('lists every component, sorted, each with docs and a basic example first', () => {
    const names = catalog.components.map((c) => c.name)
    expect(names).toEqual([...names].sort())
    expect(names).toEqual(
      expect.arrayContaining(['Card', 'Divider', 'Grid', 'Icon', 'Island', 'Page', 'Section', 'Stack', 'WeftProvider']),
    )
    for (const c of catalog.components) {
      expect(c.purpose, c.name).not.toBe('')
      expect(c.examples[0]?.name, c.name).toBe('basic')
    }
  })

  it('reads props with their TypeScript types, defaults and descriptions', () => {
    const stack = byName('Stack')
    expect(stack.props.find((p) => p.name === 'gap')).toMatchObject({ type: '0 | 1 | 2 | 3 | 4 | 6 | 8', required: false, default: '4' })
    expect(stack.props.find((p) => p.name === 'direction')).toMatchObject({ type: "'column' | 'row'", default: "'column'" })
    const icon = byName('Icon')
    expect(icon.props.find((p) => p.name === 'name')).toMatchObject({ type: 'string', required: true, default: null })
    expect(icon.props.find((p) => p.name === 'label')?.description).toMatch(/assistive technology/)
    expect(byName('Island').props.find((p) => p.name === 'when')?.type).toBe("'load' | 'idle' | 'visible' | 'media'")
    expect(byName('Grid').props.find((p) => p.name === 'cols')?.default).toBe('() => ({ narrow: 1, regular: 2, wide: 3 })')
  })

  it('reads slots from the template, in order', () => {
    expect(byName('Card').slots.map((s) => s.name)).toEqual(['header', 'actions', 'default', 'footer'])
    expect(byName('Icon').slots).toEqual([])
  })

  it('carries the docs text and the example sources', () => {
    const card = byName('Card')
    expect(card.group).toBe('layout')
    expect(card.file).toBe('packages/ui/src/layout/Card.vue')
    expect(Object.keys(card.adapts)).toEqual(['input', 'surface', 'width'])
    expect(card.examples[0]).toMatchObject({ title: 'Basic', file: 'packages/ui/examples/Card/basic.vue' })
    expect(card.examples[0]!.source).toContain('<Card')
  })
})

describe('the catalogue build refuses', () => {
  it('a component without a docs file', () => {
    expect(() => buildCatalog(repoWith({ docs: null }))).toThrow(
      'packages/ui/src/widgets/Thing.vue has no Thing.docs.md beside it; add one with group, purpose, use, avoid, adapts in its frontmatter.',
    )
  })

  it('a component without a basic example', () => {
    expect(() => buildCatalog(repoWith({ examples: {} }))).toThrow(
      'Thing has no basic example; add packages/ui/examples/Thing/basic.vue, its minimal example.',
    )
    expect(() => buildCatalog(repoWith({ examples: { 'other.vue': '<template><i /></template>' } }))).toThrow(
      'Thing has no basic example',
    )
  })

  it('an example file named like neither an example nor a helper', () => {
    expect(() => buildCatalog(repoWith({ examples: { 'basic.vue': '', 'With_Actions.vue': '' } }))).toThrow(
      'packages/ui/examples/Thing/With_Actions.vue is named like neither an example (kebab-case, as basic.vue) nor a helper component (PascalCase).',
    )
  })

  it('a group that is not the folder the component is in', () => {
    expect(() => buildCatalog(repoWith({ docs: DOCS.replace('group: widgets', 'group: basics') }))).toThrow(
      'packages/ui/src/widgets/Thing.docs.md says group basics, but Thing is in src/widgets; make them match.',
    )
  })

  it('examples or docs for something that is not a component', () => {
    const root = repoWith({})
    cpSync(join(root, 'packages/ui/examples/Thing'), join(root, 'packages/ui/examples/Thingy'), { recursive: true })
    expect(() => buildCatalog(root)).toThrow(
      'packages/ui/examples/Thingy holds examples for Thingy, which is not a component; name the folder after the component it shows.',
    )
    const other = repoWith({})
    writeFileSync(join(other, 'packages/ui/src/widgets/Other.docs.md'), DOCS)
    expect(() => buildCatalog(other)).toThrow(
      'packages/ui/src/widgets/Other.docs.md has no Other.vue beside it; name it after the component it documents.',
    )
  })

  it('reads helpers beside the examples without listing them', () => {
    const root = repoWith({ examples: { 'basic.vue': 'b', 'with-icon.vue': 'w', 'Helper.vue': 'h' } })
    const [thing] = buildCatalog(root).components
    expect(thing!.examples.map((e) => [e.name, e.title])).toEqual([['basic', 'Basic'], ['with-icon', 'With icon']])
    expect(thing!.slots).toEqual([{ name: 'default', props: [], dynamic: false }])
    expect(thing!.props).toEqual([{ name: 'on', type: 'boolean', required: false, default: null, description: null }])
  })
})

describe('the catalogue reads', () => {
  it('emits with their payloads, v-model props and declared slots', () => {
    const sfc = `<script setup vapor lang="ts">
import type { Size } from './types.ts'
interface Props {
  /** The chosen value. */
  modelValue?: string
  size?: Size
}
withDefaults(defineProps<Props>(), { size: 'md' })
defineEmits<{
  /** The user picked a value. */
  'update:modelValue': [value: string]
  close: []
}>()
const open = defineModel<boolean>('open', { default: false })
defineSlots<{ item(props: { option: string; index: number }): unknown; empty(): unknown }>()
</script>
<template><i /></template>
`
    const root = repoWith({ sfc })
    writeFileSync(join(root, 'packages/ui/src/widgets/types.ts'), "export type Size = 'sm' | 'md' | 'lg'\n")
    const [thing] = buildCatalog(root).components
    expect(thing!.props).toEqual([
      { name: 'modelValue', type: 'string', required: false, default: null, description: 'The chosen value.' },
      { name: 'size', type: "'sm' | 'md' | 'lg'", required: false, default: "'md'", description: null },
      { name: 'open', type: 'boolean', required: false, default: 'false', description: null },
    ])
    expect(thing!.emits).toEqual([
      { name: 'update:modelValue', payload: '[value: string]', description: 'The user picked a value.' },
      { name: 'close', payload: '[]', description: null },
      { name: 'update:open', payload: '[value: boolean]', description: null },
    ])
    expect(thing!.slots).toEqual([
      { name: 'item', props: ['option', 'index'], dynamic: false },
      { name: 'empty', props: [], dynamic: false },
    ])
  })

  it('defaults given by destructuring runtime props (mergeDefaults)', () => {
    const sfc = `<script setup vapor lang="ts">
const { size = 'md', n = 2 } = defineProps({ size: String, n: { type: Number, required: true }, tags: [String, Array] })
</script>
<template><i /></template>
`
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.props).toEqual([
      { name: 'size', type: 'string', required: false, default: "'md'", description: null },
      { name: 'n', type: 'number', required: true, default: '2', description: null },
      { name: 'tags', type: 'string | unknown[]', required: false, default: null, description: null },
    ])
  })

  it('prop types through interface extends and intersections', () => {
    const sfc = `<script setup vapor lang="ts">
interface Base {
  /** From the base. */
  tone?: 'a' | 'b'
}
interface Props extends Base { size?: string }
type Extra = { open?: boolean }
defineProps<Props & Extra & { last: number }>()
</script>
<template><i /></template>
`
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.props.map((p) => [p.name, p.type, p.required, p.description])).toEqual([
      ['size', 'string', false, null],
      ['tone', "'a' | 'b'", false, 'From the base.'],
      ['open', 'boolean', false, null],
      ['last', 'number', true, null],
    ])
  })

  it('a slot with a bound name as a dynamic slot, not as a prop', () => {
    const sfc = '<script setup vapor lang="ts">\n</script>\n<template><div><slot v-for="c in 2" :name="`cell-${c}`" :value="c" /></div></template>\n'
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.slots).toEqual([{ name: '`cell-${c}`', props: ['value'], dynamic: true }])
  })

  it('index-signature slots in defineSlots as dynamic slot patterns, and slot props through aliases and unions', () => {
    const sfc = [
      '<script setup vapor lang="ts">',
      'type Row = (props: { item: string; index: number }) => unknown',
      'defineSlots<{',
      '  empty?: () => unknown',
      '  row?: Row',
      '  [cell: `${string}-cell`]: ((props: { row: unknown; value: unknown }) => unknown) | undefined',
      '  [name: string]: Row | undefined',
      '}>()',
      '</script>',
      '<template><i /></template>',
      '',
    ].join('\n')
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.slots).toEqual([
      { name: 'empty', props: [], dynamic: false },
      { name: 'row', props: ['item', 'index'], dynamic: false },
      { name: '<key>-cell', props: ['row', 'value'], dynamic: true },
      { name: '<key>', props: ['item', 'index'], dynamic: true },
    ])
  })

  it("lists Table's cell and header slots and Tabs' panel slots as patterns", () => {
    const catalog = buildCatalog(repo)
    const slots = (name: string) => catalog.components.find((c) => c.name === name)!.slots
    expect(slots('Table')).toEqual(
      expect.arrayContaining([
        { name: '<key>-cell', props: ['row', 'value', 'index', 'column'], dynamic: true },
        { name: '<key>-header', props: ['column'], dynamic: true },
      ]),
    )
    expect(slots('Tabs')).toContainEqual({ name: '<key>', props: ['item', 'index', 'selected'], dynamic: true })
    expect(slots('Stepper')).toContainEqual({ name: 'title', props: ['item', 'index', 'state'], dynamic: false })
  })

  it('options from a plain <script> beside <script setup>, with or without TypeScript', () => {
    const plain = (lang: string) => `<script${lang}>
const shared = { a: String }
export default { inheritAttrs: false, props: shared }
</script>
<script setup vapor${lang}>
</script>
<template><i /></template>
`
    for (const lang of [' lang="ts"', '']) {
      const [thing] = buildCatalog(repoWith({ sfc: plain(lang) })).components
      expect(thing!.props, lang || 'plain JavaScript').toEqual([
        { name: 'a', type: 'string', required: false, default: null, description: null },
      ])
    }
  })

  it('setup props beside a plain <script> that sets other options, with or without TypeScript', () => {
    const sfc = (lang: string, props: string) => `<script${lang}>
export default { inheritAttrs: false }
</script>
<script setup vapor${lang}>
${props}
</script>
<template><i /></template>
`
    const ts = buildCatalog(repoWith({ sfc: sfc(' lang="ts"', 'defineProps<{ a?: string }>()') })).components[0]!
    const js = buildCatalog(repoWith({ sfc: sfc('', 'defineProps({ a: { type: String, default: "x" } })') })).components[0]!
    expect(ts.props).toEqual([{ name: 'a', type: 'string', required: false, default: null, description: null }])
    expect(js.props).toEqual([{ name: 'a', type: 'string', required: false, default: '"x"', description: null }])
  })

  it('refuses, in one sentence, options it cannot read', () => {
    const sfc = `<script lang="ts">
import shared from './shared.ts'
export default { props: shared }
</script>
<script setup vapor lang="ts">
</script>
<template><i /></template>
`
    const root = repoWith({ sfc })
    writeFileSync(join(root, 'packages/ui/src/widgets/shared.ts'), 'export default { a: String }\n')
    expect(() => buildCatalog(root)).toThrow(
      'packages/ui/src/widgets/Thing.vue sets component options the catalogue cannot read; declare props, emits and slots in <script setup>, or as object literals in a plain <script>.',
    )
  })

  it('slot props from the template when slots are not declared', () => {
    const sfc = '<script setup vapor lang="ts">\n</script>\n<template><ul><slot name="row" :item="1" index="2" /><slot /></ul></template>\n'
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.slots).toEqual([
      { name: 'row', props: ['item', 'index'], dynamic: false },
      { name: 'default', props: [], dynamic: false },
    ])
  })
})

describe('the compiled options', () => {
  const read = (body: string) => compiledOptions(`export default {\n  __name: 'X',\n  ${body},\n}\n`, 'X.vue')

  it('keep destructured defaults through mergeModels inside mergeDefaults', () => {
    const { props } = read(`props: _mergeDefaults(_mergeModels({ size: String }, { open: { type: Boolean } }), { size: 'md', open: true })`)
    expect(props).toEqual([
      { name: 'size', required: false, default: "'md'", runtimeType: 'String' },
      { name: 'open', required: false, default: 'true', runtimeType: 'Boolean' },
    ])
  })

  it('refuse defaults that are not an object literal', () => {
    expect(() => read('props: _mergeDefaults({ size: String }, defaults)')).toThrow(
      'X.vue sets component options the catalogue cannot read; declare props, emits and slots in <script setup>, or as object literals in a plain <script>.',
    )
  })

  it('refuse a spread of anything but an object literal', () => {
    expect(() => read('...base(), props: { a: String }')).toThrow('X.vue sets component options the catalogue cannot read')
    expect(() => read('...imported, props: { a: String }')).toThrow('X.vue sets component options the catalogue cannot read')
  })

  it('refuse a default export that is not an object', () => {
    expect(() => compiledOptions('export default makeComponent()\n', 'X.vue')).toThrow('X.vue sets component options the catalogue cannot read')
  })
})

describe('docs frontmatter', () => {
  const f = 'X.docs.md'
  it('reads every key, continued lines and the notes', () => {
    const docs = parseDocs(DOCS.replace('use: When you need a thing.', 'use: When you need\n  a thing.') + '\nNotes here.\n', f)
    expect(docs).toEqual({
      group: 'widgets',
      purpose: 'A thing.',
      use: 'When you need a thing.',
      avoid: 'When you need another thing.',
      adapts: { input: 'Nothing changes.', surface: 'Nothing changes.', width: 'Nothing changes.' },
      notes: 'Notes here.',
    })
  })

  it.each([
    ['no frontmatter', 'just text', 'X.docs.md has no frontmatter; start it with a --- block holding group, purpose, use, avoid, adapts.'],
    ['a missing key', DOCS.replace('avoid: When you need another thing.\n', ''), 'X.docs.md is missing avoid in its frontmatter.'],
    ['an empty key', DOCS.replace('purpose: A thing.', 'purpose:'), 'X.docs.md gives purpose no text.'],
    ['an unknown key', DOCS.replace('use:', 'colour: red\nuse:'), 'X.docs.md has the unknown key "colour" in its frontmatter; the keys are group, purpose, use, avoid, adapts.'],
    ['a missing axis', DOCS.replace('  width: Nothing changes.\n', ''), 'X.docs.md is missing adapts.width; say what the component does between narrow, regular and wide, even if nothing changes.'],
    ['an unknown axis', DOCS.replace('  width:', '  height:'), 'X.docs.md has the unknown axis "height" under adapts; the axes are input, surface and width.'],
    ['a line that is not key: value', DOCS.replace('use:', 'just words\nuse:'), 'X.docs.md line 4 is not "key: value"; write each frontmatter entry on its own line.'],
  ])('refuses %s, in one sentence', (_, text, sentence) => {
    expect(() => parseDocs(text, f)).toThrow(sentence)
  })
})

// The skills and the docs say how many components there are; the number is
// the catalogue's, so a component added or taken away must change them too.
describe('the component count in the skills and docs', () => {
  it('is the number of components in the catalogue', () => {
    const catalog = JSON.parse(readFileSync(join(repo, CATALOG_FILE), 'utf8')) as Catalog
    const files = ['skills/weft-components/SKILL.md', 'site-src/content/docs/index.md']
    for (const file of files) {
      const counts = [...readFileSync(join(repo, file), 'utf8').matchAll(/\b(\d+) (?:Vapor |Vue )?components\b/g)].map((m) => Number(m[1]))
      expect(counts.length, `${file} gives no count`).toBeGreaterThan(0)
      for (const n of counts) expect(n, file).toBe(catalog.components.length)
    }
  })
})
