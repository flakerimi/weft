import { fileURLToPath } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { build, type Rolldown } from 'vite'
import { describe, expect, it } from 'vitest'
import { weft } from '../src/plugin.ts'

const fixtures = fileURLToPath(new URL('./fixtures', import.meta.url))

async function buildRegistry(islands: string[]) {
  const out = (await build({
    configFile: false,
    root: fixtures,
    logLevel: 'silent',
    plugins: [vue(), weft({ islands })],
    build: {
      write: false,
      minify: false,
      modulePreload: false,
      rolldownOptions: {
        input: 'virtual:weft/islands',
        external: ['vue'],
        preserveEntrySignatures: 'strict',
      },
    },
  })) as Rolldown.RolldownOutput
  return out.output.filter((o): o is Rolldown.OutputChunk => o.type === 'chunk')
}

describe('weft() plugin', () => {
  it('maps each island name to a dynamic import of its own chunk', async () => {
    const chunks = await buildRegistry(['site/components/Counter.vue', 'runtime/Boom.vue'])
    const entry = chunks.find((c) => c.isEntry)!
    const counter = chunks.find((c) => c.facadeModuleId?.endsWith('/site/components/Counter.vue'))!
    const boom = chunks.find((c) => c.facadeModuleId?.endsWith('/runtime/Boom.vue'))!

    expect(counter.isDynamicEntry).toBe(true)
    expect(boom.isDynamicEntry).toBe(true)
    const loaderOf = (name: string) => entry.code.split('\n').find((l) => l.trim().startsWith(`"${name}":`))
    const file = (c: Rolldown.OutputChunk) => c.fileName.split('/').pop()
    expect(loaderOf('Counter')).toContain(`import("./${file(counter)}")`)
    expect(loaderOf('Boom')).toContain(`import("./${file(boom)}")`)
    expect(entry.code).not.toContain('<button') // the component lives in its chunk, not the registry
  })

  it('fails the build when two islands share a name', async () => {
    await expect(buildRegistry(['site/components/Counter.vue', 'dup/Counter.vue'])).rejects.toThrow(
      'Islands site/components/Counter.vue and dup/Counter.vue are both named Counter; rename one of them.',
    )
  })
})
