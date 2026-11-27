// weftAutoImport(): a page or component writes `<Button>` and gets
// `import { Button } from '@weft/ui'`, unless the name is the author's own.
import { spawnSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, statSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer, resolveConfig } from 'vite'
import { afterAll, beforeAll, describe, expect, it } from 'vitest'
import { autoImport, componentNames, weftAutoImport } from '../src/auto-import.ts'
import { buildSite } from '../src/build.ts'
import { siteConfig } from '../src/config.ts'

const root = fileURLToPath(new URL('./fixtures/auto-import', import.meta.url))
const indexFile = fileURLToPath(new URL('../../ui/src/index.ts', import.meta.url))
const names = new Set(['Badge', 'Button', 'Card', 'Island', 'Stack'])

/** The `<script setup>` block of an SFC, opening tag to closing tag. */
const setupOf = (code: string) => /<script\b[^>]*\bsetup\b[^>]*>[\s\S]*?<\/script>/.exec(code)?.[0]

describe('autoImport()', () => {
  it('adds the import a tag needs', () => {
    const out = autoImport('<script setup vapor lang="ts">\n</script>\n\n<template><Button /></template>\n', names)
    expect(out).not.toBeNull()
    expect(setupOf(out!.code)).toBe(`<script setup vapor lang="ts">\nimport { Button } from '@weft/ui'\n</script>`)
    expect(out!.code).toContain('<template><Button /></template>')
    expect(out!.map.mappings).not.toBe('')
  })

  it('puts it on a line of its own when the code starts on the opening tag\'s line', () => {
    const out = autoImport('<script setup vapor>const a = 1</script>\n<template><Card>{{ a }}</Card></template>\n', names)!
    expect(setupOf(out.code)).toBe(`<script setup vapor>\nimport { Card } from '@weft/ui'\nconst a = 1</script>`)
  })

  it('puts it in the <script setup> beside a plain <script>, keeping its attributes', () => {
    const code = [
      '<script lang="ts">',
      "export const getPaths = async () => [{ slug: 'a' }]",
      '</script>',
      '',
      '<script setup lang="ts" generic="T">',
      'const props = defineProps<{ slug: string }>()',
      '</script>',
      '',
      '<template><Card>{{ props.slug }}</Card></template>',
      '',
    ].join('\n')
    const out = autoImport(code, names)!
    expect(out.code.startsWith('<script lang="ts">\nexport const getPaths')).toBe(true)
    expect(setupOf(out.code)).toBe(
      `<script setup lang="ts" generic="T">\nimport { Card } from '@weft/ui'\nconst props = defineProps<{ slug: string }>()\n</script>`,
    )
  })

  describe('keeps the author\'s names', () => {
    const page = (script: string, template: string, lang = ' lang="ts"') =>
      `<script setup vapor${lang}>\n${script}\n</script>\n\n<template>${template}</template>\n`

    it('a name the script imports', () => {
      expect(autoImport(page("import Button from './Button.vue'", '<Button />'), names)).toBeNull()
      expect(autoImport(page("import { Thing as Button } from './things.ts'", '<Button />'), names)).toBeNull()
      expect(autoImport(page("import * as Button from './button.ts'", '<Button />'), names)).toBeNull()
    })

    it('a name the script declares', () => {
      expect(autoImport(page("const Card = { name: 'mine' }", '<Card />'), names)).toBeNull()
      expect(autoImport(page('function Card() { return null }', '<Card />'), names)).toBeNull()
      expect(autoImport(page('class Card {}', '<Card />'), names)).toBeNull()
      expect(autoImport(page('const { Card } = useParts()', '<Card />'), names)).toBeNull()
      expect(autoImport(page('const [Card] = useParts()', '<Card />', ''), names)).toBeNull()
      expect(autoImport(page('enum Card { A }', '<Card />'), names)).toBeNull()
    })

    it('a name a plain <script> beside it imports or declares', () => {
      const code = [
        '<script lang="ts">',
        "import Badge from './MyBadge.vue'",
        'export const Card = {}',
        '</script>',
        '<script setup vapor lang="ts">',
        '</script>',
        '<template><Badge /><Card /></template>',
      ].join('\n')
      expect(autoImport(code, names)).toBeNull()
    })

    it("the component's own name, which Vue resolves to the file itself", () => {
      const code = page('defineProps<{ depth: number }>()', '<Card v-if="depth" :depth="depth - 1" />')
      expect(autoImport(code, names, '/app/components/Card.vue')).toBeNull()
      expect(autoImport(code, names, '/app/components/Panel.vue')).not.toBeNull()
      // A page's route name is lowercase: pages/card.vue showing a <Card> means Weft's, not a page in itself.
      expect(autoImport(code, names, '/app/pages/card.vue')).not.toBeNull()
    })

    it('a lowercase tag', () => {
      expect(autoImport(page('', '<button>Save</button><card />'), names)).toBeNull()
    })

    it('a tag inside <pre v-pre>', () => {
      expect(autoImport(page('', '<pre v-pre><Button>{{ raw }}</Button></pre>'), names)).toBeNull()
      expect(autoImport(page('', '<div v-pre><section><Card /></section></div>'), names)).toBeNull()
    })

    it('a name that is not a Weft component', () => {
      expect(autoImport(page('', '<Hello /><Transition><span /></Transition>'), names)).toBeNull()
    })

    it('only the names that are not the author\'s', () => {
      const out = autoImport(page("import Button from './Button.vue'", '<Button /><Card />'), names)!
      expect(setupOf(out.code)).toContain("import { Card } from '@weft/ui'\n")
      expect(setupOf(out.code)).not.toContain('{ Button')
    })
  })

  it('adds a script when there is none', () => {
    const out = autoImport('<template>\n  <Card>Hello</Card>\n</template>\n', names)!
    expect(out.code).toBe(
      `<script setup vapor lang="ts">\nimport { Card } from '@weft/ui'\n</script>\n\n<template>\n  <Card>Hello</Card>\n</template>\n`,
    )
  })

  it('one import line for several tags, each name once, sorted', () => {
    const template = '<Stack><Card><Button /></Card><Badge /><Card><Button /></Card></Stack>'
    const out = autoImport(`<script setup vapor lang="ts">\nconst a = 1\n</script>\n<template>${template}</template>\n`, names)!
    const lines = out.code.split('\n').filter((l) => l.includes('@weft/ui'))
    expect(lines).toEqual(["import { Badge, Button, Card, Stack } from '@weft/ui'"])
  })

  it('finds tags in slots, v-if branches and nested templates', () => {
    const template = '<Stack><template #header><Badge v-if="a" /><Button v-else /></template><template v-for="x in xs"><Card /></template></Stack>'
    const out = autoImport(`<template>${template}</template>`, names)!
    expect(out.code).toContain("import { Badge, Button, Card, Stack } from '@weft/ui'")
  })

  it('leaves a file it cannot parse to the Vue plugin', () => {
    expect(autoImport('<script setup lang="ts">\nconst = \n</script>\n<template><Card /></template>', names)).toBeNull()
    expect(autoImport('<template><Card></template>', names)).toBeNull()
  })
})

