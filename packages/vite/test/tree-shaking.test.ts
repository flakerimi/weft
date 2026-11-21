// Importing one component from @weft/ui ships that component and what it
// draws with, nothing else: an app built with Vite (Vapor, production, the
// icons plugin) that mounts `{ Button }` from the index has no Dialog, no
// Table, no overlay stack and none of the overlays' default icons in its
// JavaScript, and no other component's CSS. @weft/ui's package.json makes
// this so: it declares its scripts free of side effects and its stylesheets
// as having them, which also keeps Button's CSS in the build.
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { buildAlone, type AloneBuild } from './alone.ts'

let built: AloneBuild
let js: string
const env = process.env.NODE_ENV

beforeAll(async () => {
  // A production build, as an app's `vite build` is.
  process.env.NODE_ENV = 'production'
  built = await buildAlone("import { createVaporApp } from 'vue'\nimport { Button } from '@weft/ui'\ncreateVaporApp(Button).mount('#app')\n")
  js = built.chunks.filter((c) => !c.vue).map((c) => c.code).join('\n')
}, 60_000)
afterAll(() => {
  process.env.NODE_ENV = env
})

/** The @weft/ui source files the build holds code of, from packages/ui. */
const sources = () => [...new Set(built.modules.map((m) => /packages\/ui\/(src\/[^?]+)/.exec(m.split('\\').join('/'))?.[1]).filter(Boolean))].sort()

describe('import { Button } from "@weft/ui"', () => {
  it('builds Button and what it draws with, and no other component', () => {
    expect(sources()).toEqual([
      'src/basics/Button.vue',
      'src/basics/Icon.vue',
      'src/basics/Spinner.vue',
      'src/basics/address.ts',
      'src/basics/icons.ts',
      'src/forms/form-state.ts',
    ])
    expect(js).toContain('weft-button')
  })

  it('holds no Dialog, no Table, no overlay stack and no default overlay icon', () => {
    // field.ts is the form controls' wiring: Button reads only the Form's state (form-state.ts).
    for (const file of ['Dialog.vue', 'Table.vue', 'overlayStack.ts', 'overlay.ts', 'CloseButton.vue', 'DropdownMenu.vue', 'forms/field.ts']) {
      expect(built.modules.some((m) => m.includes(file)), file).toBe(false)
    }
    for (const trace of ['weft-dialog', 'weft-table', 'weft-overlays', 'weft-close-button', 'weft-menu']) expect(js, trace).not.toContain(trace)
    // The close button's cross and the menus' chevron, drawn inline by the overlays.
    expect(js).not.toContain('M18 6 6 18')
    expect(js).not.toContain('m9 18 6-6-6-6')
  })

  it("keeps Button's CSS and those of what it draws with, and no other component's", () => {
    expect(built.css).toMatch(/\.weft-button\{/)
    expect(built.css).toMatch(/\.weft-spinner\{/)
    expect(built.css).toMatch(/\[data-tone=primary\]|\[data-tone="primary"\]/)
    for (const other of ['.weft-dialog', '.weft-table', '.weft-input', '.weft-menu', '.weft-slider', '.weft-card']) {
      expect(built.css, other).not.toContain(other)
    }
  })
})
