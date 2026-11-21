/**
 * The items of a DropdownMenu or ContextMenu, and what the levels of one
 * menu (the menu and its open submenus) share.
 */
import type { InjectionKey, Ref } from 'vue'
import type { Tone } from '../types.ts'

export interface MenuItem {
  /** What the item says; typing its start moves to it. */
  label?: string
  /**
   * `label`: a heading over its group (first in the group). `separator`: a
   * line. `checkbox`: on or off (`checked`). `radio`: one of its group.
   * Default: an action, or a link when it has `to`.
   */
  type?: 'label' | 'separator' | 'checkbox' | 'radio'
  /** A Lucide icon before the label. */
  icon?: string
  /** `danger` for an item that destroys. */
  tone?: Tone
  /** Shown and reachable, but cannot be chosen. */
  disabled?: boolean
  /** A checkbox or radio item's state. */
  checked?: boolean
  /** An address: the item is a link there. */
  to?: string
  /** With `to`: where to open it (`_blank`). */
  target?: string
  /** A submenu: its items, or groups of them. */
  children?: MenuItem[] | MenuItem[][]
  /** Chosen. The menu closes after, unless this calls `event.preventDefault()`. */
  onSelect?: (event: Event) => void
  /** A checkbox item's new state, or true for a radio item chosen. */
  onUpdateChecked?: (checked: boolean) => void
}

/** Items as groups: a flat list is one group. */
export function groupsOf(items: MenuItem[] | MenuItem[][]): MenuItem[][] {
  if (!items.length) return []
  return Array.isArray(items[0]) ? (items as MenuItem[][]) : [items as MenuItem[]]
}

export const isChoice = (item: MenuItem) => item.type === 'checkbox' || item.type === 'radio'

export function roleOf(item: MenuItem): string {
  if (item.type === 'checkbox') return 'menuitemcheckbox'
  if (item.type === 'radio') return 'menuitemradio'
  return 'menuitem'
}

/** What a submenu reads from the level that opened it. */
export interface MenuLevel {
  /** Close every level; `returnFocus` puts focus back on the menu's trigger at once. */
  closeAll: (returnFocus: boolean) => void
  /** The panels of this level and those above it (towards the root). */
  panels: () => (HTMLElement | null | undefined)[]
  /** The item that opened the submenu. */
  item: () => MenuItem | undefined
  /** Whether the submenu should focus its first item: it was opened from the keyboard or a press. */
  focusFirst: () => boolean
  /** Counts up when the submenu, already open, should focus its first item. */
  refocus: Readonly<Ref<number>>
  /** The pointer reached the submenu: keep it open. */
  keep: () => void
}

export const menuLevel: InjectionKey<MenuLevel> = Symbol('weft-menu-level')
