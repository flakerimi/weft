<script setup vapor lang="ts">
// A Frequently Asked Questions section pairing a section header (and optional
// `aside` slot) with an Accordion whose answers are rendered on the server too
// and kept in the page with hidden="until-found", so find-in-page matches text
// inside a closed answer and opens it. `items` uses AccordionItem's shape
// (`{ label, content, value, icon, trailingIcon, disabled, slot }`).
// `orientation` is `'vertical'` (default: header above the questions) or
// `'horizontal'` (header and aside in a column beside the questions when the
// block's own width is `wide`, stacked when `narrow` or `regular`).
import { computed, useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import Accordion from '../disclosure/Accordion.vue'
import { useControllable } from '../headless/useControllable.ts'
import type { FaqItem } from './types.ts'
import '../styles/base.css'
import '../styles/faq.css'

const props = withDefaults(
  defineProps<{
    /** Short eyebrow text above the section title; the headline slot replaces it. */
    headline?: string
    /** The section's heading, rendered as an h2 by default (or `h${level}`). */
    title?: string
    /** A lead sentence under the title. */
    description?: string
    /** The questions and answers (`AccordionItem[]`: `{ label, content, value, icon, trailingIcon, disabled, slot }`). */
    items?: FaqItem[]
    /** `single` (one question open at a time, default) or `multiple` (any number). */
    type?: 'single' | 'multiple'
    /** The open question's value (`single`; null when none is), or the open values (`multiple`), bound with `v-model`. */
    modelValue?: string | string[] | null
    /** The open question or questions when `v-model` is not bound. */
    defaultValue?: string | string[] | null
    /** With `single`: the open question closes again when its header is pressed. */
    collapsible?: boolean
    /** `vertical` (heading above the questions, default) or `horizontal` (heading beside the questions when wide). */
    orientation?: 'vertical' | 'horizontal'
    /** Alignment of the header when `orientation="vertical"`: `start` (default) or `center`. */
    align?: 'start' | 'center'
    /** The section's heading level, 2 to 6; question headings in the Accordion render one level deeper. */
    level?: 2 | 3 | 4 | 5 | 6
  }>(),
  {
    items: () => [],
    type: 'single',
    modelValue: undefined,
    defaultValue: undefined,
    collapsible: true,
    orientation: 'vertical',
    align: 'start',
    level: 2,
  },
)

const emit = defineEmits<{
  /** A question opened or closed. */
  'update:modelValue': [value: string | string[] | null]
}>()

defineSlots<{
  /** Above the section title, in place of `headline`. */
  headline?: () => unknown
  /** The section's heading block, in place of `title` and `description`. */
  header?: () => unknown
  /** Extra content below the header (such as a contact link). */
  aside?: () => unknown
  /** A question header's text, in place of `label`. */
  label?: (props: { item: FaqItem; index: number; open: boolean }) => unknown
  /** An answer panel's content, in place of `content`. */
  content?: (props: { item: FaqItem; index: number; open: boolean }) => unknown
  /** The answer panel of the item whose `slot` names it. */
  [name: string]: ((props: { item: FaqItem; index: number; open: boolean }) => unknown) | undefined
}>()

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root)
const model = useControllable(
  props,
  'modelValue',
  emit,
  () => props.defaultValue ?? (props.type === 'multiple' ? [] : null),
)

const questionLevel = computed(() => Math.min(props.level + 1, 6) as 2 | 3 | 4 | 5 | 6)
const namedItemSlots = computed(() => {
  const names = new Set<string>()
  for (const item of props.items) {
    if (item.slot && item.slot !== 'headline' && item.slot !== 'header' && item.slot !== 'aside' && item.slot !== 'label' && item.slot !== 'content') {
      names.add(item.slot)
    }
  }
  return [...names]
})
</script>

<template>
  <section
    ref="root"
    class="weft-faq"
    :data-orientation="orientation"
    :data-align="align"
    :data-width="width"
  >
    <header v-if="headline || title || description || $slots.headline || $slots.header || $slots.aside" class="weft-faq__header">
      <slot name="header">
        <div v-if="headline || $slots.headline" class="weft-faq__headline"><slot name="headline">{{ headline }}</slot></div>
        <component :is="`h${level}`" v-if="title" class="weft-faq__title">{{ title }}</component>
        <p v-if="description" class="weft-faq__description">{{ description }}</p>
      </slot>
      <div v-if="$slots.aside" class="weft-faq__aside"><slot name="aside" /></div>
    </header>
    <Accordion
      v-model="model"
      class="weft-faq__accordion"
      :items="items"
      :type="type"
      :collapsible="collapsible"
      :level="questionLevel"
    >
      <template v-if="$slots.label" #label="slotProps">
        <slot name="label" v-bind="slotProps" />
      </template>
      <template v-if="$slots.content" #content="slotProps">
        <slot name="content" v-bind="slotProps" />
      </template>
      <template v-for="slotName in namedItemSlots" :key="slotName" #[slotName]="slotProps">
        <slot :name="slotName" v-bind="slotProps" />
      </template>
    </Accordion>
  </section>
</template>
