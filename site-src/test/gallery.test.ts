// The gallery's model and the preview stage's query, as plain functions.
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { DEFAULT_VIEW, formatView, INPUTS, parseView, SCHEMES, SURFACES, THEMES, WIDTHS, type StageView } from '../components/gallery/stage.ts'
import { checkFresh, exampleIsland, groupsOf, inlineHTML, isInteractive, neighbours, slugOf, stageIsland, type Catalog } from '../gallery/model.ts'

const catalog = JSON.parse(readFileSync(new URL('../../catalog/components.json', import.meta.url), 'utf8')) as Catalog
const exampleFiles = () =>
  Object.fromEntries(catalog.components.flatMap((c) => c.examples.map((e) => [e.file, e.source] as const)))

describe('the stage query', () => {
  // Every view the toolbar can make.
  const views: StageView[] = []
  for (const input of [undefined, ...INPUTS])
    for (const surface of SURFACES)
      for (const theme of THEMES)
        for (const scheme of [undefined, ...SCHEMES])
          for (const width of [undefined, ...WIDTHS]) {
            const v: StageView = { surface, theme }
            if (input) v.input = input
            if (scheme) v.scheme = scheme
            if (width) v.width = width
            views.push(v)
          }

  it('round-trips every view the toolbar can make', () => {
    expect(views).toHaveLength(3 * 2 * 2 * 3 * 4)
    for (const v of views) {
      const query = formatView(v)
      expect(parseView(query), query).toEqual(v)
      expect(formatView(parseView(query))).toBe(query)
    }
  })

  it('writes only what differs from the default, in one order', () => {
    expect(formatView(DEFAULT_VIEW)).toBe('')
    expect(formatView({ input: 'touch', surface: 'landing', theme: 'lisa', scheme: 'dark', width: 390 })).toBe(
      '?input=touch&surface=landing&theme=lisa&scheme=dark&width=390',
    )
    expect(formatView({ surface: 'app', theme: 'neutral', width: 1280 })).toBe('?width=1280')
  })

  it('reads a link in any order, and leaves what it gets wrong at the default', () => {
    expect(parseView('?width=820&theme=lisa')).toEqual({ surface: 'app', theme: 'lisa', width: 820 })
    expect(parseView('?input=mouse&surface=&theme=Lisa&scheme=dim&width=391&utm=x')).toEqual(DEFAULT_VIEW)
    expect(parseView('')).toEqual(DEFAULT_VIEW)
  })
})

describe('the gallery model', () => {
  it('names each page by its component in kebab case', () => {
    expect(['Button', 'DatePicker', 'AppShell', 'InputTags', 'CommandPalette', 'WeftProvider'].map(slugOf)).toEqual([
      'button', 'date-picker', 'app-shell', 'input-tags', 'command-palette', 'weft-provider',
    ])
    // No two components share a page.
    const slugs = catalog.components.map((c) => slugOf(c.name))
    expect(new Set(slugs).size).toBe(slugs.length)
  })

  it('groups every component once, in the groups order, by name within a group', () => {
    const groups = groupsOf(catalog.components)
    expect(groups.flatMap((g) => g.components.map((c) => c.name)).sort()).toEqual(catalog.components.map((c) => c.name).sort())
    expect(groups.slice(0, 3).map((g) => g.label)).toEqual(['Layout', 'Basics', 'Forms'])
    // A group added later follows, under its own name.
    const later = groupsOf([...catalog.components, { name: 'Chart', group: 'charts' }])
    expect(later.at(-1)).toEqual({ group: 'charts', label: 'Charts', components: [{ name: 'Chart', group: 'charts' }] })
  })

  it('links each component to the ones before and after it in its group', () => {
    const forms = groupsOf(catalog.components).find((g) => g.group === 'forms')!.components.map((c) => c.name)
    const { prev, next } = neighbours(catalog.components, forms[1]!)
    expect([prev?.name, next?.name]).toEqual([forms[0], forms[2]])
    expect(neighbours(catalog.components, forms[0]!).prev).toBeUndefined()
    expect(neighbours(catalog.components, forms.at(-1)!).next).toBeUndefined()
  })

  it('gives every island its own id, none the start of another chunk name', () => {
    const ids = catalog.components.flatMap((c) => [stageIsland(c.name), ...c.examples.map((e) => exampleIsland(c.name, e.name))])
    expect(new Set(ids).size).toBe(ids.length)
    for (const id of ids) expect(id).toMatch(/^[A-Za-z0-9_]+$/)
    expect(exampleIsland('Input', 'tags-basic')).not.toBe(exampleIsland('InputTags', 'basic'))
  })

  it('hydrates an example only when it needs JavaScript', () => {
    const source = (file: string) => catalog.components.flatMap((c) => c.examples).find((e) => e.file.endsWith(`/${file}`))!.source
    expect(isInteractive(source('Button/basic.vue'))).toBe(false)
    expect(isInteractive(source('Button/variants-and-tones.vue'))).toBe(false)
    expect(isInteractive(source('Button/loading.vue'))).toBe(true)
    // A perm button is drawn only once the session is known, in the browser.
    expect(isInteractive(source('Button/by-permission.vue'))).toBe(true)
    expect(isInteractive('<template><Button label="Go" perm="x:y" /></template>')).toBe(true)
    expect(isInteractive('<template><Button label="Go" :perm="p" /></template>')).toBe(true)
    expect(isInteractive(source('Select/basic.vue'))).toBe(true)
    expect(isInteractive(source('Grid/basic.vue'))).toBe(false)
    // A comment in the template is not markup.
    expect(isInteractive('<template>\n  <!-- <Select /> @click -->\n  <Badge label="a" />\n</template>\n')).toBe(false)
    expect(isInteractive('<script setup vapor lang="ts">\nimport { Badge } from \'@weft/ui\'\n</script>\n\n<template>\n  <Badge label="a" @click="go" />\n</template>\n')).toBe(true)
    // A component added later, or a helper of the example's own, hydrates.
    expect(isInteractive('<template><Chart /></template>')).toBe(true)
  })

  it('takes a catalogue that matches the example files', () => {
    expect(() => checkFresh(catalog, exampleFiles())).not.toThrow()
    // A helper component beside the examples is not an example.
    expect(() => checkFresh(catalog, { ...exampleFiles(), 'packages/ui/examples/Toast/Counter.vue': 'x' })).not.toThrow()
  })

  it('refuses a catalogue the examples have moved on from, in one sentence', () => {
    const files = exampleFiles()
    expect(() => checkFresh(catalog, { ...files, 'packages/ui/examples/Button/new.vue': 'x' })).toThrow(
      'gallery: catalog/components.json is out of date (it does not list packages/ui/examples/Button/new.vue); run pnpm catalog.',
    )
    expect(() => checkFresh(catalog, { ...files, 'packages/ui/examples/Button/basic.vue': 'changed' })).toThrow(
      'gallery: catalog/components.json is out of date (packages/ui/examples/Button/basic.vue has changed since it was written); run pnpm catalog.',
    )
    const { ['packages/ui/examples/Button/loading.vue']: _, ...fewer } = files
    expect(() => checkFresh(catalog, fewer)).toThrow(
      'gallery: catalog/components.json is out of date (it lists packages/ui/examples/Button/loading.vue, which is gone); run pnpm catalog.',
    )
  })

  it('writes a catalogue sentence as HTML, escaped, with its code spans', () => {
    expect(inlineHTML('Use `<Link>` & "go" <b>')).toBe('Use <code>&lt;Link&gt;</code> &amp; &quot;go&quot; &lt;b&gt;')
  })
})
