/** A section of an Accordion. */
export interface AccordionItem {
  /** The header's text. */
  label?: string
  /** Names the section in `v-model`; default its index as a string. */
  value?: string
  /** A Lucide icon before the label. */
  icon?: string
  /** A Lucide icon at the end of the header, in place of the chevron. */
  trailingIcon?: string
  /** The panel's text, when no slot gives it. */
  content?: string
  /** Cannot be opened or closed; its header is skipped by the arrow keys. */
  disabled?: boolean
  /** The name of the Accordion's slot that renders this panel. */
  slot?: string
}
