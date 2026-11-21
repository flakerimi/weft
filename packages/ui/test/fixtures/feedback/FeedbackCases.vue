<script setup vapor lang="ts">
import { ref } from 'vue'
import Alert from '../../../src/feedback/Alert.vue'
import Banner from '../../../src/feedback/Banner.vue'
import EmptyState from '../../../src/feedback/EmptyState.vue'
import ErrorState from '../../../src/feedback/ErrorState.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

withDefaults(
  defineProps<{
    bannerId?: string
    bannerTo?: string
    loading?: boolean
    retrying?: boolean
    error?: unknown
    input?: Input
    orientation?: 'vertical' | 'horizontal'
    errorActions?: boolean
  }>(),
  { error: undefined },
)
const open = ref(true)
const bannerOpen = ref(true)
const log = ref<string[]>([])
defineExpose({ open, bannerOpen, log })
</script>

<template>
  <WeftProvider :input="input">
    <Alert
      v-model:open="open"
      title="Payment failed"
      description="The card was declined."
      tone="danger"
      icon="circle-alert"
      close
      :orientation="orientation"
      :actions="[{ label: 'Update card', onClick: () => log.push('update') }]"
    />
    <Alert class="plain" title="Heads up">
      <template #leading><span class="custom-leading">!</span></template>
      More <strong>text</strong>
    </Alert>
    <Banner
      v-model:open="bannerOpen"
      :id="bannerId"
      title="We are moving offices on Friday."
      :to="bannerTo"
      icon="info"
      close
      :actions="[{ label: 'Read more', to: '/news' }]"
      @close="log.push('banner-closed')"
    />
    <EmptyState
      title="No customers yet"
      description="Add one to get started."
      icon="users"
      :level="2"
      :loading="loading"
      :actions="[{ label: 'Add customer', onClick: () => log.push('add') }]"
    >
      <template #footer><a href="/help">Learn more</a></template>
    </EmptyState>
    <ErrorState :error="error" retry :loading="retrying" @retry="log.push('retry')">
      <template v-if="errorActions" #actions><button type="button" class="report">Report it</button></template>
    </ErrorState>
  </WeftProvider>
</template>
