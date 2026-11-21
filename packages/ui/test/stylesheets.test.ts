// Each component imports its own stylesheet (and base.css), so a page ships
// the CSS of what it renders. weft.css stays the all-in-one bundle: through
// the group sheets it imports every stylesheet, so an app that loads it whole
// misses nothing.
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const src = join(dirname(fileURLToPath(import.meta.url)), '../src')
const styles = join(src, 'styles')
const sheet = (name: string) => readFileSync(join(styles, name), 'utf8')
const importsIn = (name: string) => [...sheet(name).matchAll(/@import '\.\/([a-z-]+\.css)';/g)].map((m) => m[1]!)
/** A group sheet holds only comments and @imports. */
const isGroup = (name: string) => sheet(name).replace(/\/\*[\s\S]*?\*\//g, '').replace(/@import '[^']+';/g, '').trim() === ''
const all = readdirSync(styles).filter((f) => f.endsWith('.css') && f !== 'weft.css').sort()
const groups = all.filter(isGroup)
const sheets = all.filter((f) => !isGroup(f))
/** The stylesheets weft.css brings in, group imports followed, in order. */
const flatten = (name: string): string[] => importsIn(name).flatMap((f) => (isGroup(f) ? flatten(f) : [f]))
const weft = flatten('weft.css')

const components = readdirSync(src, { recursive: true, encoding: 'utf8' })
  .filter((f) => f.endsWith('.vue'))
  .map((f) => f.split('\\').join('/'))
  .sort()
/** The stylesheets a component imports, by file name, in its order. */
const importsOf = (file: string) =>
  [...readFileSync(join(src, file), 'utf8').matchAll(/^import '\.\.\/styles\/([a-z-]+\.css)'$/gm)].map((m) => m[1]!)

describe('stylesheets', () => {
  it('are split per component, under group sheets of @imports only', () => {
    for (const name of ['base.css', 'button.css', 'input.css', 'card.css', 'form-field.css', 'slider.css']) {
      expect(sheets).toContain(name)
    }
    expect(groups).toEqual(['basics.css', 'forms.css', 'layout.css'])
    expect(importsIn('weft.css')[0]).toBe('base.css')
    for (const group of groups) expect(importsIn('weft.css')).toContain(group)
  })

  it('are each in weft.css, once, base.css first', () => {
    expect([...weft].sort()).toEqual(sheets)
    expect(new Set(weft).size).toBe(weft.length)
    expect(weft[0]).toBe('base.css')
  })

  it('keep in weft.css the order each component imports them in', () => {
    for (const file of components) {
      const at = importsOf(file).map((s) => weft.indexOf(s))
      expect(at, file).toEqual([...at].sort((a, b) => a - b))
    }
  })

  it.each(components)('%s imports base.css first, and only component stylesheets', (file) => {
    const own = importsOf(file)
    expect(own[0]).toBe('base.css')
    for (const name of own) expect(sheets).toContain(name)
  })

  it.each(components)('%s has a stylesheet of its own, or its docs say it has none', (file) => {
    const own = importsOf(file).filter((s) => s !== 'base.css')
    const docs = readFileSync(join(src, file.replace(/\.vue$/, '.docs.md')), 'utf8')
    if (own.length === 0) expect(docs).toMatch(/^It has no stylesheet of its own\b/m)
    else expect(docs).not.toMatch(/^It has no stylesheet of its own\b/m)
  })

  it('are each imported by some component', () => {
    const used = new Set(components.flatMap(importsOf))
    expect(sheets.filter((s) => !used.has(s))).toEqual([])
  })
})
