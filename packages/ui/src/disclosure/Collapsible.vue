<script setup vapor lang="ts">
// Content shown and hidden by a trigger (WAI-ARIA APG, Disclosure). The
// trigger is a button with aria-expanded and aria-controls: the default
// slot's one element (a Button), or one drawn from `label`. The content is
// always rendered, on the server too, and hidden with
// `hidden="until-found"`: find-in-page reaches it and opens it
// (`beforematch`); a browser without until-found treats it as plain
// `hidden`. Its height animates open and shut in CSS, not under reduced
// motion. Disabled, the trigger opens nothing.
import { useTemplateRef } from 'vue'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useTrigger } from '../overlays/overlay.ts'
import '../styles/base.css'
import '../styles/collapsible.css'

const props = withDefaults(
  defineProps<{
    /** Open or shut, bound with `v-model:open`. */
    open?: boolean
    /** Whether it starts open when `v-model:open` is not bound. */
    defaultOpen?: boolean
    /** The text of a trigger button drawn for you, when the default slot gives none. */
    label?: string
    /** The trigger opens nothing; a drawn trigger is disabled. */
    disabled?: boolean
  }>(),
  { open: undefined, defaultOpen: false },
)

const emit = defineEmits<{
  /** Opened or shut, by the trigger or by find-in-page. */
  'update:open': [open: boolean]
}>()

defineSlots<{
  /** The trigger: one element (a Button), which toggles it. */
  default?: (props: { open: boolean }) => unknown
  /** What it shows and hides. */
  content?: () => unknown
}>()

const open = useControllable(props, 'open', emit, () => props.defaultOpen)
const id = useId('collapsible')
const triggerId = `${id}-trigger`
const wrap = useTemplateRef<HTMLElement>('wrap')
useTrigger(wrap, () => ({ 'aria-expanded': open.value ? 'true' : 'false', 'aria-controls': id }))

function toggle(event?: MouseEvent) {
  if (props.disabled || event?.defaultPrevented) return
  open.value = !open.value
}
</script>

<template>
  <div class="weft-collapsible" :data-state="open ? 'open' : 'closed'" :data-disabled="disabled ? 'true' : undefined">
    <span v-if="$slots.default" ref="wrap" class="weft-collapsible__wrap" @click="toggle"><slot :open="open" /></span>
    <button
      v-else
      :id="triggerId"
      type="button"
      class="weft-collapsible__trigger"
      :aria-expanded="open ? 'true' : 'false'"
      :aria-controls="id"
      :disabled="disabled || undefined"
      @click="toggle"
    >
      <span class="weft-collapsible__label">{{ label }}</span>
      <svg class="weft-collapsible__chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
    </button>
    <div :id="id" class="weft-collapsible__content" :hidden.attr="open ? undefined : 'until-found'" @beforematch="open = true">
      <div class="weft-collapsible__body"><slot name="content" /></div>
    </div>
  </div>
</template>
