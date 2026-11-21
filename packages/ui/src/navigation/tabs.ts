/** A tab of Tabs. */
export interface TabsItem {
  /** The tab's text. */
  label?: string
  /** Names the tab in `v-model` (and in the address with `hash`); default its index. */
  value?: string | number
  /** A Lucide icon before the label. */
  icon?: string
  /** A count or word after the label, drawn as a Badge. */
  badge?: string | number
  /** The panel's text, when no slot gives it. */
  content?: string
  /** Shown, but cannot be selected; the arrow keys skip it. */
  disabled?: boolean
  /** The name of the Tabs slot that renders this panel. */
  slot?: string
}
