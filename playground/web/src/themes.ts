import lisa from '@weft/themes/lisa.css?url'
import neutral from '@weft/themes/neutral.css?url'

export const themes = { neutral, lisa } as const
export type ThemeName = keyof typeof themes
export type Scheme = 'system' | 'light' | 'dark'

export function applyTheme(name: ThemeName): void {
  const link = document.getElementById('weft-theme') as HTMLLinkElement
  link.href = themes[name]
}

export function applyScheme(scheme: Scheme): void {
  const root = document.documentElement
  if (scheme === 'system') delete root.dataset.weftScheme
  else root.dataset.weftScheme = scheme
}
