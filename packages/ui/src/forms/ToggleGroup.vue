<script setup vapor lang="ts">
// A row (or column) of toggle buttons: one of them pressed or none, or with
// `multiple` any number. A named group (give it an aria-label) of buttons
// that say aria-pressed, one tab stop: the arrow keys move along it
// (mirrored right to left, wrapping unless `loop` is off), Home and End go to
// the ends, disabled items are passed by, and the tab stop rests on the
// pressed item. Pressing the pressed item of a single group releases it.
// With a `name` each pressed value posts with a plain form. `attached`
// joins the buttons into one bar. On touch each button is at least
// --weft-hit-min tall (Button's rule).
import { computed, onScopeDispose, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { useField } from './field.ts'
import { toggleValue, type ToggleGroupItem } from './toggle.ts'
import type { Size, Tone, Variant } from '../types.ts'
import '../styles/base.css'
import '../styles/toggle-group.css'

type Value = string | number

const props = withDefaults(
  defineProps<{
    /** The buttons: `{ label, value, icon, trailingIcon, ariaLabel, disabled }`. */
    items?: ToggleGroupItem[]
    /** The pressed value (null for none), or the pressed values with `multiple`, bound with `v-model`. */
    modelValue?: Value | null | Value[]
    /** What is pressed when `v-model` is not bound. */
    defaultValue?: Value | null | Value[]
    /** Any number can be pressed. */
    multiple?: boolean
    /** `horizontal` (default) or `vertical`: which arrow keys move. */
    orientation?: 'horizontal' | 'vertical'
    /** The arrow keys wrap round at the ends. */
    loop?: boolean
    /** Join the buttons into one bar. */
    attached?: boolean
    /** The form field name: each pressed value posts under it. */
    name?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour of the buttons not pressed. */
    tone?: Tone
    /** The variant of the buttons not pressed. */
    variant?: Variant
    /** The colour of the pressed buttons. */
    activeTone?: Tone
    /** The variant of the pressed buttons. */
    activeVariant?: Variant
    /** Every button is disabled. */
    disabled?: boolean
  }>(),
  {
    items: () => [],
    modelValue: undefined,
    defaultValue: undefined,
    orientation: 'horizontal',
    loop: true,
    tone: 'neutral',
    variant: 'ghost',
    activeTone: 'primary',
    activeVariant: 'soft',
  },
)

const emit = defineEmits<{
  /** The pressed value or values changed. */
  'update:modelValue': [value: Value | null | Value[]]
}>()

defineSlots<{
  /** A button's label, in place of `label`. */
  default?: (props: { item: ToggleGroupItem; index: number; pressed: boolean }) => unknown
}>()

const f = useField(props)
// A group of buttons is not a labelable element: in a FormField it takes the
// field's id and is named by the label's id.
f.field?.namedByLabel()
onScopeDispose(() => f.field?.namedByLabel(false))
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.multiple ? [] : null))
const pressedSet = computed(() => {
  const v = model.value
  return new Set<Value>(v == null ? [] : Array.isArray(v) ? v : [v])
})
const pressed = (item: ToggleGroupItem, index: number) => pressedSet.value.has(toggleValue(item, index))
const size = computed(() => props.size ?? f.size.value)
const off = (item: ToggleGroupItem) => f.disabled.value || !!item.disabled

function toggle(item: ToggleGroupItem, index: number) {
  if (off(item)) return
  const value = toggleValue(item, index)
  const on = pressedSet.value.has(value)
  if (props.multiple) {
    const now = [...pressedSet.value]
    model.value = on ? now.filter((v) => v !== value) : [...now, value]
  } else model.value = on ? null : value
}

const root = useTemplateRef<HTMLElement>('root')
const roving = useRovingFocus(root, {
  get orientation() {
    return props.orientation
  },
  get loop() {
    return props.loop
  },
})
// The first pressed item holds the tab stop, whoever pressed it.
const firstPressed = computed(() => props.items.findIndex((item, i) => pressed(item, i)))
watch([firstPressed, root], ([i]) => i >= 0 && roving.setActive(i), { flush: 'post' })
</script>

<template>
  <div
    ref="root"
    class="weft-toggle-group"
    role="group"
    :id="f.id.value"
    :aria-labelledby="f.field?.labelId"
    :data-orientation="orientation"
    :data-attached="attached ? 'true' : undefined"
  >
    <Button
      v-for="(item, i) in items"
      :key="toggleValue(item, i)"
      class="weft-toggle-group__item"
      data-weft-item
      :label="item.label"
      :icon="item.icon"
      :trailing-icon="item.trailingIcon"
      :aria-label="item.ariaLabel"
      :size="size"
      :tone="pressed(item, i) ? activeTone : (item.tone ?? tone)"
      :variant="pressed(item, i) ? activeVariant : (item.variant ?? variant)"
      :active="pressed(item, i)"
      :aria-pressed="pressed(item, i) ? 'true' : 'false'"
      :disabled="off(item)"
      @click="toggle(item, i)"
    >
      <template v-if="$slots.default" #default><slot :item="item" :index="i" :pressed="pressed(item, i)" /></template>
    </Button>
    <template v-if="f.name.value">
      <input v-for="v in pressedSet" :key="String(v)" type="hidden" :name="f.name.value" :value="v" />
    </template>
  </div>
</template>
