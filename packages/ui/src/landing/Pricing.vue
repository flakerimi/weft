<script setup vapor lang="ts">
// A pricing section with plans and an optional billing-cycle ToggleGroup
// (`cycles` + `v-model:cycle` / `defaultCycle`). Each plan's `price`, `period`
// and `billingNote` may be a plain string (the same for every cycle) or a
// per-cycle map (`{ monthly: '$29', yearly: '$24' }`). A `price` or `period`
// map that lacks the active cycle renders an empty price area for that plan
// (never showing a wrong cycle's price) and warns once in development naming
// the plan and the cycle. Plans render in a Grid that sizes its columns to the
// plan count and its own width.
import { computed } from 'vue'
import Badge from '../basics/Badge.vue'
import Button from '../basics/Button.vue'
import Icon from '../basics/Icon.vue'
import type { ActionItem } from '../feedback/actions.ts'
import ToggleGroup from '../forms/ToggleGroup.vue'
import type { ToggleGroupItem } from '../forms/toggle.ts'
import { useControllable } from '../headless/useControllable.ts'
import Grid from '../layout/Grid.vue'
import type { Tone } from '../types.ts'
import type { PricingCycle, PricingPlan } from './types.ts'
import '../styles/base.css'
import '../styles/tone.css'
import '../styles/pricing.css'

type Cols = number | { narrow?: number; regular?: number; wide?: number }

const warnedMissingCycle = new Set<string>()

const props = withDefaults(
  defineProps<{
    /** Short eyebrow text above the section title; the headline slot replaces it. */
    headline?: string
    /** The section's heading, rendered as an h2 by default (or `h${level}`). */
    title?: string
    /** A lead sentence under the title. */
    description?: string
    /** The pricing plans: `{ id, title, description, price, period, billingNote, badge, highlight, features, action, slot }`. */
    plans?: PricingPlan[]
    /** Optional billing cycles (such as Monthly and Yearly), rendered as a single ToggleGroup above the plans. */
    cycles?: PricingCycle[]
    /** The active billing cycle value, bound with `v-model:cycle`. */
    cycle?: string
    /** The active billing cycle when `v-model:cycle` is not bound; defaults to the first cycle's value. */
    defaultCycle?: string
    /** Accessible label for the billing cycle ToggleGroup. */
    cycleLabel?: string
    /** Columns: one number for every width, or `{ narrow, regular, wide }` of the grid's own width. */
    cols?: Cols
    /** The colour of the highlighted plan, badges and feature checkmarks. */
    tone?: Tone
    /** Alignment of the section header and cycle switch: `center` (default) or `start`. */
    align?: 'start' | 'center'
    /** The section's heading level, 2 to 6; plan titles render one level deeper. */
    level?: 2 | 3 | 4 | 5 | 6
  }>(),
  {
    plans: () => [],
    cycles: () => [],
    cycle: undefined,
    defaultCycle: undefined,
    cycleLabel: 'Billing period',
    cols: undefined,
    tone: 'primary',
    align: 'center',
    level: 2,
  },
)

const emit = defineEmits<{
  /** The selected billing cycle changed. */
  'update:cycle': [cycle: string]
}>()

defineSlots<{
  /** Above the section title, in place of `headline`. */
  headline?: () => unknown
  /** The section's heading block, in place of `title` and `description`. */
  header?: () => unknown
  /** Custom content for a plan card. */
  plan?: (props: { plan: PricingPlan; index: number; cycle: string | undefined }) => unknown
  /** A note or enterprise link below the plans. */
  footer?: () => unknown
  /** Custom body for the plan whose `slot` names it. */
  [name: string]: ((props: { plan: PricingPlan; index: number; cycle: string | undefined }) => unknown) | undefined
}>()

const activeCycle = useControllable(
  props,
  'cycle',
  emit,
  () => props.defaultCycle ?? props.cycles[0]?.value ?? '',
)

const cycleItems = computed<ToggleGroupItem[]>(() =>
  props.cycles.map((c) => ({ label: c.label, value: c.value })),
)
const cycleBadgeOf = (value: string | number | undefined) =>
  props.cycles.find((c) => c.value === value)?.badge

function onCycleChange(next: string | number | null | (string | number)[]) {
  if (typeof next === 'string' && next.length > 0) activeCycle.value = next
}

const resolvedCols = computed<Cols>(() => {
  if (props.cols !== undefined) return props.cols
  const count = Math.max(props.plans.length, 1)
  return {
    narrow: 1,
    regular: Math.min(count, 2),
    wide: Math.min(count, 4),
  }
})

const planHeading = computed(() => `h${Math.min(props.level + 1, 6)}`)

function warnMissing(plan: PricingPlan, field: 'price' | 'period', cycle: string) {
  if (!import.meta.env.DEV) return
  const name = plan.id ?? plan.title
  const key = `${name}:${field}:${cycle}`
  if (warnedMissingCycle.has(key)) return
  warnedMissingCycle.add(key)
  console.warn(
    `Pricing: plan "${name}" has no ${field} for cycle "${cycle}"; give it an entry in packages/ui/src/landing/Pricing.vue or use a plain string for every cycle.`,
  )
}

