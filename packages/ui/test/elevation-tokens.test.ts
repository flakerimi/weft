// Shadows and line widths come from the theme: an elevation shadow reads a
// --weft-shadow-* token, and a border's or an outline's width reads
// --weft-border-width, --weft-border-width-strong or --weft-focus-width, each
// with its fallback, so a theme can raise a flat design, or thicken, thin or
// drop its lines. An inset ring (a negative outline-offset) sits inside the
// edge by the ring's or the border's width, so it reads the same token. The
// few declarations that are not elevation or not a line are listed below,
// each with its reason.
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { neutral } from '@weft/themes'
import { describe, expect, it } from 'vitest'
import { breaks, declarationsIn, type Declaration, type Exception, isException, themeDefaults, tokenUses, wrongFallback as wrongFor } from './line-tokens.ts'

const src = join(dirname(fileURLToPath(import.meta.url)), '../src')

/** Left as they are: each is matched exactly, and an entry that matches nothing fails the test. */
const exceptions: Exception[] = [
  {
    file: 'styles/chip.css', property: 'box-shadow', value: '0 0 0 2px var(--weft-color-bg, Canvas)',
    reason: 'a ring in the page colour that cuts the indicator out of what it sits on; not elevation',
  },
  {
    file: 'styles/avatar-group.css', property: 'box-shadow', value: '0 0 0 2px var(--weft-color-bg, Canvas)',
    reason: 'a ring in the page colour that separates overlapping avatars; not elevation',
  },
  {
    file: 'styles/timeline.css', property: 'box-shadow', value: '0 0 0 4px color-mix(in srgb, var(--weft-tone, LinkText) 25%, transparent)',
    reason: 'a tinted halo that marks the current item; not elevation',
  },
  {
    file: 'styles/command-palette.css', property: 'box-shadow', value: 'inset 0 calc(-1 * var(--weft-focus-width, 2px)) 0 var(--weft-color-focus, Highlight)',
    reason: 'the focused search field\'s focus indicator, an underline drawn inset as wide as the focus ring; not elevation',
  },
  {
    file: 'styles/switch.css', property: 'box-shadow', value: 'inset 0 0 0 var(--weft-border-width, 1px) var(--weft-color-danger, #DC2626)',
    reason: 'an invalid switch\'s danger line, inset as wide as the hairline because the track has no border; not elevation',
  },
  {
    file: 'styles/tabs.css', property: 'box-shadow', value: 'inset 0 0 0 var(--weft-border-width, 1px) var(--weft-tone, CanvasText), var(--weft-shadow-control, 0 1px 2px color-mix(in srgb, CanvasText 25%, transparent))',
    reason: 'the selected neutral pill tab\'s ring in its tone, as wide as the hairline, before its control shadow (the token): its mark, since its text is the others\' colour',
  },
  {
    file: 'styles/toast.css', property: 'border-inline-start', value: '4px solid var(--weft-tone, CanvasText)',
    reason: 'the tone\'s accent bar, a mark of its own and not a hairline',
  },
  {
    file: 'styles/spinner.css', property: 'border', value: '2px solid color-mix(in srgb, currentColor 25%, transparent)',
    reason: 'the spinner\'s stroke is geometry, sized to the spinner, not a line a theme sets',
  },
]

/** Every .vue and .css file under src, relative to it. */
const files = readdirSync(src, { recursive: true, encoding: 'utf8' })
  .map((f) => f.split('\\').join('/'))
  .filter((f) => f.endsWith('.vue') || f.endsWith('.css'))
  .sort()
const text = (f: string) => readFileSync(join(src, f), 'utf8')
const all = files.flatMap((f) => declarationsIn(f, text(f)))
const uses = files.flatMap((f) => tokenUses(f, text(f)))
const defaults = themeDefaults(neutral)
const wrongFallback = (u: Parameters<typeof wrongFor>[0]) => wrongFor(u, defaults)
const where = (d: Declaration) => `src/${d.file}:${d.line}: ${d.property}: ${d.value}`

