// Reading what a module takes from its imports and what an index passes on,
// from the code Vite's SSR transform makes, and which packages declare their
// scripts free of side effects. page-css.test.ts shows what it is for: an
// island that takes { Button } from @weft/ui ships Button's CSS alone.
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { createServer } from 'vite'
import { expect, it } from 'vitest'
import { exportsOf, importsOf, sideEffectFree } from '../src/imports.ts'

/** The SSR transform's output for `source`, as Vite writes it for renderPage. */
async function ssr(source: string): Promise<string> {
  const server = await createServer({ configFile: false, logLevel: 'error', server: { middlewareMode: true, ws: false }, appType: 'custom' })
  try {
    return (await server.ssrTransform(source, null, '/x.js', source))!.code
  } finally {
    await server.close()
  }
}

it('reads the names taken from each import, and all for a namespace, export * or a dynamic import', async () => {
  const code = await ssr(
    [
      "import { Button, Badge as B } from '/ui.js'",
      "import Thing from '/thing.js'",
      "import * as All from '/all.js'",
      "import Both, * as Ns from '/both.js'",
      "import '/side.js'",
      "export * from '/again.js'",
      "export { x } from '/x.js'",
      "const later = () => import('/later.js')",
      'console.log(Button, B, Thing, All, Both, Ns, later)',
    ].join('\n'),
  )
  const got = Object.fromEntries([...importsOf(code)].map(([url, names]) => [url, names && [...names].sort()]))
  expect(got).toEqual({
    '/ui.js': ['Badge', 'Button'],
    '/thing.js': ['default'],
    '/all.js': null,
    '/both.js': null,
    '/side.js': null,
    '/again.js': null,
    '/x.js': ['x'],
    '/later.js': null,
  })
})

it('reads where an index takes each name it exports from', async () => {
  const code = await ssr(
    [
      "export { default as Button } from '/Button.vue'",
      "export { a, b as c } from '/ab.js'",
      "export * as Ns from '/ns.js'",
      "export * from '/rest.js'",
      'export const own = 1',
    ].join('\n'),
  )
  const { names, all } = exportsOf(code)
  expect(Object.fromEntries(names)).toEqual({
    Button: { url: '/Button.vue', name: 'default' },
    a: { url: '/ab.js', name: 'a' },
    c: { url: '/ab.js', name: 'b' },
    Ns: { url: '/ns.js', name: null },
    own: null,
  })
  expect(all).toEqual(['/rest.js'])
})

it("takes @weft/ui's scripts as free of side effects and its stylesheets as not, from its package.json", () => {
  const ui = dirname(dirname(createRequire(import.meta.url).resolve('@weft/ui')))
  expect(sideEffectFree(join(ui, 'src/index.ts'))).toBe(true)
  expect(sideEffectFree(join(ui, 'src/basics/Button.vue'))).toBe(true)
  expect(sideEffectFree(join(ui, 'src/styles/button.css'))).toBe(false)
  // A package that says nothing may have side effects (Vite's own package.json has no such field); false means none.
  expect(sideEffectFree(createRequire(import.meta.url).resolve('vite'))).toBe(false)
  expect(sideEffectFree(createRequire(import.meta.url).resolve('lucide-static'))).toBe(true)
})
