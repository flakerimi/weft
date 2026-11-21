import { inject, readonly, ref, type InjectionKey, type Ref } from 'vue'
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

/** How the nearest WeftProvider says this part of the page is being used. */
export function useWeft(): WeftContext {
  return inject(weftKey, null) ?? { input: sharedInput(), surface: appSurface, locale: noLocale }
}
