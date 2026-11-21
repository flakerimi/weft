import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { build, type Rolldown } from 'vite'
import { describe, expect, it } from 'vitest'
import { lucideBody, weftIcons, type IconsOptions } from '../src/icons.ts'
import { renderPage } from '../src/render.ts'

const root = fileURLToPath(new URL('./fixtures/icons', import.meta.url))

async function bundle(input: Record<string, string>, opts?: IconsOptions) {
  const out = (await build({
    configFile: false,
    root,
    logLevel: 'silent',
    plugins: [vue(), weftIcons(opts)],
    build: {
      write: false,
      minify: false,
      modulePreload: false,
      rolldownOptions: { input, external: ['vue'], preserveEntrySignatures: 'strict' },
    },
  })) as Rolldown.RolldownOutput
  return out.output.filter((o): o is Rolldown.OutputChunk => o.type === 'chunk')
}

/** The icon names whose drawings a chunk carries. */
const drawn = (code: string) => [...code.matchAll(/registerIcon\("([^"]+)"/g)].map((m) => m[1]).sort()

describe('weftIcons()', () => {
  it('reads Lucide drawings without the svg wrapper or the licence comment', () => {
    const body = lucideBody('trash')!
    expect(body).toContain('<path d="M3 6h18"/>')
    expect(body).not.toMatch(/<svg|<!--|\n/)
    expect(lucideBody('not-an-icon')).toBeUndefined()
    expect(lucideBody('../package')).toBeUndefined()
  })

  it('bundles exactly the icons the code names, each drawn once', async () => {
    const chunks = await bundle({ page: 'pages/index.vue' })
    const code = chunks.map((c) => c.code).join('\n')
    // name= on Icon, icon= on a component, literals in a bound name, and icon: in a script.
    expect(drawn(code)).toEqual(['check', 'house', 'plus', 'trash', 'x'])
    expect(code).toContain(JSON.stringify(lucideBody('trash')))
    expect(code).not.toContain(lucideBody('star')!)
  })

  it('puts an island’s icons in its own chunk and nowhere else', async () => {
    const chunks = await bundle({ page: 'pages/index.vue', star: 'components/Starred.vue' })
    const star = chunks.find((c) => c.facadeModuleId?.endsWith('/components/Starred.vue'))!
    const page = chunks.find((c) => c.facadeModuleId?.endsWith('/pages/index.vue'))!
    expect(drawn(star.code)).toEqual(['star'])
    expect(drawn(page.code)).not.toContain('star')
  })

  it('ships no close or chevron glyph with a bundle that imports only Button', async () => {
    // The paths as the compiled templates hold them.
    const close = 'd=\\"M18 6 6 18\\"'
    const chevron = 'd=\\"m9 18 6-6-6-6\\"'
    const only = (await bundle({ only: 'components/OnlyButton.vue' })).map((c) => c.code).join('\n')
    expect(drawn(only)).toEqual([])
    for (const name of ['x', 'chevron-right', 'chevron-down', 'check']) expect(only, name).not.toContain(lucideBody(name)!)
    expect(only).not.toContain(close)
    expect(only).not.toContain(chevron)
    // Those glyphs are drawn inline by the components that use them, and come only with them.
    const overlays = (await bundle({ overlays: 'components/WithDialog.vue' })).map((c) => c.code).join('\n')
    expect(overlays).toContain(close)
    expect(overlays).toContain(chevron)
    expect(drawn(overlays)).toEqual([])
  })

  it('bundles names listed in the icons option, for names made at run time', async () => {
    const chunks = await bundle({ star: 'components/Starred.vue' }, { icons: ['bell'] })
    expect(drawn(chunks.map((c) => c.code).join('\n'))).toEqual(['bell', 'star'])
  })

  it.each([
    ['bad/Unknown.vue', 'bad/Unknown.vue uses the icon "trashh", which Lucide does not have; pick a name from lucide.dev/icons.'],
    ['bad/UnknownProp.vue', 'bad/UnknownProp.vue uses the icon "i-lucide-home", which Lucide does not have; pick a name from lucide.dev/icons.'],
    ['bad/unknown-data.ts', 'bad/unknown-data.ts uses the icon "gears-and-cogs", which Lucide does not have; pick a name from lucide.dev/icons.'],
  ])('fails the build for an unknown name in %s, with one sentence naming the file', async (file, sentence) => {
    await expect(bundle({ bad: file })).rejects.toThrow(sentence)
  })

  it('fails for an unknown name in the icons option', async () => {
    await expect(bundle({ star: 'components/Starred.vue' }, { icons: ['nope-nope'] })).rejects.toThrow(
      'weftIcons(): the icons option lists "nope-nope", which Lucide does not have; pick a name from lucide.dev/icons.',
    )
  })

  it('draws the icons into a page rendered on the server', async () => {
    const { html } = await renderPage({ root, page: 'pages/index.vue' })
    const svgs = [...html.matchAll(/<svg class="weft-icon"[^>]*>([\s\S]*?)<\/svg>/g)].map((m) => m[1])
    expect(svgs).toEqual([lucideBody('trash'), lucideBody('plus'), lucideBody('check'), lucideBody('house')])
    expect(html).toContain('aria-hidden="true"')
  })
})
