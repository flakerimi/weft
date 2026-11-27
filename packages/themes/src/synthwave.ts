import type { Theme } from './tokens.ts'

/**
 * Synthwave: neon on a deep violet ground, the genre's night drive, with hot
 * pink for the primary, a cyan focus ring and a sunset amber; in light, the
 * same hues deepened on a pale lilac ground. Space Grotesk for text and
 * JetBrains Mono for code. An original palette. The fonts are named, not
 * shipped: a site that uses this theme serves them itself.
 */
export const synthwave: Theme = {
  name: 'synthwave',
  light: {
    bg: '#F3EEFB', surface: '#F9F6FE', 'surface-raised': '#FFFFFF', border: '#D8CCF0',
    text: '#1D1236', 'text-muted': '#5C4A80', primary: '#A3127F', 'primary-soft': '#F5D6EE',
    danger: '#B3133F', warning: '#F2B33D', success: '#00745F', info: '#2D5FD0', focus: '#7B2FD6',
  },
  dark: {
    bg: '#170E2E', surface: '#1F143D', 'surface-raised': '#281A4D', border: '#3F2C72',
    text: '#F7F2FF', 'text-muted': '#B6A6DA', primary: '#FF5CD6', 'primary-soft': '#3B1B5E',
    danger: '#FF7A97', warning: '#FFC65C', success: '#35EDB8', info: '#5CC6FF', focus: '#3DE3FF',
  },
  font: {
    sans: "'Space Grotesk Variable', system-ui, sans-serif",
    mono: "'JetBrains Mono Variable', ui-monospace, monospace",
    weightBody: 400, weightHeading: 700, weightTitle: 600,
  },
  radius: { control: 6, card: 10, panel: 14 },
  control: { sm: 28, md: 36, lg: 44 },
  shadow: {
    control: '0 1px 2px color-mix(in srgb, CanvasText 25%, transparent)',
    card: 'none',
    popup: '0 8px 28px color-mix(in srgb, CanvasText 20%, transparent)',
    dialog: '0 10px 36px color-mix(in srgb, CanvasText 28%, transparent)',
  },
  border: { width: 1, strong: 2, focus: 2 },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32 },
  motion: { control: 150, layout: 220, dialog: 260, toast: 4000 },
  ease: { standard: [0.2, 0, 0, 1], emphasized: [0.3, 0, 0, 1] },
}