describe('componentNames()', () => {
  it('componentNames reads the real index', () => {
    const source = readFileSync(indexFile, 'utf8')
    const found = componentNames(source)
    expect(found).toContain('Button')
    expect(found).toContain('Island')
    expect(found).not.toContain('useWeft')
    expect(found).not.toContain('Gap')
    // Counted a second way: the lines that re-export a .vue file's default.
    const lines = source.split('\n').filter((l) => /^export \{ default as \w+\b.*\} from '[^']+\.vue'$/.test(l.trim()))
    expect(lines.length).toBeGreaterThan(0)
    expect(found.length).toBe(lines.length)
    expect(new Set(found).size).toBe(found.length)
  })

  it('reads only `default as` re-exports of .vue files', () => {
    const source = [
      "export { default as Button, type ButtonProps } from './Button.vue'",
      "export { default as helper } from './helper.ts'",
      "export { Card } from './Card.vue'",
      "export { default as Panel } from \"./Panel.vue\"",
      "export { useThing } from './thing.ts'",
    ].join('\n')
    expect(componentNames(source)).toEqual(['Button', 'Panel'])
  })
})

describe('weftAutoImport()', () => {
  let parent: string
  beforeAll(() => {
    parent = mkdtempSync(join(tmpdir(), 'weft-auto-import-'))
  })
  afterAll(() => rmSync(parent, { recursive: true, force: true }))

  it('the declaration file names every component', async () => {
    const dts = join(parent, 'types', 'components.d.ts')
    const start = async () => {
      const server = await createServer({
        root,
        configFile: false,
        logLevel: 'silent',
        server: { middlewareMode: true, ws: false },
        optimizeDeps: { noDiscovery: true, include: [] },
        plugins: [weftAutoImport({ dts })],
      })
      await server.close()
    }
    await start()
    const text = readFileSync(dts, 'utf8')
    const all = componentNames(readFileSync(indexFile, 'utf8'))
    expect(text).toContain("declare module 'vue'")
    expect(text).toContain('interface GlobalComponents')
    for (const name of all) expect(text).toContain(`    ${name}: typeof import('@weft/ui')['${name}']\n`)
    const listed = [...text.matchAll(/^ {4}(\w+): typeof import\('@weft\/ui'\)/gm)].map((m) => m[1])
    expect(listed).toEqual([...all].sort())

    // Written only when its content changed.
    const before = statSync(dts).mtimeMs
    await new Promise((r) => setTimeout(r, 20))
    await start()
    expect(statSync(dts).mtimeMs).toBe(before)
  })

  // weft create and weft build run it, so `pnpm typecheck` checks Weft's
  // components before the first build or dev run has written the file.
  it('weft-types --root writes the declaration file the plugin writes, and prints nothing', async () => {
    const site = join(parent, 'types-site')
    mkdirSync(join(site, 'pages'), { recursive: true })
    symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(site, 'node_modules'))
    const bin = fileURLToPath(new URL('../bin/weft-types.mjs', import.meta.url))
    // Relative to where it runs, as `weft` gives it: `--root web` in the app.
    writeFileSync(
      join(site, 'pages', '[section]', '[id].vue'),
      (mkdirSync(join(site, 'pages', '[section]'), { recursive: true }), '<script lang="ts">\nexport const runtime = true\n</script>\n<template><p /></template>\n'),
    )
    writeFileSync(
      join(site, 'pages', 'docs', '[...slug].vue'),
      (mkdirSync(join(site, 'pages', 'docs'), { recursive: true }), '<script lang="ts">\nexport const runtime = true\n</script>\n<template><p /></template>\n'),
    )
    writeFileSync(
      join(site, 'pages', "it's a \"quote\"\\dir", '[id].vue'),
      (mkdirSync(join(site, 'pages', "it's a \"quote\"\\dir"), { recursive: true }), '<script lang="ts">\nexport const runtime = true\n</script>\n<template><p /></template>\n'),
    )
    writeFileSync(
      join(site, 'pages', 'blog', '[slug].vue'),
      (mkdirSync(join(site, 'pages', 'blog'), { recursive: true }), '<script lang="ts">\nexport const getPaths = async () => [{ slug: "a" }]\n</script>\n<template><p /></template>\n'),
    )
    const ran = spawnSync(process.execPath, [bin, '--root', 'types-site'], { encoding: 'utf8', cwd: parent })
    expect(ran).toMatchObject({ status: 0, stdout: '', stderr: '' })
    const written = readFileSync(join(site, '.weft-types', 'components.d.ts'), 'utf8')
    expect(written).toContain(
      `declare module '@weft/ui' {\n  interface WeftRouteParams {\n    "/:section/:id": { section: string; id: string }\n    "/docs/:...slug": { slug: string }\n    "/it's a \\"quote\\"\\\\dir/:id": { id: string }\n  }\n}`,
    )
    // Built-ahead getPaths pages are not runtime pages, so they are omitted from WeftRouteParams.
    expect(written).not.toContain('/blog/:slug')

    // Compare byte-for-byte with what the plugin writes for the same root.
    const pluginDts = join(parent, 'plugin-components.d.ts')
    const srv = await createServer({
      root: site,
      configFile: false,
      logLevel: 'silent',
      server: { middlewareMode: true, ws: false },
      optimizeDeps: { noDiscovery: true, include: [] },
      plugins: [weftAutoImport({ dts: pluginDts })],
    })
    // Trigger buildStart on the plugin to write pluginDts, then test configureServer watcher refresh.
    for (const p of srv.config.plugins) {
      if (p.name === 'weft:auto-import' && typeof p.buildStart === 'function') {
        await (p.buildStart as () => void)()
      }
    }
    expect(written).toBe(readFileSync(pluginDts, 'utf8'))

    writeFileSync(
      join(site, 'pages', 'live', '[code].vue'),
      (mkdirSync(join(site, 'pages', 'live'), { recursive: true }), '<script lang="ts">\nexport const runtime = true\n</script>\n<template><p /></template>\n'),
    )
    srv.watcher.emit('add', join(site, 'pages', 'live', '[code].vue'))
    expect(readFileSync(pluginDts, 'utf8')).toContain('"/live/:code": { code: string }')
    rmSync(join(site, 'pages', 'live', '[code].vue'))
    srv.watcher.emit('unlink', join(site, 'pages', 'live', '[code].vue'))
    expect(readFileSync(pluginDts, 'utf8')).not.toContain('"/live/:code"')
    await srv.close()

    // Type-check useParams(route) against the generated WeftRouteParams: known
    // patterns and their param names compile, and an unknown pattern, a
    // built-ahead getPaths pattern or a param name the pattern lacks fails vue-tsc.
    const vueTsc = fileURLToPath(new URL('../../ui/node_modules/.bin/vue-tsc', import.meta.url))
    writeFileSync(
      join(site, 'tsconfig.json'),
      JSON.stringify({
        compilerOptions: {
          target: 'ES2022',
          module: 'ESNext',
          moduleResolution: 'bundler',
          strict: true,
          noEmit: true,
          skipLibCheck: true,
          allowImportingTsExtensions: true,
          types: ['vite/client'],
        },
        include: ['.weft-types/components.d.ts', 'check.ts'],
      }),
    )
    writeFileSync(
      join(site, 'check.ts'),
      [
        "import { useParams } from '@weft/ui'",
        "const a: string = useParams('/:section/:id').id",
        "const b: string = useParams('/docs/:...slug').slug",
        `const q: string = useParams('/it\\'s a "quote"\\\\dir/:id').id`,
        'const c: Record<string, string> = useParams()',
        '// @ts-expect-error wrong param name on /:section/:id',
        "useParams('/:section/:id').slug",
        '// @ts-expect-error unknown route pattern',
        "useParams('/missing/:id')",
        '// @ts-expect-error built-ahead getPaths page is not in WeftRouteParams',
        "useParams('/blog/:slug')",
        'void a, b, q, c',
      ].join('\n') + '\n',
    )
    const checked = spawnSync(vueTsc, ['-p', join(site, 'tsconfig.json')], { encoding: 'utf8' })
    expect(checked.status, checked.stdout + checked.stderr).toBe(0)

    const bare = mkdtempSync(join(tmpdir(), 'weft-types-bare-'))
    try {
      // Without the NODE_PATH that pnpm's bin shim gives this test, which reaches this repository's packages.
      const env = { ...process.env }
      delete env.NODE_PATH
      const refused = spawnSync(process.execPath, [bin, '--root', bare], { encoding: 'utf8', env })
      expect(refused.status).toBe(1)
      expect(refused.stderr).toBe(`weft-types: @weft/ui cannot be found from ${bare}; run pnpm install in the app and run the command again.\n`)
      const usage = spawnSync(process.execPath, [bin], { encoding: 'utf8' })
      expect(usage.status).toBe(2)
      expect(usage.stderr).toBe('weft-types: --root is required; usage is weft-types --root <site>.\n')
    } finally {
      rmSync(bare, { recursive: true, force: true })
    }
  })

  it('siteConfig adds it once, unless the site has it, and it runs before vite:vue', async () => {
    const resolved = async (site: string, command: 'serve' | 'build') =>
      (await resolveConfig(await siteConfig(site, { command, mode: 'production' }, {}), command)).plugins
    const pluginNames = async (site: string, command: 'serve' | 'build') => (await resolved(site, command)).map((p) => p.name)
    for (const command of ['serve', 'build'] as const) {
      const listed = await pluginNames(root, command)
      expect(listed.filter((n) => n === 'weft:auto-import')).toHaveLength(1)
      expect(listed.indexOf('weft:auto-import')).toBeLessThan(listed.indexOf('vite:vue'))
    }
    const own = join(parent, 'own-config')
    mkdirSync(own)
    writeFileSync(join(own, 'vite.config.mjs'), "export default { plugins: [{ name: 'weft:auto-import', enforce: 'pre' }] }\n")
    const plugins = (await resolved(own, 'build')).filter((p) => p.name === 'weft:auto-import')
    expect(plugins).toHaveLength(1)
    expect(plugins[0]).not.toHaveProperty('transform')
  })

  it('a page without imports builds', async () => {
    const out = join(parent, 'dist')
    await buildSite({ root, outDir: out })
    const card = readFileSync(join(out, 'pages', 'card', 'index.html'), 'utf8')
    expect(card).toMatch(/<div class="weft-card"[^>]*>[\s\S]*Hello from a card/)
    expect(card).toMatch(/<style>[\s\S]*\.weft-card\b[\s\S]*<\/style>/)

    // An island's component, compiled for the server and for the browser, with no import of its own.
    const island = readFileSync(join(out, 'pages', 'island', 'index.html'), 'utf8')
    expect(island).toContain('class="weft-badge"')
    expect(island).toMatch(/<style>[\s\S]*\.weft-badge\b[\s\S]*<\/style>/)
    const assets = readdirSync(join(out, 'assets')).filter((f) => f.endsWith('.js'))
    const counter = assets.map((f) => readFileSync(join(out, 'assets', f), 'utf8')).find((js) => js.includes('weft-badge'))
    expect(counter).toBeDefined()
  })
})
