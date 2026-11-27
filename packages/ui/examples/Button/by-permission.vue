<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <!-- Edit is drawn only for a signed-in user with products:edit; with no one signed in, as here, only View is. -->
  <Stack direction="row" :gap="2">
    <Button label="View" variant="outline" />
    <Button label="Edit" icon="pencil" perm="products:edit" />
  </Stack>
</template>
