<script setup vapor lang="ts">
import { Button, Stack } from '@weft/ui'
</script>

<template>
  <!--
    Edit and Delete are drawn only for a signed-in user with products:edit and products:delete.
    The gallery's demo user may edit products, not delete them, so it sees View and Edit.
  -->
  <Stack direction="row" :gap="2">
    <Button label="View" variant="outline" />
    <Button label="Edit" icon="pencil" perm="products:edit" />
    <Button label="Delete" icon="trash" tone="danger" variant="soft" perm="products:delete" />
  </Stack>
</template>
