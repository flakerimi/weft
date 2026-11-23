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

const src = join(dirname(fileURLToPath(import.meta.url)), '../src')

interface Exception { file: string; property: string; value: string; reason: string }

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

interface Declaration { file: string; line: number; property: string; value: string }

/** The declarations in a file, comments blanked out (newlines kept, so lines still count). */
function declarations(file: string): Declaration[] {
  const text = readFileSync(join(src, file), 'utf8').replace(/\/\*[\s\S]*?\*\//g, (c) => c.replace(/[^\n]/g, ' '))
  const found: Declaration[] = []
  for (const m of text.matchAll(/(?<![\w-])((?:box-shadow|border|outline)(?:-[a-z]+)*)\s*:\s*([^;{}"'`\n]+)/g)) {
    const line = text.slice(0, m.index).split('\n').length
    found.push({ file, line, property: m[1]!, value: m[2]!.trim() })
  }
  return found
}
const all = files.flatMap(declarations)

/** The value with every var(--weft-…) removed (fallbacks too), parentheses balanced. */
function withoutTokens(value: string): string {
  let out = ''
  for (let i = 0; i < value.length; ) {
    if (value.startsWith('var(--weft-', i)) {
      let depth = 1
      for (i += 'var('.length; depth > 0 && i < value.length; i++) {
        if (value[i] === '(') depth++
        else if (value[i] === ')') depth--
      }
    } else out += value[i++]
  }
  return out
}

const isException = (d: Declaration) =>
  exceptions.some((e) => e.file === d.file && e.property === d.property && e.value === d.value)
const where = (d: Declaration) => `src/${d.file}:${d.line}: ${d.property}: ${d.value}`

/** A border or outline property that takes a width: the shorthands, their sides, and the -width longhands. */
const widthProperty = /^(?:border(?:-(?:top|right|bottom|left|block|inline)(?:-(?:start|end))?)?(?:-width)?|outline(?:-width)?)$/
const literalWidth = /(?<![\w.-])(?:\d*\.)?\d+px\b|\b(?:thin|medium|thick)\b/

/** Why a declaration breaks the rule, or undefined when it keeps it. */
function breaks(d: Declaration): string | undefined {
  if (d.property === 'box-shadow') {
    if (d.value === 'none' || /^var\(--weft-shadow-[a-z]+, /.test(d.value)) return undefined
    return 'an elevation shadow must read a --weft-shadow-* token, with its fallback'
  }
  if (widthProperty.test(d.property)) {
    const rest = withoutTokens(d.value)
    if (literalWidth.test(rest)) return 'a border or outline width must read --weft-border-width, --weft-border-width-strong or --weft-focus-width, with its fallback'
    return undefined
  }
  if (d.property === 'outline-offset' && withoutTokens(d.value).trim().startsWith('-')) {
    return 'an inset ring sits in by a line\'s width, so its offset must be calc(-1 * <that width\'s token>)'
  }
  return undefined
}

/** What each shadow and width token defaults to in the themes; a fallback must say the same. */
const defaults: Record<string, string> = {
  '--weft-border-width': `${neutral.border.width}px`,
  '--weft-border-width-strong': `${neutral.border.strong}px`,
  '--weft-focus-width': `${neutral.border.focus}px`,
  ...Object.fromEntries(Object.entries(neutral.shadow).map(([k, v]) => [`--weft-shadow-${k}`, v])),
}

interface TokenUse { file: string; line: number; token: string; fallback: string | undefined }

/** Every var() of a shadow or width token in a text, with its fallback (parentheses balanced). */
function tokenUses(file: string, text: string): TokenUse[] {
  const uses: TokenUse[] = []
  for (const m of text.matchAll(/var\((--weft-(?:border-width(?:-strong)?|focus-width|shadow-[a-z]+))\s*(,?)/g)) {
    let fallback: string | undefined
    if (m[2]) {
      let depth = 1
      let i = m.index + m[0].length
      const start = i
      for (; i < text.length; i++) {
        if (text[i] === '(') depth++
        else if (text[i] === ')' && --depth === 0) break
      }
      fallback = text.slice(start, i).trim()
    }
    uses.push({ file, line: text.slice(0, m.index).split('\n').length, token: m[1]!, fallback })
  }
  return uses
}
const uses = files.flatMap((f) => tokenUses(f, readFileSync(join(src, f), 'utf8').replace(/\/\*[\s\S]*?\*\//g, (c) => c.replace(/[^\n]/g, ' '))))

/** Why a token's use breaks the rule (no fallback, or one that is not the theme default), or undefined. */
function wrongFallback(u: TokenUse): string | undefined {
  const want = defaults[u.token]
  if (want === undefined) return `${u.token} is not a theme token`
  if (u.fallback !== want) return `its fallback is ${u.fallback === undefined ? 'missing' : `"${u.fallback}"`}; the themes' default is "${want}"`
  return undefined
}

describe('shadows and line widths', () => {
  it('reads the files it guards', () => {
    expect(files.filter((f) => f.endsWith('.css')).length).toBeGreaterThan(80)
    expect(all.some((d) => d.file === 'styles/card.css' && d.property === 'border')).toBe(true)
    expect(all.some((d) => d.file === 'styles/dialog.css' && d.property === 'box-shadow')).toBe(true)
  })

  it('come from the theme\'s tokens, except the listed exceptions', () => {
    const broken = all.filter((d) => !isException(d) && breaks(d)).map((d) => `${where(d)} (${breaks(d)})`)
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
