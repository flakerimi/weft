// Number arithmetic the number controls share: clamping, and stepping without
// floating-point dust (0.1 + 0.2 is 0.3 here).

/** How many digits `n` has after the decimal point. */
export function decimals(n: number): number {
  if (!Number.isFinite(n)) return 0
  const [mantissa = '', exponent] = String(n).toLowerCase().split('e')
  const fraction = mantissa.split('.')[1]?.length ?? 0
  return Math.max(0, fraction - Number(exponent ?? 0))
}

/** `n` rounded to the precision of the most precise of `like`. */
export function tidy(n: number, ...like: number[]): number {
  const places = Math.min(20, Math.max(0, ...like.map(decimals)))
  return Number(n.toFixed(places))
}

export function clamp(n: number, min = -Infinity, max = Infinity): number {
  return Math.min(max, Math.max(min, n))
}

/** `n` moved to the nearest multiple of `step` counted from `origin`. */
export function snap(n: number, step: number, origin = 0): number {
  if (!(step > 0)) return n
  return tidy(origin + Math.round((n - origin) / step) * step, step, origin)
}
