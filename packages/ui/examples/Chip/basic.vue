<script setup vapor lang="ts">
import { Avatar, Chip, Icon, Stack } from '@weft/ui'
</script>

<template>
  <Stack direction="row" :gap="6" align="center">
    <Chip text="3" tone="danger" label="3 unread notifications">
      <Icon name="bell" size="lg" />
    </Chip>
    <Chip tone="success" position="bottom-end" inset label="Online">
      <Avatar alt="Grace Hopper" />
    </Chip>
  </Stack>
</template>
