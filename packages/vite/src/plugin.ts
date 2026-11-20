import { relative, resolve } from 'node:path'
import type { Plugin } from 'vite'

export interface WeftOptions {
  /** The island components' `.vue` files, relative to the Vite root or absolute. */
  islands?: string[]
}

const registryId = 'virtual:weft/islands'
const resolvedRegistryId = '\0' + registryId

/** An SFC's `__name`, the way @vue/compiler-sfc derives it: the file name without its extension. */
export const sfcName = (file: string) => /([^/\\]+)\.\w+$/.exec(file)?.[1] ?? file

/**
 * Weft's Vite plugin. It serves `virtual:weft/islands`, whose default export maps
 * each island's name (its SFC `__name`, the `data-id` Island writes) to a dynamic
 * import, so every island is a chunk of its own and loads only where it is used.
 */
export function weft(opts: WeftOptions = {}): Plugin {
  let root = process.cwd()
  let islands: Map<string, string> | undefined

  function collect() {
    const byName = new Map<string, string>()
    for (const entry of opts.islands ?? []) {
      const file = resolve(root, entry)
      const name = sfcName(file)
      const other = byName.get(name)
      if (other === file) continue
      if (other) {
        throw new Error(
          `Islands ${relative(root, other)} and ${relative(root, file)} are both named ${name}; rename one of them.`,
        )
      }
      byName.set(name, file)
    }
    return byName
  }

  return {
    name: 'weft',
    configResolved(config) {
      root = config.root
    },
    buildStart() {
      islands = collect()
    },
    resolveId(id) {
      if (id === registryId) return resolvedRegistryId
    },
    load(id) {
      if (id !== resolvedRegistryId) return
      islands ??= collect()
      const entries = [...islands].map(
        ([name, file]) => `  ${JSON.stringify(name)}: () => import(${JSON.stringify(file)}).then((m) => m.default),`,
      )
      return `export default {\n${entries.join('\n')}\n}\n`
    },
  }
}
