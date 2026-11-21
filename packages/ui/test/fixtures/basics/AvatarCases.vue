<script setup vapor lang="ts">
import { Avatar, AvatarGroup } from '../../../src/index.ts'
import type { Size } from '../../../src/types.ts'
defineProps<{ case: string; max?: number; size?: Size; people?: string[] }>()
</script>

<template>
  <AvatarGroup v-if="$props.case === 'group'" :max="max" :size="size" aria-label="Team">
    <Avatar v-for="name in people" :key="name" :alt="name" />
  </AvatarGroup>
  <AvatarGroup v-else-if="$props.case === 'group-sized'" size="lg">
    <Avatar alt="Ada Lovelace" size="sm" />
  </AvatarGroup>
</template>
