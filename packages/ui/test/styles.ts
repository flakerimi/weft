// Weft's real stylesheets, loaded into the test document so getComputedStyle
// answers as a browser with the neutral theme would (happy-dom resolves var()
// and @media, but draws no layout and computes no pseudo-elements).
// (Read from disk: Vitest hands CSS imports, even `?raw`, to tests empty.)
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { neutral, themeToCss } from '@weft/themes'

const styles = join(dirname(fileURLToPath(import.meta.url)), '../src/styles')
const read = (name: string) => readFileSync(join(styles, `${name}.css`), 'utf8')

/** The stylesheets weft.css imports, in its order, in place of the group sheets that import them. */
const imports = (name: string): string[] =>
  [...read(name).matchAll(/@import '\.\/([a-z-]+)\.css';/g)].flatMap((m) => {
    const inner = imports(m[1]!)
    return inner.length ? inner : [m[1]!]
  })
const sheets = imports('weft')

/** Add a theme (neutral by default) and weft.css to the document; returns a function that removes them. */
export function loadStyles(theme = neutral): () => void {
  const style = document.createElement('style')
  style.textContent = [themeToCss(theme), ...sheets.map(read)].join('\n')
  document.head.appendChild(style)
  return () => style.remove()
}

/** The rule in the loaded stylesheets whose selector is exactly `selector`. */
export function ruleFor(selector: string): CSSStyleRule | undefined {
  for (const sheet of Array.from(document.styleSheets)) {
    for (const rule of Array.from(sheet.cssRules)) {
      if (rule instanceof CSSStyleRule && rule.selectorText.split(/\s*,\s*/).includes(selector)) return rule
    }
  }
  return undefined
}
