<script setup vapor lang="ts">
import { Badge, Chip, Kbd } from '../../../src/index.ts'
defineProps<{ case: string }>()
</script>

<template>
  <Badge v-if="$props.case === 'badge-slot'" label="ignored"><b class="mine">New</b></Badge>
  <Kbd v-else-if="$props.case === 'kbd-slot'"><b class="mine">F1</b></Kbd>
  <Chip v-else-if="$props.case === 'chip'" text="3" tone="danger" size="lg" position="bottom-start" inset>
    <i class="anchor">bell</i>
  </Chip>
  <Chip v-else-if="$props.case === 'chip-label'" label="3 unread messages" text="3"><i class="anchor">bell</i></Chip>
  <Chip v-else-if="$props.case === 'chip-dot'"><i class="anchor">bell</i></Chip>
  <Chip v-else-if="$props.case === 'chip-hidden'" :show="false"><i class="anchor">bell</i></Chip>
</template>
