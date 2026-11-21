import type { Theme } from './tokens.ts'

/**
 * weft.base.al's theme, from the approved design (docs/design/site-2026-11-21,
 * its raised surface white since the second pass, site-2026-11-21-v2):
 * an indigo ink on a cool ground, saffron for the thread, Bricolage Grotesque
 * for text and JetBrains Mono for code. The fonts are named, not shipped: a
 * site that uses this theme serves them itself.
 */
export const weft: Theme = {
  name: 'weft',
  light: {
    bg: '#E8ECF4', surface: '#F4F6FA', 'surface-raised': '#FFFFFF', border: '#C9D0E0',
    text: '#141C3A', 'text-muted': '#5A6482', primary: '#2B3D8F', 'primary-soft': '#D6DCF0',
    // Saffron is the thread; as a focus ring on the light ground it is under 3:1, so focus is indigo there.
    danger: '#B3261E', warning: '#E2A11B', success: '#246B43', info: '#2F6DB5', focus: '#2B3D8F',
  },
  // The text is a shade lighter than the design's #E6EAF5, so that the indigo's
  // pressed fill (in the light scheme, under this light ink) keeps 4.5:1.
  dark: {
    bg: '#0F1530', surface: '#161E40', 'surface-raised': '#1C2448', border: '#2A3563',
    text: '#F0F2F9', 'text-muted': '#95A0C2', primary: '#9FB0F2', 'primary-soft': '#1F2A5C',
    danger: '#F2A09A', warning: '#E2A11B', success: '#6CCB94', info: '#86B6EE', focus: '#E2A11B',
  },
  font: {
    sans: "'Bricolage Grotesque Variable', system-ui, sans-serif",
    mono: "'JetBrains Mono Variable', ui-monospace, monospace",
    weightBody: 400, weightHeading: 700, weightTitle: 600,
  },
  radius: { control: 8, card: 12, panel: 16 },
  control: { sm: 28, md: 36, lg: 44 },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32 },
  motion: { control: 150, layout: 220, dialog: 260, toast: 4000 },
  ease: { standard: [0.2, 0, 0, 1], emphasized: [0.3, 0, 0, 1] },
}
