import { computed, getCurrentScope, hasInjectionContext, inject, onMounted, provide, readonly, ref, shallowRef, type InjectionKey, type Ref } from 'vue'
import { sharedInput, type Input, type Surface } from './detect.ts'

export interface WeftContext {
  input: Readonly<Ref<Input>>
  surface: Readonly<Ref<Surface>>
  /** The BCP 47 locale a WeftProvider set for dates, numbers and names inside it; undefined when none did. */
  locale: Readonly<Ref<string | undefined>>
  /**
   * Whether this part of the page is a preview of a page inside another (a
   * gallery's stage or thumbnail): a WeftProvider marked `inset` said so.
   * Page and AppShell then draw no `<main>` landmark, Page's title is an h2
   * and AppShell keeps no collapsed state, so the page around keeps one main
   * and one h1, and a preview leaves nothing behind for the next.
   */
  inset: Readonly<Ref<boolean>>
}

export const weftKey: InjectionKey<WeftContext> = Symbol('weft')

const appSurface = readonly(ref<Surface>('app'))
const noLocale = readonly(ref<string | undefined>(undefined))
const notInset = readonly(ref(false))

/** Whether the nearest WeftProvider marked this part of the page `inset`, a preview of a page; read once, at setup. */
export function useInset(): boolean {
  return inject(weftKey, null)?.inset.value ?? false
}

/**
 * How the nearest WeftProvider says this part of the page is being used.
 *
 * The detected input (no provider forcing one) is `pointer` until the first
 * component of this part of the page that asked has mounted, then follows
 * `(pointer: coarse)` live. The server cannot know the input and draws the
 * pointer form, so a phone hydrates that same markup and switches once
 * mounted. The context is provided on, so a part created later (a dialog's
 * body, a branch shown on click) reads the detected input at once.
 *
 * Outside a component (a router guard, app code in `app.runWithContext`) there
 * is nothing to hydrate, so it returns the live detection. A component's setup
 * always runs in an effect scope; plain `runWithContext` code does not. Code in
 * another effect scope that is not a component (a Pinia setup store) cannot be
 * told apart from a component in Vue 3.6, so it should read `detectInput()`
 * instead of calling `useWeft()`.
 */
export function useWeft(): WeftContext {
  const given = inject(weftKey, null)
  if (given) return given
  // Outside a component's setup there is nothing to hydrate: the detection as it is.
  if (!hasInjectionContext() || !getCurrentScope()) return { input: sharedInput(), surface: appSurface, locale: noLocale, inset: notInset }
  const mounted = shallowRef(false)
  onMounted(() => (mounted.value = true))
  const detected = sharedInput()
  const context: WeftContext = {
    input: computed(() => (mounted.value ? detected.value : 'pointer')),
    surface: appSurface,
    locale: noLocale,
    inset: notInset,
  }
  provide(weftKey, context)
  return context
}
