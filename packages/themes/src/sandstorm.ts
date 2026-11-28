import type { Theme } from './tokens.ts'

/**
 * Sandstorm: warm desert sand, dune ochre and a dusty terracotta, soft and
 * rounded, with larger radii and softer shadows; in dark, the same hues at
 * dusk on a deep umber ground. Quicksand for text and JetBrains Mono for
 * code. An original palette. The fonts are named, not shipped: a site that
 * uses this theme serves them itself.
 */
export const sandstorm: Theme = {
  name: 'sandstorm',
  light: {
    bg: '#F1E7D8', surface: '#F8F1E6', 'surface-raised': '#FFFBF4', border: '#DCC9AC',
    text: '#3A291B', 'text-muted': '#6E5843', primary: '#A2462F', 'primary-soft': '#F2DACD',
    danger: '#A33029', warning: '#D9A441', success: '#4C7442', info: '#456A8C', focus: '#A2462F',
  },
  dark: {
    bg: '#1F1711', surface: '#281E16', 'surface-raised': '#31261C', border: '#4B3A2B',
    text: '#F6EBDC', 'text-muted': '#C3AE95', primary: '#E89B7B', 'primary-soft': '#4A2C21',
    danger: '#F28D80', warning: '#E8BA5C', success: '#A1C792', info: '#97B8D6', focus: '#E8BA5C',
  },
  font: {
    sans: "'Quicksand Variable', system-ui, sans-serif",
    mono: "'JetBrains Mono Variable', ui-monospace, monospace",
    weightBody: 500, weightHeading: 700, weightTitle: 600,
  },
  radius: { control: 14, card: 20, panel: 28 },
  control: { sm: 28, md: 36, lg: 44 },
  shadow: {
    control: '0 1px 3px color-mix(in srgb, CanvasText 18%, transparent)',
    card: 'none',
    popup: '0 10px 30px color-mix(in srgb, CanvasText 14%, transparent)',
    dialog: '0 12px 40px color-mix(in srgb, CanvasText 20%, transparent)',
  },
  border: { width: 1, strong: 2, focus: 2 },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32 },
  motion: { control: 180, layout: 260, dialog: 300, toast: 4000 },
  ease: { standard: [0.2, 0, 0, 1], emphasized: [0.3, 0, 0, 1] },
}
