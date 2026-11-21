<script setup vapor lang="ts">
import { Avatar, Stack } from '@weft/ui'

// An inline drawing, so the example needs no network.
const picture =
  'data:image/svg+xml,' +
  encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" fill="#6B7D2E"/><circle cx="16" cy="13" r="6" fill="#F8FAFC"/></svg>')
</script>

<template>
  <!-- An image, initials from alt, then an icon when there is neither. -->
  <Stack direction="row" :gap="2" align="center">
    <Avatar :src="picture" alt="Ada Lovelace" />
    <Avatar alt="Grace Hopper" />
    <Avatar />
  </Stack>
</template>
