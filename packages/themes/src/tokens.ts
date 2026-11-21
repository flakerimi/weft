export const colorKeys = [
  'bg', 'surface', 'surface-raised', 'border', 'text', 'text-muted',
  'primary', 'primary-soft', 'danger', 'warning', 'success', 'info', 'focus',
] as const

/**
 * The tones a component can be drawn in, besides neutral (the text colour).
 * For each, `themeToCss` works out at build two colours a theme does not
 * give: `--weft-color-<tone>-fg`, the text on a solid fill (the better of the
 * theme's two inks, light.text and dark.text), and `--weft-color-<tone>-text`,
 * the tone as text on the page and on its tints. Both reach WCAG AA (4.5:1),
 * or the build refuses.
 */
export const tones = ['primary', 'danger', 'warning', 'success'] as const
export type Tone = (typeof tones)[number]

/**
 * The tints, in percent of the tone over the ground behind, that tinted
 * variants draw under their text: a soft fill, the soft fill on hover, and
 * the hover of outline and ghost. @weft/ui's CSS uses exactly these.
 */
export const tints = { soft: 14, softHover: 22, quiet: 10 } as const

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
   * px: the drawn height of a control at each size, for pointer and touch
   * alike, as `--weft-control-sm/md/lg`; these are the only control heights a
   * theme sets. On touch a control's hit area is at least `--weft-hit-min`:
   * an md Button or Input grows to it (`max(md, hit-min)`); a small or square
   * control keeps its size and extends its hit area with a pseudo-element.
   */
  control: { sm: number; md: number; lg: number }
  /** px */
  space: { 1: number; 2: number; 3: number; 4: number; 6: number; 8: number }
  /** ms */
  motion: { control: number; layout: number; dialog: number }
  ease: { standard: Curve; emphasized: Curve }
}
