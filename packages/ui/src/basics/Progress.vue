<script setup vapor lang="ts">
// A bar showing how far a task has got: determinate with a value,
// indeterminate (a sweeping bar) without one. A `progressbar` in the WAI-ARIA
// sense; give it a `label` (or aria-labelledby) to name what it measures.
// The bar's length is logical, so it fills from the right in a right-to-left
// document. Under reduced motion the indeterminate bar stands still.
import { computed } from 'vue'
import type { Size, Tone } from '../types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/progress.css'

const props = withDefaults(
  defineProps<{
    /** How far along, from 0 to `max`; `null` or unset for unknown (indeterminate). */
    modelValue?: number | null
    /** The value at completion. */
    max?: number
    /** What it measures, for assistive technology ("Uploading report.pdf"). */
    label?: string
    /** The bar's colour and what it says. */
    tone?: Tone
    /** `sm`, `md` or `lg`: the bar's thickness. */
    size?: Size
    /** Show the percentage beside the bar (determinate only). */
    status?: boolean
  }>(),
  { modelValue: null, max: 100, tone: 'primary', size: 'md' },
)

const determinate = computed(() => typeof props.modelValue === 'number' && Number.isFinite(props.modelValue))
const value = computed(() => Math.min(Math.max(props.modelValue ?? 0, 0), props.max))
const percent = computed(() => (props.max > 0 ? Math.round((value.value / props.max) * 100) : 0))
</script>

<template>
  <div
    class="weft-progress"
    role="progressbar"
    :aria-label="label"
    aria-valuemin="0"
    :aria-valuemax="max"
    :aria-valuenow="determinate ? value : undefined"
    :data-state="determinate ? 'determinate' : 'indeterminate'"
    :data-tone="tone"
    :data-size="size"
  >
    <div class="weft-progress__track">
      <div class="weft-progress__bar" :style="determinate ? { inlineSize: `${percent}%` } : undefined"></div>
    </div>
    <span v-if="status && determinate" class="weft-progress__status">{{ percent }}%</span>
  </div>
</template>
