<script setup vapor lang="ts">
// Avatars overlapping in a row, such as the people on a project. With `max`,
// the first `max` show and a last avatar counts the rest, drawn "+3" and
// said "3 more". Name the
// group with an aria-label ("Project members").
import { provideAvatarGroup } from './avatarGroup.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/avatar.css'
import '../styles/avatar-group.css'

const props = withDefaults(
  defineProps<{
    /** How many avatars to show; the rest are counted. Unset or 0 shows all. */
    max?: number
    /** The size of every avatar in the group. */
    size?: Size
  }>(),
  { size: 'md' },
)

const more = provideAvatarGroup(
  () => props.size,
  () => props.max,
)
</script>

<template>
  <div class="weft-avatar-group" role="group" :data-size="size">
    <slot />
    <span v-if="more > 0" class="weft-avatar weft-avatar-group__more" :data-size="size">
      <span aria-hidden="true">+{{ more }}</span>
      <span class="weft-visually-hidden">{{ more }} more</span>
    </span>
  </div>
</template>
