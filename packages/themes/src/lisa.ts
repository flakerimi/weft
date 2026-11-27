import { mix } from './contrast.ts'
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
  const stop = (name: string, n: number) => c('accent', `accent-${name}-${n}`)
  // The contract pairs each ramp's stops with their use ($solid, $selectionOutline):
  // solid fills and text take 700 on light and 300 on dark, outlines 500 on light
  // and 300 on dark. Danger takes the red ramp the same way: semantic.error reads
  // at 4.2:1 with warm-white and 4.1:1 with ink-900, under AA either way. The text
  // on each fill is chosen at build (themeToCss), from the contract's two inks.
  // Muted text: the contract's ink-500 reads at 4.34:1 on paper, under AA;
  // 5% of ink-700 mixed in (#627188) is the least that reaches 4.5:1 on paper
  // (4.52:1) and on the surfaces (4.74:1).
  const muted = mix(c('ink', 'ink-700'), c('ink', 'ink-500'), 0.05)
  const semantic = {
    warning: c('semantic', 'warning'),
    success: c('semantic', 'success'),
    info: c('semantic', 'info'),
  }
  return {
    name: accent === 'olive' ? 'lisa' : `lisa-${accent}`,
    light: {
      bg: c('ink', 'paper'), surface: c('ink', 'surface'), 'surface-raised': c('ink', 'surface'),
      border: c('ink', 'line-200'), text: c('ink', 'ink-900'), 'text-muted': muted,
      primary: stop(accent, 700), 'primary-soft': stop(accent, 300), focus: stop(accent, 500),
      danger: stop('red', 700), ...semantic,
    },
    dark: {
      bg: c('dark', 'base'), surface: c('dark', 'elevated'), 'surface-raised': c('dark', 'card'),
      border: c('dark', 'line'), text: c('dark', 'text-primary'), 'text-muted': c('dark', 'text-secondary'),
      primary: stop(accent, 300), 'primary-soft': stop(accent, 700), focus: stop(accent, 300),
      danger: stop('red', 300), ...semantic,
    },
    font: {
      sans: `${tokens.font.ui.value}, system-ui, sans-serif`,
      mono: `${tokens.font.mono.value}, ui-monospace, monospace`,
      weightBody: pick<number>(tokens.type, 'body-weight', 'type'),
      weightHeading: pick<number>(tokens.type, 'heading-weight', 'type'),
      weightTitle: pick<number>(tokens.type, 'title-weight', 'type'),
    },
    radius: { control: m('radius-control'), card: m('radius-card'), panel: m('radius-panel') },
    // A toolbar control, a standard control, a touch target.
    control: { sm: m('toolbar-control'), md: m('control-min'), lg: m('touch-target') },
    // Weft's defaults, not the contract's: its shadow-*-y and -blur metrics
    // and shadow-alpha-* are not mapped yet, so Lisa draws as the other themes do.
    shadow: {
      control: '0 1px 2px color-mix(in srgb, CanvasText 25%, transparent)',
      card: 'none',
      popup: '0 8px 24px color-mix(in srgb, CanvasText 16%, transparent)',
      dialog: '0 8px 32px color-mix(in srgb, CanvasText 24%, transparent)',
    },
    border: { width: 1, strong: 2, focus: 2 },
    space: { 1: m('space-1'), 2: m('space-2'), 3: m('space-3'), 4: m('space-4'), 6: m('space-6'), 8: m('space-8') },
    motion: {
      control: pick<number>(tokens.motion, 'control', 'motion'),
      layout: pick<number>(tokens.motion, 'layout', 'motion'),
      dialog: pick<number>(tokens.motion, 'dialog', 'motion'),
      toast: pick<number>(tokens.motion, 'toast-timeout', 'motion'),
    },
    ease: {
      standard: pick<Curve>(tokens.curve, 'standard', 'curve'),
      emphasized: pick<Curve>(tokens.curve, 'emphasized', 'curve'),
    },
  }
}
