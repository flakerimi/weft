// The landing page's model: the benchmark bars from the newest results file
// in bench/results/, and the catalogue excerpt the agents section shows, as
// plain functions of what the page reads at build time.
import { readdirSync, readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { benchFacts, benchRows, catalogExcerpt, formatCount, newestResults, type BenchResults } from '../landing/model.ts'
import type { Catalog } from '../gallery/model.ts'

const resultsDir = new URL('../../bench/results/', import.meta.url)
const resultFiles = Object.fromEntries(
  readdirSync(resultsDir)
    .filter((f) => f.endsWith('.json'))
    .map((f) => [`../../bench/results/${f}`, JSON.parse(readFileSync(new URL(f, resultsDir), 'utf8')) as BenchResults]),
)
const catalog = JSON.parse(readFileSync(new URL('../../catalog/components.json', import.meta.url), 'utf8')) as Catalog
const datePicker = catalog.components.find((c) => c.name === 'DatePicker')!

describe('the newest benchmark results', () => {
  const run = (date: string) => ({ date }) as BenchResults

  it('is the file with the latest date in its name', () => {
    const files = {
      '../../bench/results/2026-11-20.json': run('2026-11-20'),
      '../../bench/results/2027-01-02.json': run('2027-01-02'),
      '../../bench/results/2026-12-31.json': run('2026-12-31'),
    }
    expect(newestResults(files)).toEqual({ file: 'bench/results/2027-01-02.json', results: run('2027-01-02') })
  })

  it('ignores files not named for a date, and refuses a folder with none', () => {
    expect(newestResults({ '../../bench/results/notes.json': run('x'), '../../bench/results/2026-11-20.json': run('2026-11-20') }).file).toBe(
      'bench/results/2026-11-20.json',
    )
    expect(() => newestResults({})).toThrow(/bench\/results.*no results/)
  })

  it('reads the repository as it is today', () => {
    const newest = readdirSync(resultsDir).filter((f) => /^\d{4}-\d{2}-\d{2}\.json$/.test(f)).sort().at(-1)
    expect(newestResults(resultFiles).file).toBe(`bench/results/${newest}`)
  })
})

describe('the benchmark bars', () => {
  const { results } = newestResults(resultFiles)
  const rows = benchRows(results)

  it('has a bar for each site behind the server it ships with, fastest first, Weft marked as ours', () => {
    const shown = results.targets.filter((t) => (t.kind ?? 'own') === 'own')
    expect(shown.length).toBeGreaterThan(1)
    expect(rows.map((r) => r.id).sort()).toEqual(shown.map((t) => t.id).sort())
    const rps = rows.map((r) => r.rps)
    expect(rps).toEqual([...rps].sort((a, b) => b - a))
    expect(rows.filter((r) => r.ours).map((r) => r.id)).toEqual(['weft'])
  })

  it('takes each number from the results, the median of the load runs', () => {
    for (const row of rows) {
      const t = results.targets.find((x) => x.id === row.id)!
      expect(row.rps).toBe(t.load.rps.median)
      expect(row.label).toBe(t.label)
      expect(row.server).toBe(t.server)
    }
  })

  it('draws each bar as a share of the fastest', () => {
    expect(rows[0]!.share).toBe(1)
    for (const row of rows) expect(row.share).toBeCloseTo(row.rps / rows[0]!.rps, 6)
  })

  it('writes counts with thousands separators', () => {
    expect(formatCount(124503)).toBe('124,503')
    expect(formatCount(8206)).toBe('8,206')
  })

  it('states the facts beside the bars from the same file', () => {
    const weft = results.targets.find((t) => t.id === 'weft')!
    const plain = results.targets.find((t) => t.id === 'astro-plain-preview')!
    const facts = benchFacts(results)
    expect(facts.ttfb).toBe(`${weft.ttfb.median_ms.toFixed(2)} ms`)
    expect(facts.runs).toBe(results.method.load.runs)
    expect(facts.machine).toBe(results.machine.chip)
    expect(facts.date).toBe(results.date)
    expect(facts.weftJS).toBe(weft.bytes.js_br)
    expect(facts.plainJS).toBe(plain.bytes.js_br + plain.bytes.js_inline_raw)
  })

  it('leaves the plain Go servers out of the bars and states their range beside them', () => {
    const target = (id: string, kind: 'own' | 'gzip' | 'packed', rps: number) =>
      ({ ...results.targets[0]!, id, kind, load: { rps: { median: rps, min: rps, max: rps } } })
    const made = {
      ...results,
      targets: [target('weft', 'own', 100), target('a-own', 'own', 40), target('a-static', 'gzip', 10), target('a-packed', 'packed', 130), target('b-packed', 'packed', 110)],
    }
    expect(benchRows(made).map((r) => [r.id, r.share])).toEqual([['weft', 1], ['a-own', 0.4]])
    expect(benchFacts({ ...made, targets: [...made.targets, target('astro-plain-preview', 'own', 20)] }).packed).toEqual({ min: 110, max: 130 })
  })

  it('has no pre-compressed range for results that measured none', () => {
    const old = { ...results, targets: results.targets.filter((t) => t.kind !== 'packed') }
    expect(benchFacts(old).packed).toBeNull()
  })

  it('refuses results without a Weft target', () => {
    expect(() => benchRows({ ...results, targets: results.targets.filter((t) => t.id !== 'weft') })).toThrow(/no weft target/)
  })
})

describe('the catalogue excerpt', () => {
  const excerpt = catalogExcerpt(datePicker)
  const full = JSON.stringify(datePicker, null, 2).split('\n')

  it('shows the name, group, file, purpose and how it adapts to input, line for line as the JSON has them', () => {
    const lines = excerpt.split('\n')
    for (const line of lines) {
      if (line.trim() === '…') continue
      expect(full, line).toContain(line)
    }
    expect(excerpt).toContain(`"name": "DatePicker"`)
    expect(excerpt).toContain(JSON.stringify(datePicker.purpose))
    expect(excerpt).toContain(JSON.stringify(datePicker.adapts.input))
    expect(lines[0]).toBe('{')
    expect(lines.at(-1)).toBe('}')
  })

  it('marks what it leaves out', () => {
    expect(excerpt.split('\n').filter((l) => l.trim() === '…')).toHaveLength(3)
  })
})
