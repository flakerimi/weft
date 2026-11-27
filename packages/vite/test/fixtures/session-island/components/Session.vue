<script setup vapor lang="ts">
// Buttons by permission in an island: the server draws none of them (it has
// no session), hydration's first render draws none either, and once the
// session is known the browser draws the ones the user may press.
import { Button } from '@weft/ui'
</script>

<template>
  <div class="session">
    <Button label="Always" />
    <Button label="Edit" perm="products:edit" />
    <Button label="Users" perm="users:read" />
  </div>
</template>
