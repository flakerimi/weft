// @vitest-environment node
// Text drawn on a tint of the text colour (a soft surface, a pill tab list,
// a hovered, active, current or open item, initials on their disc) reaches
// WCAG AA (4.5:1) in every theme and scheme. The muted colour does not, on
// those tints, so the text there is drawn in the full text colour. Each case
// reads, from the stylesheets, the colour the text takes in that state (the
// first of its selectors that sets one: the text's own rule for that state,
// else its own rule, else the rule of the item it inherits from) and the
// tint behind it.
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { contrast, lisa, mix, neutral, themeToCss, weft, type LisaTokens, type Theme } from '@weft/themes'
import tokens from '../../themes/src/lisa/tokens.json' with { type: 'json' }
import { describe, expect, it } from 'vitest'

const styles = join(dirname(fileURLToPath(import.meta.url)), '../src/styles')
const css = readdirSync(styles)
  .filter((f) => f.endsWith('.css'))
  .sort()
  .map((f) => readFileSync(join(styles, f), 'utf8'))
  .join('\n')
  .replace(/\/\*[\s\S]*?\*\//g, '')

/** A selector list split at its top-level commas (not those inside `:not(…)`). */
function selectors(list: string): string[] {
  const out: string[] = []
  let depth = 0
  let start = 0
  for (let i = 0; i < list.length; i++) {
    if (list[i] === '(') depth++
    else if (list[i] === ')') depth--
    else if (list[i] === ',' && depth === 0) {
      out.push(list.slice(start, i).trim())
      start = i + 1
    }
  }
  return [...out, list.slice(start).trim()]
}

/** The declarations of every rule whose selector list holds `selector` exactly, in source order. */
function declarations(selector: string): string[] {
  const out: string[] = []
  for (const m of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (selectors(m[1]!).includes(selector)) out.push(m[2]!)
  }
  return out
}

/** The colour token (`text`, `text-muted`, …) the first of `selectors` with a `color` sets. */
function textToken(selectors: string[]): string {
  for (const selector of selectors) {
    for (const body of declarations(selector).reverse()) {
      const m = /(?:^|;|\s)color:\s*var\(--weft-color-([\w-]+)/.exec(body)
      if (m) return m[1]!
    }
  }
  throw new Error(`no rule colours ${selectors.join(' or ')}`)
}

/** The share of the text colour the rule `selector` lays over the ground, and whether it mixes with the surface (opaque) rather than over whatever is behind. */
function tintOf(selector: string): { share: number; over?: 'surface' } {
  for (const body of declarations(selector)) {
    const m = /background:\s*color-mix\(in srgb, var\(--weft-color-text, CanvasText\) (\d+)%, (transparent|var\(--weft-color-surface)/.exec(body)
    if (m) return { share: Number(m[1]) / 100, over: m[2] === 'transparent' ? undefined : 'surface' }
  }
  throw new Error(`no rule tints ${selector} with the text colour`)
}

const HOVER = ':not(:disabled, [aria-disabled="true"]):hover'
const ALL = ['bg', 'surface', 'surface-raised'] as const
const RAISED = ['surface', 'surface-raised'] as const

const cases: { name: string; text: string[]; tint: string; grounds: readonly string[] }[] = [
  { name: 'Tabs pill: an unselected tab on the list', text: ['.weft-tabs[data-variant="pill"] .weft-tabs__tab', '.weft-tabs__tab'], tint: '.weft-tabs[data-variant="pill"] > .weft-tabs__list', grounds: ALL },
  { name: 'EmptyState soft: the description', text: ['.weft-empty-state[data-variant="soft"] .weft-empty-state__description', '.weft-empty-state__description'], tint: '.weft-empty-state[data-variant="soft"]', grounds: ALL },
  { name: 'EmptyState soft: the footer', text: ['.weft-empty-state[data-variant="soft"] .weft-empty-state__footer', '.weft-empty-state__footer'], tint: '.weft-empty-state[data-variant="soft"]', grounds: ALL },
  { name: 'NavigationMenu: a hovered item\'s description', text: ['.weft-nav__description', `.weft-nav .weft-nav__item${HOVER}`], tint: `.weft-nav .weft-nav__item${HOVER}`, grounds: ALL },
  { name: 'NavigationMenu: the current item\'s description', text: ['.weft-nav__description', '.weft-nav .weft-nav__item[aria-current="page"]'], tint: '.weft-nav__panel .weft-nav__item[aria-current="page"]', grounds: ALL },
  { name: 'Listbox: the active option\'s description', text: ['.weft-listbox__option[data-active] .weft-listbox__description', '.weft-listbox__description'], tint: '.weft-listbox__option[data-active]', grounds: RAISED },
  { name: 'Listbox: the active option\'s leading text', text: ['.weft-listbox__option[data-active] .weft-listbox__leading', '.weft-listbox__leading'], tint: '.weft-listbox__option[data-active]', grounds: RAISED },
  { name: 'Listbox: the active "Create …" option', text: ['.weft-listbox__create[data-active]', '.weft-listbox__create'], tint: '.weft-listbox__option[data-active]', grounds: RAISED },
  { name: 'DropdownMenu: a focused item\'s description', text: ['.weft-menu__item:focus .weft-menu__description', '.weft-menu__description'], tint: '.weft-menu__item:focus', grounds: RAISED },
  { name: 'DropdownMenu: an open item\'s description', text: ['.weft-menu__item[aria-expanded="true"] .weft-menu__description', '.weft-menu__description'], tint: '.weft-menu__item[aria-expanded="true"]', grounds: RAISED },
  { name: 'DropdownMenu: a focused Back', text: ['.weft-menu__back:focus', '.weft-menu__back'], tint: '.weft-menu__item:focus', grounds: RAISED },
  { name: 'Calendar: a hovered day outside the month', text: ['.weft-calendar__day[data-outside]:hover:not(:disabled, [data-disabled], [data-selected])', '.weft-calendar__day[data-outside]'], tint: '.weft-calendar__day:hover:not(:disabled, [data-disabled], [data-selected])', grounds: ALL },
  { name: 'Avatar: the initials on their disc', text: ['.weft-avatar'], tint: '.weft-avatar', grounds: ['surface'] },
]

/** The custom properties of the first block whose selector is `selector`. */
function block(out: string, selector: string): Record<string, string> {
  const start = out.indexOf(`${selector} {`)
  const body = out.slice(start, out.indexOf('}', start))
  return Object.fromEntries([...body.matchAll(/--weft-([\w-]+): ([^;]+);/g)].map((m) => [m[1]!, m[2]!]))
}

const accents = Object.keys(tokens.color.accent).filter((k) => /^accent-.+-500$/.test(k)).map((k) => k.slice(7, -4))
const themes: Theme[] = [neutral, weft, ...accents.map((accent) => lisa(tokens as LisaTokens, { accent }))]

describe.each(themes.map((t) => [t.name, t] as const))('%s', (_, theme) => {
  const out = themeToCss(theme)
  const light = block(out, ':where(:root, [data-weft-scheme="light"])')
  const schemes = { light, dark: { ...light, ...block(out, ':where([data-weft-scheme="dark"])') } }
  for (const [scheme, v] of Object.entries(schemes)) {
    it.each(cases.map((c) => [c.name, c] as const))(`${scheme}: %s reads at 4.5:1`, (_, c) => {
      const token = textToken(c.text)
      const { share, over } = tintOf(c.tint)
      const text = v[`color-${token}`]!
      for (const ground of over ? [over] : c.grounds) {
        const behind = mix(v['color-text']!, v[`color-${ground}`]!, share)
        expect(contrast(text, behind), `${token} ${text} on ${ground} with ${share * 100}% text`).toBeGreaterThanOrEqual(4.5)
      }
    })
  }
})
