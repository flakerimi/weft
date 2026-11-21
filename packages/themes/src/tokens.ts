export const colorKeys = [
  'bg', 'surface', 'surface-raised', 'border', 'text', 'text-muted',
  'primary', 'primary-fg', 'primary-soft', 'danger', 'danger-fg', 'warning', 'warning-fg',
  'success', 'success-fg', 'info', 'focus',
] as const

export type ColorKey = (typeof colorKeys)[number]
export type ColorSet = Record<ColorKey, string>
export type Curve = [number, number, number, number]

export interface Theme {
  name: string
  light: ColorSet
  dark: ColorSet
  font: { sans: string; mono: string; weightBody: number; weightHeading: number; weightTitle: number }
  /** px */
  radius: { control: number; card: number; panel: number }
  /**
   * px: the drawn height of a control at each size. It is the same for pointer
   * and touch; on touch a control's hit area grows to `--weft-hit-min` instead.
   */
  control: { sm: number; md: number; lg: number }
  /** px */
  space: { 1: number; 2: number; 3: number; 4: number; 6: number; 8: number }
  /** ms */
  motion: { control: number; layout: number; dialog: number }
  ease: { standard: Curve; emphasized: Curve }
}
