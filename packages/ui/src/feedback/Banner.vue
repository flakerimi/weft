<script setup vapor lang="ts">
// A strip across the top of a page or an app for news that concerns
// everyone (maintenance, a new feature): an icon, a title, actions, a close
// button, in a tone. With `to` the title is a link. With an `id` a closed
// banner stays closed on later visits (localStorage, read once mounted, so
// the server always draws it). On touch the close button's hit area is at
// least --weft-hit-min; when narrow the actions wrap under the title.
import { onMounted, ref } from 'vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import Link from '../basics/Link.vue'
import CloseButton from '../overlays/CloseButton.vue'
import type { Tone } from '../types.ts'
import type { ActionItem } from './actions.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/banner.css'

const props = withDefaults(
  defineProps<{
    /** Remembers being closed under this id, in localStorage; without it, it shows again on the next page. */
    id?: string
    /** The news. */
    title?: string
    /** A Lucide icon before the title. */
    icon?: string
    /** An address: the title links there. */
    to?: string
    /** The colour. */
    tone?: Tone
    /** Buttons after the title: `{ label, onClick, to, icon, tone, variant }`. */
    actions?: ActionItem[]
    /** Show a close button. */
    close?: boolean
    /** The close button's name for assistive technology. */
    closeLabel?: string
  }>(),
  { tone: 'primary', closeLabel: 'Close' },
)

const emit = defineEmits<{
  /** Closed with its button. */
  close: []
}>()

defineSlots<{
  /** The title, in place of `title`. */
  default?: () => unknown
  /** Before the title, in place of `icon`. */
  leading?: () => unknown
  /** The buttons, in place of `actions`. */
  actions?: () => unknown
}>()

const storeKey = () => `weft-banner:${props.id}`
const shown = ref(true)
onMounted(() => {
  if (!props.id) return
  try {
    if (localStorage.getItem(storeKey()) === 'closed') shown.value = false
  } catch {
    // Storage refused (private mode, a sandbox): it shows.
  }
})
function dismiss() {
  shown.value = false
  if (props.id) {
    try {
      localStorage.setItem(storeKey(), 'closed')
    } catch {
      // Not remembered; it is closed for this page.
    }
  }
  emit('close')
}
</script>

<template>
  <div v-if="shown" class="weft-banner" :data-tone="tone" :data-id="id">
    <div class="weft-banner__inner">
      <span v-if="$slots.leading || icon" class="weft-banner__leading"><slot name="leading"><Icon v-if="icon" :name="icon" /></slot></span>
      <Link v-if="to" class="weft-banner__title" :to="to"><slot>{{ title }}</slot></Link>
      <span v-else class="weft-banner__title"><slot>{{ title }}</slot></span>
      <div v-if="$slots.actions || actions?.length" class="weft-banner__actions">
        <slot name="actions">
          <Button
            v-for="action in actions"
            :key="action.label"
            size="sm"
            :label="action.label"
            :icon="action.icon"
            :to="action.to"
            :disabled="action.disabled"
            :loading="action.loading"
            :tone="action.tone ?? 'neutral'"
            :variant="action.variant ?? 'solid'"
            @click="action.onClick?.($event)"
          />
        </slot>
      </div>
    </div>
    <CloseButton v-if="close" class="weft-banner__close" :label="closeLabel" @click="dismiss" />
  </div>
</template>
