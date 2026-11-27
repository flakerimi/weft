// The three showcase themes, synthwave, editor and sandstorm: real themes an
// app can build with, each with a look of its own, and the checks weft.ts has
// (every colour as #RRGGBB, focus at 3:1, every derived colour at AA).
import { describe, expect, it } from 'vitest'
import { contrast } from '../src/contrast.ts'
import { editor } from '../src/editor.ts'
import { sandstorm } from '../src/sandstorm.ts'
import { synthwave } from '../src/synthwave.ts'
import { themeToCss } from '../src/to-css.ts'
import { colorKeys, type Theme } from '../src/tokens.ts'
import * as index from '../src/index.ts'

const schemes = ['light', 'dark'] as const

describe.each([
  ['synthwave', synthwave],
  ['editor', editor],
  ['sandstorm', sandstorm],
] as [string, Theme][])('%s', (name, theme) => {
  it('is named for its file and exported from the package', () => {
    expect(theme.name).toBe(name)
    expect((index as Record<string, unknown>)[name]).toBe(theme)
  })

  it('has every colour key as #RRGGBB in both schemes', () => {
    for (const scheme of schemes) {
      for (const key of colorKeys) expect(theme[scheme][key], `${scheme}.${key}`).toMatch(/^#[0-9A-F]{6}$/)
    }
  })

  // A focus ring is a control's boundary: 3:1 against the page (WCAG 1.4.11).
  it('draws focus at 3:1 or more on the page, light and dark', () => {
    for (const scheme of schemes) {
      for (const ground of [theme[scheme].bg, theme[scheme].surface, theme[scheme]['surface-raised']]) {
        expect(contrast(theme[scheme].focus, ground), `${scheme} focus on ${ground}`).toBeGreaterThanOrEqual(3)
      }
    }
  })

  // A raised surface (a tile, a stage) separates from the surface below it.
  it('raises surfaces a step above the surface', () => {
    for (const scheme of schemes) {
      expect(contrast(theme[scheme].surface, theme[scheme]['surface-raised']), scheme).toBeGreaterThan(1.03)
    }
  })

  it('renders to CSS, every derived colour reaching AA', () => {
    expect(() => themeToCss(theme)).not.toThrow()
  })
})

it('synthwave: neon on deep violet, the same hues on pale lilac, Space Grotesk for text', () => {
  // Deep violet: blue and red well over green, and dark.
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(synthwave.dark.bg.slice(i, i + 2), 16)) as [number, number, number]
  expect(b > g && r > g && b < 0x40).toBe(true)
  expect(contrast(synthwave.light.bg, '#FFFFFF')).toBeLessThan(1.15)
  expect(synthwave.font.sans).toMatch(/^'Space Grotesk Variable', /)
  expect(synthwave.font.mono).toMatch(/^'JetBrains Mono Variable', /)
})

it('editor: monospace everywhere, JetBrains Mono for text too', () => {
  expect(editor.font.sans).toMatch(/^'JetBrains Mono Variable', /)
  expect(editor.font.mono).toMatch(/^'JetBrains Mono Variable', /)
  // A warm charcoal: red at least green at least blue, and dark.
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(editor.dark.bg.slice(i, i + 2), 16)) as [number, number, number]
  expect(r >= g && g >= b && r < 0x40).toBe(true)
})

it('sandstorm: soft and rounded, larger radii than weft, Quicksand for text', () => {
  expect(sandstorm.font.sans).toMatch(/^'Quicksand Variable', /)
  expect(sandstorm.radius.control).toBeGreaterThan(8)
  expect(sandstorm.radius.card).toBeGreaterThan(12)
  expect(sandstorm.radius.panel).toBeGreaterThan(16)
  // Sand by day, deep umber at dusk: warm both times.
  for (const bg of [sandstorm.light.bg, sandstorm.dark.bg]) {
    const [r, g, b] = [1, 3, 5].map((i) => parseInt(bg.slice(i, i + 2), 16)) as [number, number, number]
    expect(r > g && g > b, bg).toBe(true)
  }
})
