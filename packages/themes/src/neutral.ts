import type { Theme } from './tokens.ts'

export const neutral: Theme = {
  name: 'neutral',
  light: {
    bg: '#FAFAFA', surface: '#FFFFFF', 'surface-raised': '#FFFFFF', border: '#E4E4E7',
    text: '#18181B', 'text-muted': '#71717A', primary: '#2563EB', 'primary-fg': '#FFFFFF',
    'primary-soft': '#DBEAFE', danger: '#DC2626', warning: '#D97706', success: '#16A34A',
    info: '#0284C7', focus: '#2563EB',
  },
  dark: {
    bg: '#09090B', surface: '#18181B', 'surface-raised': '#27272A', border: '#27272A',
    text: '#FAFAFA', 'text-muted': '#A1A1AA', primary: '#3B82F6', 'primary-fg': '#FFFFFF',
    'primary-soft': '#172554', danger: '#EF4444', warning: '#F59E0B', success: '#22C55E',
    info: '#38BDF8', focus: '#60A5FA',
  },
  font: {
    sans: 'Inter, system-ui, sans-serif', mono: 'ui-monospace, monospace',
    weightBody: 400, weightHeading: 600, weightTitle: 700,
  },
  radius: { control: 8, card: 12, panel: 16 },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32 },
  motion: { control: 150, layout: 220, dialog: 260 },
  ease: { standard: [0.2, 0, 0, 1], emphasized: [0.3, 0, 0, 1] },
}
