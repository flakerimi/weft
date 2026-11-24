// The icon tags: every page's <head>, the not-found page's included, names
// the icon files the site's public/ folder holds, with no setting: the .ico
// first, then the SVG (a browser takes the last icon it understands), then
// the apple-touch icon. A file that is not there writes no tag.
import { cpSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { afterAll, describe, expect, it } from 'vitest'
import { buildSite, type Manifest } from '../src/build.ts'
import { iconTags } from '../src/document.ts'

const ICO = '<link rel="icon" href="/favicon.ico" sizes="32x32">'
const SVG = '<link rel="icon" href="/favicon.svg" type="image/svg+xml">'
const APPLE = '<link rel="apple-touch-icon" href="/apple-touch-icon.png">'
const FILES = [
  ['favicon.ico', ICO],
  ['favicon.svg', SVG],
  ['apple-touch-icon.png', APPLE],
] as const

const temps: string[] = []
afterAll(() => {
  for (const dir of temps) rmSync(dir, { recursive: true, force: true })
})
const tempDir = () => {
  const dir = mkdtempSync(join(tmpdir(), 'weft-favicon-'))
  temps.push(dir)
  return dir
}

/** A copy of the fixture whose public/ folder holds `names`, resolving packages through this package's node_modules. */
function siteWith(names: readonly string[]): string {
  const root = join(tempDir(), 'site')
  cpSync(fileURLToPath(new URL('./fixtures/favicon-site', import.meta.url)), root, { recursive: true })
  symlinkSync(fileURLToPath(new URL('../node_modules', import.meta.url)), join(root, 'node_modules'))
  if (names.length) mkdirSync(join(root, 'public'))
  for (const name of names) writeFileSync(join(root, 'public', name), `bytes of ${name}`)
  return root
}

/** Every icon tag in the page's <head>, in order. */
const iconsIn = (html: string) => [.../<head>([\s\S]*?)<\/head>/.exec(html)![1]!.matchAll(/<link rel="(?:icon|apple-touch-icon|shortcut icon)"[^>]*>/g)].map((m) => m[0])

/** Each combination of the three files, as the files and the tags they give. */
const combinations = [0, 1, 2, 3, 4, 5, 6, 7].map((bits) => {
  const chosen = FILES.filter((_, i) => bits & (1 << i))
  const names = chosen.map(([name]) => name)
  return { names, label: names.join(', ') || 'nothing', tags: chosen.map(([, tag]) => tag) }
})

describe('the icon tags a build writes', { timeout: 60_000 }, () => {
  it.each(combinations)('with public/ holding $label: its tags, in order, on a page and on the 404 page', async ({ names, tags }) => {
    const root = siteWith(names)
    const out = join(tempDir(), 'dist')
    const manifest: Manifest = await buildSite({ root, outDir: out })
    for (const page of ['pages/index/index.html', 'pages/404/index.html']) {
      expect(iconsIn(readFileSync(join(out, page), 'utf8')), page).toEqual(tags)
    }
    for (const name of names) {
      const entry = manifest.files['/' + name]!
      expect(entry.type).toBe({ 'favicon.ico': 'image/x-icon', 'favicon.svg': 'image/svg+xml', 'apple-touch-icon.png': 'image/png' }[name])
      // Named without a hash: revalidated, not cached for a year.
      expect(entry.immutable).toBe(false)
    }
  })
})

describe('iconTags', () => {
  it('writes none for a site with no public/ folder', () => {
    expect(iconTags(siteWith([]))).toEqual([])
  })

  it('takes only a file by its exact name: not a folder, nor another case', () => {
    const root = siteWith([])
    mkdirSync(join(root, 'public/favicon.svg'), { recursive: true })
    writeFileSync(join(root, 'public/FAVICON.ICO'), 'bytes')
    writeFileSync(join(root, 'public/apple-touch-icon.png.txt'), 'bytes')
    expect(iconTags(root)).toEqual([])
  })

  it('takes a symlink to a file inside public/ as the file', () => {
    const root = siteWith(['real.svg'])
    symlinkSync(join(root, 'public/real.svg'), join(root, 'public/favicon.svg'))
    expect(iconTags(root)).toEqual([SVG])
  })
})
