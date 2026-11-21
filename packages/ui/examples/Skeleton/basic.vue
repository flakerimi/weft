<script setup vapor lang="ts">
import { Skeleton, Stack } from '@weft/ui'
</script>

<template>
  <!-- The loading region says so with aria-busy; the skeletons are decorative. -->
  <Stack direction="row" :gap="3" align="center" aria-busy="true">
    <Skeleton circle width="40px" height="40px" />
    <Stack :gap="2" style="flex: 1">
      <Skeleton width="40%" />
      <Skeleton width="80%" />
    </Stack>
  </Stack>
</template>
