import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { lisa, type LisaTokens } from '../src/lisa.ts'
import { neutral } from '../src/neutral.ts'
import { themeToCss } from '../src/to-css.ts'
import { weft } from '../src/weft.ts'
import { synthwave } from '../src/synthwave.ts'
import { editor } from '../src/editor.ts'
import { sandstorm } from '../src/sandstorm.ts'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const tokens = JSON.parse(readFileSync(join(root, 'src/lisa/tokens.json'), 'utf8')) as LisaTokens
mkdirSync(join(root, 'dist'), { recursive: true })
const themes = [neutral, lisa(tokens), weft, synthwave, editor, sandstorm]
for (const theme of themes) writeFileSync(join(root, `dist/${theme.name}.css`), themeToCss(theme))
console.log(`wrote ${themes.map((t) => `dist/${t.name}.css`).join(', ')}`)
