import type { Theme } from './tokens.ts'

/**
 * Editor: a warm charcoal code-editor look with candy accents, pink for the
 * primary, lime, amber and violet, in the spirit of the classic dark editor
 * themes; in light, the same accents deepened on a warm paper ground.
 * Monospace everywhere: JetBrains Mono for text and code alike, tighter
 * corners. An original palette. The font is named, not shipped: a site that
 * uses this theme serves it itself.
 */
export const editor: Theme = {
  name: 'editor',
  light: {
    bg: '#F5F0E6', surface: '#FAF7F0', 'surface-raised': '#FFFFFF', border: '#DED4C2',
    text: '#29241E', 'text-muted': '#665C4F', primary: '#B8195A', 'primary-soft': '#F5D5E1',
    danger: '#B23A12', warning: '#E3A41F', success: '#4A7000', info: '#6143C9', focus: '#6143C9',
  },
  dark: {
    bg: '#23201C', surface: '#2B2823', 'surface-raised': '#34302A', border: '#4A443B',
    text: '#F5F0E6', 'text-muted': '#B3A999', primary: '#FF6A9E', 'primary-soft': '#4A2533',
    danger: '#FF8A65', warning: '#F2BE4A', success: '#B5E354', info: '#B9A2FF', focus: '#F2BE4A',
  },
  font: {
    sans: "'JetBrains Mono Variable', ui-monospace, monospace",
    mono: "'JetBrains Mono Variable', ui-monospace, monospace",
    weightBody: 400, weightHeading: 700, weightTitle: 600,
  },
  radius: { control: 4, card: 6, panel: 8 },
  control: { sm: 28, md: 36, lg: 44 },
  shadow: {
    control: '0 1px 2px color-mix(in srgb, CanvasText 25%, transparent)',
    card: 'none',
    popup: '0 8px 24px color-mix(in srgb, CanvasText 18%, transparent)',
    dialog: '0 8px 32px color-mix(in srgb, CanvasText 26%, transparent)',
  },
  border: { width: 1, strong: 2, focus: 2 },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32 },
  motion: { control: 120, layout: 200, dialog: 240, toast: 4000 },
  ease: { standard: [0.2, 0, 0, 1], emphasized: [0.3, 0, 0, 1] },
}
