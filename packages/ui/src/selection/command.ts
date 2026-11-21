/**
 * The commands of a CommandPalette, and how a search finds them.
 *
 * - A command has a `label` and may have an `id` (what recent commands are
 *   kept by; the label otherwise; CommandPalette's `label-key`,
 *   `description-key` and `value-key` read other keys), `icon`, `prefix` and `suffix` (drawn
 *   before and after the label), `description`, `keywords` (searched, not
 *   shown), `kbds` (its shortcut, shown with Kbd), `disabled`, `loading`,
 *   `to` and `target` (a link it follows), `children` (a page of commands it
 *   opens, with `placeholder` for that page's search) and `onSelect(event)`.
 * - Groups have an `id`, a `label` (their heading), `items`; one with
 *   `ignoreFilter` shows its items whatever the search, and `postFilter(term,
 *   items)` has the last word on any group's items.
 *
 * `searchGroups` matches each command fuzzily (fuzzy.ts) against its label,
 * and, worth half as much, its description, suffix and keywords; keeps the
 * matches best first, at most `limit` per group; orders the groups by their
 * best match (or keeps their order with `preserveGroupOrder`), a group that
 * ignores the filter staying after the groups it came after; and returns
 * where each label matched, for marking. An empty search keeps everything,
 * in order, unlimited.
 */
import { fuzzyMatch } from './fuzzy.ts'
import { get } from './items.ts'

/** Where a command's label, description and key are read (CommandPalette's `label-key`, `description-key`, `value-key`). */
export interface CommandKeys {
  labelKey: string
  descriptionKey: string
  valueKey?: string
}
export const DEFAULT_KEYS: CommandKeys = { labelKey: 'label', descriptionKey: 'description' }

const text = (v: unknown) => (v == null ? '' : String(v))
/** A command's label, read at `labelKey`. */
export const labelOfCommand = (item: CommandItem, keys: CommandKeys = DEFAULT_KEYS) => text(get(item, keys.labelKey))
/** A command's description, read at `descriptionKey`. */
export const descriptionOfCommand = (item: CommandItem, keys: CommandKeys = DEFAULT_KEYS) => text(get(item, keys.descriptionKey)) || undefined

export interface CommandItem {
  id?: string
  label?: string
  prefix?: string
  suffix?: string
  description?: string
  icon?: string
  keywords?: string[]
  kbds?: string[]
  disabled?: boolean
  loading?: boolean
  to?: string
  target?: string
  children?: CommandItem[]
  placeholder?: string
  onSelect?: (event: Event) => void
  [key: string]: unknown
}

export interface CommandGroup {
  id: string
  label?: string
  items?: CommandItem[]
  ignoreFilter?: boolean
  postFilter?: (term: string, items: CommandItem[]) => CommandItem[]
}

export interface Found {
  groups: { group: CommandGroup; items: CommandItem[] }[]
  /** Where each command's label matched the search. */
  marks: Map<CommandItem, [number, number][]>
}

/** How well `item` matches `term`, and where its label did; null when it does not. */
function scoreOf(item: CommandItem, term: string, keys: CommandKeys): { score: number; ranges: [number, number][] } | null {
  const label = fuzzyMatch(labelOfCommand(item, keys), term)
  let best = label ? { score: label.score, ranges: label.ranges } : null
  for (const other of [descriptionOfCommand(item, keys), item.suffix, ...(item.keywords ?? [])]) {
    if (!other) continue
    const m = fuzzyMatch(other, term)
    if (m && (!best || m.score / 2 > best.score)) best = { score: m.score / 2, ranges: label?.ranges ?? [] }
  }
  return best
}

export function searchGroups(
  groups: CommandGroup[],
  term: string,
  opts: { limit: number; preserveGroupOrder: boolean; keys?: CommandKeys },
): Found {
  const keys = opts.keys ?? DEFAULT_KEYS
  const marks = new Map<CommandItem, [number, number][]>()
  const t = term.trim()
  type Result = { group: CommandGroup; items: CommandItem[]; best: number; at: number }
  const matched: Result[] = []
  const ignored: Result[] = []
  groups.forEach((group, at) => {
    const all = group.items ?? []
    let items: CommandItem[]
    let best = -Infinity
    if (!t || group.ignoreFilter) items = all
    else {
      const scored: { item: CommandItem; score: number }[] = []
      for (const item of all) {
        const s = scoreOf(item, t, keys)
        if (!s) continue
        scored.push({ item, score: s.score })
        if (s.ranges.length) marks.set(item, s.ranges)
      }
      scored.sort((a, b) => b.score - a.score)
      items = scored.map((s) => s.item)
      best = scored[0]?.score ?? -Infinity
    }
    if (group.postFilter) items = group.postFilter(t, items)
    if (t && !group.ignoreFilter) items = items.slice(0, opts.limit)
    if (!items.length) return
    ;(group.ignoreFilter && t ? ignored : matched).push({ group, items, best, at })
  })
  if (t && !opts.preserveGroupOrder) matched.sort((a, b) => b.best - a.best || a.at - b.at)
  const out = [...matched]
  if (opts.preserveGroupOrder) out.push(...ignored)
  else {
    // A group that ignores the filter goes after the last group shown that came before it.
    for (const g of ignored) {
      let after = -1
      out.forEach((r, i) => r.at < g.at && (after = i))
      out.splice(after + 1, 0, g)
    }
  }
  if (opts.preserveGroupOrder) out.sort((a, b) => a.at - b.at)
  return { groups: out.map(({ group, items }) => ({ group, items })), marks }
}

/** What a recent command is kept by: its key at `valueKey`, else its `id`, else its label. */
export const keyOf = (item: CommandItem, keys: CommandKeys = DEFAULT_KEYS) =>
  text(keys.valueKey ? get(item, keys.valueKey) : undefined) || item.id || labelOfCommand(item, keys)

/** Every command in `groups`, children included, by its key. */
export function commandsByKey(groups: CommandGroup[], keys: CommandKeys = DEFAULT_KEYS): Map<string, CommandItem> {
  const out = new Map<string, CommandItem>()
  const walk = (items: CommandItem[] | undefined) => {
    for (const item of items ?? []) {
      const k = keyOf(item, keys)
      if (!out.has(k)) out.set(k, item)
      walk(item.children)
    }
  }
  groups.forEach((g) => walk(g.items))
  return out
}
