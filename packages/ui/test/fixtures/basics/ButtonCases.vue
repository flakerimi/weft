<script setup vapor lang="ts">
import { Button, WeftProvider } from '../../../src/index.ts'
import type { Input } from '../../../src/adaptive/detect.ts'
defineProps<{ case: string; input?: Input; loading?: boolean; onPress?: () => void; onSubmit?: () => void }>()
</script>

<template>
  <Button v-if="$props.case === 'slot'" label="ignored" icon="plus"><b class="mine">Add</b></Button>
  <WeftProvider v-else-if="$props.case === 'adaptive'" :input="input">
    <Button size="sm" label="Save"><i class="hit-probe" style="min-block-size: var(--weft-hit-min)"></i></Button>
  </WeftProvider>
  <form v-else-if="$props.case === 'form'" @submit.prevent="onSubmit?.()">
    <Button type="submit" label="Send" :loading="loading" @click="onPress?.()" />
  </form>
</template>
