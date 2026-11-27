<script setup vapor lang="ts">
// Events in order (an order's progress, an account's history): an ordered
// list, each item a dot or icon on a line, with its date, title and
// description. Given the current item (`v-model`, its value or its place),
// the items before it are done (said as "Done:" to assistive technology)
// and it is the current step (aria-current="step"); `reverse` lists the
// newest first. Not interactive. Horizontal when asked and its own box is
// wide enough; vertical when narrow, in CSS from `data-width` (so before
// hydration too). Its type scale follows the surface.
import { computed, useTemplateRef } from 'vue'
import Icon from '../basics/Icon.vue'
import { useWidth } from '../adaptive/useWidth.ts'
import type { Size, Tone } from '../types.ts'
import { TIMELINE_LABELS, type TimelineItem, type TimelineLabels } from './timeline.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/timeline.css'

const props = withDefaults(
  defineProps<{
    /** The events: `{ date, title, description, icon, value }`, oldest first. */
    items?: TimelineItem[]
    /** The current item's value (or its place, from 0): those before it are done. */
    modelValue?: string | number
    /** The current item when `v-model` is not bound. */
    defaultValue?: string | number
    /** `vertical` (default) or `horizontal`; vertical whenever its box is narrow. */
    orientation?: 'vertical' | 'horizontal'
    /** List the newest first. */
    reverse?: boolean
    /** The colour of the done and current items. */
    tone?: Tone
    /** `sm`, `md` or `lg`: the dots and the text. */
    size?: Size
    /** The words it says; give some or all to translate them. */
    labels?: Partial<TimelineLabels>
  }>(),
  { items: () => [], modelValue: undefined, defaultValue: undefined, orientation: 'vertical', tone: 'primary', size: 'md' },
)

defineSlots<{
  /** An item's dot, in place of its icon. */
  indicator?: (props: { item: TimelineItem; index: number }) => unknown
  /** An item's date. */
  date?: (props: { item: TimelineItem; index: number }) => unknown
  /** An item's title. */
  title?: (props: { item: TimelineItem; index: number }) => unknown
  /** An item's description. */
  description?: (props: { item: TimelineItem; index: number }) => unknown
}>()

const t = computed<TimelineLabels>(() => ({ ...TIMELINE_LABELS, ...props.labels }))
const root = useTemplateRef<HTMLElement>('root')
// Only a horizontal timeline depends on its width. It says its width class in
// `data-width`, which the page's pre-paint script keeps current before
// hydration, and CSS draws it vertical on `[data-width="narrow"]`, so a
// static page adapts too.
const adapts = props.orientation === 'horizontal'
const width = useWidth(root, { prepaint: adapts })
const widthId = width.id
const layout = computed(() => (props.orientation === 'horizontal' && width.value !== 'narrow' ? 'horizontal' : 'vertical'))
const value = computed(() => props.modelValue ?? props.defaultValue)
/** The current item's place in `items`, -1 for none. */
const current = computed(() => {
  const v = value.value
  if (v === undefined) return -1
  const at = props.items.findIndex((item) => item.value === v)
  return at >= 0 ? at : typeof v === 'number' && v < props.items.length ? v : -1
})
/** The items in the order drawn, each with its place in `items`. */
const order = computed(() => {
  const all = props.items.map((item, index) => ({ item, index }))
  return props.reverse ? all.reverse() : all
})
const state = (index: number) => (current.value < 0 ? undefined : index < current.value ? 'done' : index === current.value ? 'current' : undefined)
</script>

<template>
  <ol ref="root" class="weft-timeline" :data-orientation="layout" :data-width="adapts ? width : undefined" :data-weft-w="widthId" :data-tone="tone" :data-size="size">
    <li
      v-for="{ item, index } in order"
      :key="item.value ?? index"
      class="weft-timeline__item"
      :data-state="state(index)"
      :aria-current="state(index) === 'current' ? 'step' : undefined"
    >
      <div class="weft-timeline__marker" aria-hidden="true">
        <span class="weft-timeline__indicator" aria-hidden="true"><slot name="indicator" :item="item" :index="index"><Icon v-if="item.icon" :name="item.icon" /></slot></span>
        <span class="weft-timeline__line"></span>
      </div>
      <div class="weft-timeline__content">
        <p v-if="item.date || $slots.date" class="weft-timeline__date"><slot name="date" :item="item" :index="index">{{ item.date }}</slot></p>
        <p v-if="item.title || $slots.title" class="weft-timeline__title">
          <span v-if="state(index) === 'done'" class="weft-visually-hidden">{{ t.done }}</span>
          <slot name="title" :item="item" :index="index">{{ item.title }}</slot>
        </p>
        <div v-if="item.description || $slots.description" class="weft-timeline__description"><slot name="description" :item="item" :index="index">{{ item.description }}</slot></div>
      </div>
    </li>
  </ol>
</template>
