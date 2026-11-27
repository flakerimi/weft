<script setup vapor lang="ts">
// Several checkboxes for one list of chosen values: a fieldset, named by its
// legend or its FormField's label, whose value is the list. Give the options
// as `items`, or put Checkboxes with a `value` in the default slot. Tab
// visits each box, as native checkboxes do; Space ticks.
import { computed, onScopeDispose } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import Checkbox from './Checkbox.vue'
import { choice, provideCheckboxGroup, type ChoiceItem } from './choice.ts'
import { leavesGroup } from './dom.ts'
import { hideField, useField } from './field.ts'
import type { Size, Tone } from '../types.ts'
import '../styles/base.css'
import '../styles/checkbox-group.css'

const props = withDefaults(
  defineProps<{
    /** The chosen values, bound with `v-model`. */
    modelValue?: (string | number)[]
    /** The values chosen at first when `v-model` is not bound. */
    defaultValue?: (string | number)[]
    /** The options: values, or `{ value, label, description, disabled }`. */
    items?: ChoiceItem[]
    /** The group's name, shown above it; inside a FormField the field's label names it instead. */
    legend?: string
    /** The form field name each box posts under; a FormField's applies when not set. */
    name?: string
    /** `sm`, `md` or `lg`, for every box. */
    size?: Size
    /** The colour of the ticked boxes. */
    tone?: Tone
    /** `vertical` (a column) or `horizontal` (a wrapping row). */
    orientation?: 'vertical' | 'horizontal'
    /** Disables every box. */
    disabled?: boolean
  }>(),
  { modelValue: undefined, defaultValue: () => [], orientation: 'vertical', tone: 'primary' },
)

const emit = defineEmits<{
  /** A box was ticked or unticked: the new list. */
  'update:modelValue': [value: (string | number)[]]
}>()

defineSlots<{
  /** Checkboxes with a `value`, in place of `items`. */
  default?: () => unknown
  /** In place of `legend`. */
  legend?: () => unknown
}>()

const f = useField(props)
// A fieldset is not a labelable element: the label names it with aria-labelledby.
f.field?.namedByLabel()
onScopeDispose(() => f.field?.namedByLabel(false))
const model = useControllable(props, 'modelValue', emit, () => [...props.defaultValue])
const options = computed(() => (props.items ?? []).map(choice))

provideCheckboxGroup({
  has: (value) => model.value.includes(value as string | number),
  toggle(value, on) {
    const v = value as string | number
    const without = model.value.filter((x) => x !== v)
    model.value = on ? [...without, v] : without
    f.input()
  },
  name: () => f.name.value,
  size: () => f.size.value,
  tone: () => props.tone,
  disabled: () => f.disabled.value,
})
hideField()
</script>

<template>
  <fieldset
    class="weft-checkbox-group"
    :data-orientation="orientation"
    :data-size="f.size.value"
    :disabled="f.disabled.value || undefined"
    :id="f.id.value"
    :aria-labelledby="f.field && !legend && !$slots.legend ? f.field.labelId : undefined"
    :aria-describedby="f.aria.value['aria-describedby']"
    :aria-invalid="f.aria.value['aria-invalid']"
    @focusout="leavesGroup($event) && f.blur()"
  >
    <legend v-if="legend || $slots.legend" class="weft-checkbox-group__legend"><slot name="legend">{{ legend }}</slot></legend>
    <div class="weft-checkbox-group__items">
      <slot>
        <Checkbox
          v-for="item in options"
          :key="String(item.value)"
          :value="item.value"
          :label="item.label"
          :description="item.description"
          :disabled="item.disabled"
        />
      </slot>
    </div>
  </fieldset>
</template>
