// @vitest-environment node
// The catalogue (catalog/components.json) must match what scripts/catalog.ts
// builds from the components, their docs files and their examples.
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildCatalog, CATALOG_FILE, parseDocs, serialise, type Catalog, type CatalogComponent } from '../../../scripts/catalog.ts'

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
    expect(thing!.slots).toEqual([{ name: 'default', props: [] }])
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
      { name: 'item', props: ['option', 'index'] },
      { name: 'empty', props: [] },
    ])
  })

  it('slot props from the template when slots are not declared', () => {
    const sfc = '<script setup vapor lang="ts">\n</script>\n<template><ul><slot name="row" :item="1" index="2" /><slot /></ul></template>\n'
    const [thing] = buildCatalog(repoWith({ sfc })).components
    expect(thing!.slots).toEqual([
      { name: 'row', props: ['item', 'index'] },
      { name: 'default', props: [] },
    ])
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
