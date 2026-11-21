import { contrast, isHex, mix } from './contrast.ts'
import { colorKeys, tints, tones, type Theme } from './tokens.ts'

const spaceKeys = ['1', '2', '3', '4', '6', '8'] as const
const radiusKeys = ['control', 'card', 'panel'] as const
const motionKeys = ['control', 'layout', 'dialog'] as const
const controlKeys = ['sm', 'md', 'lg'] as const

/** Read a value by dotted path, or throw the one sentence a theme author needs. */
function need(theme: Theme, path: string): unknown {
  let value: unknown = theme
  for (const key of path.split('.')) {
    value = value != null && typeof value === 'object' ? (value as Record<string, unknown>)[key] : undefined
  }
  const empty = value === undefined || value === null || value === '' || (typeof value === 'number' && Number.isNaN(value))
  if (empty) throw new Error(`theme "${theme.name}" is missing ${path}`)
  return value
}

const curve = (theme: Theme, path: string) => {
  const c = need(theme, path)
  if (!Array.isArray(c) || c.length !== 4) throw new Error(`theme "${theme.name}" is missing ${path}`)
  return `cubic-bezier(${c.join(', ')})`
}

const block = (selector: string, lines: string[]) => `${selector} {\n${lines.join('\n')}\n}`
const colors = (theme: Theme, scheme: 'light' | 'dark') => [
  ...colorKeys.map((key) => `  --weft-color-${key}: ${need(theme, `${scheme}.${key}`)};`),
  ...toneColors(theme, scheme),
]

const AA = 4.5

/** A colour the contrast checks read, which must be #RRGGBB. */
function hexOf(theme: Theme, path: string): string {
  const value = String(need(theme, path))
  if (!isHex(value)) throw new Error(`theme "${theme.name}": ${path} is "${value}"; give it as #RRGGBB so its contrast can be checked.`)
  return value
}

/**
 * Each tone's `-fg` (text on its solid fill) and `-text` (the tone as text on
 * the page and its tints), chosen to reach AA. The fg is the better of the
 * theme's light text (dark.text) and dark text (light.text). The text is the
 * tone pulled toward the scheme's text, in 5% steps, just far enough to read
 * on bg, surface and surface-raised and on each tint over them.
 */
function toneColors(theme: Theme, scheme: 'light' | 'dark'): string[] {
  const lightInk = hexOf(theme, 'dark.text')
  const darkInk = hexOf(theme, 'light.text')
  const text = hexOf(theme, `${scheme}.text`)
  const grounds = (['bg', 'surface', 'surface-raised'] as const).map((g) => hexOf(theme, `${scheme}.${g}`))
  const lines: string[] = []
  for (const tone of tones) {
    const fill = hexOf(theme, `${scheme}.${tone}`)
    const fg = contrast(fill, lightInk) >= contrast(fill, darkInk) ? lightInk : darkInk
    if (contrast(fill, fg) < AA) {
      throw new Error(
        `theme "${theme.name}": neither its light text (${lightInk}) nor its dark text (${darkInk}) reaches 4.5:1 on ${scheme}.${tone} (${fill}); pick a ${tone} colour one of them can be read on.`,
      )
    }
    const behind = grounds.flatMap((g) => [0, ...Object.values(tints)].map((t) => mix(fill, g, t / 100)))
    let toned: string | undefined
    for (let share = 100; share >= 0 && !toned; share -= 5) {
      const candidate = mix(fill, text, share / 100)
      if (behind.every((b) => contrast(candidate, b) >= AA)) toned = candidate
    }
    if (!toned) throw new Error(`theme "${theme.name}": ${scheme}.text (${text}) does not reach 4.5:1 on its own grounds; darken or lighten it.`)
    lines.push(`  --weft-color-${tone}-fg: ${fg};`, `  --weft-color-${tone}-text: ${toned};`)
  }
  return lines
}

const pointerSizes = ['  --weft-control-height: 36px;', '  --weft-hit-min: 24px;']
const touchSizes = ['  --weft-control-height: 44px;', '  --weft-hit-min: 44px;']
const appDensity = ['  --weft-text-base: 0.875rem;', '  --weft-page-max: none;', '  --weft-section-gap: var(--weft-space-6);']
const landingDensity = ['  --weft-text-base: 1.125rem;', '  --weft-page-max: 1200px;', '  --weft-section-gap: var(--weft-space-8);']

/**
 * Render a theme as CSS custom properties: scheme blocks, then density blocks.
 * Without a WeftProvider the input comes from `(pointer: coarse)`; a provider's
 * `data-weft-input` overrides it for its subtree. Throws for any missing token.
 */
export function themeToCss(theme: Theme): string {
  const shared = [
    '  color-scheme: light;',
    ...colors(theme, 'light'),
    ...spaceKeys.map((k) => `  --weft-space-${k}: ${need(theme, `space.${k}`)}px;`),
    ...radiusKeys.map((k) => `  --weft-radius-${k}: ${need(theme, `radius.${k}`)}px;`),
    ...controlKeys.map((k) => `  --weft-control-${k}: ${need(theme, `control.${k}`)}px;`),
    ...motionKeys.map((k) => `  --weft-duration-${k}: ${need(theme, `motion.${k}`)}ms;`),
    `  --weft-ease-standard: ${curve(theme, 'ease.standard')};`,
    `  --weft-ease-emphasized: ${curve(theme, 'ease.emphasized')};`,
    `  --weft-font-sans: ${need(theme, 'font.sans')};`,
    `  --weft-font-mono: ${need(theme, 'font.mono')};`,
    `  --weft-weight-body: ${need(theme, 'font.weightBody')};`,
    `  --weft-weight-heading: ${need(theme, 'font.weightHeading')};`,
    `  --weft-weight-title: ${need(theme, 'font.weightTitle')};`,
    ...pointerSizes,
    ...appDensity,
  ]
  const dark = ['  color-scheme: dark;', ...colors(theme, 'dark')]
  return [
    `/* Weft theme: ${theme.name}. Generated by @weft/themes; do not edit. */`,
    block(':root, [data-weft-scheme="light"]', shared),
    block('[data-weft-scheme="dark"]', dark),
    `@media (prefers-color-scheme: dark) {\n${block(':root:not([data-weft-scheme="light"])', dark)}\n}`,
    `@media (pointer: coarse) {\n${block(':root:not([data-weft-input])', touchSizes)}\n}`,
    block('[data-weft-input="pointer"]', pointerSizes),
    block('[data-weft-input="touch"]', touchSizes),
    block('[data-weft-surface="landing"]', landingDensity),
    block('[data-weft-surface="app"]', appDensity),
    '',
  ].join('\n\n')
}
