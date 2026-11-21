<script setup vapor lang="ts">
import { Button, Separator, Stack } from '@weft/ui'
</script>

<template>
  <Stack :gap="3">
    <Button block label="Continue with email" icon="mail" />
    <Separator label="or" />
    <Button block label="Continue with a passkey" variant="outline" />
  </Stack>
</template>
