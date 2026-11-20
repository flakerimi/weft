/**
 * useControllable(props, key, emit, defaultValue)
 *
 * The value behind a component's `v-model`, which works whether or not the
 * parent binds it:
 *
 * - Bound (`props[key]` is not undefined): reading gives the prop; writing
 *   emits `update:<key>` and the value changes when the parent passes it back.
 * - Not bound: the component keeps its own value, starting from
 *   `defaultValue` (a function is called once, for objects and arrays);
 *   writing changes it and still emits `update:<key>`.
 *
 * Writing the value it already has emits nothing. When the parent stops
 * binding, the component keeps the last value it was given.
 *
 *     const props = withDefaults(defineProps<{ open?: boolean }>(), { open: undefined })
 *     const emit = defineEmits<{ 'update:open': [open: boolean] }>()
 *     const open = useControllable(props, 'open', emit, false)
 *
 * Let TypeScript infer `T` and `K` from the arguments.
 *
 * A boolean model must default to `undefined` in `withDefaults`, as above:
 * Vue turns an absent boolean prop into `false`, which would read as bound.
 */
import { computed, ref, watch, type Ref } from 'vue'

export function useControllable<T, K extends string>(
  props: { readonly [P in K]?: T | undefined },
  key: K,
  emit: (event: `update:${K}`, value: T) => void,
  defaultValue: T | (() => T),
): Ref<T> {
  const read = () => props[key] as T | undefined
  const own = ref(read() ?? (typeof defaultValue === 'function' ? (defaultValue as () => T)() : defaultValue)) as Ref<T>
  // Follow the bound value, so a parent that stops binding leaves the last one.
  watch(read, (v) => {
    if (v !== undefined) own.value = v
  }, { flush: 'sync' })

  return computed({
    get: () => read() ?? own.value,
    set(value: T) {
      const before = read() ?? own.value
      if (read() === undefined) own.value = value
      if (!Object.is(before, value)) emit(`update:${key}`, value)
    },
  })
}
