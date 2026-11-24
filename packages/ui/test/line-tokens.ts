// The checks behind the shadow and line-width guards: this package's
// (elevation-tokens.test.ts, over packages/ui/src) and the site's
// (site-src/test/line-tokens.test.ts, over its own styles). An elevation
// shadow reads a --weft-shadow-* token, a border's or an outline's width reads
// --weft-border-width, --weft-border-width-strong or --weft-focus-width, an
// inset ring's negative offset reads the width it sits in by, and every one of
// those tokens falls back to the themes' default. It imports nothing but
// types, so the site can use it without depending on @weft/themes.

export interface Declaration { file: string; line: number; property: string; value: string }
export interface Exception { file: string; property: string; value: string; reason: string }
export interface TokenUse { file: string; line: number; token: string; fallback: string | undefined }

/** The text with its comments blanked out, newlines kept, so lines still count. */
export const withoutComments = (text: string) => text.replace(/\/\*[\s\S]*?\*\//g, (c) => c.replace(/[^\n]/g, ' '))

/** A .vue file's <style> blocks, everything else blanked out (newlines kept). */
export function styleBlocks(text: string): string {
  let out = ''
  let at = 0
  for (const m of text.matchAll(/(<style\b[^>]*>)([\s\S]*?)<\/style>/g)) {
    const start = m.index + m[1]!.length
    out += text.slice(at, start).replace(/[^\n]/g, ' ') + m[2]!
    at = start + m[2]!.length
  }
  return out + text.slice(at).replace(/[^\n]/g, ' ')
}

/** The shadow, border and outline declarations in a stylesheet's text. */
export function declarationsIn(file: string, text: string): Declaration[] {
  const css = withoutComments(text)
  const found: Declaration[] = []
  for (const m of css.matchAll(/(?<![\w-])((?:box-shadow|border|outline)(?:-[a-z]+)*)\s*:\s*([^;{}"'`\n]+)/g)) {
    found.push({ file, line: css.slice(0, m.index).split('\n').length, property: m[1]!, value: m[2]!.trim() })
  }
  return found
}

/** The value with every var(--weft-…) removed (fallbacks too), parentheses balanced. */
export function withoutTokens(value: string): string {
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

export const isException = (d: Declaration, exceptions: Exception[]) =>
  exceptions.some((e) => e.file === d.file && e.property === d.property && e.value === d.value)

/** A border or outline property that takes a width: the shorthands, their sides, and the -width longhands. */
const widthProperty = /^(?:border(?:-(?:top|right|bottom|left|block|inline)(?:-(?:start|end))?)?(?:-width)?|outline(?:-width)?)$/
const literalWidth = /(?<![\w.-])(?:\d*\.)?\d+px\b|\b(?:thin|medium|thick)\b/

/** Why a declaration breaks the rule, or undefined when it keeps it. */
export function breaks(d: Declaration): string | undefined {
  if (d.property === 'box-shadow') {
    if (d.value === 'none' || /^var\(--weft-shadow-[a-z]+, /.test(d.value)) return undefined
    return 'an elevation shadow must read a --weft-shadow-* token, with its fallback'
  }
  if (widthProperty.test(d.property)) {
    if (literalWidth.test(withoutTokens(d.value))) return 'a border or outline width must read --weft-border-width, --weft-border-width-strong or --weft-focus-width, with its fallback'
    return undefined
  }
  if (d.property === 'outline-offset' && withoutTokens(d.value).trim().startsWith('-')) {
    return 'an inset ring sits in by a line\'s width, so its offset must be calc(-1 * <that width\'s token>)'
  }
  return undefined
}

/** What a theme gives each shadow and width token, as its CSS writes them. */
export function themeDefaults(theme: { border: { width: number; strong: number; focus: number }; shadow: Record<string, string> }): Record<string, string> {
  return {
    '--weft-border-width': `${theme.border.width}px`,
    '--weft-border-width-strong': `${theme.border.strong}px`,
    '--weft-focus-width': `${theme.border.focus}px`,
    ...Object.fromEntries(Object.entries(theme.shadow).map(([k, v]) => [`--weft-shadow-${k}`, v])),
  }
}

/** Every var() of a shadow or width token in a text, with its fallback (parentheses balanced). */
export function tokenUses(file: string, text: string): TokenUse[] {
  const css = withoutComments(text)
  const uses: TokenUse[] = []
  for (const m of css.matchAll(/var\((--weft-(?:border-width(?:-strong)?|focus-width|shadow-[a-z]+))\s*(,?)/g)) {
    let fallback: string | undefined
    if (m[2]) {
      let depth = 1
      let i = m.index + m[0].length
      const start = i
      for (; i < css.length; i++) {
        if (css[i] === '(') depth++
        else if (css[i] === ')' && --depth === 0) break
      }
      fallback = css.slice(start, i).trim()
    }
    uses.push({ file, line: css.slice(0, m.index).split('\n').length, token: m[1]!, fallback })
  }
  return uses
}

/** Why a token's use breaks the rule (no fallback, or one that is not the theme default), or undefined. */
export function wrongFallback(u: TokenUse, defaults: Record<string, string>): string | undefined {
  const want = defaults[u.token]
  if (want === undefined) return `${u.token} is not a theme token`
  if (u.fallback !== want) return `its fallback is ${u.fallback === undefined ? 'missing' : `"${u.fallback}"`}; the themes' default is "${want}"`
  return undefined
}
