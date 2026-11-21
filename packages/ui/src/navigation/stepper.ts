/** A step of a Stepper. */
export interface StepperItem {
  /** What the step is called. */
  title?: string
  /** A sentence under the title. */
  description?: string
  /** Names the step in `v-model`; default its index. */
  value?: string | number
  /** A Lucide icon in the indicator, in place of the step's number. */
  icon?: string
  /** The step's content, when no slot gives it. */
  content?: string
  /** Never reached: skipped by the keys, `next()` and `prev()`. */
  disabled?: boolean
  /** The name of the Stepper slot that renders this step's content. */
  slot?: string
}

/** Where a step stands: before the current one, the current one, or after it. */
export type StepState = 'completed' | 'active' | 'inactive'
