// What a module takes from the modules it imports, and what an index passes
// on, read from the code Vite's SSR transform makes of it. renderPage uses it
// to follow an island's named imports through an index (such as @weft/ui's)
// to the modules that define those names, and no further, as a bundler's
// tree-shaking does for a package that declares its scripts free of side
// effects.
//
// The SSR transform writes every static import as
//
//     const __vite_ssr_import_3__ = await __vite_ssr_import__("/src/index.ts", {"importedNames":["Button"]});
//
// (no metadata when nothing is named: `import * as`, `export *`, a bare
// import), every export as
//
//     __vite_ssr_exportName__("Button", () => { try { return __vite_ssr_import_3__.default } catch {} });
//
// and `export * from` as `__vite_ssr_exportAll__(__vite_ssr_import_4__);`.
import { readFileSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'

const IMPORT = /const (__vite_ssr_import_\d+__) = await __vite_ssr_import__\(("(?:[^"\\]|\\.)*")(?:, (\{.*?\}))?\);/g
const DYNAMIC = /__vite_ssr_dynamic_import__\((?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)')/g
const EXPORT = /__vite_ssr_exportName__\(("(?:[^"\\]|\\.)*"), \(\) => \{ try \{ return (.*?) \} catch \{\} \}\);/g
const EXPORT_ALL = /__vite_ssr_exportAll__\((__vite_ssr_import_\d+__)\)/g
const BINDING = /^(__vite_ssr_import_\d+__)(?:\.([\w$]+)|\[("(?:[^"\\]|\\.)*")\])?$/

/** Each import's binding (`__vite_ssr_import_3__`) → the URL it imports. */
function bindings(code: string): Map<string, string> {
  const out = new Map<string, string>()
  for (const m of code.matchAll(IMPORT)) out.set(m[1]!, JSON.parse(m[2]!) as string)
  return out
}

/**
 * The URLs a module imports, each with the names it takes from it; `null`
 * when it may use all of them: a namespace import, `export *`, a bare import,
 * a dynamic import, or a binding the code passes around whole.
 */
export function importsOf(code: string): Map<string, Set<string> | null> {
  const out = new Map<string, Set<string> | null>()
  const all = (url: string) => out.set(url, null)
  for (const m of code.matchAll(IMPORT)) {
    const [, id, quoted, meta] = m
    const url = JSON.parse(quoted!) as string
    const names = meta ? ((JSON.parse(meta) as { importedNames?: string[] }).importedNames ?? null) : null
    // The binding used other than as `binding.name` or `binding["name"]`: a namespace, whole.
    const whole = new RegExp(`${id}(?![.[\\w$])`, 'g')
    const uses = [...code.matchAll(whole)].length
    if (!names?.length || uses > 1) {
      all(url)
      continue
    }
    const had = out.get(url)
    if (had === null) continue
    out.set(url, new Set([...(had ?? []), ...names]))
  }
  // As the source quoted it; a URL built at run time (a template literal) is not known here.
  for (const m of code.matchAll(DYNAMIC)) all(m[1] ?? m[2]!)
  return out
}

/** Where a module's exports come from: each name's source, and the URLs it re-exports everything of. */
export interface Exports {
  /**
   * An exported name → the URL and name it passes on (`name: null` for a
   * whole module, `export * as ns`), or `null` when the module defines it.
   */
  names: Map<string, { url: string; name: string | null } | null>
  /** The URLs of its `export * from`. */
  all: string[]
}

export function exportsOf(code: string): Exports {
  const urls = bindings(code)
  const names: Exports['names'] = new Map()
  for (const m of code.matchAll(EXPORT)) {
    const name = JSON.parse(m[1]!) as string
    const b = BINDING.exec(m[2]!.trim())
    const url = b && urls.get(b[1]!)
    names.set(name, url ? { url, name: b[2] ?? (b[3] ? (JSON.parse(b[3]) as string) : null) } : null)
  }
  const all = [...code.matchAll(EXPORT_ALL)].map((m) => urls.get(m[1]!)).filter((u): u is string => !!u)
  return { names, all }
}

/** A glob from package.json `sideEffects` as a regular expression over the path from the package's folder. */
function globRegExp(glob: string): RegExp {
  // As bundlers read it: a pattern without a slash matches the file name in any folder.
  const pattern = glob.replace(/^\.\//, '')
  const full = pattern.includes('/') ? pattern : `**/${pattern}`
  let re = ''
  for (let i = 0; i < full.length; i++) {
    const c = full[i]!
    if (c === '*' && full[i + 1] === '*') {
      i++
      if (full[i + 1] === '/') {
        i++
        re += '(?:.*/)?'
      } else re += '.*'
    } else if (c === '*') re += '[^/]*'
    else if (c === '?') re += '[^/]'
    else re += c.replace(/[.+^${}()|[\]\\]/g, '\\$&')
  }
  return new RegExp(`^${re}$`)
}

const packages = new Map<string, { dir: string; sideEffects: unknown } | null>()

/** The nearest package.json above `dir` that names a package, with its `sideEffects`. */
function packageOf(dir: string): { dir: string; sideEffects: unknown } | null {
  if (packages.has(dir)) return packages.get(dir)!
  let found: { dir: string; sideEffects: unknown } | null = null
  try {
    const pkg = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8')) as { name?: string; sideEffects?: unknown }
    if (pkg.name !== undefined) found = { dir, sideEffects: pkg.sideEffects }
  } catch {
    // No package.json here, or not one that parses: look further up.
  }
  if (!found && dirname(dir) !== dir) found = packageOf(dirname(dir))
  packages.set(dir, found)
  return found
}

/**
 * Whether the package `file` is in declares it free of side effects, through
 * package.json `sideEffects`: `false`, or a list of globs it does not match.
 * Without the field, a module may have side effects.
 */
export function sideEffectFree(file: string): boolean {
  const pkg = packageOf(dirname(file))
  if (!pkg) return false
  if (pkg.sideEffects === false) return true
  if (!Array.isArray(pkg.sideEffects)) return false
  const path = relative(pkg.dir, file).split('\\').join('/')
  return !pkg.sideEffects.some((g) => typeof g === 'string' && globRegExp(g).test(path))
}
