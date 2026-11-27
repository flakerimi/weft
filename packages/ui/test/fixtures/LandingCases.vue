<script setup vapor lang="ts">
import { ref } from 'vue'
import {
  Cta,
  Faq,
  FeatureGrid,
  Hero,
  Pricing,
  Testimonials,
  WeftProvider,
  type FaqItem,
  type FeatureItem,
  type PricingCycle,
  type PricingPlan,
  type TestimonialItem,
} from '../../src/index.ts'

defineProps<{ case: string }>()

const clicks = ref<string[]>([])
const cycle = ref('monthly')
const openFaq = ref<string | string[] | null>(null)

const features: FeatureItem[] = [
  { icon: 'zap', title: 'Fast', description: 'Vapor DOM', to: '/fast', linkLabel: 'Learn more' },
  { icon: 'shield', title: 'Typed', description: 'End to end', to: '/typed' },
  { title: 'Custom', slot: 'special' },
]

const quotes: TestimonialItem[] = [
  { quote: 'One binary changed our deploys.', author: 'Ada Lovelace', role: 'CTO', to: '/ada' },
  { quote: 'Zero drift between Go and Vue.', author: 'Grace Hopper', avatar: 'https://example.com/grace.png' },
]

const cycles: PricingCycle[] = [
  { label: 'Monthly', value: 'monthly' },
  { label: 'Yearly', value: 'yearly', badge: 'Save 20%' },
]

const plans: PricingPlan[] = [
  {
    title: 'Starter',
    description: 'For individuals',
    price: '$0',
    period: '/mo',
    features: ['1 project', 'SQLite'],
    action: { label: 'Start free', onClick: () => clicks.value.push('starter') },
  },
  {
    title: 'Pro',
    description: 'For teams',
    price: { monthly: '$29', yearly: '$24' },
    period: { monthly: '/mo', yearly: '/mo' },
    billingNote: { yearly: 'Billed $288 yearly' },
    badge: 'Popular',
    highlight: true,
    features: ['Unlimited projects', 'Roles'],
    action: { label: 'Go Pro', onClick: () => clicks.value.push('pro') },
  },
]

const missingCyclePlans: PricingPlan[] = [
  {
    title: 'AnnualOnly',
    price: { yearly: '$99' },
    period: { yearly: '/yr' },
  },
]

const missingCycleProdPlans: PricingPlan[] = [
  {
    title: 'AnnualOnlyProd',
    price: { yearly: '$199' },
    period: { yearly: '/yr' },
  },
]

const faqs: FaqItem[] = [
  { value: 'js', label: 'Does it ship JS?', content: 'Only for islands.' },
  { value: 'db', label: 'Which DB?', content: 'SQLite today.', slot: 'custom-db' },
]

defineExpose({ clicks, cycle, openFaq })
</script>

<template>
  <Hero
    v-if="$props.case === 'hero-default'"
    headline="Pre-alpha"
    title="One UI system"
    description="Go and Vue in one binary."
    :actions="[
      { label: 'Install', onClick: () => clicks.push('install') },
      { label: 'Docs', to: '/docs' },
    ]"
  />
  <WeftProvider v-else-if="$props.case === 'hero-inset'" inset>
    <Hero title="Inset hero" description="Inside gallery preview" />
  </WeftProvider>
  <Hero v-else-if="$props.case === 'hero-slot-vertical'" title="Still vertical">
    <div class="visual">Preview frame</div>
  </Hero>
  <Hero
    v-else-if="$props.case === 'hero-horizontal'"
    orientation="horizontal"
    :level="3"
    title="Side by side"
  >
    <template #background><span class="bg-glow" /></template>
    <template #headline><b class="custom-headline">New</b></template>
    <template #caption><span class="cap">v0.1.0</span></template>
    <div class="visual">Visual</div>
  </Hero>
  <div v-else-if="$props.case === 'all-blocks-page'" class="full-landing">
    <Hero title="Main page heading" />
    <FeatureGrid title="Capabilities" :items="features" />
    <Pricing title="Plans" :plans="plans" />
    <Testimonials title="What people say" :items="quotes" />
    <Faq title="Questions" :items="faqs" />
    <Cta title="Ready to build?" />
  </div>
  <Cta
    v-else-if="$props.case === 'cta-default'"
    headline="Next step"
    title="Ship one binary"
    description="Try Weft today."
    :actions="[
      { label: 'Create app', onClick: () => clicks.push('create') },
      { label: 'Read docs', to: '/docs' },
    ]"
  />
  <Cta
    v-else-if="$props.case === 'cta-solid-horizontal'"
    variant="solid"
    tone="danger"
    orientation="horizontal"
    :level="3"
    title="Solid band"
    :actions="[{ label: 'Act now' }]"
  />
  <Cta
    v-else-if="$props.case === 'cta-aside'"
    orientation="horizontal"
    title="With aside"
    :actions="[{ label: 'Start' }]"
  >
    <template #aside><div class="loom">Loom</div></template>
  </Cta>
  <FeatureGrid
    v-else-if="$props.case === 'features-default'"
    headline="Why Weft"
    title="Built together"
    description="Four pieces."
    :items="features"
  >
    <template #actions><button class="hdr-act">All</button></template>
    <template #special="{ item }"><div class="special-slot">{{ item.title }}!</div></template>
  </FeatureGrid>
  <FeatureGrid
    v-else-if="$props.case === 'features-plain'"
    variant="plain"
    align="center"
    :level="3"
    title="Plain features"
    :items="features"
  />
  <Testimonials
    v-else-if="$props.case === 'testimonials-default'"
    headline="Voices"
    title="Loved by engineers"
    description="Real quotes."
    :items="quotes"
  />
  <Pricing
    v-else-if="$props.case === 'pricing-cycles'"
    v-model:cycle="cycle"
    headline="Pricing"
    title="Pick a plan"
    description="Cancel anytime."
    :cycles="cycles"
    :plans="plans"
  >
    <template #footer><span class="price-foot">All prices in USD</span></template>
  </Pricing>
  <Pricing
    v-else-if="$props.case === 'pricing-missing'"
    v-model:cycle="cycle"
    title="Missing cycle test"
    :cycles="cycles"
    :plans="missingCyclePlans"
  />
  <Pricing
    v-else-if="$props.case === 'pricing-missing-prod'"
    v-model:cycle="cycle"
    title="Missing cycle prod test"
    :cycles="cycles"
    :plans="missingCycleProdPlans"
  />
  <Faq
    v-else-if="$props.case === 'faq-default'"
    v-model="openFaq"
    headline="Support"
    title="Frequently asked"
    description="Answers inside."
    :items="faqs"
  >
    <template #aside><span class="faq-contact">Email us</span></template>
    <template #custom-db="{ item }"><b class="db-slot">{{ item.content }}</b></template>
  </Faq>
  <Faq
    v-else-if="$props.case === 'faq-horizontal'"
    orientation="horizontal"
    type="multiple"
    :default-value="['js']"
    :level="3"
    title="Split FAQ"
    :items="faqs"
  />
</template>
