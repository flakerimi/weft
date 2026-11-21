// Colour arithmetic for checking contrast at theme build time, on #RRGGBB
// colours in encoded sRGB: the space `color-mix(in srgb, …)` and the
// browser's compositing of a translucent fill both work in.

type RGB = [number, number, number]

const HEX = /^#[0-9a-f]{6}$/i

/** Whether `value` is a #RRGGBB colour, the only kind the contrast checks read. */
export const isHex = (value: string): boolean => HEX.test(value)

const rgb = (hex: string): RGB => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)) as RGB
const hex = (c: RGB) => '#' + c.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('').toUpperCase()

/** `a` and `b` mixed in encoded sRGB, `share` (0–1) of `a`: `color-mix(in srgb, a share, b)`. */
export function mix(a: string, b: string, share: number): string {
  const [x, y] = [rgb(a), rgb(b)]
  return hex(x.map((v, i) => v * share + y[i]! * (1 - share)) as RGB)
}

/** WCAG 2 relative luminance. */
function luminance(color: string): number {
  const [r, g, b] = rgb(color).map((v) => v / 255).map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!
}

/** WCAG 2 contrast ratio of two #RRGGBB colours, from 1 to 21. */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi! + 0.05) / (lo! + 0.05)
}
