import { readonly, ref, type Ref } from 'vue'

export type Input = 'pointer' | 'touch'
export type Surface = 'landing' | 'app'
export type Width = 'narrow' | 'regular' | 'wide'

type MatchMedia = (query: string) => MediaQueryList

const COARSE = '(pointer: coarse)'

let override: MatchMedia | undefined
let shared: Readonly<Ref<Input>> | undefined

function currentMatchMedia(): MatchMedia | undefined {
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
  list.addEventListener('change', (e) => {
    input.value = e.matches ? 'touch' : 'pointer'
  })
  return readonly(input)
}

/** One detection shared by every consumer outside an explicit override. */
export function sharedInput(): Readonly<Ref<Input>> {
  shared ??= detectInput()
  return shared
}

/** Width class of a container: under 640 px narrow, under 1024 px regular, else wide. */
export function widthOf(px: number): Width {
  if (px < 640) return 'narrow'
  if (px < 1024) return 'regular'
  return 'wide'
}
