<script setup vapor lang="ts">
// One choice among a few, all in view: a radiogroup of native radio inputs,
// as the WAI-ARIA radio group pattern asks. Tab enters the group once, on
// the chosen radio (or the first); the arrow keys move to the next or
// previous radio and choose it, wrapping at the ends and skipping disabled
// ones; Right and Left swap in a right-to-left document. Space chooses the
// focused radio. Each radio's input lies over its drawn circle at least
// --weft-hit-min square, so on touch it takes a 44 px press.
import { computed, onMounted, onScopeDispose, useTemplateRef, watch } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import { choice, type ChoiceItem } from './choice.ts'
import { leavesGroup } from './dom.ts'
import { hideField, useField } from './field.ts'
import type { Size, Tone } from '../types.ts'

const props = withDefaults(
  defineProps<{
    /** The chosen value, bound with `v-model`; undefined or null when none is. */
    modelValue?: string | number | null
    /** The value chosen at first when `v-model` is not bound. */
    defaultValue?: string | number | null
    /** The options: values, or `{ value, label, description, disabled }`. */
    items: ChoiceItem[]
    /** The group's name, shown above it; inside a FormField the field's label names it instead. */
    legend?: string
    /** The form field name the radios share; a FormField's applies, else one is generated. */
    name?: string
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** The colour of the chosen radio. */
    tone?: Tone
    /** `vertical` (a column) or `horizontal` (a wrapping row). The arrow keys work the same either way. */
    orientation?: 'vertical' | 'horizontal'
    /** Disables every radio. */
    disabled?: boolean
    /** A choice must be made: the native `required` on the radios. */
    required?: boolean
  }>(),
  { modelValue: undefined, defaultValue: null, orientation: 'vertical', tone: 'primary' },
)

const emit = defineEmits<{
  /** A radio was chosen, by press, Space or an arrow key. */
  'update:modelValue': [value: string | number | null]
}>()

defineSlots<{
  /** In place of `legend`. */
  legend?: () => unknown
}>()

const f = useField(props)
hideField()
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue)
const options = computed(() => props.items.map(choice))
const base = useId('radio')
const name = computed(() => f.name.value ?? base)

const root = useTemplateRef<HTMLElement>('root')
const roving = useRovingFocus(root, { orientation: 'horizontal', items: '.weft-radio__input' })

function choose(value: string | number) {
  model.value = value
  f.input()
  // A bound value the parent did not change stays chosen.
  root.value?.querySelectorAll<HTMLInputElement>('.weft-radio__input').forEach((el, i) => {
    el.checked = options.value[i]?.value === model.value
  })
}
const indexOf = (value: unknown) => options.value.findIndex((o) => o.value === value)
// The chosen radio holds the tab stop. Watching the element too, after
// useRovingFocus does, this runs on mount once the group has its items.
watch([model, root], ([value]) => {
  if (indexOf(value) >= 0) roving.setActive(indexOf(value))
}, { flush: 'post' })

// An arrow key moves focus (useRovingFocus moves Left and Right; Up and Down
// are handled here), and the radio it lands on is chosen. Tab or a click
// moving focus chooses nothing by itself.
let arrowed = false
const ARROWS = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight']
function onKeydownCapture(event: KeyboardEvent) {
  if (!ARROWS.includes(event.key)) return
  arrowed = true
  queueMicrotask(() => (arrowed = false))
}
function onKeydown(event: KeyboardEvent) {
  if (event.defaultPrevented || (event.key !== 'ArrowUp' && event.key !== 'ArrowDown')) return
  if (event.altKey || event.ctrlKey || event.metaKey) return
  const enabled = options.value.map((o) => !o.disabled && !f.disabled.value)
  const count = enabled.length
  const at = roving.active.value
  const step = event.key === 'ArrowDown' ? 1 : -1
  for (let n = 1; n <= count; n++) {
    const i = (((at + step * n) % count) + count) % count
    if (enabled[i]) {
      event.preventDefault()
      roving.focus(i)
      return
    }
  }
}
function onFocusin(event: FocusEvent) {
  if (!arrowed) return
  const i = Number((event.target as HTMLElement).dataset.index)
  const o = options.value[i]
  if (o && !o.disabled) choose(o.value)
}
let listening: HTMLElement | null = null
onMounted(() => {
  listening = root.value
  listening?.addEventListener('keydown', onKeydownCapture, true)
})
onScopeDispose(() => listening?.removeEventListener('keydown', onKeydownCapture, true))
</script>

<template>
  <fieldset
    ref="root"
    class="weft-radio-group"
    role="radiogroup"
    :data-orientation="orientation"
    :data-size="f.size.value"
    :data-tone="tone"
    :disabled="f.disabled.value || undefined"
    :aria-labelledby="f.field && !legend && !$slots.legend ? f.field.labelId : undefined"
    :aria-describedby="f.aria.value['aria-describedby']"
    :aria-invalid="f.aria.value['aria-invalid']"
    :aria-required="f.aria.value['aria-required'] ?? (required ? 'true' : undefined)"
    @keydown="onKeydown"
    @focusin="onFocusin"
    @focusout="leavesGroup($event) && f.blur()"
  >
    <legend v-if="legend || $slots.legend" class="weft-radio-group__legend"><slot name="legend">{{ legend }}</slot></legend>
    <div class="weft-radio-group__items">
      <div
        v-for="(item, i) in options"
        :key="String(item.value)"
        class="weft-radio"
        :data-state="model === item.value ? 'checked' : 'unchecked'"
        :data-disabled="item.disabled || f.disabled.value ? 'true' : undefined"
      >
        <span class="weft-radio__control">
          <input
            :id="`${base}-${i}`"
            class="weft-radio__input"
            type="radio"
            :name="name"
            :value="item.value"
            :checked="model === item.value"
            :disabled="item.disabled || undefined"
            :required="required || undefined"
            :data-index="i"
            :aria-describedby="item.description ? `${base}-${i}-description` : undefined"
            @change="choose(item.value)"
          />
          <span class="weft-radio__circle" aria-hidden="true"></span>
        </span>
        <span class="weft-radio__text">
          <label class="weft-radio__label" :for="`${base}-${i}`">{{ item.label }}</label>
          <span v-if="item.description" :id="`${base}-${i}-description`" class="weft-radio__description">{{ item.description }}</span>
        </span>
      </div>
    </div>
  </fieldset>
</template>
