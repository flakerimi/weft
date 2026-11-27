<script setup vapor lang="ts">
// The end of a page, in a <footer> (the contentinfo landmark at the end of
// a page): an optional band above, then a row with content at the start, in
// the middle and at the end (copyright, links, social buttons), then an
// optional band below. When its own width is narrow the row stacks, centred.
// It has nothing interactive of its own.
import { useTemplateRef } from 'vue'
import { useWidth } from '../adaptive/useWidth.ts'
import '../styles/base.css'
import '../styles/footer.css'

defineSlots<{
  /** A band above the row (columns of links, a newsletter form). */
  top?: () => unknown
  /** The start of the row (the copyright). */
  leading?: () => unknown
  /** The middle of the row (links). */
  default?: () => unknown
  /** The end of the row (social links). */
  trailing?: () => unknown
  /** A band below the row. */
  bottom?: () => unknown
}>()

const root = useTemplateRef<HTMLElement>('root')
const width = useWidth(root, { prepaint: false })
</script>

<template>
  <footer ref="root" class="weft-footer" :data-width="width">
    <div v-if="$slots.top" class="weft-footer__top"><slot name="top" /></div>
    <div class="weft-footer__row">
      <div v-if="$slots.leading" class="weft-footer__leading"><slot name="leading" /></div>
      <div v-if="$slots.default" class="weft-footer__center"><slot /></div>
      <div v-if="$slots.trailing" class="weft-footer__trailing"><slot name="trailing" /></div>
    </div>
    <div v-if="$slots.bottom" class="weft-footer__bottom"><slot name="bottom" /></div>
  </footer>
</template>
