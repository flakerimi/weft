/** What AppShell keeps apart from its template: its words, where it keeps the sidebar's state, and how it fills the bottom bar. */
import type { NavItem } from '../navigation/nav.ts'

/** The words AppShell says, for assistive technology; give some or all to translate them. */
export interface AppShellLabels {
  /** The sidebar toggle's name while the sidebar is expanded. */
  collapse: string
  /** The sidebar toggle's name while the sidebar is collapsed. */
  expand: string
  /** The menu button's name on a narrow pointer screen. */
  menu: string
  /** The bottom bar's last button, and the title of the sheet it opens. */
  more: string
}

export const APP_SHELL_LABELS: AppShellLabels = {
  collapse: 'Collapse sidebar',
  expand: 'Expand sidebar',
  menu: 'Menu',
  more: 'More',
}

/** The stored collapsed state under `key`: undefined when none is stored, or storage cannot be read (the server, a private window that refuses it). */
export function readCollapsed(key: string): boolean | undefined {
  try {
    const value = globalThis.localStorage?.getItem(key)
    return value === 'true' ? true : value === 'false' ? false : undefined
  } catch {
    return undefined
  }
}

/** Store the collapsed state under `key`; nothing happens where storage cannot be written. */
export function writeCollapsed(key: string, collapsed: boolean): void {
  try {
    globalThis.localStorage?.setItem(key, String(collapsed))
  } catch {
    // Storage refused (quota, a private window): the state lasts for this page only.
  }
}

/** The most items a bottom bar holds, its More button included. */
export const BAR_MAX = 5

/**
 * The bottom bar's links and what goes under More. Links without children
 * go in the bar in order; when they are more than five, or any item is not
 * such a link (a group, a button), the bar shows the first four and More
 * holds the rest.
 */
export function splitBar(items: NavItem[]): { shown: NavItem[]; more: NavItem[] } {
  const links = items.filter((item) => item.to && !item.children?.length)
  if (links.length === items.length && items.length <= BAR_MAX) return { shown: items, more: [] }
  const shown = links.slice(0, BAR_MAX - 1)
  return { shown, more: items.filter((item) => !shown.includes(item)) }
}
