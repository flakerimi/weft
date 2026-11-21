/**
 * The items of a Tree, and the flat list of rows it draws: the visible
 * nodes in order, each with the level, set size and position APG asks a
 * tree to state when it is not drawn as nested lists.
 */

/** A node of a Tree. */
export interface TreeItem {
  /** What the node says; typing its start moves to it. */
  label: string
  /** Names the node in `v-model` and `v-model:expanded`; default its labels from the top, joined with `/`. */
  value?: string
  /** A Lucide icon before the label. */
  icon?: string
  /** Its children: the node opens and closes. */
  children?: TreeItem[]
  /** Open at first, when `v-model:expanded` and `default-expanded` say nothing. */
  defaultExpanded?: boolean
  /** Reached by the keys, but cannot be chosen, opened or closed. */
  disabled?: boolean
}

export interface TreeRow {
  item: TreeItem
  key: string
  /** From 1 at the top. */
  level: number
  setsize: number
  posinset: number
  /** The row of its parent, -1 at the top. */
  parent: number
  /** Has children. */
  branch: boolean
  open: boolean
}

/** The key of `item` under `parentKey` (empty at the top). */
export const keyOf = (item: TreeItem, parentKey: string) => item.value ?? (parentKey ? `${parentKey}/${item.label}` : item.label)

/** The visible rows of `items`, with the nodes whose keys are in `open` opened. */
export function flatten(items: TreeItem[], open: ReadonlySet<string>): TreeRow[] {
  const rows: TreeRow[] = []
  const walk = (list: TreeItem[], level: number, parent: number, parentKey: string) => {
    list.forEach((item, i) => {
      const key = keyOf(item, parentKey)
      const branch = !!item.children?.length
      const isOpen = branch && open.has(key)
      rows.push({ item, key, level, setsize: list.length, posinset: i + 1, parent, branch, open: isOpen })
      if (isOpen) walk(item.children!, level + 1, rows.length - 1, key)
    })
  }
  walk(items, 1, -1, '')
  return rows
}

/** The keys of the nodes marked `defaultExpanded`, at any depth. */
export function defaultOpen(items: TreeItem[], parentKey = ''): string[] {
  return items.flatMap((item) => {
    const key = keyOf(item, parentKey)
    const inner = item.children ? defaultOpen(item.children, key) : []
    return item.defaultExpanded && item.children?.length ? [key, ...inner] : inner
  })
}
