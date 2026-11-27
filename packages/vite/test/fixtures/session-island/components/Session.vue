<script setup vapor lang="ts">
// Buttons and Guards by permission in an island: the server draws none of
// the buttons and only the Guards' loading (it has no session), hydration's
// first render draws the same, and once the session is known the browser
// draws what the user may see and press.
import { Button, Guard } from '@weft/ui'
</script>

<template>
  <div class="session">
    <Button label="Always" />
    <Button label="Edit" perm="products:edit" />
    <Button label="Users" perm="users:read" />
    <Guard perm="products:edit"><p class="stock">Stock</p></Guard>
    <Guard perm="users:read"><p class="people">People</p></Guard>
  </div>
</template>
