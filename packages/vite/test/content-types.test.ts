// A site types its pages' reads of virtual:weft/content from @weft/vite/virtual,
// as it would with @weft/vite in its node_modules.
import { mkdirSync, mkdtempSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'
import { afterAll, expect, it } from 'vitest'

const temps: string[] = []
afterAll(() => {
  for (const t of temps) rmSync(t, { recursive: true, force: true })
})

it('gives a site the types of virtual:weft/content through @weft/vite/virtual', () => {
  const site = mkdtempSync(join(tmpdir(), 'weft-content-types-'))
  temps.push(site)
  mkdirSync(join(site, 'node_modules/@weft'), { recursive: true })
  symlinkSync(fileURLToPath(new URL('..', import.meta.url)), join(site, 'node_modules/@weft/vite'))
  writeFileSync(
    join(site, 'page.ts'),
    [
      "import { contentPaths, useCollection, useContent } from 'virtual:weft/content'",
      "export const title: string = useContent('docs', '').frontmatter.title",
      "export const paths: Record<string, string>[] = contentPaths('docs')",
      "export const first: string | undefined = useCollection('docs')[0]?.path",
      "export const wrong: number = useContent('docs', '').html",
      '',
    ].join('\n'),
  )
  const tsconfig = {
    compilerOptions: {
      strict: true,
      noEmit: true,
      target: 'es2022',
      module: 'esnext',
      moduleResolution: 'bundler',
      allowImportingTsExtensions: true,
      skipLibCheck: true,
      types: ['@weft/vite/virtual'],
    },
    files: ['page.ts'],
  }
  writeFileSync(join(site, 'tsconfig.json'), JSON.stringify(tsconfig))
  const config = ts.getParsedCommandLineOfConfigFile(join(site, 'tsconfig.json'), {}, { ...ts.sys, onUnRecoverableConfigFileDiagnostic: () => {} })!
  const program = ts.createProgram(config.fileNames, config.options)
  const errors = ts.getPreEmitDiagnostics(program).map((d) => {
    const line = d.file ? d.file.getLineAndCharacterOfPosition(d.start ?? 0).line + 1 : 0
    return `${d.file ? d.file.fileName.slice(site.length + 1) : ''}:${line} TS${d.code}`
  })
  // Only the line that misuses a type: the types are real, not any.
  expect(errors).toEqual(['page.ts:5 TS2322'])
})
