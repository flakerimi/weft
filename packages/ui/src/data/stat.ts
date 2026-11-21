// The words a StatCard says about its delta.

/** What a StatCard says, for assistive technology; give some or all to translate them. */
export interface StatLabels {
  /** A rise, given the amount with its unit: "up 4%". */
  up: (amount: string) => string
  /** A fall: "down 2%". */
  down: (amount: string) => string
  /** A delta of 0. */
  flat: string
}

export const STAT_LABELS: StatLabels = {
  up: (amount) => `up ${amount}`,
  down: (amount) => `down ${amount}`,
  flat: 'no change',
}
