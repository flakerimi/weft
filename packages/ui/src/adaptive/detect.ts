import { effectScope, getCurrentScope, onScopeDispose, readonly, ref, type Ref } from 'vue'

export type Input = 'pointer' | 'touch'
export type Surface = 'landing' | 'app'
export type Width = 'narrow' | 'regular' | 'wide'

type MatchMedia = (query: string) => MediaQueryList

const COARSE = '(pointer: coarse)'

let override: MatchMedia | undefined
let shared: Readonly<Ref<Input>> | undefined

/** The media-query source: the one setMatchMedia gave, else the browser's, else none (the server). */
export function currentMatchMedia(): MatchMedia | undefined {
  if (override) return override
  return typeof window !== 'undefined' && window.matchMedia ? window.matchMedia.bind(window) : undefined
}

/** Replace the media-query source (tests, previews). `undefined` restores the browser's. */
export function setMatchMedia(mq: MatchMedia | undefined): void {
  override = mq
  shared = undefined
}

/**
 * The primary input, from what the browser reports about the pointer — never the device name.
 * `touch` exactly when `(pointer: coarse)` matches. Follows changes live.
 */
export function detectInput(mq: MatchMedia | undefined = currentMatchMedia()): Readonly<Ref<Input>> {
  const input = ref<Input>('pointer')
  if (!mq) return readonly(input)
  const list = mq(COARSE)
  input.value = list.matches ? 'touch' : 'pointer'
  const onChange = (e: { matches: boolean }) => {
    input.value = e.matches ? 'touch' : 'pointer'
  }
  list.addEventListener('change', onChange)
  // Inside a component or effect scope, stop listening when it ends.
  if (getCurrentScope()) onScopeDispose(() => list.removeEventListener('change', onChange))
  return readonly(input)
}

/** One detection shared by every consumer outside an explicit override. */
export function sharedInput(): Readonly<Ref<Input>> {
  // Created outside any scope so it lives for the page, not for the first component that asked.
  shared ??= effectScope(true).run(() => detectInput())!
  return shared
}

/** Breakpoints (in px) where a container's width class moves from narrow to regular and from regular to wide. */
export const NARROW_MAX = 640
export const REGULAR_MAX = 1024

/** Width class of a container: under 640 px narrow, under 1024 px regular, else wide. */
export function widthOf(px: number): Width {
  if (px < NARROW_MAX) return 'narrow'
  if (px < REGULAR_MAX) return 'regular'
  return 'wide'
}
