// The site's own styles draw lines and focus rings with the theme's tokens, as
// @weft/ui's components do: a hairline reads --weft-border-width, an
// emphasised line --weft-border-width-strong, a focus ring --weft-focus-width,
// each falling back to the weft theme's default, so nothing changes on screen
// at the defaults. The checks are packages/ui/test/line-tokens.ts, shared
// with the components' guard (packages/ui/test/elevation-tokens.test.ts). The
// site is a designed page: its shadows have values of their own, and stay, as
// listed below with the other exceptions, each with its reason.
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { weft } from '../../packages/themes/src/weft.ts'
import { breaks, declarationsIn, type Declaration, type Exception, isException, styleBlocks, themeDefaults, tokenUses, wrongFallback } from '../../packages/ui/test/line-tokens.ts'

const root = fileURLToPath(new URL('..', import.meta.url))

const exceptions: Exception[] = [
  {
    file: 'styles/site.css', property: 'border-block-start', value: '4px solid',
    reason: 'the footer\'s saffron bar, drawn through border-image: a mark of the site\'s design, not a hairline',
  },
  {
    file: 'styles/landing.css', property: 'outline', value: 'calc(var(--weft-focus-width, 2px) + 1px) solid var(--site-saffron)',
    reason: 'the landing\'s saffron focus ring is the theme\'s focus width and 1px more, on purpose: it follows the token and stays heavier than the docs\'',
  },
  {
    file: 'styles/landing.css', property: 'box-shadow', value: '0 30px 80px -20px color-mix(in srgb, black 60%, transparent), 0 0 0 1px color-mix(in srgb, var(--weft-color-text) 12%, transparent)',
    reason: 'the hero frames\' deep drop and edge, a designed value no shadow token has',
  },
  {
    file: 'styles/landing.css', property: 'box-shadow', value: 'inset 0 -5px var(--site-saffron)',
    reason: 'the saffron stripe in the drawn Northline logo: part of the drawing, not elevation',
  },
  {
    file: 'styles/landing.css', property: 'border', value: '8px solid color-mix(in srgb, black 85%, var(--weft-color-primary))',
    reason: 'the phone frame\'s bezel: geometry of the drawing, not a line a theme sets',
  },
  {
    file: 'styles/landing.css', property: 'box-shadow', value: '0 20px 50px -20px color-mix(in srgb, black 40%, transparent)',
    reason: 'the search scene\'s card, a designed value no shadow token has',
  },
  {
    file: 'styles/landing.css', property: 'box-shadow', value: '0 12px 30px -14px color-mix(in srgb, black 35%, transparent)',
    reason: 'the adaptive demo\'s drawn popover, a designed value no shadow token has',
  },
  {
    file: 'styles/landing.css', property: 'box-shadow', value: '0 -10px 24px -16px color-mix(in srgb, black 50%, transparent)',
    reason: 'the adaptive demo\'s drawn sheet, an upward shadow no shadow token has',
  },
]

/** The site's stylesheets and its .vue files (their <style> blocks), relative to site-src. */
const skip = /(^|\/)(node_modules|dist|public)(\/|$)/
const files = readdirSync(root, { recursive: true, encoding: 'utf8' })
  .map((f) => f.split('\\').join('/'))
  .filter((f) => !skip.test(f) && ((f.startsWith('styles/') && f.endsWith('.css')) || f.endsWith('.vue')))
  .sort()
const css = (f: string) => {
  const text = readFileSync(join(root, f), 'utf8')
  return f.endsWith('.vue') ? styleBlocks(text) : text
}
const all = files.flatMap((f) => declarationsIn(f, css(f)))
const uses = files.flatMap((f) => tokenUses(f, css(f)))
const defaults = themeDefaults(weft)
const where = (d: Declaration) => `site-src/${d.file}:${d.line}: ${d.property}: ${d.value}`

describe('the site\'s lines and shadows', () => {
  it('reads the files it guards', () => {
    for (const f of ['styles/landing.css', 'styles/docs.css', 'styles/shell.css', 'components/CopyCommand.vue']) expect(files).toContain(f)
    expect(all.some((d) => d.file === 'components/CopyCommand.vue' && d.property === 'border')).toBe(true)
    // Only a .vue file's <style> is read, on its own lines.
    const vue = '<template>\n<p style="border: 1px solid">x</p>\n</template>\n<style>\na { border: 0 }\n</style>\n'
    expect(declarationsIn('x.vue', styleBlocks(vue))).toEqual([{ file: 'x.vue', line: 5, property: 'border', value: '0' }])
  })

  it('come from the theme\'s tokens, except the listed exceptions', () => {
    expect(all.filter((d) => !isException(d, exceptions) && breaks(d)).map((d) => `${where(d)} (${breaks(d)})`)).toEqual([])
  })

  it('fall back to the weft theme\'s defaults wherever a token is read', () => {
    expect(uses.filter((u) => wrongFallback(u, defaults)).map((u) => `site-src/${u.file}:${u.line}: ${u.token} (${wrongFallback(u, defaults)})`)).toEqual([])
  })

  it('list only exceptions that are still there, each with a reason', () => {
    for (const e of exceptions) {
      expect(e.reason.length, `${e.file} ${e.property}`).toBeGreaterThan(10)
      expect(all.some((d) => d.file === e.file && d.property === e.property && d.value === e.value), `${e.file} ${e.property}: ${e.value}`).toBe(true)
    }
  })
})
