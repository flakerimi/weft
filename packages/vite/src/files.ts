import type { Plugin } from 'vite'

const filesId = 'virtual:weft/files'
const resolvedFilesId = '\0' + filesId

/**
 * Serves `virtual:weft/files` to pages on the server: `useFile(path, body)`
 * records a file the page build writes beside the pages, at `path` from the
 * site root (`/catalog.json`), such as the raw data a page shows. It records
 * into the server render's context; the build checks the path and writes
 * the file, compressed and in the manifest, revalidated rather than cached
 * for good. Several pages may give the same file with the same bytes. An
 * island runs in the browser, where there is no build to write to, so it is
 * refused there.
 */
export function weftFiles(): Plugin {
  return {
    name: 'weft:files',
    resolveId(id) {
      if (id === filesId) return resolvedFilesId
    },
    load(id, options) {
      if (id !== resolvedFilesId) return
      if (!options?.ssr) {
        throw new Error(`${filesId} is used by pages on the server; an island runs in the browser, where there is no build to write a file to.`)
      }
      return [
        "import { useSSRContext } from 'vue'",
        'export function useFile(path, body) {',
        "  if (typeof body !== 'string') throw new Error(`useFile: the body of ${path} is not a string; give the file's text.`)",
        '  const ctx = useSSRContext()',
        "  if (!ctx) throw new Error(`useFile: ${path} is given outside a page's render; call useFile in a page's setup.`)",
        '  ;(ctx.files ??= []).push({ path: String(path), body })',
        '}',
        '',
      ].join('\n')
    },
  }
}
