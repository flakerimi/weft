<script setup vapor lang="ts">
import { Link, Stack } from '@weft/ui'
</script>

<template>
  <!-- In the browser a Link marks itself when it is the current page; active says so on the server too. -->
  <nav aria-label="Account">
    <Stack direction="row" :gap="4">
      <Link standalone to="/account" active>Profile</Link>
      <Link standalone to="/account/billing">Billing</Link>
      <Link standalone to="/account/team" disabled>Team</Link>
    </Stack>
  </nav>
</template>
