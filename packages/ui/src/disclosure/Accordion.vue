<script setup vapor lang="ts">
// Stacked sections whose content opens under its header (WAI-ARIA APG,
// Accordion). Each header is a heading (h3 unless `level` says) holding a
// button with aria-expanded and aria-controls; each panel is a region named
// by its header. Enter and Space toggle a section; Down and Up move between
// headers, wrapping, Home and End go to the first and last, disabled ones
// skipped, and every header stays in the tab order. `single` keeps one
// section open; `collapsible` (the default) lets it close again, and when it
// may not, its header says aria-disabled. Panels are rendered, on the server
// too, and hidden with hidden="until-found": find-in-page opens the section
// it finds text in. Heights animate in CSS, not under reduced motion.
import { computed, useTemplateRef } from 'vue'
import Icon from '../basics/Icon.vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useRovingFocus } from '../headless/useRovingFocus.ts'
import type { AccordionItem } from './accordion.ts'
import '../styles/base.css'
import '../styles/collapsible.css'
import '../styles/accordion.css'

const props = withDefaults(
  defineProps<{
    /** The sections: `{ label, value, icon, content, disabled, slot }`. */
    items?: AccordionItem[]
    /** `single`: one section open at a time. `multiple`: any number. */
    type?: 'single' | 'multiple'
    /** The open section's value (`single`; null when none is), or the open values (`multiple`), bound with `v-model`. A section's value is its `value`, else its index as a string. */
    modelValue?: string | string[] | null
    /** The open section or sections when `v-model` is not bound. */
    defaultValue?: string | string[] | null
    /** With `single`: the open section closes again when its header is pressed. */
    collapsible?: boolean
    /** Every section: none opens or closes. */
    disabled?: boolean
    /** The heading level of the headers, 2 to 6, to fit the page's outline. */
    level?: 2 | 3 | 4 | 5 | 6
    /** A Lucide icon at the end of each header, in place of the chevron; an item's `trailingIcon` wins. */
    trailingIcon?: string
  }>(),
  { items: () => [], type: 'single', modelValue: undefined, defaultValue: undefined, collapsible: true, level: 3 },
)

const emit = defineEmits<{
  /** A section opened or closed. */
  'update:modelValue': [value: string | string[] | null]
}>()

defineSlots<{
  /** A header's text, in place of `label`. */
  label?: (props: { item: AccordionItem; index: number; open: boolean }) => unknown
  /** A panel's content, in place of `content`. */
  content?: (props: { item: AccordionItem; index: number; open: boolean }) => unknown
  /** The panel of the item whose `slot` names it. */
  [name: string]: ((props: { item: AccordionItem; index: number; open: boolean }) => unknown) | undefined
}>()

const model = useControllable(props, 'modelValue', emit, () => props.defaultValue ?? (props.type === 'multiple' ? [] : null))
const id = useId('accordion')
const root = useTemplateRef<HTMLElement>('root')
useRovingFocus(root, { orientation: 'vertical', items: '.weft-accordion__trigger', tabStops: 'all' })

const valueOf = (item: AccordionItem, index: number) => item.value ?? String(index)
const openValues = computed<string[]>(() => {
  const v = model.value
  return v == null ? [] : Array.isArray(v) ? v : [v]
})
const isOpen = (value: string) => openValues.value.includes(value)
/** A single, non-collapsible accordion's open section cannot be closed by its header. */
const locked = (value: string) => props.type === 'single' && !props.collapsible && isOpen(value)

function set(value: string, open: boolean) {
  if (props.type === 'multiple') {
    const now = openValues.value.filter((v) => v !== value)
    model.value = open ? [...now, value] : now
  } else if (open) model.value = value
  else if (props.collapsible) model.value = null
}
function toggle(item: AccordionItem, index: number) {
  const value = valueOf(item, index)
  if (props.disabled || item.disabled || locked(value)) return
  set(value, !isOpen(value))
}
function reveal(item: AccordionItem, index: number) {
  const value = valueOf(item, index)
  if (!isOpen(value)) set(value, true)
}
</script>

<template>
  <div ref="root" class="weft-accordion" :data-disabled="disabled ? 'true' : undefined">
    <div
      v-for="(item, i) in items"
      :key="valueOf(item, i)"
      class="weft-accordion__item"
      :data-state="isOpen(valueOf(item, i)) ? 'open' : 'closed'"
      :data-disabled="disabled || item.disabled ? 'true' : undefined"
    >
      <component :is="`h${level}`" class="weft-accordion__header">
        <button
          :id="`${id}-${i}-header`"
          type="button"
          class="weft-accordion__trigger"
          :aria-expanded="isOpen(valueOf(item, i)) ? 'true' : 'false'"
          :aria-controls="`${id}-${i}`"
          :aria-disabled="locked(valueOf(item, i)) ? 'true' : undefined"
          :disabled="disabled || item.disabled || undefined"
          @click="toggle(item, i)"
        >
          <Icon v-if="item.icon" :name="item.icon" class="weft-accordion__icon" />
          <span class="weft-accordion__label"><slot name="label" :item="item" :index="i" :open="isOpen(valueOf(item, i))">{{ item.label }}</slot></span>
          <Icon v-if="item.trailingIcon || trailingIcon" :name="(item.trailingIcon || trailingIcon)!" class="weft-accordion__trailing" />
          <svg v-else class="weft-accordion__trailing weft-accordion__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
        </button>
      </component>
      <div
        :id="`${id}-${i}`"
        class="weft-collapsible__content weft-accordion__content"
        role="region"
        :aria-labelledby="`${id}-${i}-header`"
        :hidden.attr="isOpen(valueOf(item, i)) ? undefined : 'until-found'"
        @beforematch="reveal(item, i)"
      >
        <div class="weft-collapsible__body weft-accordion__body">
          <slot v-if="item.slot && $slots[item.slot]" :name="item.slot" :item="item" :index="i" :open="isOpen(valueOf(item, i))" />
          <slot v-else name="content" :item="item" :index="i" :open="isOpen(valueOf(item, i))">{{ item.content }}</slot>
        </div>
      </div>
    </div>
  </div>
</template>
