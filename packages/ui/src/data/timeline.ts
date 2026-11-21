// The events of a Timeline, and the words it says.

/** One event of a Timeline. */
export interface TimelineItem {
  /** When, as it should read: "Mar 4", "2 hours ago". */
  date?: string
  title?: string
  description?: string
  /** A Lucide icon in its dot. */
  icon?: string
  /** What `v-model` names it by; default its place. */
  value?: string | number
}

/** What a Timeline says, for assistive technology; give some or all to translate them. */
export interface TimelineLabels {
  /** Before the title of an item that is done. */
  done: string
}

export const TIMELINE_LABELS: TimelineLabels = { done: 'Done: ' }
