// Toned components draw their text in colours @weft/themes checks for AA at
// build: --weft-color-<tone>-fg on a solid fill, --weft-color-<tone>-text on
// the page and on tints. The tints here must be the ones the themes check.
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { neutral, themeToCss, tints } from '@weft/themes'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import Badge from '../src/basics/Badge.vue'
import Button from '../src/basics/Button.vue'
import { mount } from './mount.ts'
import { loadStyles } from './styles.ts'

// Every stylesheet: tone.css sets the tones, the components' own sheets tint them.
const styles = join(dirname(fileURLToPath(import.meta.url)), '../src/styles')
const css = readdirSync(styles)
  .filter((f) => f.endsWith('.css'))
  .sort()
  .map((f) => readFileSync(join(styles, f), 'utf8'))
  .join('\n')

describe('tone colours', () => {
  it('tint only by the percentages the themes check text against', () => {
    const used = [...css.matchAll(/background: color-mix\(in srgb, var\(--weft-tone, [^)]+\) (\d+)%, transparent\)/g)].map((m) => Number(m[1]))
    expect(used.length).toBeGreaterThan(0)
    for (const share of used) expect(Object.values(tints)).toContain(share)
  })

  it('draw tinted text and solid text in the colours the theme chose', () => {
    for (const tone of ['primary', 'danger', 'warning', 'success']) {
      const rule = new RegExp(`\\[data-tone="${tone}"\\] \\{[^}]*\\}`).exec(css)![0]
      expect(rule).toContain(`--weft-tone-text: var(--weft-color-${tone}-text,`)
      expect(rule).toContain(`--weft-tone-fg: var(--weft-color-${tone}-fg,`)
    }
    expect(css).not.toMatch(/--weft-tone-text: color-mix/)
  })

  describe('in the document', () => {
    let unload: () => void
    let mounted: { unmount(): void }[] = []
    beforeEach(() => { unload = loadStyles() })
    afterEach(() => {
      mounted.forEach((m) => m.unmount())
      mounted = []
      unload()
    })
    const value = (name: string) => new RegExp(`--weft-color-${name}: (#[0-9A-F]{6});`).exec(themeToCss(neutral))![1]!
    const colourOf = (component: object, props: Record<string, unknown>) => {
      const m = mount(component as never, props)
      mounted.push(m)
      return getComputedStyle(m.el.firstElementChild!).color.toUpperCase()
    }

    it('soft, outline and ghost text is the tone\'s text colour; solid text its fg', () => {
      for (const variant of ['soft', 'outline', 'ghost']) {
        expect(colourOf(Button, { label: 'x', tone: 'warning', variant }), variant).toBe(value('warning-text'))
        expect(colourOf(Badge, { label: 'x', tone: 'success', variant }), variant).toBe(value('success-text'))
      }
      expect(colourOf(Button, { label: 'x', tone: 'danger' })).toBe(value('danger-fg'))
    })
  })
})
