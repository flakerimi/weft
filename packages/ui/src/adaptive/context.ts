import { computed, hasInjectionContext, inject, onMounted, provide, readonly, ref, shallowRef, type InjectionKey, type Ref } from 'vue'
import { sharedInput, type Input, type Surface } from './detect.ts'

export interface WeftContext {
  input: Readonly<Ref<Input>>
  surface: Readonly<Ref<Surface>>
  /** The BCP 47 locale a WeftProvider set for dates, numbers and names inside it; undefined when none did. */
  locale: Readonly<Ref<string | undefined>>
}

export const weftKey: InjectionKey<WeftContext> = Symbol('weft')

const appSurface = readonly(ref<Surface>('app'))
const noLocale = readonly(ref<string | undefined>(undefined))

/**
 * How the nearest WeftProvider says this part of the page is being used.
 *
 * The detected input (no provider forcing one) is `pointer` until the first
 * component of this part of the page that asked has mounted, then follows
 * `(pointer: coarse)` live. The server cannot know the input and draws the
 * pointer form, so a phone hydrates that same markup and switches once
 * mounted. The context is provided on, so a part created later (a dialog's
 * body, a branch shown on click) reads the detected input at once.
 */
export function useWeft(): WeftContext {
  const given = inject(weftKey, null)
  if (given) return given
  // Outside a component's setup there is nothing to hydrate: the detection as it is.
  if (!hasInjectionContext()) return { input: sharedInput(), surface: appSurface, locale: noLocale }
  const mounted = shallowRef(false)
  onMounted(() => (mounted.value = true))
  const detected = sharedInput()
  const context: WeftContext = {
    input: computed(() => (mounted.value ? detected.value : 'pointer')),
    surface: appSurface,
    locale: noLocale,
  }
  provide(weftKey, context)
  return context
}
