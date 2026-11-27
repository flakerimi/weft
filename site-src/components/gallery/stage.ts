// The preview stage's view, as the URL query keeps it, so a link reproduces
// it: ?input=touch&surface=landing&theme=lisa&scheme=dark&width=390. A value
// left out is the default: input and scheme follow the device and the page,
// the surface is app, the theme is the site's (the one the header's Theme
// menu picked), and the stage fills its column.

export const INPUTS = ['pointer', 'touch'] as const
export const SURFACES = ['app', 'landing'] as const
/** `site`: the page's own theme; neutral and lisa are forced on the canvas. */
export const THEMES = ['site', 'neutral', 'lisa'] as const
export const SCHEMES = ['light', 'dark'] as const
/** Phone, tablet and desktop widths, in CSS pixels. */
export const WIDTHS = [390, 820, 1280] as const

export interface StageView {
  /** Forced input, or undefined to follow the device. */
  input?: (typeof INPUTS)[number]
  surface: (typeof SURFACES)[number]
  theme: (typeof THEMES)[number]
  /** Forced scheme, or undefined to follow the page. */
  scheme?: (typeof SCHEMES)[number]
  /** The stage's width in pixels, or undefined to fill the column. */
  width?: (typeof WIDTHS)[number]
}

export const DEFAULT_VIEW: StageView = { surface: 'app', theme: 'site' }

const pick = <T extends string | number>(list: readonly T[], value: string | null): T | undefined =>
  list.find((v) => String(v) === value)

/** The view a query string (`?a=b`, or `location.search`) asks for; what it gets wrong is left at the default. */
export function parseView(search: string): StageView {
  const q = new URLSearchParams(search)
  const view: StageView = {
    surface: pick(SURFACES, q.get('surface')) ?? DEFAULT_VIEW.surface,
    theme: pick(THEMES, q.get('theme')) ?? DEFAULT_VIEW.theme,
  }
  const input = pick(INPUTS, q.get('input'))
  const scheme = pick(SCHEMES, q.get('scheme'))
  const width = pick(WIDTHS, q.get('width'))
  if (input) view.input = input
  if (scheme) view.scheme = scheme
  if (width) view.width = width
  return view
}

/** The query string for a view, `?…` with only what differs from the default, in a fixed order; '' for the default view. */
export function formatView(view: StageView): string {
  const q = new URLSearchParams()
  if (view.input) q.set('input', view.input)
  if (view.surface !== DEFAULT_VIEW.surface) q.set('surface', view.surface)
  if (view.theme !== DEFAULT_VIEW.theme) q.set('theme', view.theme)
  if (view.scheme) q.set('scheme', view.scheme)
  if (view.width) q.set('width', String(view.width))
  const s = q.toString()
  return s ? `?${s}` : ''
}
