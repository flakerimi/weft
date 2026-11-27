<script setup vapor lang="ts">
// The steps of a task done in order (a checkout, a setup): an ordered list
// of step buttons, each with a numbered indicator (a check once done), its
// title and description, and below them the current step's content. The
// current step says aria-current="step"; the ones before it are completed.
// The step buttons are one tab stop, moved with the arrow keys (mirrored
// right to left) and chosen with Enter, Space or a press. `linear` (the
// default) lets a press reach only the steps done and the next one;
// disabled steps are never reached. `next()` and `prev()` move from code (a
// form's Continue button). A horizontal stepper stacks vertically when its
// own box is narrow, in CSS from `data-width` (so before hydration too); on
// touch every step button is at least --weft-hit-min tall.
import { computed, useTemplateRef, watch } from 'vue'
import Icon from '../basics/Icon.vue'
import { useWidth } from '../adaptive/useWidth.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import type { Size, Tone } from '../types.ts'
import type { StepperItem, StepState } from './stepper.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/stepper.css'

const props = withDefaults(
  defineProps<{
    /** The steps: `{ title, description, value, icon, content, disabled, slot }`. */
    items: StepperItem[]
    /** The current step's value, bound with `v-model`: its `value`, else its index. */
    modelValue?: string | number
    /** The step at first when `v-model` is not bound; default the first. */
    defaultValue?: string | number
    /** Steps go in order: a press reaches only the steps done and the next one. */
    linear?: boolean
    /** `horizontal` (a row, stacked when narrow) or `vertical` (a column). */
    orientation?: 'horizontal' | 'vertical'
    /** The colour of the current and completed steps. */
    tone?: Tone
    /** `sm`, `md` or `lg`. */
    size?: Size
    /** No step can be chosen. */
    disabled?: boolean
  }>(),
  { modelValue: undefined, defaultValue: undefined, linear: true, orientation: 'horizontal', tone: 'primary', size: 'md' },
)

const emit = defineEmits<{
  /** The current step changed. */
  'update:modelValue': [value: string | number]
  /** `next()` moved on, to this step. */
  next: [value: string | number]
  /** `prev()` moved back, to this step. */
  prev: [value: string | number]
}>()

defineSlots<{
  /** A step's indicator, in place of its number, icon or check. */
  indicator?: StepSlot
  /** A step's title, in place of `title`. */
  title?: StepSlot
  /** A step's description, in place of `description`. */
  description?: StepSlot
  /** The current step's content, in place of `content`. */
  content?: StepSlot
  /** The content of the step whose `slot` names it. */
  [name: string]: StepSlot | undefined
}>()

type StepSlot = (props: { item: StepperItem; index: number; state: StepState }) => unknown

const valueOf = (item: StepperItem, index: number) => item.value ?? index
const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.items[0] ? valueOf(props.items[0], 0) : 0))
const current = computed(() => Math.max(0, props.items.findIndex((s, i) => valueOf(s, i) === model.value)))
const currentItem = computed(() => props.items[current.value])

const root = useTemplateRef<HTMLElement>('root')
const list = useTemplateRef<HTMLElement>('list')
// Only a horizontal stepper depends on its width. It says its width class in
// `data-width`, which the page's pre-paint script keeps current before
// hydration, and CSS stacks it on `[data-width="narrow"]`, so a static page
// adapts too; `orientation` (for the arrow keys) follows once mounted.
const adapts = props.orientation === 'horizontal'
const width = useWidth(root, { prepaint: adapts })
const widthId = width.id
const orientation = computed(() => (props.orientation === 'horizontal' && width.value !== 'narrow' ? 'horizontal' : 'vertical'))
const roving = useRovingFocus(list, {
  get orientation() {
    return orientation.value
  },
  items: '.weft-stepper__trigger',
})
watch([current, list], ([i]) => roving.setActive(i), { flush: 'post' })

const stateOf = (i: number): StepState => (i < current.value ? 'completed' : i === current.value ? 'active' : 'inactive')
const reachable = (i: number) => !props.disabled && !props.items[i]?.disabled && (!props.linear || i <= current.value + 1)

/** The nearest step from `from` in direction `d` that is not disabled. */
function seek(from: number, d: 1 | -1): number {
  for (let i = from + d; i >= 0 && i < props.items.length; i += d) if (!props.items[i]!.disabled) return i
  return -1
}
const hasNext = computed(() => !props.disabled && seek(current.value, 1) >= 0)
const hasPrev = computed(() => !props.disabled && seek(current.value, -1) >= 0)

function goTo(i: number) {
  const item = props.items[i]
  if (item) model.value = valueOf(item, i)
}
function next() {
  const i = hasNext.value ? seek(current.value, 1) : -1
  if (i < 0) return
  goTo(i)
  emit('next', valueOf(props.items[i]!, i))
}
function prev() {
  const i = hasPrev.value ? seek(current.value, -1) : -1
  if (i < 0) return
  goTo(i)
  emit('prev', valueOf(props.items[i]!, i))
}
function choose(i: number) {
  if (reachable(i)) goTo(i)
}

defineExpose({ next, prev, hasNext, hasPrev })
</script>

<template>
  <div ref="root" class="weft-stepper" :data-orientation="orientation" :data-width="adapts ? width : undefined" :data-weft-w="widthId" :data-size="size" :data-tone="tone">
    <ol ref="list" class="weft-stepper__list">
      <li v-for="(item, i) in items" :key="String(valueOf(item, i))" class="weft-stepper__item" :data-state="stateOf(i)">
        <button
          type="button"
          class="weft-stepper__trigger"
          :aria-current="i === current ? 'step' : undefined"
          :disabled="!reachable(i) && i !== current ? true : undefined"
          @click="choose(i)"
        >
          <span class="weft-stepper__indicator" aria-hidden="true">
            <slot name="indicator" :item="item" :index="i" :state="stateOf(i)">
              <svg v-if="i < current" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
              <Icon v-else-if="item.icon" :name="item.icon" />
              <template v-else>{{ i + 1 }}</template>
            </slot>
          </span>
          <span class="weft-stepper__text">
            <span class="weft-stepper__title"><slot name="title" :item="item" :index="i" :state="stateOf(i)">{{ item.title }}</slot></span>
            <span v-if="item.description || $slots.description" class="weft-stepper__description"><slot name="description" :item="item" :index="i" :state="stateOf(i)">{{ item.description }}</slot></span>
          </span>
        </button>
        <span v-if="i < items.length - 1" class="weft-stepper__separator" aria-hidden="true"></span>
      </li>
    </ol>
    <div v-if="currentItem && (currentItem.content || currentItem.slot || $slots.content)" class="weft-stepper__content">
      <slot v-if="currentItem.slot && $slots[currentItem.slot]" :name="currentItem.slot" :item="currentItem" :index="current" state="active" />
      <slot v-else name="content" :item="currentItem" :index="current" state="active">{{ currentItem.content }}</slot>
    </div>
  </div>
</template>
