<script setup vapor lang="ts">
// The landing page's adaptive demo, an island that starts when it comes into
// view: VisitDay.vue, the example whose source the page shows, inside a
// WeftProvider whose input a Pointer/Touch toggle sets. On pointer the
// DatePicker's button opens its popover calendar; on touch the field is one
// button that opens its bottom sheet. The toggle starts at what the device
// reports and can be flipped while the calendar is open.
//
// The page draws both forms as static previews beside it, so before it
// starts, and without JavaScript, it shows only a note: a field and toggle
// that could do nothing are not offered.
import { onMounted, ref } from 'vue'
import { ToggleGroup, WeftProvider, detectInput, type InputMethod, type ToggleGroupItem } from '@weft/ui'
import VisitDay from './VisitDay.vue'

const mounted = ref(false)
const input = ref<InputMethod>('pointer')
const inputs: ToggleGroupItem[] = [
  { label: 'Pointer', value: 'pointer' },
  { label: 'Touch', value: 'touch' },
]

onMounted(() => {
  input.value = detectInput().value
  mounted.value = true
})

/** A press on the pressed button would leave neither; it is ignored. */
function choose(value: unknown) {
  if (value === 'pointer' || value === 'touch') input.value = value
}
</script>

<template>
  <section class="landing-try" aria-labelledby="landing-try-title">
    <h3 id="landing-try-title" class="landing-try__title">Try it</h3>
    <p v-if="!mounted" class="landing-try__note">The previews above are the real components drawn by the server, not ones you can use. With JavaScript on, the real date picker opens here, as a calendar or as a sheet.</p>
    <template v-else>
      <ToggleGroup :items="inputs" :model-value="input" size="sm" attached aria-label="Input" @update:model-value="choose" />
      <WeftProvider :input="input" surface="app">
        <VisitDay />
      </WeftProvider>
    </template>
  </section>
</template>
