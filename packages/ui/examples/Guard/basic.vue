<script setup vapor lang="ts">
import { Guard, Stack } from '@weft/ui'
</script>

<template>
  <!--
    Each Guard shows what it holds only to a signed-in user with its permission, and sends a visitor
    with no session to /login and back. The gallery's demo user may edit products, not read users.
  -->
  <Stack :gap="3">
    <Guard perm="products:edit">
      <template #default="{ user }">
        <p>{{ user.name }} may edit products.</p>
      </template>
    </Guard>
    <Guard perm="users:read">
      <p>The users of this app.</p>
    </Guard>
  </Stack>
</template>
