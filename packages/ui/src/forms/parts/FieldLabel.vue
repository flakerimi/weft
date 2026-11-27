<script setup vapor lang="ts">
// A FormField's label, a part only FormField draws: a <label> for the
// field's control, or, over a control that is not a labelable element (a
// group, a slider's thumbs, a listbox), a <span> that control is
// aria-labelledby, since a <label> must label a form control.
//
// The control says which in its setup, and the field draws its label before
// the control. In the browser the label follows when the control says so.
// The server writes each element once, so there this part is drawn last:
// a server prefetch makes the server render it once the rest of the render
// has run synchronously, the field's control (and its setup) included, and
// then it knows. A control that sets up later than that (inside a component
// with an async setup or a prefetch of its own) is not waited for: the server
// draws a <label for> over it, and the island puts that right once it
// hydrates.
//
// When the label turns from one element into the other (InputTime switching
// between its parts and the native time input), what the label slot holds is
// set up again, and loses its state.
//
// What the label slot holds is no control of the field's (a Checkbox there
// takes neither the field's id nor its name), and, in an island, takes its ids
// from a scope of the label's own: the server draws it after the rest of the
// island, and its ids must not move the island's count.
import { inject, onServerPrefetch, provide } from 'vue'
import { ID_SCOPE, type IdScope } from '../../headless/useId.ts'
import { hideField } from '../field.ts'

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
hideField()
if (inject<IdScope | null>(ID_SCOPE, null)) provide<IdScope>(ID_SCOPE, { prefix: props.id, next: 0 })
</script>

<template>
  <span v-if="named()" :id="id" class="weft-form-field__label"><slot /></span>
  <label v-else :id="id" :for="props.control" class="weft-form-field__label"><slot /></label>
</template>
