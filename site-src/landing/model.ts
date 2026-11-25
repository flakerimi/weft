// The landing page's model, as plain functions of what the page reads at
// build time: the benchmark bars and the facts beside them from the newest
// results file in bench/results/ (never typed by hand), and the excerpt of a
// catalogue entry the agents section shows.
import type { CatalogEntry } from '../gallery/model.ts'

/** The parts of a bench/results/<date>.json the landing page reads (bench/tools/measure.mjs writes it). */
export interface BenchResults {
  date: string
  machine: { chip: string }
  method: { load: { runs: number; connections: number; duration: string } }
  targets: {
    id: string
    label: string
    server: string
    /**
     * What serves the site: the server it ships with (`own`), or one of the
     * benchmark's plain Go servers, gzipping on the fly (`gzip`) or sending
     * brotli stored at build from memory (`packed`). Results from before
     * 2026-11-25 have no kind, and every target in them counts as `own`.
     */
    kind?: 'own' | 'gzip' | 'packed'
    load: { rps: { median: number; min: number; max: number } }
    ttfb: { median_ms: number }
    bytes: { js_br: number; js_inline_raw: number }
  }[]
}

const DATED = /(?:^|\/)(\d{4}-\d{2}-\d{2})\.json$/

/**
 * The newest results among `files` (as import.meta.glob gives them, by a
 * path relative to the page): the one with the latest date in its name. Its
 * path is returned from the repository root.
 */
export function newestResults(files: Record<string, BenchResults>): { file: string; results: BenchResults } {
  const dated = Object.keys(files)
    .filter((f) => DATED.test(f))
    .sort((a, b) => (DATED.exec(a)![1]! < DATED.exec(b)![1]! ? -1 : 1))
  const last = dated.at(-1)
  if (!last) throw new Error('landing: bench/results/ has no results named <date>.json; run bench/run.sh, or restore the folder.')
  return { file: last.replace(/^(\.\.\/)+/, ''), results: files[last]! }
}

/** One bar: a target's requests per second, as a share of the fastest. */
export interface BenchRow {
  id: string
  label: string
  server: string
  rps: number
  /** rps / the fastest's, 0 to 1. */
  share: number
  /** Weft's own bar. */
  ours: boolean
}

/**
 * A bar for each site behind the server it ships with, fastest first: the
 * median requests per second of its load runs. The benchmark's plain Go
 * servers are not bars; benchFacts gives the pre-compressed one's range.
 */
export function benchRows(results: BenchResults): BenchRow[] {
  if (!results.targets.some((t) => t.id === 'weft')) throw new Error(`landing: bench/results/${results.date}.json has no weft target.`)
  const shown = results.targets.filter((t) => (t.kind ?? 'own') === 'own')
  const top = Math.max(...shown.map((t) => t.load.rps.median))
  return shown
    .map((t) => ({ id: t.id, label: t.label, server: t.server, rps: t.load.rps.median, share: t.load.rps.median / top, ours: t.id === 'weft' }))
    .sort((a, b) => b.rps - a.rps)
}

/** A count as the page writes it, with thousands separators: 12,345. */
export const formatCount = (n: number) => Math.round(n).toLocaleString('en-US')

/** What the page says beside the bars, from the same file. */
export function benchFacts(results: BenchResults) {
  const target = (id: string) => {
    const t = results.targets.find((x) => x.id === id)
    if (!t) throw new Error(`landing: bench/results/${results.date}.json has no ${id} target.`)
    return t
  }
  const weft = target('weft')
  const plain = target('astro-plain-preview')
  const packed = results.targets.filter((t) => t.kind === 'packed').map((t) => t.load.rps.median)
  return {
    date: results.date,
    machine: results.machine.chip,
    runs: results.method.load.runs,
    /** Weft's median time to first byte. */
    ttfb: `${weft.ttfb.median_ms.toFixed(2)} ms`,
    /** The JavaScript Weft's page loads, brotli bytes. */
    weftJS: weft.bytes.js_br,
    /** The JavaScript Astro's page with no framework island loads: files (brotli) and inline (raw). */
    plainJS: plain.bytes.js_br + plain.bytes.js_inline_raw,
    /**
     * The slowest and fastest of the other sites' static builds behind a Go
     * server that sends brotli stored at build from memory, as Weft's does,
     * in requests per second; null when the run measured none.
     */
    packed: packed.length ? { min: Math.min(...packed), max: Math.max(...packed) } : null,
  }
}

/** Bytes as the page writes them: 237 B, 20.5 KB (1024 bytes). */
export const formatBytes = (n: number) => (n < 1024 ? `${n} B` : `${(n / 1024).toFixed(1)} KB`)

/**
 * An excerpt of a catalogue entry as /components/<slug>.json has it: its
 * name, group, file and purpose, and how it adapts to input, each line as
 * that file writes it, with … where lines are left out.
 */
export function catalogExcerpt(entry: CatalogEntry): string {
  const line = (indent: number, key: string, value: unknown, more: boolean) => `${' '.repeat(indent)}${JSON.stringify(key)}: ${JSON.stringify(value)}${more ? ',' : ''}`
  return [
    '{',
    line(2, 'name', entry.name, true),
    line(2, 'group', entry.group, true),
    line(2, 'file', entry.file, true),
    line(2, 'purpose', entry.purpose, true),
    '  …',
    '  "adapts": {',
    line(4, 'input', entry.adapts.input, true),
    '    …',
    '  },',
    '  …',
    '}',
  ].join('\n')
}
