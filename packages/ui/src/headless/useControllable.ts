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
 * A bound write reads back at once, until the parent's answer arrives (Vue
 * passes a parent's new value down at its next flush, after the event that
 * wrote it): two key presses in one task step twice, not once. At the next
 * tick the prop rules again, so a parent that refused the value shows its own.
 *
 *     const props = withDefaults(defineProps<{ open?: boolean }>(), { open: undefined })
 *     const emit = defineEmits<{ 'update:open': [open: boolean] }>()
 *     const open = useControllable(props, 'open', emit, false)
 *
 * The value's type is the prop's, less undefined, and the key alone names the
 * prop: `emit` may be a component's whole `defineEmits` overload set (other
 * events, other models) and the default is checked against the prop, never
 * used to widen it.
 *
 * A boolean model must default to `undefined` in `withDefaults`, as above:
 * Vue turns an absent boolean prop into `false`, which would read as bound.
 */
import { computed, nextTick, ref, shallowRef, watch, type Ref } from 'vue'

/** The value behind prop `K` of `P`: its type, less undefined (which means not bound). */
type Model<P, K extends keyof P> = Exclude<P[K], undefined>

export function useControllable<P extends object, K extends keyof P & string>(
  props: P,
  key: K,
  emit: NoInfer<(event: `update:${K}`, value: Model<P, K>) => void>,
  defaultValue: NoInfer<Model<P, K> | (() => Model<P, K>)>,
): Ref<Model<P, K>> {
  type T = Model<P, K>
  const read = () => props[key] as T | undefined
  const own = ref(read() ?? (typeof defaultValue === 'function' ? (defaultValue as () => T)() : defaultValue)) as Ref<T>
  // Follow the bound value, so a parent that stops binding leaves the last one.
  watch(read, (v) => {
    if (v !== undefined) own.value = v
  }, { flush: 'sync' })

  // A bound write, shown until the parent's answer arrives.
  const pending = shallowRef<{ value: T } | null>(null)
  const current = () => (pending.value ? pending.value.value : (read() ?? own.value))

  return computed({
    get: current,
    set(value: T) {
      const before = current()
      if (Object.is(before, value)) return
      if (read() === undefined) own.value = value
      else {
        const mine = { value }
        pending.value = mine
        void nextTick(() => {
          if (Object.is(pending.value, mine)) pending.value = null
        })
      }
      emit(`update:${key}`, value)
    },
  })
}
