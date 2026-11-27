<script setup vapor lang="ts">
// A FormField's label, a part only FormField draws: a <label> for the
// field's control, or, over a control that is not a labelable element (a
// group, a slider's thumbs, a listbox), a <span> that control is
// aria-labelledby, since a <label> must label a form control.
//
// The control says which in its setup, and the field draws its label before
// the control. In the browser the label follows when the control says so.
// The server writes each element once, so there this part is drawn last:
// a server prefetch makes the server render it once the rest of the page has
// rendered, the field's control (and its setup) included, and then it knows.
import { onServerPrefetch } from 'vue'

const props = defineProps<{
  /** The label's id. */
  id: string
  /** The control's id, the <label>'s `for`. */
  control: string
  /** True while the control names itself by the label's id. */
  named: () => boolean
}>()

defineSlots<{
  /** The label's text and the required marker. */
  default?: () => unknown
}>()

if (import.meta.env.SSR) onServerPrefetch(() => {})
</script>

<template>
  <span v-if="named()" :id="id" class="weft-form-field__label"><slot /></span>
  <label v-else :id="id" :for="props.control" class="weft-form-field__label"><slot /></label>
</template>
