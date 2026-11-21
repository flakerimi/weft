<script setup vapor lang="ts">
import { Avatar, AvatarGroup } from '@weft/ui'

const team = ['Ada Lovelace', 'Grace Hopper', 'Alan Turing', 'Edsger Dijkstra', 'Barbara Liskov']
</script>

<template>
  <AvatarGroup :max="3" aria-label="Project members">
    <Avatar v-for="name in team" :key="name" :alt="name" />
  </AvatarGroup>
</template>