function priceOf(plan: PricingPlan): string {
  if (typeof plan.price === 'string') return plan.price
  const c = activeCycle.value
  if (c && Object.prototype.hasOwnProperty.call(plan.price, c)) return plan.price[c]!
  if (c) warnMissing(plan, 'price', c)
  return ''
}

function periodOf(plan: PricingPlan): string {
  if (!plan.period) return ''
  if (typeof plan.period === 'string') return plan.period
  const c = activeCycle.value
  if (c && Object.prototype.hasOwnProperty.call(plan.period, c)) return plan.period[c]!
  if (c) warnMissing(plan, 'period', c)
  return ''
}

function noteOf(plan: PricingPlan): string {
  if (!plan.billingNote) return ''
  if (typeof plan.billingNote === 'string') return plan.billingNote
  const c = activeCycle.value
  return (c && plan.billingNote[c]) || ''
}

function planAction(plan: PricingPlan, action: ActionItem) {
  return {
    ...action,
    tone: action.tone ?? (plan.highlight ? props.tone : 'neutral'),
    variant: action.variant ?? (plan.highlight ? 'solid' : 'outline'),
  }
}
</script>

<template>
  <section class="weft-pricing" :data-tone="tone" :data-align="align">
    <header v-if="headline || title || description || cycles.length || $slots.headline || $slots.header" class="weft-pricing__header">
      <slot name="header">
        <div v-if="headline || $slots.headline" class="weft-pricing__headline"><slot name="headline">{{ headline }}</slot></div>
        <component :is="`h${level}`" v-if="title" class="weft-pricing__title">{{ title }}</component>
        <p v-if="description" class="weft-pricing__description">{{ description }}</p>
      </slot>
      <ToggleGroup
        v-if="cycles.length"
        class="weft-pricing__cycles"
        :aria-label="cycleLabel"
        :items="cycleItems"
        :model-value="activeCycle"
        :active-tone="tone"
        active-variant="solid"
        variant="ghost"
        @update:model-value="onCycleChange"
      >
        <template #default="{ item, pressed }">
          <span class="weft-pricing__cycle-label">{{ item.label }}</span>
          <Badge
            v-if="cycleBadgeOf(item.value)"
            class="weft-pricing__cycle-badge"
            :label="cycleBadgeOf(item.value)"
            :tone="pressed ? 'neutral' : tone"
            variant="soft"
            size="sm"
          />
        </template>
      </ToggleGroup>
    </header>
    <Grid class="weft-pricing__grid" :cols="resolvedCols" :gap="4">
      <div
        v-for="(plan, i) in plans"
        :key="plan.id || plan.title || i"
        class="weft-pricing__plan"
        :data-highlight="plan.highlight ? 'true' : undefined"
      >
        <slot v-if="plan.slot && $slots[plan.slot]" :name="plan.slot" :plan="plan" :index="i" :cycle="activeCycle || undefined" />
        <slot v-else name="plan" :plan="plan" :index="i" :cycle="activeCycle || undefined">
          <div class="weft-pricing__plan-top">
            <div class="weft-pricing__plan-header">
              <component :is="planHeading" class="weft-pricing__plan-title">{{ plan.title }}</component>
              <Badge v-if="plan.badge" class="weft-pricing__plan-badge" :label="plan.badge" :tone="tone" variant="soft" size="sm" />
            </div>
            <p v-if="plan.description" class="weft-pricing__plan-description">{{ plan.description }}</p>
          </div>
          <div class="weft-pricing__price-block">
            <div v-if="priceOf(plan)" class="weft-pricing__price-row">
              <span class="weft-pricing__price">{{ priceOf(plan) }}</span>
              <span v-if="periodOf(plan)" class="weft-pricing__period">{{ periodOf(plan) }}</span>
            </div>
            <div v-if="noteOf(plan)" class="weft-pricing__note">{{ noteOf(plan) }}</div>
          </div>
          <ul v-if="plan.features?.length" class="weft-pricing__features">
            <li v-for="(feat, fi) in plan.features" :key="fi" class="weft-pricing__feature">
              <span class="weft-pricing__feature-icon" aria-hidden="true">
                <Icon name="check" />
              </span>
              <span class="weft-pricing__feature-text">{{ feat }}</span>
            </li>
          </ul>
          <div v-if="plan.action" class="weft-pricing__action">
            <Button
              class="weft-pricing__button"
              :label="planAction(plan, plan.action).label"
              :to="planAction(plan, plan.action).to"
              :icon="planAction(plan, plan.action).icon"
              :tone="planAction(plan, plan.action).tone"
              :variant="planAction(plan, plan.action).variant"
              :disabled="planAction(plan, plan.action).disabled"
              :loading="planAction(plan, plan.action).loading"
              block
              @click="plan.action.onClick?.($event)"
            />
          </div>
        </slot>
      </div>
    </Grid>
    <footer v-if="$slots.footer" class="weft-pricing__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>
