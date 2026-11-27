// Every theme @weft/themes builds is one a site can be built and served in:
// the page build and the dev server read its stylesheet as the first CSS of
// every page.
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { readStyles, THEMES } from '../src/document.ts'

const root = fileURLToPath(new URL('./fixtures/css-site', import.meta.url))

describe('the themes', () => {
  it('are the six @weft/themes builds', () => {
    expect(THEMES).toEqual(['neutral', 'lisa', 'weft', 'synthwave', 'editor', 'sandstorm'])
  })

  it.each(THEMES)('%s is read as the first CSS of a page', async (theme) => {
    const { css, files } = await readStyles(root, theme)
    expect(css.startsWith(`/* Weft theme: ${theme}.`)).toBe(true)
    expect(files[0]).toMatch(new RegExp(`[/\\\\]${theme}\\.css$`))
  })
})
