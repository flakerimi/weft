<script setup vapor lang="ts">
import { ref } from 'vue'
import Breadcrumb from '../../../src/navigation/Breadcrumb.vue'
import Pagination from '../../../src/navigation/Pagination.vue'
import WeftProvider from '../../../src/adaptive/WeftProvider.vue'
import type { Input } from '../../../src/adaptive/detect.ts'

const props = withDefaults(
  defineProps<{ total?: number; siblingCount?: number; showEdges?: boolean; links?: boolean; disabled?: boolean; input?: Input; dir?: 'ltr' | 'rtl' }>(),
  { total: 200, siblingCount: 1 },
)
const page = ref(5)
const crumbs = [
  { label: 'Home', to: '/', icon: 'house' },
  { label: 'Docs', to: '/docs' },
  { label: 'Components' },
  { label: 'Breadcrumb', to: '/docs/components/breadcrumb' },
]
const to = props.links ? (n: number) => `/list?page=${n}` : undefined
defineExpose({ page })
</script>

<template>
  <WeftProvider :input="input">
    <div :dir="dir">
      <Breadcrumb :items="crumbs" />
      <Pagination v-model:page="page" :total="total" :items-per-page="10" :sibling-count="siblingCount" :show-edges="showEdges" :to="to" :disabled="disabled" />
    </div>
  </WeftProvider>
</template>
