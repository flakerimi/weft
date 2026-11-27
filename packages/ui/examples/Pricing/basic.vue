<script setup vapor lang="ts">
import { Pricing, type PricingPlan } from '@weft/ui'

const plans: PricingPlan[] = [
  {
    title: 'Starter',
    description: 'For side projects and small internal tools.',
    price: '$0',
    period: '/month',
    features: ['Unlimited pages', 'SQLite data store', 'Community support'],
    action: { label: 'Start for free', to: '/docs/getting-started/install' },
  },
  {
    title: 'Team',
    description: 'For product teams running customer-facing apps.',
    price: '$29',
    period: '/month',
    badge: 'Popular',
    highlight: true,
    features: ['Everything in Starter', 'Role permissions', 'Automated backups', 'Priority support'],
    action: { label: 'Start free trial', to: '/docs/getting-started/install' },
  },
]
</script>

<template>
  <Pricing
    headline="Pricing"
    title="Simple, predictable plans"
    description="Start free and upgrade when your team grows."
    :plans="plans"
  />
</template>
