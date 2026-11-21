/**
 * The items of a NavigationMenu, and of AppShell's navigation, which draws
 * them with one, and how to tell the current page among them.
 */
import { isCurrentPage } from '../basics/address.ts'

/** A place to go, or a group of them. */
export interface NavItem {
  /** What it says. */
  label?: string
  /** A Lucide icon before the label; the only thing shown in a collapsed sidebar and, with the label under it, in a bottom bar. */
  icon?: string
  /** The address. An item with neither `to` nor `children` is a button that calls `onSelect`. */
  to?: string
  /** With `to`: where to open it (`_blank`). */
  target?: string
  /** This is the current page: aria-current="page". Unset, a link compares its address with the browser's after mounting (Link's rule). */
  active?: boolean
  /** Shown, not followed. */
  disabled?: boolean
  /** A count or word after the label (unread messages). */
  badge?: string | number
  /** A sentence under the label, in a dropdown. */
  description?: string
  /** `label`: a heading over the items after it (vertical only). */
  type?: 'label'
  /** Places under this one: a list its disclosure button opens, floating (horizontal, or a collapsed sidebar) or below it (vertical). */
  children?: NavItem[]
  /** Vertical: its children start shown. */
  defaultOpen?: boolean
  /** Chosen: a link followed or a button pressed. */
  onSelect?: (event: Event) => void
}

/** Items as groups: a flat list is one group. */
export function navGroups(items: NavItem[] | NavItem[][]): NavItem[][] {
  if (!items.length) return []
  return Array.isArray(items[0]) ? (items as NavItem[][]) : [items as NavItem[]]
}

/** Whether the item is the current page: its `active`, else (once mounted, in the browser) its address against the browser's, as Link has it. */
export const isCurrent = (item: NavItem, mounted: boolean): boolean =>
  item.active ?? (mounted && isCurrentPage(item.to))

/** Whether the item, or one of its children, is the current page. */
export const holdsCurrent = (item: NavItem, mounted = false): boolean =>
  isCurrent(item, mounted) || !!item.children?.some((child) => holdsCurrent(child, mounted))
