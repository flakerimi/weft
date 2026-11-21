<script setup vapor lang="ts">
// Every basic component in one island: server-rendered, then hydrated in
// place (basics-island.test.ts) and checked for mismatches.
import { ref } from 'vue'
import { Avatar, AvatarGroup, Badge, Button, Chip, Container, Kbd, Link, Progress, Separator, Skeleton, Spinner } from '@weft/ui'

const steps = ref(1)
const busy = ref(false)
// Bound, not static: vue-tsc 3.3.14 loses the script's bindings for the rest
// of a vapor template after a static component prop holding "//".
const vue = 'https://vuejs.org/'
</script>

<template>
  <Container>
    <Button label="Step" icon="plus" :loading="busy" @click="steps++" />
    <Button label="Busy" variant="outline" @click="busy = !busy" />
    <Link to="/">Home</Link>
    <Link :to="vue">Vue</Link>
    <Badge :label="steps" tone="success" variant="soft" />
    <AvatarGroup :max="2" aria-label="Team">
      <Avatar alt="Ada Lovelace" />
      <Avatar alt="Grace Hopper" />
      <Avatar />
    </AvatarGroup>
    <Chip :text="steps" tone="danger"><Kbd value="command" /></Chip>
    <Separator label="or" />
    <Progress :model-value="steps" :max="4" label="Steps" status />
    <Progress label="Waiting" />
    <Spinner label="Working" />
    <Skeleton width="4rem" />
  </Container>
</template>
