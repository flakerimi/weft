/**
 * fuzzyMatch(text, query): how well `text` matches a fuzzy `query`, as a
 * command palette searches ("gtst" finds "Git: Stash"), and where.
 *
 * - Each word of the query must appear in the text with its letters in
 *   order, gaps allowed; case and accents are ignored. Words may match in
 *   any order.
 * - The score rewards a match at the start of the text or of a word
 *   (after a space, `-`, `_`, `/`, `.`, `:` or a lower-to-upper case change)
 *   and runs of consecutive letters, and costs a little for each letter
 *   skipped; higher is better. For each word the best placement is found.
 * - `ranges` are the matched characters, as `[start, end)` pairs into
 *   `text`, merged and sorted, for drawing the match (`segments`).
 *
 * Returns null when some word does not match. An empty query matches
 * everything with score 0.
 */
import { fold } from './items.ts'

export interface FuzzyMatch {
  score: number
  ranges: [number, number][]
}

const SEPARATOR = /[\s\-_/.:]/
const START = 10
const WORD = 8
const RUN = 6
const GAP = 0.2

function boundary(text: string, j: number): number {
  if (j === 0) return START
  const prev = text[j - 1]!
  if (SEPARATOR.test(prev)) return WORD
  const ch = text[j]!
  // camelCase: a capital after a small letter starts a word.
  if (prev !== prev.toUpperCase() && ch !== ch.toLowerCase()) return WORD
  return 0
}

/** The best placement of `word` in `text`, as a score and the matched indices. */
function matchWord(text: string, folded: string, word: string): { score: number; at: number[] } | null {
  const n = folded.length
  const m = word.length
  if (m === 0) return { score: 0, at: [] }
  if (m > n) return null
  const bonus = Array.from({ length: n }, (_, j) => boundary(text, j))
  // best[i][j]: the best score of the word's first i+1 letters, letter i at j.
  const best: number[][] = []
  const from: number[][] = []
  for (let i = 0; i < m; i++) {
    const row = new Array<number>(n).fill(-Infinity)
    const back = new Array<number>(n).fill(-1)
    const prev = best[i - 1]
    // The best (score + GAP·k) over k ≤ j - 2, for a jump with a gap.
    let gapBest = -Infinity
    let gapAt = -1
    for (let j = 0; j < n; j++) {
      if (prev && j >= 2 && prev[j - 2]! + GAP * (j - 2) > gapBest) {
        gapBest = prev[j - 2]! + GAP * (j - 2)
        gapAt = j - 2
      }
      if (folded[j] !== word[i]) continue
      const own = 1 + bonus[j]!
      if (!prev) {
        row[j] = own - GAP * j * 0.25
        continue
      }
      const run = j >= 1 ? prev[j - 1]! + own + RUN : -Infinity
      const jump = gapAt >= 0 ? gapBest - GAP * (j - 1) + own : -Infinity
      if (run >= jump && run > -Infinity) {
        row[j] = run
        back[j] = j - 1
      } else if (jump > -Infinity) {
        row[j] = jump
        back[j] = gapAt
      }
    }
    best.push(row)
    from.push(back)
  }
  const last = best[m - 1]!
  let end = -1
  for (let j = 0; j < n; j++) if (last[j]! > -Infinity && (end < 0 || last[j]! > last[end]!)) end = j
  if (end < 0) return null
  const at: number[] = []
  for (let i = m - 1, j = end; i >= 0; j = from[i]![j]!, i--) at.unshift(j)
  return { score: last[end]!, at }
}

export function fuzzyMatch(text: string, query: string): FuzzyMatch | null {
  const words = fold(query).split(/\s+/).filter(Boolean)
  if (!words.length) return { score: 0, ranges: [] }
  const folded = fold(text)
  let total = 0
  const hit = new Set<number>()
  for (const word of words) {
    const m = matchWord(text, folded, word)
    if (!m) return null
    total += m.score
    m.at.forEach((j) => hit.add(j))
  }
  const sorted = [...hit].sort((a, b) => a - b)
  const ranges: [number, number][] = []
  for (const j of sorted) {
    const r = ranges[ranges.length - 1]
    if (r && r[1] === j) r[1] = j + 1
    else ranges.push([j, j + 1])
  }
  // A shorter text that matches as well is the closer match.
  return { score: total - text.length * 0.01, ranges }
}

/** `text` cut into runs, each matched or not, for drawing with <mark>. */
export function segments(text: string, ranges: [number, number][]): { text: string; match: boolean }[] {
  const out: { text: string; match: boolean }[] = []
  let at = 0
  for (const [start, end] of ranges) {
    if (start > at) out.push({ text: text.slice(at, start), match: false })
    out.push({ text: text.slice(start, end), match: true })
    at = end
  }
  if (at < text.length || !out.length) out.push({ text: text.slice(at), match: false })
  return out
}
