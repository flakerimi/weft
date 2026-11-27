// Rewrites the pinned server HTML of the island fixtures
// (test/fixtures/*/markup.ts) from what renderPage draws now, after a Vue
// update changes the hydration markers a server render writes. It refuses a
// fixture whose markup changed in anything but those comments: a changed
// element or text is a change in what a component draws, for a person to
// look at, never to repin. Each file keeps its comment, its export and the
// lines it was split into. Run from packages/vite:
//   node test/island-markup.ts [fixture ...]   (default: every fixture with a markup.ts)
// then read the diff and run the tests.
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { lucideBody } from '../src/icons.ts'
import { renderPage } from '../src/render.ts'

const fixtures = fileURLToPath(new URL('./fixtures', import.meta.url))
// What stands for a parametrised fixture's argument (an icon's drawing) while the markup is compared.
const HOLE = '\u0000'

/** The HTML as tokens: comments, tags and runs of text. */
const tokens = (html: string) => html.match(/<!--[\s\S]*?-->|<[^>]*>|[^<]+/g) ?? []
const isComment = (t: string) => t.startsWith('<!--')
const withoutComments = (html: string) => tokens(html).filter((t) => !isComment(t)).join('')

/** The first `<weft-island>` of `html`, with what is inside it. */
function firstIsland(html: string, file: string): string {
  const start = html.indexOf('<weft-island')
  if (start < 0) throw new Error(`island-markup: ${file} renders no <weft-island>; check its page.`)
  let depth = 0
  for (const m of html.slice(start).matchAll(/<weft-island\b|<\/weft-island>/g)) {
    depth += m[0] === '</weft-island>' ? -1 : 1
    if (depth === 0) return html.slice(start, start + m.index + m[0].length)
  }
  throw new Error(`island-markup: ${file} renders a <weft-island> that is never closed.`)
}

/** A markup.ts: what comes before its string, its pieces (one a line), the argument's name, and the quotes. */
function parse(source: string, file: string) {
  const at = source.search(/["`]<weft-island/)
  if (at < 0) throw new Error(`island-markup: ${file} holds no "<weft-island…" string; it is not a fixture this script rewrites.`)
  const head = source.slice(0, at)
  const param = /\((\w+): string\)\s*=>\s*$/.exec(head)?.[1]
  const quote = source[at] as '"' | '`'
  const body = source.slice(at).trimEnd()
  const pieces = body.split(/ \+\n\s*/)
  const indent = /\n(\s*)["`]/.exec(body)?.[1] ?? '  '
  const values = pieces.map((p) => new Function(param ?? '_', `return ${p}`)(HOLE) as string)
  return { head, param, quote, pieces: values, indent }
}

/** Write `piece` in the file's quotes, the argument where the hole is. */
function quoted(piece: string, quote: '"' | '`', param?: string): string {
  if (quote === '"') {
    if (piece.includes(HOLE)) throw new Error('island-markup: a double-quoted fixture cannot take an argument.')
    return JSON.stringify(piece).replace(/[\u0080-￿]/g, (c) => `\\u${c.charCodeAt(0).toString(16).padStart(4, '0')}`)
  }
  const text = piece.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\$\{/g, '\\${')
  return '`' + text.split(HOLE).join('${' + param + '}') + '`'
}

/**
 * The new markup cut where the old was: before the element or text that began
 * each old line, with the range openers just ahead of it, as the old lines began.
 */
function cut(old: string[], fresh: string): string[] {
  const all = tokens(fresh)
  const starts: number[] = []
  let seen = 0
  for (const piece of old.slice(0, -1)) {
    seen += tokens(piece).filter((t) => !isComment(t)).length
    let i = -1
    for (let n = 0, j = 0; j < all.length; j++) {
      if (isComment(all[j]!)) continue
      if (n++ === seen) {
        i = j
        break
      }
    }
    if (i < 0) return [fresh]
    while (i > 0 && /^<!--\[/.test(all[i - 1]!)) i--
    starts.push(all.slice(0, i).join('').length)
  }
  const out: string[] = []
  let from = 0
  for (const s of starts) {
    if (s <= from) continue
    out.push(fresh.slice(from, s))
    from = s
  }
  out.push(fresh.slice(from))
  return out
}

const names = process.argv.length > 2 ? process.argv.slice(2) : readdirSync(fixtures).filter((d) => existsSync(join(fixtures, d, 'markup.ts')))
let failed = false
for (const name of names) {
  const file = join(fixtures, name, 'markup.ts')
  const rel = `test/fixtures/${name}/markup.ts`
  try {
    const old = parse(readFileSync(file, 'utf8'), rel)
    // The pieces must read as the module's own export does.
    const mod = (await import(pathToFileURL(file).href)) as Record<string, unknown>
    const exported = Object.values(mod)[0]
    const whole = typeof exported === 'function' ? (exported as (s: string) => string)(HOLE) : (exported as string)
    if (whole !== old.pieces.join('')) throw new Error(`island-markup: ${rel} is not read as its export reads; rewrite it by hand.`)
    const { html } = await renderPage({ root: join(fixtures, name), page: 'pages/index.vue' })
    let fresh = firstIsland(html, rel)
    if (old.param) {
      const drawing = lucideBody(old.param)
      if (!drawing || !fresh.includes(drawing)) throw new Error(`island-markup: ${rel} takes the drawing of the icon ${old.param}, which its render does not hold.`)
      fresh = fresh.split(drawing).join(HOLE)
    }
    if (withoutComments(fresh) !== withoutComments(whole)) {
      throw new Error(`island-markup: ${rel} draws other elements or text than it pins, not only other markers; look at what changed before you repin it.`)
    }
    if (fresh === whole) {
      console.log(`${rel}: unchanged`)
      continue
    }
    const lines = cut(old.pieces, fresh).map((p) => quoted(p, old.quote, old.param))
    const joined = old.pieces.length > 1 ? lines.join(` +\n${old.indent}`) : lines.join('')
    writeFileSync(file, old.head + joined + '\n')
    console.log(`${rel}: rewritten`)
  } catch (e) {
    failed = true
    console.error(e instanceof Error ? e.message : String(e))
  }
}
process.exit(failed ? 1 : 0)
