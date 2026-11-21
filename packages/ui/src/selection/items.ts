/**
 * The items of Weft's selection components (Select, Combobox, Listbox), the
 * shape Nuxt UI's Select and SelectMenu take:
 *
 * - An item is a value (`'Apple'`, `3`, `true`) or an object
 *   (`{ label, value, description, icon, disabled }`, more keys allowed).
 *   `type: 'label'` makes an object a group's heading and
 *   `type: 'separator'` a line; neither can be chosen.
 * - `items` is a list, or a list of groups (an array of arrays), drawn with
 *   separators between.
 * - An object's label is read at `labelKey` (default `label`), its
 *   description at `descriptionKey`, its value at `valueKey`: without one the
 *   object itself is the value (Select reads `value` by default). Keys may be
 *   dotted paths (`user.name`).
 * - Two values are the same when `by` says so (a key both have, or a
 *   function), else when they are equal in depth (objects key by key).
 *
 * Filtering (`filterGroups`) keeps the items whose label (or any of
 * `fields`) contains the search, ignoring case and accents, and sorts each
 * run of items between headings: the label that is the search, then those
 * that start with it, then the rest; a group left with no item goes.
 */
export type SelectValue = string | number | boolean | bigint | null | undefined | Record<string, unknown>

export interface SelectOption {
  label?: string
  description?: string
  /** A Lucide icon before the label. */
  icon?: string
  /** `label`: a heading over its group. `separator`: a line. Default: an item to choose. */
  type?: 'label' | 'separator' | 'item'
  /** What choosing it gives `v-model`, when the component reads values at `value` (Select, or `value-key="value"`). */
  value?: unknown
  /** Shown, never chosen. */
  disabled?: boolean
  /** Chosen. */
  onSelect?: (event: Event) => void
  [key: string]: unknown
}

export type SelectItem = string | number | boolean | SelectOption
export type SelectItems = SelectItem[] | SelectItem[][]
/** How two values are compared: a key both have, or a function. */
export type By = string | ((a: unknown, b: unknown) => boolean)

export const isObject = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v)

/** Items as groups: a flat list is one group. */
export function groupsOf(items: SelectItems | undefined): SelectItem[][] {
  if (!items?.length) return []
  return Array.isArray(items[0]) ? (items as SelectItem[][]) : [items as SelectItem[]]
}

/** A heading or a separator: drawn, never chosen. */
export const isStructural = (item: SelectItem) => isObject(item) && (item.type === 'label' || item.type === 'separator')

/** The value at a dotted path of `obj`. */
export function get(obj: unknown, path: string | undefined): unknown {
  if (!path) return undefined
  let at: unknown = obj
  for (const part of path.split('.')) {
    if (!isObject(at) && !Array.isArray(at)) return undefined
    at = (at as Record<string, unknown>)[part]
  }
  return at
}

/** What choosing `item` gives v-model. */
export const valueOf = (item: SelectItem, valueKey?: string): unknown =>
  isObject(item) ? (valueKey ? get(item, valueKey) : item) : item

/** What `item` says. */
export function labelOf(item: SelectItem, labelKey = 'label'): string {
  if (!isObject(item)) return String(item)
  const label = get(item, labelKey)
  return label == null ? '' : String(label)
}

export function deepEqual(a: unknown, b: unknown): boolean {
  if (Object.is(a, b)) return true
  if (Array.isArray(a) && Array.isArray(b)) return a.length === b.length && a.every((v, i) => deepEqual(v, b[i]))
  if (isObject(a) && isObject(b)) {
    const ka = Object.keys(a)
    return ka.length === Object.keys(b).length && ka.every((k) => deepEqual(a[k], b[k]))
  }
  return false
}

export function sameValue(a: unknown, b: unknown, by?: By): boolean {
  if (typeof by === 'function') return by(a, b)
  if (by && isObject(a) && isObject(b)) return deepEqual(get(a, by), get(b, by))
  return deepEqual(a, b)
}

/** The item among `items` whose value is `value`. */
export function itemOf(items: SelectItems | undefined, value: unknown, opts: { valueKey?: string; by?: By }): SelectItem | undefined {
  for (const group of groupsOf(items)) {
    for (const item of group) {
      if (!isStructural(item) && sameValue(valueOf(item, opts.valueKey), value, opts.by)) return item
    }
  }
  return undefined
}

/** What a value shows: its item's label, else the value itself when it is text or a number. */
export function labelOfValue(items: SelectItems | undefined, value: unknown, opts: { valueKey?: string; labelKey?: string; by?: By }): string {
  if (value === undefined || value === null || value === '') return ''
  const item = itemOf(items, value, opts)
  if (item !== undefined) return labelOf(item, opts.labelKey)
  if (isObject(value)) return labelOf(value as SelectOption, opts.labelKey)
  return String(value)
}

/** Text folded for comparing: no case, no accents. Each character maps to one, so indices line up with the original. */
export function fold(text: string): string {
  let out = ''
  for (const ch of text) {
    const base = ch.normalize('NFD').replace(/\p{M}/gu, '').toLowerCase()
    out += base.length === ch.length ? base : ch
  }
  return out
}

/** How well `text` matches `term`: 0 is it, 1 starts with it, 2 contains it, null no match. */
export function score(text: string, term: string): number | null {
  const t = fold(text)
  const q = fold(term)
  if (!t.includes(q)) return null
  if (t === q) return 0
  return t.startsWith(q) ? 1 : 2
}

function scoreItem(item: SelectItem, term: string, fields: string[]): number | null {
  if (!isObject(item)) return score(String(item), term)
  let best: number | null = null
  for (const field of fields) {
    const v = get(item, field)
    if (v == null) continue
    for (const one of Array.isArray(v) ? v : [v]) {
      const s = score(String(one), term)
      if (s !== null && (best === null || s < best)) best = s
    }
  }
  return best
}

/**
 * Which items of each group match `term`, as places: `{ group, items }`
 * with the group's index and its items' indices, best first within each
 * run between headings (headings kept); a group with no match is left out.
 * Every item matches an empty term.
 */
export function filterPlaces(groups: SelectItem[][], term: string, fields: string[]): { group: number; items: number[] }[] {
  const out: { group: number; items: number[] }[] = []
  groups.forEach((group, g) => {
    const items: number[] = []
    let run: { i: number; s: number }[] = []
    const flush = () => {
      items.push(...run.sort((a, b) => a.s - b.s).map((r) => r.i))
      run = []
    }
    let matched = false
    group.forEach((item, i) => {
      if (isStructural(item)) {
        flush()
        items.push(i)
        return
      }
      const s = term ? scoreItem(item, term, fields) : 0
      if (s === null) return
      matched = true
      run.push({ i, s })
    })
    flush()
    if (matched) out.push({ group: g, items })
  })
  return out
}

/** The groups with only the items matching `term` (see filterPlaces). */
export function filterGroups(groups: SelectItem[][], term: string, fields: string[]): SelectItem[][] {
  if (!term) return groups
  return filterPlaces(groups, term, fields).map((p) => p.items.map((i) => groups[p.group]![i]!))
}

/** A value as a hidden form input submits it. */
export const formValue = (v: unknown): string => (v === null || v === undefined ? '' : typeof v === 'object' ? JSON.stringify(v) : String(v))
