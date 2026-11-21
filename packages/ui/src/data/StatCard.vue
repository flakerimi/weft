<script setup vapor lang="ts">
// One figure that matters, on a card: its label, its value (a number is
// formatted for the locale), how it changed (a delta whose sign picks the
// tone, green up and red down, swapped by `invert` for figures where less
// is better; drawn as +4% with an arrow and said as "up 4%"), a sentence
// after the delta ("vs last month") and an optional sparkline (drawing
// only, hidden from assistive technology). A description list, so the
// label names the value. While loading it is busy, a skeleton in place of
// the value. Its type scale follows the surface; it fills its grid cell.
import { computed } from 'vue'
import Icon from '../basics/Icon.vue'
import Skeleton from '../basics/Skeleton.vue'
import { useWeft } from '../adaptive/context.ts'
import { STAT_LABELS, type StatLabels } from './stat.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/stat-card.css'

const props = withDefaults(
  defineProps<{
    /** What the figure is: "Revenue". */
    label: string
    /** The figure: a number (formatted for the locale) or text as it should read ("$1.2M"). */
    value?: string | number
    /** How much it changed, as a number: 4 for a 4% rise, -2 for a fall. */
    delta?: number
    /** After the delta's number: `%` (default), ` pts`, or nothing. */
    deltaUnit?: string
    /** Less is better (churn, errors): a fall is good news, drawn in the success tone. */
    invert?: boolean
    /** A sentence after the delta: "vs last month". */
    description?: string
    /** A Lucide icon before the label. */
    icon?: string
    /** Still counting: a skeleton in place of the value, and aria-busy. */
    loading?: boolean
    /** The words it says; give some or all to translate them. */
    labels?: Partial<StatLabels>
  }>(),
  { value: undefined, delta: undefined, deltaUnit: '%' },
)

defineSlots<{
  /** A small chart beside the figure, drawn only. */
  sparkline?: () => unknown
  /** Before the label, in place of `icon`. */
  leading?: () => unknown
  /** Under everything: a link to the details. */
  footer?: () => unknown
}>()

const weft = useWeft()
const t = computed<StatLabels>(() => ({ ...STAT_LABELS, ...props.labels }))
// A fixed fallback locale, so the server and the browser format alike.
const locale = computed(() => weft.locale.value ?? 'en-US')
const shown = computed(() => (typeof props.value === 'number' ? new Intl.NumberFormat(locale.value).format(props.value) : (props.value ?? '')))
const amount = computed(() =>
  props.delta === undefined ? '' : `${new Intl.NumberFormat(locale.value, { maximumFractionDigits: 2 }).format(Math.abs(props.delta))}${props.deltaUnit}`,
)
const direction = computed(() => (props.delta === undefined ? undefined : props.delta > 0 ? 'up' : props.delta < 0 ? 'down' : 'flat'))
const tone = computed(() => {
  if (direction.value === 'flat') return 'neutral'
  return (direction.value === 'up') !== props.invert ? 'success' : 'danger'
})
const sign = computed(() => (direction.value === 'up' ? '+' : direction.value === 'down' ? '−' : ''))
const said = computed(() => (direction.value === 'up' ? t.value.up(amount.value) : direction.value === 'down' ? t.value.down(amount.value) : t.value.flat))
</script>

<template>
  <div class="weft-stat-card" :aria-busy="loading ? 'true' : undefined">
    <dl class="weft-stat-card__stat">
      <dt class="weft-stat-card__label">
        <span v-if="$slots.leading || icon" class="weft-stat-card__icon"><slot name="leading"><Icon v-if="icon" :name="icon" /></slot></span>
        {{ label }}
      </dt>
      <dd class="weft-stat-card__value">
        <Skeleton v-if="loading" width="6ch" height="1em" />
        <template v-else>{{ shown }}</template>
      </dd>
      <dd v-if="!loading && direction" class="weft-stat-card__delta" :data-tone="tone">
        <span class="weft-stat-card__badge">
          <svg class="weft-stat-card__arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
            <path v-if="direction === 'up'" d="M7 17 17 7M7 7h10v10" />
            <path v-else-if="direction === 'down'" d="M7 7l10 10M17 7v10H7" />
            <path v-else d="M5 12h14" />
          </svg>
          <span class="weft-stat-card__change" aria-hidden="true">{{ sign }}{{ amount }}</span>
          <span class="weft-visually-hidden">{{ said }}</span>
        </span>
        <span v-if="description" class="weft-stat-card__description">{{ description }}</span>
      </dd>
    </dl>
    <div v-if="$slots.sparkline" class="weft-stat-card__sparkline" aria-hidden="true"><slot name="sparkline" /></div>
    <div v-if="$slots.footer" class="weft-stat-card__footer"><slot name="footer" /></div>
  </div>
</template>
