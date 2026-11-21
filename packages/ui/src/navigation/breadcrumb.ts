/** A step of a Breadcrumb's path. */
export interface BreadcrumbItem {
  /** What the step says. */
  label?: string
  /** Its address; without one the step is text. */
  to?: string
  /** With `to`: where to open it (`_blank`). */
  target?: string
  /** A Lucide icon before the label. */
  icon?: string
  /** Shown, not followed. */
  disabled?: boolean
  /** This step is the current page. Unset, the last step is, and a link is compared with the browser's address after mounting (Link's rule). */
  active?: boolean
}
