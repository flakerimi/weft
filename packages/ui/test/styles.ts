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

/** Add the neutral theme and weft.css to the document; returns a function that removes them. */
export function loadStyles(): () => void {
  const style = document.createElement('style')
  style.textContent = [themeToCss(neutral), read('base'), read('layout'), read('basics')].join('\n')
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
