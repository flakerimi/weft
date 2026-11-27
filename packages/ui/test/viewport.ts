/** Resize happy-dom's window to `width` px: innerWidth and every `(max-width)` media query follow, as a browser's do. */
export function setViewport(width: number) {
  ;(window as unknown as { happyDOM: { setViewport(v: { width: number }): void } }).happyDOM.setViewport({ width })
}

/** happy-dom's default window width, which every test starts from. */
export const DEFAULT_VIEWPORT = 1024
