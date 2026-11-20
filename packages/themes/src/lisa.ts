import type { Curve, Theme } from './tokens.ts'

interface Entry<T> { value: T }
type Group = Record<string, unknown>

/** The parts of Lisa's docs/contract/tokens.json the theme reads. */
export interface LisaTokens {
  color: {
    accent: Group
    ink: Group
    dark: Group
    semantic: Group
  }
  font: { ui: Entry<string>; mono: Entry<string> }
  metric: Group
  motion: Group
  curve: Group
  type: Group
}

function pick<T>(group: Group, key: string, path: string): T {
  const entry = group[key] as Entry<T> | undefined
  if (!entry || entry.value === undefined) throw new Error(`Lisa tokens have no ${path}.${key}`)
  return entry.value
}

/** Build the Lisa theme from Lisa's token contract. */
export function lisa(tokens: LisaTokens, opts: { accent?: string } = {}): Theme {
  const accent = opts.accent ?? 'olive'
  const { color } = tokens
  if (!(`accent-${accent}-500` in color.accent)) throw new Error(`unknown Lisa accent "${accent}"`)
  const c = (group: keyof LisaTokens['color'], key: string) => pick<string>(color[group], key, `color.${group}`)
  const m = (key: string) => pick<number>(tokens.metric, key, 'metric')
  const primary = c('accent', `accent-${accent}-500`)
  const common = {
    primary,
    'primary-fg': c('ink', 'warm-white'),
    'primary-soft': c('accent', `accent-${accent}-300`),
    danger: c('semantic', 'error'),
    warning: c('semantic', 'warning'),
    success: c('semantic', 'success'),
    info: c('semantic', 'info'),
    focus: primary,
  }
  return {
    name: accent === 'olive' ? 'lisa' : `lisa-${accent}`,
    light: {
      bg: c('ink', 'paper'), surface: c('ink', 'surface'), 'surface-raised': c('ink', 'surface'),
      border: c('ink', 'line-200'), text: c('ink', 'ink-900'), 'text-muted': c('ink', 'ink-500'),
      ...common,
    },
    dark: {
      bg: c('dark', 'base'), surface: c('dark', 'elevated'), 'surface-raised': c('dark', 'card'),
      border: c('dark', 'line'), text: c('dark', 'text-primary'), 'text-muted': c('dark', 'text-secondary'),
      ...common,
    },
    font: {
      sans: `${tokens.font.ui.value}, system-ui, sans-serif`,
      mono: `${tokens.font.mono.value}, ui-monospace, monospace`,
      weightBody: pick<number>(tokens.type, 'body-weight', 'type'),
      weightHeading: pick<number>(tokens.type, 'heading-weight', 'type'),
      weightTitle: pick<number>(tokens.type, 'title-weight', 'type'),
    },
    radius: { control: m('radius-control'), card: m('radius-card'), panel: m('radius-panel') },
    space: { 1: m('space-1'), 2: m('space-2'), 3: m('space-3'), 4: m('space-4'), 6: m('space-6'), 8: m('space-8') },
    motion: {
      control: pick<number>(tokens.motion, 'control', 'motion'),
      layout: pick<number>(tokens.motion, 'layout', 'motion'),
      dialog: pick<number>(tokens.motion, 'dialog', 'motion'),
    },
    ease: {
      standard: pick<Curve>(tokens.curve, 'standard', 'curve'),
      emphasized: pick<Curve>(tokens.curve, 'emphasized', 'curve'),
    },
  }
}