describe('shadows and line widths', () => {
  it('reads the files it guards', () => {
    expect(files.filter((f) => f.endsWith('.css')).length).toBeGreaterThan(80)
    expect(all.some((d) => d.file === 'styles/card.css' && d.property === 'border')).toBe(true)
    expect(all.some((d) => d.file === 'styles/dialog.css' && d.property === 'box-shadow')).toBe(true)
  })

  it('come from the theme\'s tokens, except the listed exceptions', () => {
    const broken = all.filter((d) => !isException(d, exceptions) && breaks(d)).map((d) => `${where(d)} (${breaks(d)})`)
    expect(broken).toEqual([])
  })

  it('fall back to the themes\' defaults wherever a token is read', () => {
    expect(uses.length).toBeGreaterThan(100)
    expect(uses.filter(wrongFallback).map((u) => `src/${u.file}:${u.line}: ${u.token} (${wrongFallback(u)})`)).toEqual([])
  })

  it('catch a fallback that differs from the default, or none', () => {
    const [ok] = tokenUses('x.css', 'border: var(--weft-border-width, 1px) solid red;')
    expect(wrongFallback(ok!)).toBeUndefined()
    const [thick] = tokenUses('x.css', 'border: var(--weft-border-width, 2px) solid red;')
    expect(wrongFallback(thick!)).toContain('the themes\' default is "1px"')
    const [shadow] = tokenUses('x.css', 'box-shadow: var(--weft-shadow-popup, 0 8px 24px color-mix(in srgb, CanvasText 20%, transparent));')
    expect(shadow!.fallback).toBe('0 8px 24px color-mix(in srgb, CanvasText 20%, transparent)')
    expect(wrongFallback(shadow!)).toBeDefined()
    const [bare] = tokenUses('x.css', 'outline: var(--weft-focus-width) solid red;')
    expect(wrongFallback(bare!)).toContain('missing')
  })

  it('list only exceptions that are still there, each with a reason', () => {
    for (const e of exceptions) {
      expect(e.reason.length, `${e.file} ${e.property}`).toBeGreaterThan(10)
      expect(all.some((d) => d.file === e.file && d.property === e.property && d.value === e.value), `${e.file} ${e.property}: ${e.value}`).toBe(true)
    }
  })

  it('catch a hard-coded shadow, border, focus ring and inset offset', () => {
    const d = (property: string, value: string): Declaration => ({ file: 'styles/x.css', line: 1, property, value })
    expect(breaks(d('box-shadow', '0 8px 24px color-mix(in srgb, CanvasText 16%, transparent)'))).toBeDefined()
    expect(breaks(d('box-shadow', '0 0 0 2px red'))).toBeDefined()
    expect(breaks(d('border', '1px solid var(--weft-color-border, GrayText)'))).toBeDefined()
    expect(breaks(d('border-block-end', '2px solid transparent'))).toBeDefined()
    expect(breaks(d('border-block-end-width', '2px'))).toBeDefined()
    expect(breaks(d('outline', '2px solid var(--weft-color-focus, Highlight)'))).toBeDefined()
    expect(breaks(d('outline', 'thin solid red'))).toBeDefined()
    expect(breaks(d('outline-offset', '-2px'))).toBeDefined()
    // A token's own fallback is not a literal width.
    expect(breaks(d('box-shadow', 'var(--weft-shadow-popup, 0 8px 24px color-mix(in srgb, CanvasText 16%, transparent))'))).toBeUndefined()
    expect(breaks(d('border', 'var(--weft-border-width, 1px) solid var(--weft-color-border, GrayText)'))).toBeUndefined()
    expect(breaks(d('outline', 'var(--weft-focus-width, 2px) solid var(--weft-color-focus, Highlight)'))).toBeUndefined()
    expect(breaks(d('outline-offset', 'calc(-1 * var(--weft-focus-width, 2px))'))).toBeUndefined()
    // No width, or a gap outside the edge, is not a line's width.
    for (const [p, v] of [['border', '0'], ['outline', 'none'], ['border-radius', '8px'], ['border-color', 'red'], ['outline-offset', '2px'], ['border', 'inherit']] as const) {
      expect(breaks(d(p, v)), `${p}: ${v}`).toBeUndefined()
    }
  })
})
