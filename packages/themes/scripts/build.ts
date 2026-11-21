import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { lisa, type LisaTokens } from '../src/lisa.ts'
import { neutral } from '../src/neutral.ts'
import { themeToCss } from '../src/to-css.ts'
import { weft } from '../src/weft.ts'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const tokens = JSON.parse(readFileSync(join(root, 'src/lisa/tokens.json'), 'utf8')) as LisaTokens
mkdirSync(join(root, 'dist'), { recursive: true })
writeFileSync(join(root, 'dist/neutral.css'), themeToCss(neutral))
writeFileSync(join(root, 'dist/lisa.css'), themeToCss(lisa(tokens)))
writeFileSync(join(root, 'dist/weft.css'), themeToCss(weft))
console.log('wrote dist/neutral.css, dist/lisa.css and dist/weft.css')
