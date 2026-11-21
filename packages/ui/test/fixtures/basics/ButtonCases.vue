<script setup vapor lang="ts">
import { Button, WeftProvider } from '../../../src/index.ts'
import type { Input } from '../../../src/adaptive/detect.ts'
import type { Size } from '../../../src/types.ts'
defineProps<{ case: string; input?: Input; size?: Size; loading?: boolean; onPress?: () => void; onSubmit?: () => void }>()
</script>

<template>
  <Button v-if="$props.case === 'slot'" label="ignored" icon="plus"><b class="mine">Add</b></Button>
  <Button v-else-if="$props.case === 'ends'" label="Inbox" icon="plus" trailing-icon="arrow-right">
    <template #leading><i class="lead">L</i></template>
    <template #trailing><i class="trail">3</i></template>
  </Button>
  <WeftProvider v-else-if="$props.case === 'adaptive'" :input="input">
    <Button class="plain" :size="size" label="Save" />
    <Button class="square" :size="size" icon="plus" aria-label="Add" />
    <!-- Resolves --weft-hit-min where the buttons and their ::before do. -->
    <i class="hit-probe" style="min-block-size: var(--weft-hit-min)"></i>
  </WeftProvider>
  <form v-else-if="$props.case === 'form'" @submit.prevent="onSubmit?.()">
    <Button type="submit" label="Send" :loading="loading" @click="onPress?.()" />
  </form>
</template>
