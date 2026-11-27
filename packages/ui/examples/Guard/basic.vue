<script setup vapor lang="ts">
import { Guard } from '@weft/ui'
</script>

<template>
  <!--
    Shown only to a signed-in user with users:read. A visitor with no session is sent to /login and
    back; here, where no API answers, it says so.
  -->
  <Guard perm="users:read">
    <template #default="{ user }">
      <p>Signed in as {{ user.email }}.</p>
    </template>
  </Guard>
</template>
