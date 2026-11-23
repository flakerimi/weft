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
    file: 'styles/command-palette.css', property: 'box-shadow', value: 'inset 0 -2px 0 var(--weft-color-focus, Highlight)',
    reason: 'the focused search field\'s underline, drawn inset; not elevation',
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
