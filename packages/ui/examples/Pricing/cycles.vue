<script setup vapor lang="ts">
import { ref } from 'vue'
import { Pricing, type PricingCycle, type PricingPlan } from '@weft/ui'

const cycle = ref('yearly')

const cycles: PricingCycle[] = [
  { label: 'Monthly', value: 'monthly' },
  { label: 'Yearly', value: 'yearly', badge: 'Save 20%' },
]

const plans: PricingPlan[] = [
  {
    title: 'Hobby',
    description: 'One project on a single server.',
    price: '$0',
    period: '/month',
    features: ['Single binary deploy', 'SQLite store', 'All 82 components'],
    action: { label: 'Install Weft' },
  },
  {
    title: 'Pro',
    description: 'For growing products and teams.',
    price: { monthly: '$30', yearly: '$24' },
    period: '/month',
    billingNote: { monthly: 'Billed monthly', yearly: 'Billed $288 yearly' },
    badge: 'Recommended',
    highlight: true,
    features: ['Unlimited projects', 'Team roles and audit log', 'Priority email support'],
    action: { label: 'Start Pro trial' },
  },
]
</script>

<template>
  <Pricing
    v-model:cycle="cycle"
    title="Switch between monthly and yearly billing"
    :cycles="cycles"
    :plans="plans"
  />
</template>
